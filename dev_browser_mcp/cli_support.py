from __future__ import annotations

import json
import os
import subprocess
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path
from typing import Any, Optional

from .paths import now_ms, platform_cache_dir, platform_state_dir, safe_artifact_path


def state_dir(profile: str) -> Path:
    return platform_state_dir() / "dev-browser-mcp" / profile


def state_file(profile: str) -> Path:
    return state_dir(profile) / "daemon.json"


def artifact_dir(profile: str) -> Path:
    return platform_cache_dir() / "dev-browser-mcp" / profile / "artifacts"


def read_state(profile: str) -> Optional[dict[str, Any]]:
    path = state_file(profile)
    if not path.exists():
        return None
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except Exception:
        return None


def daemon_base_url(profile: str) -> Optional[str]:
    info = read_state(profile)
    if not info:
        return None
    host = info.get("host")
    port = info.get("port")
    if not isinstance(host, str) or not isinstance(port, int):
        return None
    return f"http://{host}:{port}"


def http_json(method: str, url: str, body: Optional[dict[str, Any]] = None, timeout_s: float = 3.0) -> dict[str, Any]:
    data = None if body is None else json.dumps(body, ensure_ascii=False).encode("utf-8")
    req = urllib.request.Request(url, method=method)
    if data is not None:
        req.add_header("Content-Type", "application/json; charset=utf-8")
        req.data = data
    try:
        with urllib.request.urlopen(req, timeout=timeout_s) as resp:
            raw = resp.read()
    except urllib.error.HTTPError as exc:
        raw = exc.read()
    return json.loads(raw.decode("utf-8"))


def is_daemon_healthy(profile: str) -> bool:
    base = daemon_base_url(profile)
    if not base:
        return False
    try:
        data = http_json("GET", f"{base}/health", None, timeout_s=1.5)
    except Exception:
        return False
    return bool(data.get("ok") is True)


def daemon_script_path() -> Path:
    env_path = os.environ.get("DEV_BROWSER_DAEMON_PATH")
    if env_path:
        path = Path(env_path).expanduser().resolve()
        if path.exists():
            return path

    here = Path(__file__).resolve()
    candidates = [here.parent.parent / "daemon.py"]
    for parent in here.parents:
        candidates.append(parent / "libexec/dev-browser-mcp/daemon.py")
    for cand in candidates:
        if cand.exists():
            return cand
    raise RuntimeError("dev-browser daemon script not found; reinstall the package")


def start_daemon(profile: str, headless: bool) -> None:
    if is_daemon_healthy(profile):
        return

    state_dir(profile).mkdir(parents=True, exist_ok=True)
    log_path = state_dir(profile) / "daemon.log"

    cmd = [sys.executable, "-u", str(daemon_script_path()), "--profile", profile]
    if headless:
        cmd.append("--headless")

    with open(log_path, "a", encoding="utf-8") as log:
        subprocess.Popen(
            cmd,
            stdout=log,
            stderr=log,
            stdin=subprocess.DEVNULL,
            start_new_session=True,
            close_fds=True,
        )

    deadline = time.time() + 30.0
    while time.time() < deadline:
        if is_daemon_healthy(profile):
            return
        time.sleep(0.2)

    raise RuntimeError(f"Timed out waiting for dev-browser daemon (profile={profile}). See {log_path}")


def stop_daemon(profile: str) -> bool:
    info = read_state(profile)
    base = daemon_base_url(profile)
    if not info or not base:
        return False

    try:
        http_json("POST", f"{base}/shutdown", {})
    except Exception:
        pid = info.get("pid")
        if isinstance(pid, int):
            try:
                os.kill(pid, 15)
            except Exception:
                pass

    deadline = time.time() + 5.0
    while time.time() < deadline:
        if not is_daemon_healthy(profile):
            break
        time.sleep(0.2)

    try:
        state_file(profile).unlink()
    except FileNotFoundError:
        pass

    return True


def ensure_page(profile: str, headless: bool, page: str) -> dict[str, Any]:
    start_daemon(profile, headless=headless)
    base = daemon_base_url(profile)
    if not base:
        raise RuntimeError("Daemon state missing after start")
    data = http_json("POST", f"{base}/pages", {"name": page}, timeout_s=10.0)
    ws_endpoint = data.get("wsEndpoint")
    target_id = data.get("targetId")
    if not isinstance(ws_endpoint, str) or not ws_endpoint:
        raise RuntimeError("Daemon did not return wsEndpoint")
    if not isinstance(target_id, str) or not target_id:
        raise RuntimeError("Daemon did not return targetId")
    return {"base": base, "wsEndpoint": ws_endpoint, "targetId": target_id}


def write_output(profile: str, output_mode: str, result: dict[str, Any], out_path: Optional[str]) -> None:
    if output_mode == "json":
        sys.stdout.write(json.dumps(result, ensure_ascii=False, indent=2) + "\n")
        return

    if output_mode == "summary":
        if "snapshot" in result and isinstance(result["snapshot"], str):
            sys.stdout.write(result["snapshot"] + "\n")
            return
        if "path" in result and isinstance(result["path"], str):
            sys.stdout.write(result["path"] + "\n")
            return
        sys.stdout.write(json.dumps(result, ensure_ascii=False) + "\n")
        return

    if output_mode == "path":
        path = safe_artifact_path(artifact_dir(profile), out_path, f"cli-{now_ms()}.json")
        path.write_text(json.dumps(result, ensure_ascii=False, indent=2), encoding="utf-8")
        sys.stdout.write(str(path) + "\n")
        return

    raise ValueError(f"Unknown output mode: {output_mode}")


def parse_args_json(args_json: str) -> dict[str, Any]:
    try:
        data = json.loads(args_json)
    except Exception as exc:
        raise ValueError("Invalid JSON for --args") from exc
    if not isinstance(data, dict):
        raise ValueError("--args must be a JSON object")
    return data

