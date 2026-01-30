# UI/UX Feature Roadmap

Features needed to make `dev-browser-go` feature-complete for UI/UX development work.

## High Priority (Core DevTools)

### 1. CSS Inspector
- [x] View computed styles for elements (via `style-capture` + `js-eval getComputedStyle`)
- [x] Show CSS rules and inline styles (via `style-capture` bundle/inline + `save-html`/`asset-snapshot`)
- [ ] Color picker (extract hex/rgb values)
- [ ] Font inspector (family, size, weight, line-height)
- [x] Box model visualization (margin, border, padding, content) (via `bounds` + `js-eval getBoundingClientRect`)

### 2. JavaScript Console / REPL
- [x] Execute JavaScript in browser context (`js-eval`)
- [x] Inspect variables and objects (`js-eval` JSON output)
- [x] View return values (`js-eval`)
- [ ] Multi-line script support (workaround: use `inject --file` for larger scripts)

### 3. Network Monitor
- [x] List network requests (URL, method, status) (`network-monitor`)
- [x] View request headers and payload (`--headers`, `--bodies`)
- [x] View response headers and body (`--headers`, `--bodies`)
- [x] Filter by request type/status (`--type`, `--status`, `--status-min`, `--status-max`, `--failed`)
- [x] Search/filter by URL (`--url-contains`)

### 4. Live DOM Inspector
- [x] Traverse element hierarchy (parent/child/sibling) (via `snapshot --format tree`)
- [x] View element attributes (via `inspect-ref`)
- [x] Get CSS selectors and XPath for elements (via `inspect-ref`)
- [x] Interactive element picker (headless) (use `snapshot` refs + `inspect-ref` / `test-selector` / `test-xpath`)

## Medium Priority (Enhanced Workflow)

### 5. Visual Diff / Regression Testing
- [x] Compare screenshots against baseline (`visual-diff`)
- [x] Highlight differences
- [x] Save/update baselines (`save-baseline`)
- [x] Compare DOM structure (`save-dom-baseline` + `dom-diff`)

### 6. Performance Metrics
- [x] Core Web Vitals (LCP, CLS best-effort) (`perf-metrics`)
- [x] Page load timing (`perf-metrics` navigation timing)
- [x] Resource timing breakdown (`perf-metrics` resource summary)
- [x] FPS monitoring (`perf-metrics` rAF sample)

### 7. Element Picker / Selector Generator
- [x] Select element (headless) (`snapshot` ref)
- [x] Generate CSS selector (`inspect-ref`)
- [x] Generate XPath (`inspect-ref`)
- [x] Test selector matches (`test-selector`, `test-xpath`)

### 8. Responsive Preview
- [x] Quick viewport presets (mobile, tablet, desktop) (`--device` + `devices`)
- [x] Custom viewport dimensions (`--window-size WxH`)
- [x] Orientation toggle (portrait/landscape) (via device profiles / WxH swap)
- [ ] Side-by-side device comparison

## Nice to Have

### 9. Color Contrast Checker
- [ ] Calculate contrast ratios
- [ ] WCAG compliance checking (AA/AAA)
- [ ] Visual pass/fail indicators

### 10. Live Reload / HMR
- [ ] Watch file system for changes
- [ ] Auto-refresh pages on change
- [ ] Support for HMR frameworks

### 11. Annotation Tools
- [ ] Add notes to screenshots
- [ ] Draw shapes/markers
- [ ] Export annotated images

### 12. Session Export / Import
- [ ] Save browser state to file
- [ ] Restore browser session
- [ ] Share sessions with team

## Implementation Notes

- Each feature should follow existing command pattern (Cobra CLI)
- Support `--output` flags: `summary`, `json`, `path`
- Use existing runner architecture for browser interaction
- Device emulation already exists - can be leveraged for responsive testing
