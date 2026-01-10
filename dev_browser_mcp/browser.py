import json
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Optional, TYPE_CHECKING

from .chromium_flags import chromium_launch_args
from .paths import platform_cache_dir, platform_state_dir, safe_artifact_path

if TYPE_CHECKING:  # pragma: no cover
    from playwright.sync_api import BrowserContext, Page, Playwright


INTERACTIVE_ROLES: set[str] = {
    "button",
    "checkbox",
    "combobox",
    "link",
    "listbox",
    "menuitem",
    "option",
    "radio",
    "searchbox",
    "slider",
    "spinbutton",
    "switch",
    "tab",
    "textbox",
    "treeitem",
}


@dataclass(frozen=True)
class RefTarget:
    role: str
    name: Optional[str]
    nth: int


def extract_refs(
    ax: Optional[dict[str, Any]],
    *,
    interactive_only: bool,
    max_items: int,
) -> tuple[list[dict[str, Any]], dict[str, RefTarget], bool]:
    if not ax:
        return [], {}, False

    items: list[dict[str, Any]] = []
    refs: dict[str, RefTarget] = {}
    counts: dict[tuple[str, str], int] = {}
    truncated = False

    def walk(node: dict[str, Any]) -> None:
        nonlocal truncated
        if len(items) >= max_items:
            truncated = True
            return

        role = node.get("role")
        name = node.get("name")
        value = node.get("value")

        if isinstance(role, str):
            role_norm = role.strip().lower()
            is_interactive = role_norm in INTERACTIVE_ROLES
            if is_interactive or not interactive_only:
                name_str = name if isinstance(name, str) and name.strip() else ""
                key = (role_norm, name_str)
                nth = counts.get(key, 0)
                counts[key] = nth + 1

                ref = f"e{len(items) + 1}"
                refs[ref] = RefTarget(role=role_norm, name=name_str or None, nth=nth)

                item: dict[str, Any] = {"ref": ref, "role": role_norm}
                if name_str:
                    item["name"] = name_str
                if isinstance(value, str) and value:
                    item["value"] = value
                items.append(item)

        children = node.get("children")
        if isinstance(children, list):
            for child in children:
                if not isinstance(child, dict):
                    continue
                walk(child)
                if truncated:
                    return

    walk(ax)
    return items, refs, truncated


def format_snapshot(items: list[dict[str, Any]], *, max_chars: int, truncated: bool) -> str:
    lines: list[str] = []
    for item in items:
        ref = item["ref"]
        role = item.get("role", "unknown")
        name = item.get("name")
        value = item.get("value")
        parts = [f"[{ref}] {role}"]
        if isinstance(name, str) and name:
            parts.append(f"name={json.dumps(name)}")
        if isinstance(value, str) and value:
            parts.append(f"value={json.dumps(value)}")
        lines.append(" ".join(parts))

    if truncated:
        lines.append("[…] truncated (max_items reached)")

    text = "\n".join(lines)
    if len(text) > max_chars:
        return text[: max(0, max_chars - 24)] + "\n[… ] truncated (max_chars)"
    return text


class BrowserManager:
    def __init__(self, *, profile: str, headless: bool) -> None:
        self._profile = profile
        self._headless = headless

        self._playwright: Optional["Playwright"] = None
        self._context: Optional["BrowserContext"] = None

        self._pages: dict[str, "Page"] = {}
        self._refs: dict[str, dict[str, RefTarget]] = {}

        base_state = platform_state_dir() / "dev-browser-mcp" / profile
        self._user_data_dir = base_state / "chromium-profile"

        base_cache = platform_cache_dir() / "dev-browser-mcp" / profile
        self._artifact_dir = base_cache / "artifacts"

    def close(self) -> None:
        for page in list(self._pages.values()):
            try:
                if not page.is_closed():
                    page.close()
            except Exception:
                pass

        self._pages.clear()
        self._refs.clear()

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

    def ensure_started(self) -> None:
        if self._context is not None:
            return

        try:
            from playwright.sync_api import sync_playwright  # type: ignore
        except Exception as exc:  # pragma: no cover
            raise RuntimeError(
                "Playwright is not available. Launch the server via Nix so Playwright, the driver, and browsers are present."
            ) from exc

        self._user_data_dir.mkdir(parents=True, exist_ok=True)

        self._playwright = sync_playwright().start()
        self._context = self._playwright.chromium.launch_persistent_context(
            user_data_dir=str(self._user_data_dir),
            headless=self._headless,
            ignore_https_errors=True,
            accept_downloads=True,
            args=chromium_launch_args(),
        )
        self._context.set_default_timeout(15_000)

        pages = list(self._context.pages)
        main = pages[0] if pages else self._context.new_page()
        for extra in pages[1:]:
            try:
                extra.close()
            except Exception:
                pass

        self._pages["main"] = main

    def get_page(self, name: str) -> "Page":
        self.ensure_started()

        page = self._pages.get(name)
        if page is not None and not page.is_closed():
            return page

        assert self._context is not None
        page = self._context.new_page()
        self._pages[name] = page
        return page

    def list_pages(self) -> list[dict[str, Any]]:
        out: list[dict[str, Any]] = []
        for name, page in self._pages.items():
            if page.is_closed():
                continue
            try:
                title = page.title()
            except Exception:
                title = ""
            out.append({"name": name, "url": page.url, "title": title})
        return out

    def close_page(self, name: str) -> None:
        page = self._pages.get(name)
        if page is None:
            return
        try:
            if not page.is_closed():
                page.close()
        finally:
            self._pages.pop(name, None)
            self._refs.pop(name, None)

    def artifact_path(self, *, path_arg: Optional[str], default_name: str) -> Path:
        return safe_artifact_path(self._artifact_dir, path_arg, default_name)

    def set_refs(self, page_name: str, refs: dict[str, RefTarget]) -> None:
        self._refs[page_name] = refs

    def get_ref(self, page_name: str, ref: str) -> RefTarget:
        page_refs = self._refs.get(page_name)
        if not page_refs or ref not in page_refs:
            raise KeyError(f"Unknown ref '{ref}' for page '{page_name}'. Run snapshot first.")
        return page_refs[ref]
