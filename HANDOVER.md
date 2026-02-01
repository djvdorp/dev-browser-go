# Handover

State: Go-only (`dev-browser-go`) with embedded daemon flag. Python path removed. Snapshot JS vendored under `internal/devbrowser/snapshot_assets*`; cache/state under `~/Library/Caches/dev-browser-go/<profile>/artifacts` and `~/Library/Application Support/dev-browser-go/<profile>` (XDG respected).
Notes: Added Playwright device profiles (`--device`, `devices`). Device/viewport flags apply on daemon start; stop to switch. `DEV_BROWSER_WINDOW_SIZE` honored.

## Review learnings (keep)

From PR #12 + Copilot review follow-ups:

- **Event-driven concurrency**: treat Playwright `OnRequest/OnResponse/...` handlers as concurrent; keep shared state updates under a mutex consistently (prefer `defer mu.Unlock()` patterns).
- **CLI flag validation**: don’t duplicate `MarkFlagRequired` checks inside `RunE`/`PreRunE` if the flag is already marked required.
- **Dead code discipline**: remove unused helpers (e.g. perf normalization helpers) before PR; keep exported surface small.
- **Injected JS safety**:
  - Escape XPath literals properly (IDs can contain quotes).
  - Type-guard DOM nodes (XPath results aren’t always Elements).
  - Handle missing APIs gracefully (e.g. `canvas.getContext('2d')` can be null).
- **Tests as a contract**: add unit tests for new “pure” helpers + output schema shapes (filters, parsers, struct outputs) in the same PR.
- **Commit hygiene**: use Conventional Commits consistently.

Goals next:
- CI: build/test matrix (darwin/linux, amd64/arm64) with Playwright browsers installed; attach single binary + checksums to GitHub Release on tag (SemVer `v0.y.z`).
- Smoke: run `HEADLESS=1 ./dev-browser-go goto https://example.com` then `./dev-browser-go snapshot` inside Nix dev shell (Playwright present).
- Packaging: Nix flake exposes only Go binary and skill output.
