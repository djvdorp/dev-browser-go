import os
import sys
import time
from pathlib import Path
from typing import Optional


def now_ms() -> int:
    return int(time.time() * 1000)


def platform_cache_dir() -> Path:
    home = Path.home()

    xdg_cache_home = os.environ.get("XDG_CACHE_HOME")
    if xdg_cache_home:
        return Path(xdg_cache_home)

    if sys.platform == "darwin":
        return home / "Library" / "Caches"

    return home / ".cache"


def platform_state_dir() -> Path:
    home = Path.home()

    xdg_state_home = os.environ.get("XDG_STATE_HOME")
    if xdg_state_home:
        return Path(xdg_state_home)

    if sys.platform == "darwin":
        return home / "Library" / "Application Support"

    return home / ".local" / "state"


def safe_artifact_path(artifact_dir: Path, path_arg: Optional[str], default_name: str) -> Path:
    allow_unsafe = os.environ.get("DEV_BROWSER_ALLOW_UNSAFE_PATHS", "").lower() in {"1", "true", "yes"}

    if not path_arg:
        artifact_dir.mkdir(parents=True, exist_ok=True)
        return artifact_dir / default_name

    expanded = Path(os.path.expandvars(os.path.expanduser(path_arg)))
    resolved = expanded if expanded.is_absolute() else (artifact_dir / expanded)
    resolved = resolved.resolve()

    if allow_unsafe:
        resolved.parent.mkdir(parents=True, exist_ok=True)
        return resolved

    artifact_dir_resolved = artifact_dir.resolve()
    if artifact_dir_resolved not in resolved.parents and resolved != artifact_dir_resolved:
        raise ValueError(
            f"Refusing to write outside artifact dir: {resolved} (allowed under {artifact_dir_resolved}). "
            "Set DEV_BROWSER_ALLOW_UNSAFE_PATHS=1 to override."
        )

    resolved.parent.mkdir(parents=True, exist_ok=True)
    return resolved

