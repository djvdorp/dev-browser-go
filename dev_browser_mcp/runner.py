from __future__ import annotations

import time
from contextlib import contextmanager
from pathlib import Path
from typing import Any, Iterator, Optional, TYPE_CHECKING

from .cdp import connect_over_cdp, find_page_by_target_id
from .paths import now_ms, safe_artifact_path
from .snapshot_refs import clear_ref_overlay, draw_ref_overlay, get_snapshot, select_ref

if TYPE_CHECKING:  # pragma: no cover
    from playwright.sync_api import Browser, BrowserContext, Page


@contextmanager
def open_page(ws_endpoint: str, target_id: str) -> Iterator[tuple["Browser", "BrowserContext", "Page"]]:
    with connect_over_cdp(ws_endpoint) as (_pw, browser):
        page = find_page_by_target_id(browser, target_id)
        if page is None:
            raise RuntimeError(f"Page with targetId={target_id} not found")
        yield browser, page.context, page


def run_call(page: "Page", name: str, args: dict[str, Any], *, artifact_dir: Path) -> dict[str, Any]:
    if name == "goto":
        url = _require_str(args, "url")
        wait_until = args.get("wait_until", "domcontentloaded")
        if not isinstance(wait_until, str) or not wait_until:
            raise ValueError("Expected string 'wait_until'")
        timeout_ms = _optional_int(args, "timeout_ms", 45_000)
        page.goto(url, wait_until=wait_until, timeout=timeout_ms)
        return {"url": page.url, "title": _safe_title(page)}

    if name == "snapshot":
        engine = args.get("engine", "simple")
        if not isinstance(engine, str) or not engine:
            raise ValueError("Expected string 'engine'")
        format = args.get("format", "list")
        if not isinstance(format, str) or not format:
            raise ValueError("Expected string 'format'")
        interactive_only = _optional_bool(args, "interactive_only", True)
        include_headings = _optional_bool(args, "include_headings", True)
        max_items = _optional_int(args, "max_items", 80)
        max_chars = _optional_int(args, "max_chars", 8000)
        snap = get_snapshot(
            page,
            engine=engine,
            format=format,
            interactive_only=interactive_only,
            include_headings=include_headings,
            max_items=max_items,
            max_chars=max_chars,
        )
        yaml = snap.get("yaml") if isinstance(snap, dict) else None
        items = snap.get("items") if isinstance(snap, dict) else None
        return {
            "url": page.url,
            "title": _safe_title(page),
            "engine": engine,
            "format": format,
            "snapshot": yaml or "",
            "items": items or [],
        }

    if name == "click_ref":
        ref = _require_str(args, "ref")
        timeout_ms = _optional_int(args, "timeout_ms", 15_000)
        element = select_ref(page, ref)
        try:
            element.click(timeout=timeout_ms)
        finally:
            try:
                element.dispose()
            except Exception:
                pass
        return {"ref": ref, "clicked": True}

    if name == "fill_ref":
        ref = _require_str(args, "ref")
        text = _require_str(args, "text")
        timeout_ms = _optional_int(args, "timeout_ms", 15_000)
        element = select_ref(page, ref)
        try:
            element.fill(text, timeout=timeout_ms)
        finally:
            try:
                element.dispose()
            except Exception:
                pass
        return {"ref": ref, "filled": True}

    if name == "press":
        key = _require_str(args, "key")
        page.keyboard.press(key)
        return {"key": key, "pressed": True}

    if name == "wait":
        strategy = args.get("strategy", "playwright")
        if not isinstance(strategy, str) or not strategy:
            raise ValueError("Expected non-empty string 'strategy'")
        state = args.get("state", "load")
        if not isinstance(state, str) or not state:
            raise ValueError("Expected non-empty string 'state'")
        allowed = {"load", "domcontentloaded", "networkidle", "commit"}
        if state not in allowed:
            raise ValueError(f"Invalid state '{state}' (expected one of: {', '.join(sorted(allowed))})")
        timeout_ms = _optional_int(args, "timeout_ms", 10_000)
        min_wait_ms = _optional_int(args, "min_wait_ms", 0)

        if strategy == "playwright":
            start = time.monotonic()
            if min_wait_ms:
                page.wait_for_timeout(min_wait_ms)

            timed_out = False
            try:
                page.wait_for_load_state(state=state, timeout=timeout_ms)
            except Exception as exc:  # noqa: BLE001
                if exc.__class__.__name__ == "TimeoutError":
                    timed_out = True
                else:
                    raise

            waited_ms = int((time.monotonic() - start) * 1000)
            ready_state = ""
            try:
                rs = page.evaluate("() => document.readyState")
                if isinstance(rs, str):
                    ready_state = rs
            except Exception:
                pass

            return {
                "ok": not timed_out,
                "strategy": strategy,
                "state": state,
                "timed_out": timed_out,
                "waited_ms": waited_ms,
                "ready_state": ready_state,
            }

        if strategy == "perf":
            result = _wait_perf(page, state=state, timeout_ms=timeout_ms, min_wait_ms=min_wait_ms)
            result["strategy"] = strategy
            result["state"] = state
            return result

        raise ValueError("Invalid strategy (expected 'playwright' or 'perf')")

    if name == "screenshot":
        path_arg = _optional_str(args, "path")
        full_page = _optional_bool(args, "full_page", True)
        annotate_refs = _optional_bool(args, "annotate_refs", False)
        path = safe_artifact_path(artifact_dir, path_arg, f"screenshot-{now_ms()}.png")
        if annotate_refs:
            draw_ref_overlay(page)
            page.wait_for_timeout(50)
        try:
            page.screenshot(path=str(path), full_page=full_page)
        finally:
            if annotate_refs:
                try:
                    clear_ref_overlay(page)
                except Exception:
                    pass
        return {"path": str(path)}

    if name == "save_html":
        path_arg = _optional_str(args, "path")
        path = safe_artifact_path(artifact_dir, path_arg, f"page-{now_ms()}.html")
        path.write_text(page.content(), encoding="utf-8")
        return {"path": str(path)}

    raise ValueError(f"Unknown call '{name}'")


