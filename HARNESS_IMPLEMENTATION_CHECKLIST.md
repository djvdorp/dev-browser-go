# Harness MVP — Implementation Checklist

Owner: Daniel

Goal: enable a CLI agent to iteratively build and verify a React frontend against a Go backend autonomously.

Guiding principles:
- **Headless-first** (no required human interaction)
- **Structured outputs** (JSON schemas are contracts)
- **Low flake** (stable waits, deterministic ordering)
- **Few high-signal commands** (harness > toolbox)

---

## MVP: `diagnose` (report-only)

### CLI + wiring
- [ ] Add `diagnose` cobra command (cmd/dev-browser-go/diagnose.go)
- [ ] Add runner tool name: `diagnose` → RunCall switch
- [ ] Support flags:
  - [ ] `--url` (optional; if set, run goto)
  - [ ] `--page` (default main)
  - [ ] `--wait` (load|domcontentloaded|networkidle|commit)
  - [ ] `--timeout-ms`, `--min-wait-ms`
  - [ ] `--snapshot-engine` (simple|aria)
  - [ ] `--net-bodies` + `--net-max-body-bytes`
  - [ ] `--perf-sample-ms` + `--perf-top-n`
  - [ ] `--artifact-mode` (none|minimal|full)

### Report schema + stability
- [ ] Define `internal/devbrowser/diagnose.go` structs:
  - [ ] `DiagnoseReport` + `DiagnoseMeta` + `DiagnoseSummary`
  - [ ] Include `runId` (uuid) + timestamp
  - [ ] Include deterministic ordering for lists
- [ ] Produce a single JSON report containing:
  - [ ] console entries + counts
  - [ ] network summary (optionally bodies)
  - [ ] perf metrics
  - [ ] snapshot yaml + items
  - [ ] artifacts paths (if written)

### Artifacts
- [ ] Create per-run artifact dir (under existing artifact root) when artifact-mode != none
- [ ] Screenshot full page
- [ ] Optionally write: console.json, network.json, report.json

### Exit codes
- [ ] `diagnose` always exits `0` (report-only)

---

## MVP: `assert` (gating)

### CLI + rules
- [ ] Add `assert` cobra command (cmd/dev-browser-go/assert.go)
- [ ] Add `--rules` flag supporting:
  - [ ] raw JSON string
  - [ ] `@path/to/rules.json`
- [ ] Reuse `diagnose` internally (do not duplicate collection logic)

### Rule schema (MVP)
- [ ] Implement struct parsing for:
  - [ ] `maxConsole` (by level)
  - [ ] `network.maxFailed`
  - [ ] `network.maxStatus` (min + count)
  - [ ] `selectors[]` with {selector,min,max}
  - [ ] `perf` with {lcpMaxMs, clsMax}

### Selector checks
- [ ] Add a cheap selector count evaluator (JS) returning count only
- [ ] Deterministic preview for failures (first N elements)

### Output + exit codes
- [ ] Output JSON:
  - [ ] `passed` bool
  - [ ] `failedChecks[]` (id/message/context)
  - [ ] minimal diagnostic context (console counts, status counts, perf summary)
- [ ] Exit code: `0` pass, `2` fail

---

## MVP: `html-validate` (lite, report-only)

### CLI
- [ ] Add `html-validate` cobra command (cmd/dev-browser-go/html_validate.go)

### Checks (MVP)
- [ ] Duplicate IDs
- [ ] Missing alt on img
- [ ] Inputs missing accessible name (aria-label / labelledby / label)
- [ ] A few common invalid nesting checks (best-effort)

### Output
- [ ] JSON findings list with:
  - [ ] rule id
  - [ ] message
  - [ ] selector-ish location
  - [ ] snippet

### Exit code
- [ ] Always `0` (report-only)

---

## Cross-cutting

### Correlation + determinism
- [ ] Ensure deterministic ordering (sort by url/method/start time, etc.)
- [ ] Include timestamps consistently (ms since start or ISO)

### Tests
- [ ] Unit tests for rule parsing + assert evaluation (pure logic)
- [ ] Unit tests for html-validate rule functions (pure JS output shape where possible)

### Docs
- [ ] Update README command table with `diagnose`, `assert`, `html-validate`
- [ ] Add examples for a local dev server + CI usage

### Safety/perf
- [ ] Avoid huge bodies by default
- [ ] Truncate big strings
- [ ] Keep default artifact mode minimal

---

## Post-MVP backlog (keep visible)

- [ ] `assert --report ./diagnose.json` (no browser run)
- [ ] Correlate console ↔ network with request IDs
- [ ] Dev overlay detection (Vite/Next/CRA)
- [ ] Axe-core optional a11y audit
- [ ] Long tasks / INP best-effort
