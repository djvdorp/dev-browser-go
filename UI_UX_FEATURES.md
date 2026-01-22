# UI/UX Feature Gaps

Owner: Josh. Style: telegraph; short clauses ok.

Goal: feature complete for UI/UX change workflows.

## Execution List

- [ ] JS eval tool. Read DOM, computed styles, box model, runtime text.
- [ ] JS/CSS injection. Prototype tweaks in-page, no rebuild.
- [ ] Style capture. Bundle computed styles or inline critical CSS.
- [ ] Asset snapshot. Save HTML plus linked CSS/JS/assets for offline review.
- [x] Visual diff. Before/after screenshots and simple pixel diff.
- [x] Output mode. Return HTML payload on stdout/JSON, not file only.

## Notes

- Keep paths inside artifact dir unless `DEV_BROWSER_ALLOW_UNSAFE_PATHS=1`.
- Prefer single binary flow; tools via `RunCall`/CLI subcommands.
