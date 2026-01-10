import json
import sys
from typing import Any

from .browser import BrowserManager
from .schema import TOOLS
from .tools import handle_tools_call


PROTOCOL_VERSION = "2024-11-05"


def jsonrpc_result(request_id: Any, result: Any) -> dict[str, Any]:
    return {"jsonrpc": "2.0", "id": request_id, "result": result}


def jsonrpc_error(request_id: Any, *, code: int, message: str) -> dict[str, Any]:
    return {"jsonrpc": "2.0", "id": request_id, "error": {"code": code, "message": message}}


def write_message(message: dict[str, Any]) -> None:
    sys.stdout.write(json.dumps(message, ensure_ascii=False) + "\n")
    sys.stdout.flush()


def serve_stdio(manager: BrowserManager) -> int:
    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue

        try:
            message = json.loads(line)
        except json.JSONDecodeError:
            continue

        request_id = message.get("id")
        method = message.get("method")
        params = message.get("params", {})
        if params is None:
            params = {}

        if not isinstance(method, str):
            if request_id is not None:
                write_message(jsonrpc_error(request_id, code=-32600, message="Invalid Request"))
            continue

        if method == "initialize":
            if request_id is None:
                continue
            result = {
                "protocolVersion": PROTOCOL_VERSION,
                "capabilities": {"tools": {"listChanged": False}},
                "serverInfo": {"name": "dev-browser-mcp", "version": "0.1.0"},
            }
            write_message(jsonrpc_result(request_id, result))
            continue

        if method == "notifications/initialized":
            continue

        if method == "tools/list":
            if request_id is None:
                continue
            write_message(jsonrpc_result(request_id, {"tools": TOOLS}))
            continue

        if method == "tools/call":
            if request_id is None:
                continue
            if not isinstance(params, dict):
                write_message(jsonrpc_result(request_id, {"isError": True, "content": [{"type": "text", "text": "Invalid params"}]}))
                continue
            write_message(jsonrpc_result(request_id, handle_tools_call(manager, params)))
            continue

        if request_id is not None:
            write_message(jsonrpc_error(request_id, code=-32601, message=f"Method not found: {method}"))

    return 0
