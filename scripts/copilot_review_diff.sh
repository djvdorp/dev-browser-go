#!/usr/bin/env bash
set -euo pipefail

BASE_REF="${1:-origin/main}"
RANGE="${2:-${BASE_REF}...HEAD}"

if ! command -v gh >/dev/null 2>&1; then
  echo "gh CLI not found; cannot run Copilot review." >&2
  exit 1
fi

DIFF=$(git diff --no-color "$RANGE")
if [[ -z "${DIFF}" ]]; then
  echo "No diff for range: ${RANGE}" >&2
  exit 0
fi

PROMPT=$'Review this git diff for correctness, determinism/stable output, backwards compatibility, and test gaps.\nReturn a concise bullet list with severity (blocker/non-blocker).\n\n---\n'

# Note: gh copilot does not support --stdin; pass the diff as part of the prompt.
gh copilot -p "${PROMPT}${DIFF}"
