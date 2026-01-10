from __future__ import annotations

from contextlib import contextmanager
from typing import Iterator, Optional, TYPE_CHECKING

if TYPE_CHECKING:  # pragma: no cover
    from playwright.sync_api import Browser, BrowserContext, Page, Playwright


@contextmanager
def connect_over_cdp(ws_endpoint: str) -> Iterator[tuple["Playwright", "Browser"]]:
    try:
        from playwright.sync_api import sync_playwright  # type: ignore
    except Exception as exc:  # pragma: no cover
        raise RuntimeError("Playwright is not available; launch via Nix so Playwright/driver/browsers are present.") from exc

    playwright = sync_playwright().start()
    try:
        browser = playwright.chromium.connect_over_cdp(ws_endpoint)
        yield playwright, browser
    finally:
        try:
            browser.close()
        except Exception:
            pass
        try:
            playwright.stop()
        except Exception:
            pass


def find_page_by_target_id(browser: "Browser", target_id: str) -> Optional["Page"]:
    for context in browser.contexts:
        page = _find_in_context(context, target_id)
        if page is not None:
            return page
    return None


def _find_in_context(context: "BrowserContext", target_id: str) -> Optional["Page"]:
    for page in context.pages:
        session = None
        try:
            session = context.new_cdp_session(page)
            info = session.send("Target.getTargetInfo")
            ti = info.get("targetInfo") if isinstance(info, dict) else None
            tid = ti.get("targetId") if isinstance(ti, dict) else None
            if tid == target_id:
                return page
        except Exception:
            continue
        finally:
            if session is not None:
                try:
                    session.detach()
                except Exception:
                    pass
    return None

