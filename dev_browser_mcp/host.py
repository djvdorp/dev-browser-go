import json
import time
import urllib.request
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Optional, TYPE_CHECKING

from .chromium_flags import DEFAULT_WINDOW_SIZE, chromium_launch_args, window_size_from_env
from .paths import platform_state_dir
from .snapshot_base_script import get_base_script

if TYPE_CHECKING:  # pragma: no cover
    from playwright.sync_api import BrowserContext, Page, Playwright


@dataclass(frozen=True)
class PageEntry:
    name: str
    target_id: str


def _fetch_json(url: str, *, timeout_s: float) -> dict:
    with urllib.request.urlopen(url, timeout=timeout_s) as resp:
        raw = resp.read()
    data = json.loads(raw.decode("utf-8"))
    if not isinstance(data, dict):
        raise ValueError("Expected JSON object")
    return data


def _get_ws_endpoint(cdp_port: int, *, timeout_s: float = 10.0) -> str:
    url = f"http://127.0.0.1:{cdp_port}/json/version"
    deadline = time.time() + timeout_s
    last_error: Optional[Exception] = None
    while time.time() < deadline:
        try:
            data = _fetch_json(url, timeout_s=1.5)
            ws = data.get("webSocketDebuggerUrl")
            if isinstance(ws, str) and ws:
                return ws
        except Exception as exc:  # noqa: BLE001
            last_error = exc
        time.sleep(0.2)
    raise RuntimeError(f"Timed out waiting for Chromium CDP endpoint at {url}: {last_error}")


class BrowserHost:
    def __init__(self, *, profile: str, headless: bool, cdp_port: int) -> None:
        self.profile = profile
        self.headless = headless
        self.cdp_port = cdp_port

        self._playwright: Optional["Playwright"] = None
        self._context: Optional["BrowserContext"] = None
        self._ws_endpoint: Optional[str] = None

        self._registry: dict[str, tuple["Page", str]] = {}

        base_state = platform_state_dir() / "dev-browser-mcp" / profile
        self._user_data_dir = base_state / "chromium-profile"

    @property
    def ws_endpoint(self) -> str:
        if not self._ws_endpoint:
            raise RuntimeError("Host not started")
        return self._ws_endpoint

    @property
    def user_data_dir(self) -> Path:
        return self._user_data_dir

    def start(self) -> None:
        if self._context is not None:
            return

        try:
            from playwright.sync_api import sync_playwright  # type: ignore
        except Exception as exc:  # pragma: no cover
            raise RuntimeError("Playwright is not available; launch via Nix so Playwright/driver/browsers are present.") from exc

        self._user_data_dir.mkdir(parents=True, exist_ok=True)

        self._playwright = sync_playwright().start()
        context_kwargs: dict[str, Any] = {}
        window_size = window_size_from_env(default=DEFAULT_WINDOW_SIZE)
        if window_size is not None:
            width, height = window_size
            context_kwargs["viewport"] = {"width": width, "height": height}
            context_kwargs["screen"] = {"width": width, "height": height}

        self._context = self._playwright.chromium.launch_persistent_context(
            user_data_dir=str(self._user_data_dir),
            headless=self.headless,
            ignore_https_errors=True,
            accept_downloads=True,
            args=chromium_launch_args(cdp_port=self.cdp_port),
            **context_kwargs,
        )
        self._context.set_default_timeout(15_000)
        self._context.add_init_script(get_base_script())

        self._ws_endpoint = _get_ws_endpoint(self.cdp_port)

        pages = list(self._context.pages)
        if not pages:
            pages = [self._context.new_page()]
        self._register_page("main", pages[0])
        for extra in pages[1:]:
            try:
                extra.close()
            except Exception:
                pass

    def stop(self) -> None:
        for name in list(self._registry.keys()):
            try:
                self.close_page(name)
            except Exception:
                pass
        self._registry.clear()

        try:
            if self._context is not None:
                self._context.close()
        except Exception:
            pass
        self._context = None

        try:
            if self._playwright is not None:
                self._playwright.stop()
        except Exception:
            pass
        self._playwright = None
        self._ws_endpoint = None

    def list_pages(self) -> list[str]:
        return [name for name, (page, _tid) in self._registry.items() if not page.is_closed()]

    def close_page(self, name: str) -> bool:
        entry = self._registry.get(name)
        if not entry:
            return False
        page, _tid = entry
        try:
            if not page.is_closed():
                page.close()
        finally:
            self._registry.pop(name, None)
        return True

    def get_or_create_page(self, name: str) -> PageEntry:
        self.start()
        assert self._context is not None

        entry = self._registry.get(name)
        if entry and not entry[0].is_closed():
            return PageEntry(name=name, target_id=entry[1])

        page = self._context.new_page()
        target_id = self._get_target_id(page)
        self._registry[name] = (page, target_id)
        page.on("close", lambda: self._registry.pop(name, None))
        return PageEntry(name=name, target_id=target_id)

    def _register_page(self, name: str, page: "Page") -> None:
        target_id = self._get_target_id(page)
        self._registry[name] = (page, target_id)
        page.on("close", lambda: self._registry.pop(name, None))

    def _get_target_id(self, page: "Page") -> str:
        assert self._context is not None
        session = self._context.new_cdp_session(page)
        try:
            info = session.send("Target.getTargetInfo")
        finally:
            try:
                session.detach()
            except Exception:
                pass
        target_info = info.get("targetInfo") if isinstance(info, dict) else None
        target_id = target_info.get("targetId") if isinstance(target_info, dict) else None
        if not isinstance(target_id, str) or not target_id:
            raise RuntimeError("Failed to resolve page targetId")
        return target_id