def run_actions(page: "Page", calls: list[dict[str, Any]], *, artifact_dir: Path) -> list[dict[str, Any]]:
    results: list[dict[str, Any]] = []
    for call in calls:
        if not isinstance(call, dict):
            raise ValueError("Each call must be an object")
        name = call.get("name")
        if not isinstance(name, str) or not name:
            raise ValueError("Each call must include non-empty string 'name'")
        args = call.get("arguments", {})
        if args is None:
            args = {}
        if not isinstance(args, dict):
            raise ValueError("Call 'arguments' must be an object")
        results.append({"name": name, "result": run_call(page, name, args, artifact_dir=artifact_dir)})
    return results


def _safe_title(page: "Page") -> str:
    try:
        return page.title()
    except Exception:
        return ""


def _require_str(args: dict[str, Any], key: str) -> str:
    value = args.get(key)
    if not isinstance(value, str) or not value:
        raise ValueError(f"Expected non-empty string '{key}'")
    return value


def _optional_str(args: dict[str, Any], key: str) -> Optional[str]:
    value = args.get(key)
    if value is None:
        return None
    if not isinstance(value, str):
        raise ValueError(f"Expected string '{key}'")
    return value


def _optional_bool(args: dict[str, Any], key: str, default: bool) -> bool:
    value = args.get(key, default)
    if isinstance(value, bool):
        return value
    raise ValueError(f"Expected boolean '{key}'")


def _optional_int(args: dict[str, Any], key: str, default: int) -> int:
    value = args.get(key, default)
    if isinstance(value, int) and value >= 0:
        return value
    raise ValueError(f"Expected non-negative integer '{key}'")


_PERF_LOAD_STATE_JS = r"""
() => {
  const doc = globalThis.document;
  const perf = globalThis.performance;
  const readyState = doc && typeof doc.readyState === "string" ? doc.readyState : "unknown";
  if (!perf || typeof perf.getEntriesByType !== "function" || typeof perf.now !== "function") {
    return { readyState, pendingRequests: 0 };
  }

  const now = perf.now();
  const resources = perf.getEntriesByType("resource") || [];

  const adPatterns = [
    "doubleclick.net",
    "googlesyndication.com",
    "googletagmanager.com",
    "google-analytics.com",
    "facebook.net",
    "connect.facebook.net",
    "analytics",
    "ads",
    "tracking",
    "pixel",
    "hotjar.com",
    "clarity.ms",
    "mixpanel.com",
    "segment.com",
    "newrelic.com",
    "nr-data.net",
    "/tracker/",
    "/collector/",
    "/beacon/",
    "/telemetry/",
    "/log/",
    "/events/",
    "/track.",
    "/metrics/",
  ];

  const nonCriticalTypes = ["img", "image", "icon", "font"];

  let pending = 0;
  for (const entry of resources) {
    if (!entry || entry.responseEnd !== 0) continue;
    const url = String(entry.name || "");

    if (!url || url.startsWith("data:") || url.length > 500) continue;
    if (adPatterns.some((p) => url.includes(p))) continue;

    const loadingDuration = now - (entry.startTime || 0);
    if (loadingDuration > 10000) continue;

    const resourceType = String(entry.initiatorType || "unknown");
    if (nonCriticalTypes.includes(resourceType) && loadingDuration > 3000) continue;

    const isImageUrl = /\.(jpg|jpeg|png|gif|webp|svg|ico)(\\?|$)/i.test(url);
    if (isImageUrl && loadingDuration > 3000) continue;

    pending++;
  }
  return { readyState, pendingRequests: pending };
}
"""


def _ready_state_satisfies(ready_state: str, state: str) -> bool:
    rs = ready_state.lower()
    if state in {"domcontentloaded", "commit"}:
        return rs in {"interactive", "complete"}
    return rs == "complete"


def _wait_perf(page: "Page", *, state: str, timeout_ms: int, min_wait_ms: int) -> dict[str, Any]:
    poll_interval_ms = 50

    start = time.monotonic()
    if min_wait_ms:
        page.wait_for_timeout(min_wait_ms)

    deadline = start + (timeout_ms / 1000.0)
    last_ready_state = ""
    last_pending = 0
    success = False

    while time.monotonic() < deadline:
        try:
            data = page.evaluate(_PERF_LOAD_STATE_JS)
        except Exception:  # noqa: BLE001
            page.wait_for_timeout(poll_interval_ms)
            continue

        if isinstance(data, dict):
            rs = data.get("readyState")
            pending = data.get("pendingRequests")
            if isinstance(rs, str):
                last_ready_state = rs
            if isinstance(pending, int):
                last_pending = pending

        if last_ready_state and _ready_state_satisfies(last_ready_state, state) and last_pending == 0:
            success = True
            break

        page.wait_for_timeout(poll_interval_ms)

    waited_ms = int((time.monotonic() - start) * 1000)
    return {
        "ok": success,
        "timed_out": not success,
        "waited_ms": waited_ms,
        "ready_state": last_ready_state,
        "pending_requests": last_pending,
    }
