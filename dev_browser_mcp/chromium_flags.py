from __future__ import annotations

import os
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

    return args


def _env_truthy(name: str) -> bool:
    return os.environ.get(name, "").strip().lower() in {"1", "true", "yes", "on"}

