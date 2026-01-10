from __future__ import annotations

from typing import Any, TYPE_CHECKING

from .snapshot_base_script import get_base_script
from .snapshot_script import get_aria_script

if TYPE_CHECKING:  # pragma: no cover
    from playwright.sync_api import ElementHandle, Page


def ensure_injected(page: "Page", *, engine: str = "simple") -> None:
    try:
        present = page.evaluate("() => Boolean(globalThis.__devBrowser_getAISnapshot)")
    except Exception:
        present = False
    if not present:
        page.evaluate(get_base_script())

    if engine.lower() == "aria":
        try:
            aria_present = page.evaluate("() => Boolean(globalThis.__devBrowser_getAISnapshotAria)")
        except Exception:
            aria_present = False
        if not aria_present:
            page.evaluate(get_aria_script())


def get_snapshot(
    page: "Page",
    *,
    engine: str = "simple",
    format: str = "list",
    interactive_only: bool = True,
    include_headings: bool = True,
    max_items: int = 80,
    max_chars: int = 8000,
) -> dict[str, Any]:
    ensure_injected(page, engine=engine)
    return page.evaluate(
        "(opts) => globalThis.__devBrowser_getAISnapshot(opts)",
        {
            "engine": str(engine),
            "format": str(format),
            "interactiveOnly": bool(interactive_only),
            "includeHeadings": bool(include_headings),
            "maxItems": int(max_items),
            "maxChars": int(max_chars),
        },
    )


def select_ref(page: "Page", ref: str, *, engine: str = "simple") -> "ElementHandle":
    ensure_injected(page, engine=engine)
    handle = page.evaluate_handle("(ref) => globalThis.__devBrowser_selectSnapshotRef(ref)", ref)
    element = handle.as_element()
    if element is None:
        handle.dispose()
        raise RuntimeError(f"Ref '{ref}' did not resolve to an element")
    return element


def draw_ref_overlay(page: "Page", *, max_refs: int = 80, engine: str = "simple") -> dict[str, Any]:
    ensure_injected(page, engine=engine)
    return page.evaluate("(opts) => globalThis.__devBrowser_drawRefOverlay(opts)", {"maxRefs": int(max_refs)})


def clear_ref_overlay(page: "Page", *, engine: str = "simple") -> None:
    ensure_injected(page, engine=engine)
    page.evaluate("() => globalThis.__devBrowser_clearRefOverlay()")
