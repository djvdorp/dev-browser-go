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

if ((status != 0)); then
  exit "$status"
fi

echo "docs-audit: README.md and SKILL.md match the current CLI surface"
