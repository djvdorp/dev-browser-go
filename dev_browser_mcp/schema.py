from typing import Any


TOOLS: list[dict[str, Any]] = [
    {
        "name": "page",
        "description": "Get or create a named page (default: main).",
        "inputSchema": {
            "type": "object",
            "properties": {
                "page": {"type": "string", "description": "Page name (default: main)."},
                "bring_to_front": {"type": "boolean", "description": "Bring the page to front (headed only).", "default": False},
            },
        },
    },
    {
        "name": "list_pages",
        "description": "List known pages (name, url, title).",
        "inputSchema": {"type": "object", "properties": {}},
    },
    {
        "name": "close_page",
        "description": "Close a named page (no-op if missing).",
        "inputSchema": {"type": "object", "properties": {"page": {"type": "string"}}},
    },
    {
        "name": "goto",
        "description": "Navigate a page to a URL.",
        "inputSchema": {
            "type": "object",
            "properties": {
                "page": {"type": "string"},
                "url": {"type": "string"},
                "wait_until": {
                    "type": "string",
                    "description": "Playwright wait_until: load|domcontentloaded|networkidle|commit.",
                    "default": "domcontentloaded",
                },
                "timeout_ms": {"type": "integer", "default": 45000},
            },
            "required": ["url"],
        },
    },
    {
        "name": "snapshot",
        "description": "Token-light accessibility snapshot with stable refs (e1, e2, ...).",
        "inputSchema": {
            "type": "object",
            "properties": {
                "page": {"type": "string"},
                "interactive_only": {"type": "boolean", "default": True},
                "interesting_only": {"type": "boolean", "description": "Playwright accessibility snapshot interesting_only.", "default": True},
                "max_items": {"type": "integer", "default": 80},
                "max_chars": {"type": "integer", "default": 8000},
            },
        },
    },
    {
        "name": "click_ref",
        "description": "Click an element ref from the most recent snapshot on that page.",
        "inputSchema": {
            "type": "object",
            "properties": {"page": {"type": "string"}, "ref": {"type": "string"}, "timeout_ms": {"type": "integer", "default": 15000}},
            "required": ["ref"],
        },
    },
    {
        "name": "fill_ref",
        "description": "Fill an input element ref from the most recent snapshot on that page.",
        "inputSchema": {
            "type": "object",
            "properties": {
                "page": {"type": "string"},
                "ref": {"type": "string"},
                "text": {"type": "string"},
                "timeout_ms": {"type": "integer", "default": 15000},
            },
            "required": ["ref", "text"],
        },
    },
    {
        "name": "press",
        "description": "Send a keyboard press to the page (e.g., Enter, Escape, Control+L).",
        "inputSchema": {"type": "object", "properties": {"page": {"type": "string"}, "key": {"type": "string"}}, "required": ["key"]},
    },
    {
        "name": "screenshot",
        "description": "Save a PNG screenshot to the artifact dir and return the path.",
        "inputSchema": {
            "type": "object",
            "properties": {
                "page": {"type": "string"},
                "path": {"type": "string", "description": "Optional relative path under the artifact dir."},
                "full_page": {"type": "boolean", "default": True},
            },
        },
    },
    {
        "name": "save_html",
        "description": "Save the current page HTML to the artifact dir and return the path.",
        "inputSchema": {
            "type": "object",
            "properties": {"page": {"type": "string"}, "path": {"type": "string", "description": "Optional relative path under the artifact dir."}},
        },
    },
    {
        "name": "actions",
        "description": "Run multiple tool calls in a single round-trip. Each call is {name, arguments}.",
        "inputSchema": {
            "type": "object",
            "properties": {
                "calls": {
                    "type": "array",
                    "items": {
                        "type": "object",
                        "properties": {"name": {"type": "string"}, "arguments": {"type": "object"}},
                        "required": ["name"],
                    },
                }
            },
            "required": ["calls"],
        },
    },
]

