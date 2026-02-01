#!/usr/bin/env bash
set -euo pipefail

BASE_REF="${1:-origin/main}"
RANGE="${2:-${BASE_REF}...HEAD}"

if ! command -v gh >/dev/null 2>&1; then
  echo "gh CLI not found; cannot run Copilot review." >&2
  exit 1
fi

if ! gh copilot --help >/dev/null 2>&1; then
  echo "gh copilot not available (is the Copilot extension installed and authenticated?)." >&2
  echo "Try: gh extension install github/gh-copilot" >&2
  exit 1
fi

DIFF=$(git diff --no-color "$RANGE")
if [[ -z "${DIFF}" ]]; then
  echo "No diff for range: ${RANGE}" >&2
  exit 0
fi

# Keep the prompt short and let Copilot see the full diff.
PROMPT=$'Please do a code review of the following git diff.\n\nFocus on:\n- correctness / edge cases\n- determinism / stable output\n- backwards compatibility\n- test coverage gaps\n- security / footguns\n\nReturn a concise bullet list with actionable recommendations and note severity (blocker/non-blocker).\n\n---\n'

# Use copilot explain as a general-purpose analysis tool.
printf "%s%s" "$PROMPT" "$DIFF" | gh copilot explain --stdin
