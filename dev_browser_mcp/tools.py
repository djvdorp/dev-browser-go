import json
import logging
from typing import Any, Optional

from .browser import BrowserManager, extract_refs, format_snapshot
from .paths import now_ms


LOGGER = logging.getLogger("dev-browser-mcp")


def mcp_error(text: str) -> dict[str, Any]:
    return {"isError": True, "content": [{"type": "text", "text": text}]}


def require_str(args: dict[str, Any], key: str) -> str:
    value = args.get(key)
    if not isinstance(value, str) or not value:
        raise ValueError(f"Expected non-empty string '{key}'")
    return value


def optional_str(args: dict[str, Any], key: str) -> Optional[str]:
    value = args.get(key)
    if value is None:
        return None
    if not isinstance(value, str):
        raise ValueError(f"Expected string '{key}'")
    return value


def optional_bool(args: dict[str, Any], key: str, default: bool) -> bool:
    value = args.get(key, default)
    if isinstance(value, bool):
        return value
    raise ValueError(f"Expected boolean '{key}'")


def optional_int(args: dict[str, Any], key: str, default: int) -> int:
    value = args.get(key, default)
    if isinstance(value, int) and value >= 0:
        return value
    raise ValueError(f"Expected non-negative integer '{key}'")


def optional_crop(args: dict[str, Any]) -> Optional[dict[str, float]]:
    raw = args.get("crop")
    if raw is None:
        return None

    def _from_seq(seq: list[Any]) -> tuple[int, int, int, int]:
        if len(seq) != 4:
            raise ValueError("crop must have 4 items: x,y,width,height")
        vals: list[int] = []
        for item in seq:
            if not isinstance(item, int):
                raise ValueError("crop values must be integers")
            if item < 0:
                raise ValueError("crop values must be non-negative")
            vals.append(int(item))
        return vals[0], vals[1], vals[2], vals[3]

    if isinstance(raw, str):
        parts = [p.strip() for p in raw.split(",") if p.strip()]
        if len(parts) != 4:
            raise ValueError("crop must be x,y,width,height")
        seq: list[int] = []
        for part in parts:
            if not part.isdigit():
                raise ValueError("crop values must be integers")
            seq.append(int(part))
        x, y, w, h = _from_seq(seq)
    elif isinstance(raw, dict):
        try:
            seq = [raw["x"], raw["y"], raw["width"], raw["height"]]
        except Exception as exc:
            raise ValueError("crop object must include x,y,width,height") from exc
        x, y, w, h = _from_seq(seq)
    elif isinstance(raw, (list, tuple)):
        x, y, w, h = _from_seq(list(raw))
    else:
        raise ValueError("crop must be string, array, or object")

    if w < 1 or h < 1:
        raise ValueError("crop width/height must be positive")
    max_wh = 2000
    w = min(w, max_wh)
    h = min(h, max_wh)

    return {"x": float(x), "y": float(y), "width": float(w), "height": float(h)}


def page_name(args: dict[str, Any]) -> str:
    name = args.get("page", "main")
    if not isinstance(name, str) or not name:
        raise ValueError("Expected string 'page'")
    return name

