# Harness MVP (Decision + Backlog)

## Decision

We will prioritize a **harness** UX over a toolbox UX.

**Goal:** enable a CLI agent to iteratively build and verify a React frontend against a Go backend **autonomously**, using a small number of high-signal commands that:

- produce **structured JSON** outputs
- write **artifacts** (screenshots/snapshots/logs) into a run directory
- provide **deterministic pass/fail** via exit codes

This is intentionally *not* a human DevTools clone.

---

## MVP scope (v0)

### 1) `diagnose`

**Intent:** One-call “what’s broken?” report suitable for agent loops.

**CLI**

```bash
# Basic
dev-browser-go diagnose --url http://localhost:5173 --output json

# Use an existing page (no navigation)
dev-browser-go diagnose --page main --output json

# Tuning
dev-browser-go diagnose --url http://localhost:5173 \
  --wait networkidle --timeout-ms 45000 --min-wait-ms 250 \
  --net-bodies --net-max-body-bytes 32768 \
  --snapshot-engine aria \
  --perf-sample-ms 1200 --perf-top-n 20 \
  --out ./artifacts/run-001 --output json
```

**What it runs (internally)**

- optional `goto` (if `--url` provided)
- `wait` (default `networkidle`)
- `console` (capture info/warn/error; include stack/source when present)
- `network-monitor` (requests/status/headers; bodies optional)
- `perf-metrics` (timing/resources + CWV best-effort + FPS)
- `snapshot` (default engine `simple`, option `aria`) + include `items`
- `screenshot` (full page)

**Output schema (JSON)**

Top-level fields:

```json
{
  "meta": {
    "url": "...",
    "page": "main",
    "profile": "default",
    "ts": "ISO-8601",
    "runId": "uuid",
    "artifactDir": "..."
  },
  "console": { "entries": [], "counts": {"error":0,"warning":0,"info":0} },
  "network": { "total": 0, "matched": 0, "entries": [] },
  "perf": { /* perf-metrics output */ },
  "snapshot": { "engine": "simple|aria", "yaml": "...", "items": [] },
  "artifacts": {
    "screenshot": "path",
    "snapshot": "path(optional)",
    "network": "path(optional)",
    "console": "path(optional)"
  },
  "summary": {
    "hasConsoleErrors": true,
    "hasHttp4xx5xx": true,
    "hasFailedRequests": true
  }
}
```

**Exit code**

- `0` always for `diagnose` (report-only; never “fails”)

---

### 2) `assert`

**Intent:** Deterministic gating for agents/CI. Fails with non-zero exit code if expectations aren’t met.

**CLI**

```bash
# Inline rules file (recommended)
dev-browser-go assert --url http://localhost:5173 --rules ./assert.json

# Or pass rules as JSON string
dev-browser-go assert --url http://localhost:5173 --rules @./assert.json
```

**Rules (JSON) — MVP**

```json
{
  "maxConsole": {"error": 0},
  "network": {
    "maxFailed": 0,
    "maxStatus": {"min": 400, "count": 0}
  },
  "selectors": [
    {"selector": ".error", "max": 0},
    {"selector": "[data-testid='app-root']", "min": 1}
  ],
  "perf": {"lcpMaxMs": 2500, "clsMax": 0.1}
}
```

**Behavior**

- Runs `diagnose` internally (or uses a saved report later as a follow-up enhancement).
- Evaluates rule checks.

**Output**

- JSON with `passed: true|false`, a list of failed checks, plus a subset of diagnostic context.

**Exit code**

- `0` if all checks pass
- `2` if any check fails

---

### 3) `html-validate` (lite)

**Intent:** Catch obvious markup issues without requiring an external W3C service.

**Checks (MVP)**

- duplicate IDs
- missing `alt` on `img`
- form controls missing accessible name (very basic)
- invalid nesting for a few common cases (best-effort)

**Output**

- JSON list of findings with selector-ish location hints.

**Exit code**

- `0` always (report-only) in MVP; later wire into `assert`.

---

## Non-goals (MVP)

- Full Chrome DevTools feature parity
- Full W3C validator parity
- Headed UI workflows (must work headless)

---

## Backlog (post-MVP)

### Harness quality
- Correlate console entries ↔ network requests (timestamps + request IDs)
- Persist `diagnose` reports + allow `assert --report ./report.json`
- Add `--artifact-mode` (none|minimal|full)

### React/dev-server specifics
- Detect common dev overlays (Next/Vite/CRA) and expose as `summary.devOverlayDetected`
- Better stack/source capture from console logs

### Performance
- Add best-effort INP/longtask metrics
- Add resource budget summaries (total JS/CSS bytes, top offenders)

### Accessibility
- Expand a11y lint checks; optionally integrate axe-core (vendored, headless)

---

## Notes

- Always favor: **structured outputs**, **stable schemas**, **low flake**, **headless-first**.
- Prefer single-run “report” commands over many tiny commands for agent loops.
