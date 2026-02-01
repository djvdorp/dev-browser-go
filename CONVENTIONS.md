# Conventions (Dev-browser-go)

This repo is built for **headless, agent-driven browser workflows**.

These conventions are here to keep behavior predictable for CLI agents, CI, and automation.

## Principles

- **Headless-first**: no required human interaction or headed UI.
- **Harness-friendly**: prefer a few high-signal report/gating commands that compose primitives.
- **Structured output**: JSON schemas are contracts.
- **Deterministic**: stable ordering of lists and stable keys.
- **Low flake**: prefer reliable waits and bounded outputs.

## CLI semantics

### Output modes
- Commands should honor the global `--output` (`summary|json|html|path`) and `--out` (path when `--output=path`).
- JSON output should be **stable** (field presence/order) and **deterministic** (sorted lists).

### Exit codes
- Most commands:
  - `0` on success
  - non-zero only for runtime/tool failures
- Harness/gating commands may intentionally return non-zero for failed checks:
  - `assert`: `0` pass, `2` fail
- Report-only commands should always exit `0`:
  - `diagnose`, `html-validate`

## Artifacts

- Artifacts must be written under the configured artifact root (profile-scoped).
- Use a per-run directory: `run-<timestamp>-<uuid8>/`.
- Avoid writing huge payloads by default:
  - bodies are opt-in and bounded
  - snippets are truncated

## Determinism rules

- Any list in JSON output should be sorted deterministically.
  - network: by `started_ms` then URL/method
  - console: by time then id
  - findings/diffs: by rule/id then location
- Prefer stable IDs/keys in outputs.

## JavaScript injection + Evaluate

- Prefer adding reusable helpers into the injected bundle (`internal/devbrowser/snapshot_assets/base_snapshot.js`) under `globalThis.__devBrowser_*`.
- When using `page.Evaluate`, **do not use `arguments` inside arrow functions**.
  - Good: `(sel) => document.querySelectorAll(sel).length`
  - Avoid: `() => arguments[0]`
- Always type-guard DOM nodes and handle missing APIs:
  - XPath results can be non-Element nodes
  - `canvas.getContext('2d')` can be null
  - IDs may contain quotes (escape XPath literals properly)

## Concurrency

- Treat Playwright event handlers as concurrent.
- Shared state mutated from event handlers must be guarded by a mutex.
- Prefer `defer mu.Unlock()` patterns to prevent early-return unlock bugs.

## Testing

- Add unit tests for new pure logic and schema shapes:
  - rules parsing + evaluation
  - filters
  - validators
- Keep tests deterministic (avoid timing-based assertions unless necessary).

## Where to put things

- Primitive browser operations live in `internal/devbrowser/`.
- CLI commands live in `cmd/dev-browser-go/` and should be thin wrappers.
- Harness commands should compose primitives rather than re-implementing them.
