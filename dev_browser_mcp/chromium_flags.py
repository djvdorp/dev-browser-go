from __future__ import annotations

import os
import re
from typing import Optional


def chromium_launch_args(*, cdp_port: Optional[int] = None) -> list[str]:
    args: list[str] = []
    if cdp_port is not None:
        args.append(f"--remote-debugging-port={int(cdp_port)}")

    # On macOS, Chromium stores the encryption key for cookies/passwords in the
    # user's Keychain ("Chromium Safe Storage"). When launched from automation
    # (Playwright's `headless_shell`), this can trigger a Keychain prompt.
    #
    # Default: avoid the prompt with a mock keychain (stable across runs).
    # Opt-in: set DEV_BROWSER_USE_KEYCHAIN=1 to use the real Keychain.
    if not _env_truthy("DEV_BROWSER_USE_KEYCHAIN"):
        args.append("--use-mock-keychain")

    window_size = window_size_from_env(default=DEFAULT_WINDOW_SIZE)
    if window_size is not None:
        width, height = window_size
        args.append(f"--window-size={width},{height}")

    return args


def _env_truthy(name: str) -> bool:
    return os.environ.get(name, "").strip().lower() in {"1", "true", "yes", "on"}


_WINDOW_SIZE_RE = re.compile(r"^\s*(\d+)\s*[xX*,]\s*(\d+)\s*$")
DEFAULT_WINDOW_SIZE = (2500, 1920)


def window_size_from_env(*, default: Optional[tuple[int, int]] = None) -> Optional[tuple[int, int]]:
    raw = os.environ.get("DEV_BROWSER_WINDOW_SIZE", "").strip()
    if not raw:
        return default
    normalized = raw.lower().replace("px", "")
    match = _WINDOW_SIZE_RE.match(normalized)
    if not match:
        raise ValueError("DEV_BROWSER_WINDOW_SIZE must be WIDTHxHEIGHT (e.g. 2500x1920)")
    width = int(match.group(1))
    height = int(match.group(2))
    if width < 1 or height < 1:
        raise ValueError("DEV_BROWSER_WINDOW_SIZE must be positive (e.g. 2500x1920)")
    return width, height
