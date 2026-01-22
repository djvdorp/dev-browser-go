# UI/UX Feature Roadmap

Features needed to make `dev-browser-go` feature-complete for UI/UX development work.

## High Priority (Core DevTools)

### 1. CSS Inspector
- [ ] View computed styles for elements
- [ ] Show CSS rules and inline styles
- [ ] Color picker (extract hex/rgb values)
- [ ] Font inspector (family, size, weight, line-height)
- [ ] Box model visualization (margin, border, padding, content)

### 2. JavaScript Console / REPL
- [ ] Execute JavaScript in browser context
- [ ] Inspect variables and objects
- [ ] View return values
- [ ] Multi-line script support

### 3. Network Monitor
- [ ] List network requests (URL, method, status)
- [ ] View request headers and payload
- [ ] View response headers and body
- [ ] Filter by request type/status
- [ ] Search/filter by URL

### 4. Live DOM Inspector
- [ ] Traverse element hierarchy (parent/child/sibling)
- [ ] View element attributes
- [ ] Get CSS selectors and XPath for elements
- [ ] Interactive element picker (click to select)

## Medium Priority (Enhanced Workflow)

### 5. Visual Diff / Regression Testing
- [ ] Compare screenshots against baseline
- [ ] Highlight differences
- [ ] Save/update baselines
- [ ] Compare DOM structure

### 6. Performance Metrics
- [ ] Core Web Vitals (LCP, FID, CLS)
- [ ] Page load timing
- [ ] Resource timing breakdown
- [ ] FPS monitoring

### 7. Element Picker / Selector Generator
- [ ] Click element to select it
- [ ] Generate CSS selector
- [ ] Generate XPath
- [ ] Test selector matches

### 8. Responsive Preview
- [ ] Quick viewport presets (mobile, tablet, desktop)
- [ ] Custom viewport dimensions
- [ ] Orientation toggle (portrait/landscape)
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
