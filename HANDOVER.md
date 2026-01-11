# Handover

State: Go-only (`dev-browser-go`) with embedded daemon flag. Python path removed. Snapshot JS vendored under `internal/devbrowser/snapshot_assets*`; cache/state under `~/Library/Caches/dev-browser-go/<profile>/artifacts` and `~/Library/Application Support/dev-browser-go/<profile>` (XDG respected).

Goals next:
- CI: build/test matrix (darwin/linux, amd64/arm64) with Playwright browsers installed; attach single binary + checksums to GitHub Release on tag (SemVer `v0.y.z`).
- Smoke: run `HEADLESS=1 ./dev-browser-go goto https://example.com` then `./dev-browser-go snapshot` inside Nix dev shell (Playwright present).
- Packaging: Nix flake exposes only Go binary and skill output.