def run_tool(manager: BrowserManager, tool_name: str, args: dict[str, Any]) -> dict[str, Any]:
    if tool_name == "page":
        name = page_name(args)
        bring_to_front = optional_bool(args, "bring_to_front", False)
        page = manager.get_page(name)
        if bring_to_front:
            try:
                page.bring_to_front()
            except Exception:
                pass
        try:
            title = page.title()
        except Exception:
            title = ""
        return {"page": name, "url": page.url, "title": title}

    if tool_name == "list_pages":
        return {"pages": manager.list_pages()}

    if tool_name == "close_page":
        name = page_name(args)
        manager.close_page(name)
        return {"page": name, "closed": True}

    if tool_name == "goto":
        name = page_name(args)
        url = require_str(args, "url")
        wait_until = args.get("wait_until", "domcontentloaded")
        if not isinstance(wait_until, str) or not wait_until:
            raise ValueError("Expected string 'wait_until'")
        timeout_ms = optional_int(args, "timeout_ms", 45_000)

        page = manager.get_page(name)
        page.goto(url, wait_until=wait_until, timeout=timeout_ms)
        try:
            title = page.title()
        except Exception:
            title = ""
        return {"page": name, "url": page.url, "title": title}

    if tool_name == "snapshot":
        name = page_name(args)
        interactive_only = optional_bool(args, "interactive_only", True)
        interesting_only = optional_bool(args, "interesting_only", True)
        max_items = optional_int(args, "max_items", 80)
        max_chars = optional_int(args, "max_chars", 8000)

        page = manager.get_page(name)
        ax = page.accessibility.snapshot(interesting_only=interesting_only)
        items, refs, truncated = extract_refs(ax, interactive_only=interactive_only, max_items=max_items)
        manager.set_refs(name, refs)
        snapshot_text = format_snapshot(items, max_chars=max_chars, truncated=truncated)
        try:
            title = page.title()
        except Exception:
            title = ""
        return {"page": name, "url": page.url, "title": title, "items": items, "snapshot": snapshot_text}

    if tool_name == "click_ref":
        name = page_name(args)
        ref = require_str(args, "ref")
        timeout_ms = optional_int(args, "timeout_ms", 15_000)
        target = manager.get_ref(name, ref)
        page = manager.get_page(name)
        locator = page.get_by_role(target.role, name=target.name) if target.name else page.get_by_role(target.role)
        locator.nth(target.nth).click(timeout=timeout_ms)
        return {"page": name, "ref": ref, "clicked": True}

    if tool_name == "fill_ref":
        name = page_name(args)
        ref = require_str(args, "ref")
        text = require_str(args, "text")
        timeout_ms = optional_int(args, "timeout_ms", 15_000)
        target = manager.get_ref(name, ref)
        page = manager.get_page(name)
        locator = page.get_by_role(target.role, name=target.name) if target.name else page.get_by_role(target.role)
        locator.nth(target.nth).fill(text, timeout=timeout_ms)
        return {"page": name, "ref": ref, "filled": True}

    if tool_name == "press":
        name = page_name(args)
        key = require_str(args, "key")
        page = manager.get_page(name)
        page.keyboard.press(key)
        return {"page": name, "key": key, "pressed": True}

    if tool_name == "screenshot":
        name = page_name(args)
        path_arg = optional_str(args, "path")
        full_page = optional_bool(args, "full_page", True)
        crop = optional_crop(args)
        path = manager.artifact_path(path_arg=path_arg, default_name=f"screenshot-{now_ms()}.png")
        page = manager.get_page(name)
        options: dict[str, Any] = {"path": str(path), "full_page": full_page}
        if crop is not None:
            options["clip"] = crop
            options["full_page"] = False
        page.screenshot(**options)
        return {"page": name, "path": str(path)}

    if tool_name == "save_html":
        name = page_name(args)
        path_arg = optional_str(args, "path")
        path = manager.artifact_path(path_arg=path_arg, default_name=f"page-{now_ms()}.html")
        page = manager.get_page(name)
        path.write_text(page.content(), encoding="utf-8")
        return {"page": name, "path": str(path)}

    if tool_name == "actions":
        calls = args.get("calls")
        if not isinstance(calls, list):
            raise ValueError("Expected array 'calls'")

        results: list[dict[str, Any]] = []
        for call in calls:
            if not isinstance(call, dict):
                raise ValueError("Each call must be an object")
            name = call.get("name")
            if not isinstance(name, str) or not name:
                raise ValueError("Each call must include non-empty string 'name'")
            if name == "actions":
                raise ValueError("Nested actions are not supported")
            call_args = call.get("arguments", {})
            if call_args is None:
                call_args = {}
            if not isinstance(call_args, dict):
                raise ValueError("Call 'arguments' must be an object")
            results.append({"name": name, "result": run_tool(manager, name, call_args)})

        return {"results": results}

    raise ValueError(f"Unknown tool '{tool_name}'")


def handle_tools_call(manager: BrowserManager, params: dict[str, Any]) -> dict[str, Any]:
    name = params.get("name")
    if not isinstance(name, str) or not name:
        return mcp_error("Missing tool name")

    args = params.get("arguments", {})
    if args is None:
        args = {}
    if not isinstance(args, dict):
        return mcp_error("Tool arguments must be an object")

    try:
        data = run_tool(manager, name, args)
    except Exception as exc:
        LOGGER.exception("tool_failed name=%s", name)
        return {"isError": True, "structuredContent": {"error": str(exc)}, "content": [{"type": "text", "text": str(exc)}]}

    if name == "snapshot" and isinstance(data.get("snapshot"), str):
        text = data["snapshot"]
    elif isinstance(data.get("path"), str):
        text = data["path"]
    else:
        text = json.dumps(data, ensure_ascii=False)
    return {"structuredContent": data, "content": [{"type": "text", "text": text}]}
