#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

tmp_bin="${TMPDIR:-/tmp}/dev-browser-go-doccheck-$$"
cleanup() {
  rm -f "$tmp_bin"
}
trap cleanup EXIT

go build -o "$tmp_bin" ./cmd/dev-browser-go >/dev/null

mapfile -t commands < <(
  "$tmp_bin" --help | awk '
    /^Available Commands:/ { in_cmds=1; next }
    in_cmds && /^Flags:/ { exit }
    in_cmds && NF > 0 {
      cmd = $1
      if (cmd == "help" || cmd == "completion") next
      print cmd
    }
  ' | sort -u
)

if ((${#commands[@]} == 0)); then
  echo "docs-audit: failed to discover command surface from CLI help" >&2
  exit 1
fi

status=0
for doc in README.md SKILL.md; do
  missing=()
  for cmd in "${commands[@]}"; do
    if ! grep -Fqi "$cmd" "$doc"; then
      missing+=("$cmd")
    fi
  done

  if ((${#missing[@]} > 0)); then
    echo "docs-audit: $doc is missing command mentions: ${missing[*]}" >&2
    status=1
  fi
done

if rg -n '\{"tool":|\{"args":' README.md SKILL.md >/dev/null; then
  echo "docs-audit: stale actions JSON shape found; use name/arguments, not tool/args" >&2
  rg -n '\{"tool":|\{"args":' README.md SKILL.md >&2 || true
  status=1
fi

if rg -n 'asset-snapshot[^\n]*--selector' README.md SKILL.md >/dev/null; then
  echo "docs-audit: asset-snapshot docs mention unsupported --selector flag" >&2
  rg -n 'asset-snapshot[^\n]*--selector' README.md SKILL.md >&2 || true
  status=1
fi

if rg -n 'self-contained HTML|offline capable' README.md SKILL.md >/dev/null; then
  echo "docs-audit: asset-snapshot docs overstate current implementation" >&2
  rg -n 'self-contained HTML|offline capable' README.md SKILL.md >&2 || true
  status=1
fi

python3 - "$tmp_bin" README.md SKILL.md <<'PY' || status=1
import re
import shlex
import subprocess
import sys
from pathlib import Path

bin_path = sys.argv[1]
docs = [Path(p) for p in sys.argv[2:]]

HELP_FLAG_RE = re.compile(r'^\s*(?:-[A-Za-z],\s+)?(--[a-z0-9-]+)(?:\s+([a-zA-Z][a-zA-Z0-9_-]*))?', re.I)
CODE_FENCE_RE = re.compile(r'^```')


def run_help(args):
    out = subprocess.check_output([bin_path, *args, "--help"], text=True)
    return out.splitlines()


def parse_flags(lines):
    flags = {}
    in_flags = False
    for line in lines:
        if line.startswith("Flags:") or line.startswith("Global Flags:"):
            in_flags = True
            continue
        if in_flags and not line.strip():
            continue
        if in_flags and re.match(r'^[A-Z][A-Za-z ]+:$', line):
            in_flags = False
            continue
        if not in_flags:
            continue
        m = HELP_FLAG_RE.match(line)
        if not m:
            continue
        flag = m.group(1)
        type_name = (m.group(2) or "").lower()
        takes_value = type_name in {"string", "strings", "int", "float", "duration"}
        flags[flag] = takes_value
    return flags


root_help = subprocess.check_output([bin_path, "--help"], text=True).splitlines()
global_flags = parse_flags(root_help)

commands = []
in_cmds = False
for line in root_help:
    if line.startswith("Available Commands:"):
        in_cmds = True
        continue
    if in_cmds and line.startswith("Flags:"):
        break
    if in_cmds and line.strip():
        cmd = line.split()[0]
        if cmd not in {"help", "completion"}:
            commands.append(cmd)

command_flags = {}
for cmd in commands:
    command_flags[cmd] = parse_flags(run_help([cmd]))


def join_continuations(lines):
    out = []
    buf = ""
    for line in lines:
        stripped = line.rstrip()
        if buf:
            buf += stripped.lstrip()
        else:
            buf = stripped
        if buf.endswith("\\"):
            buf = buf[:-1] + " "
            continue
        out.append(buf)
        buf = ""
    if buf:
        out.append(buf)
    return out


def extract_commands(doc: Path):
    lines = doc.read_text().splitlines()
    in_block = False
    block = []
    examples = []
    for line in lines:
        if CODE_FENCE_RE.match(line):
            if in_block:
                examples.extend(join_continuations(block))
                block = []
                in_block = False
            else:
                in_block = True
            continue
        if in_block:
            block.append(line)
    return examples


def split_shell(line):
    try:
        return shlex.split(line, comments=True, posix=True)
    except ValueError:
        return None


def truncate_shell(tokens):
    stop = {"&&", "||", "|", ";", "then", "do"}
    out = []
    for token in tokens:
        if token in stop:
            break
        out.append(token)
    return out


def parse_example(line):
    if "dev-browser-go" not in line and "./dev-browser-go" not in line:
        return None
    stripped = line.strip()
    shell_like_prefixes = (
        "dev-browser-go ",
        "./dev-browser-go ",
        "HEADLESS=",
        "DEV_BROWSER_",
        "if ",
        "echo ",
    )
    if not stripped.startswith(shell_like_prefixes) and "| dev-browser-go" not in stripped and "| ./dev-browser-go" not in stripped:
        return None
    if "<" in line:
        return None
    tokens = split_shell(line)
    if not tokens:
        return None
    tokens = truncate_shell(tokens)
    try:
        idx = next(i for i, tok in enumerate(tokens) if tok in {"dev-browser-go", "./dev-browser-go"})
    except StopIteration:
        return None
    tokens = tokens[idx + 1 :]
    if not tokens:
        return None
    return tokens


def validate_tokens(doc, line, tokens):
    errors = []
    cmd = None
    i = 0
    while i < len(tokens):
        tok = tokens[i]
        if cmd is None and tok.startswith("--"):
            takes = global_flags.get(tok)
            if takes is None:
                errors.append(f"{doc}: invalid global flag {tok!r} in example: {line}")
                break
            i += 2 if takes else 1
            continue
        if cmd is None:
            cmd = tok
            if cmd in {"help", "completion", "--help", "--version"}:
                return errors
            if cmd not in command_flags:
                errors.append(f"{doc}: unknown command {cmd!r} in example: {line}")
                return errors
            i += 1
            continue
        if tok.startswith("--"):
            flags = {**global_flags, **command_flags[cmd]}
            takes = flags.get(tok)
            if takes is None:
                errors.append(f"{doc}: invalid flag {tok!r} for command {cmd!r} in example: {line}")
                break
            i += 2 if takes else 1
            continue
        i += 1
    return errors


all_errors = []
for doc in docs:
    for line in extract_commands(doc):
        parsed = parse_example(line)
        if parsed is None:
            continue
        all_errors.extend(validate_tokens(doc.name, line, parsed))

if all_errors:
    for err in all_errors:
        print(f"docs-audit: {err}", file=sys.stderr)
    sys.exit(1)

print("docs-audit: example flags validated against live CLI help")
PY

if ((status != 0)); then
  exit "$status"
fi

echo "docs-audit: README.md and SKILL.md match the current CLI surface"
