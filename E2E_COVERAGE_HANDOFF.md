# E2E Coverage Handoff

Last updated: 2026-03-29

## Why this file exists

This repo had a real regression where the spawned daemon died after `goto`, which made later `screenshot` calls capture a blank white page. The root cause was fixed by detaching the spawned daemon process, but the bigger problem was missing real spawned-binary end-to-end coverage on the actual CLI + daemon lifecycle path.

This handoff records:

- what was fixed
- what e2e coverage was added
- what remains lower priority
- what to do next if more coverage work resumes later

## Important recent history

### Regression and release work

- `77d6c2c` `fix(daemon): detach spawned browser process`
- `223b306` `chore(release): bump version to 0.2.2`
- `dc3fda6` `docs(devices): default mobile examples to Galaxy S9+`

### E2E expansion already landed

- `b81cf8f` `test(e2e): cover cli reconnect flows`
- `e1d912a` `test(e2e): expand cli workflow coverage`
- `ff838e7` `feat(network-monitor): allow capture during navigation`
- `6e2717e` `test(e2e): cover network and asset snapshot workflows`
- `0595b25` `test(e2e): cover generic call and actions workflows`
- `9bb93b3` `test(e2e): cover selector refs and interaction flows`
- `96fcf22` `test(e2e): cover perf metrics and dom diff workflows`

## Current state

All of the following are now covered by real CLI e2e tests that build the binary, run commands against fixture pages, and exercise the spawned daemon path:

- desktop screenshot workflow
- mobile screenshot workflow
- `save-html` reconnect workflow
- positional `js-eval`
- daemon lifecycle: `start`, `status`, `list-pages`, named pages, `close-page`, `stop`
- `inject` + `console`
- `save-baseline`, `visual-diff`, `diff-images`
- `diagnose`, `assert`, `html-validate`
- `network-monitor`
- `asset-snapshot`
- generic `call`
- generic `actions`, including stdin input
- `snapshot`, `inspect-ref`, `test-selector`, `test-xpath`, `bounds`
- `fill-ref`, `press`, `wait`, `click-ref`
- `perf-metrics`
- `save-dom-baseline`, `dom-diff`

Supporting e2e test files live in:

- [cmd/dev-browser-go/e2e_browser_workflows_test.go](/Users/daniel/dev/misc/dev-browser-go/cmd/dev-browser-go/e2e_browser_workflows_test.go)
- [cmd/dev-browser-go/e2e_analysis_workflows_test.go](/Users/daniel/dev/misc/dev-browser-go/cmd/dev-browser-go/e2e_analysis_workflows_test.go)
- [cmd/dev-browser-go/e2e_batch_workflows_test.go](/Users/daniel/dev/misc/dev-browser-go/cmd/dev-browser-go/e2e_batch_workflows_test.go)
- [cmd/dev-browser-go/e2e_selector_ref_workflows_test.go](/Users/daniel/dev/misc/dev-browser-go/cmd/dev-browser-go/e2e_selector_ref_workflows_test.go)
- [cmd/dev-browser-go/e2e_perf_dom_workflows_test.go](/Users/daniel/dev/misc/dev-browser-go/cmd/dev-browser-go/e2e_perf_dom_workflows_test.go)
- [cmd/dev-browser-go/e2e_fixtures_test.go](/Users/daniel/dev/misc/dev-browser-go/cmd/dev-browser-go/e2e_fixtures_test.go)
- [cmd/dev-browser-go/e2e_helpers_test.go](/Users/daniel/dev/misc/dev-browser-go/cmd/dev-browser-go/e2e_helpers_test.go)

CI also installs Playwright Chromium before running tests:

- [.github/workflows/ci.yml](/Users/daniel/dev/misc/dev-browser-go/.github/workflows/ci.yml)

## Remaining lower-priority gaps

These are the remaining commands I would look at next, in this order:

1. `style-capture`
Reason: HTML/CSS serialization path, file output, and mode differences (`inline` vs `bundle`) are integration-heavy.

2. `color-info`
Reason: depends on snapshot refs plus computed style extraction.

3. `font-info`
Reason: similar risk profile to `color-info`; cheap to cover once ref fixture exists.

4. `devices`
Reason: lower risk, but one smoke test guarding expected device list output could catch docs/example drift.

5. `loop`
Reason: useful, but its value depends on whether it is actively used in production workflows. It may need a more careful harness than the other commands.

## Recommended next batch

If resuming later, I would do this first:

### Batch A

- `style-capture` e2e:
  - run `goto` on a fixture page with a small styled subtree
  - exercise `--mode inline`
  - exercise `--mode bundle --css-path ...`
  - assert output files exist
  - assert captured HTML/CSS contain expected marker content

- `color-info` e2e:
  - use the existing interactive fixture or a tiny dedicated fixture
  - get a ref from `snapshot`
  - call `color-info --ref <ref>`
  - assert expected color keys exist and parse sanely

- `font-info` e2e:
  - same pattern as `color-info`
  - assert expected font keys exist and contain non-empty values

This would give good coverage density with minimal new fixture complexity.

## Notes from the debugging work

- The original blank screenshot issue was not caught because older tests used in-process daemon paths instead of the real spawned binary path.
- `network-monitor` needed an optional navigation input (`--url`) to make real capture deterministic in e2e tests.
- `snapshot` defaults matter. Some generic-path tests needed `interactive_only=false` to avoid empty outputs on simple fixtures.
- `wait --strategy playwright --state load` is not the right assertion after pure in-page interactions. For interaction workflows, `--strategy perf` is the better fit.
- Keep e2e files split. One monolithic `e2e_test.go` became too large; it was intentionally broken apart to keep the suite maintainable.

## Verification baseline

Before starting another batch, re-run:

```bash
go test ./...
go build ./cmd/dev-browser-go
```

If touching browser-backed e2e behavior or CI:

- confirm `.github/workflows/ci.yml` still installs Playwright Chromium
- keep tests on the real built binary path, not only direct package-level helpers

## Resume checklist

1. Read this file.
2. Read the newest e2e test files under [cmd/dev-browser-go](/Users/daniel/dev/misc/dev-browser-go/cmd/dev-browser-go).
3. Pick the next small batch, not a huge sweep.
4. Add fixtures/helpers only when reusable.
5. Run `go test ./...` and `go build ./cmd/dev-browser-go`.
6. Commit and push after each batch.
