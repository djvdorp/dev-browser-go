# dev-browser-go

Token-light browser automation via Playwright-Go. **CLI-first** design for LLM agent workflows.

Uses ref-based interaction: get a compact accessibility snapshot, then click/fill by ref ID. Keeps context small. Single Go binary with embedded daemon.

## Acknowledgments

Inspired by [SawyerHood/dev-browser](https://github.com/SawyerHood/dev-browser). ARIA snapshot extraction is vendored from that project. Thanks to Sawyer Hood for the original work and ref-based model.

Thanks to Daniel van Dorp (@djvdorp) for early contributions (pip packaging, console logs) and legacy MCP cleanup work.

## Comparison

| Feature | SawyerHood/dev-browser | dev-browser-go |
|---------|------------------------|----------------|
| Language | TypeScript | Go |
| Runtime | Bun + browser extension | Playwright-Go |
| Interface | Browser extension skill | CLI + daemon |
| Install | `.plugin` | Go binary / Nix |
| Best for | Desktop skill users | CLI/LLM agents, Nix users |
| Snapshot engine | ARIA (JS) | Same (vendored) |

## Why CLI (no MCP)

- Lower latency: direct subprocess, no JSON-RPC framing
- Easier debugging: run commands yourself, see stdout/stderr
- Simpler integration: any agent that can shell out works
- Persistent sessions: daemon keeps browser alive between calls

## Install

Playwright browsers are required. The Nix package wraps `PLAYWRIGHT_BROWSERS_PATH` to the packaged Chromium; dev shell includes the driver/browsers. Outside Nix, Playwright-Go will download on first run.

### Nix (flake)

```bash
nix run github:joshp123/dev-browser-go#dev-browser-go -- goto https://example.com
nix profile install github:joshp123/dev-browser-go#dev-browser-go
```

### Go build

```bash
go build ./cmd/dev-browser-go
./dev-browser-go goto https://example.com
./dev-browser-go snapshot
```

## CLI Usage

```bash
dev-browser-go --help              # Full usage
dev-browser-go --version           # Version

dev-browser-go goto https://example.com
dev-browser-go snapshot            # Get refs (e1, e2, ...)
dev-browser-go click-ref e3        # Click ref
dev-browser-go fill-ref e5 "text"  # Fill input
dev-browser-go screenshot          # Capture
dev-browser-go press Enter         # Keyboard
```

The daemon starts automatically on first command and keeps the browser session alive.

### Global Flags

```
--profile <name>    Browser profile (default: "default", env DEV_BROWSER_PROFILE)
--headless          Run headless (default)
--headed            Disable headless
--window-size WxH   Viewport size (default 7680x2160 ultrawide)
--window-scale S    Viewport scale preset (1, 0.75, 0.5)
--device <name>     Device profile name (Playwright)
--output <format>   Output format: summary|json|html|path (default: summary)
--out <path>        Write output to file (with --output=path)
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `DEV_BROWSER_PROFILE` | Browser profile name |
| `HEADLESS` | Override headless default (1/true/yes to enable, 0/false to disable) |
| `DEV_BROWSER_WINDOW_SIZE` | Default viewport size (WxH) |
| `DEV_BROWSER_ALLOW_UNSAFE_PATHS` | Allow artifact writes outside cache dir |

### Viewport + Device Emulation

Viewport only (responsive CSS):
```bash
dev-browser-go --window-size 412x915 goto https://example.com
```

Device profile (UA + DPR + touch + viewport/screen):
```bash
dev-browser-go --device "Galaxy S20 Ultra" goto https://example.com
```
Do not combine `--device` with `--window-size` or `--window-scale`.

List available profiles:
```bash
dev-browser-go devices
```

Note: device profiles use Playwright names; device/viewport flags apply when the daemon starts. Stop the daemon to switch.

## Commands

| Command | Description |
|---------|-------------|
| `goto <url>` | Navigate to URL |
| `snapshot` | Accessibility tree with refs |
| `click-ref <ref>` | Click element by ref |
| `fill-ref <ref> "text"` | Fill input by ref |
| `press <key>` | Keyboard input |
| `screenshot` | Save screenshot (full-page or element crop with padding; crops clamp to 2000x2000) |
| `style-capture` | Capture computed styles (inline or bundled CSS) |
| `bounds` | Get element bounding box (selector/ARIA) |
| `console` | Read page console logs (default levels: info,warning,error) |
| `save-html` | Save page HTML |
| `js-eval` | Evaluate JavaScript and return results |
| `inject` | Inject JavaScript or CSS into page |
| `asset-snapshot` | Save HTML with linked assets for offline review |
| `visual-diff` | Compare current screenshot against baseline |
| `diff-images` | Capture before/after screenshots and save diff image |
| `save-baseline` | Save current page state as visual baseline |
| `devices` | List device profile names |
| `wait` | Wait for page state |
| `list-pages` | Show open pages |
| `close-page <name>` | Close named page |
| `call <tool>` | Generic tool call with JSON args |
| `actions` | Batch tool calls from JSON |
| `status` | Daemon status |
| `start` | Start daemon |
| `stop` | Stop daemon |

Run `dev-browser-go <command> --help` for command-specific options.

### JavaScript Evaluation

Evaluate JavaScript in the page context:
```bash
dev-browser-go js-eval --expr "document.title"
dev-browser-go js-eval --expr "Array.from(document.querySelectorAll('a')).map(a => a.href)"
```

Scope evaluation to an element:
```bash
dev-browser-go js-eval --selector ".header" --expr "this.getBoundingClientRect()"
dev-browser-go js-eval --aria-role "button" --aria-name "Submit" --expr "this.disabled"
```

Read computed styles:
```bash
dev-browser-go js-eval --selector ".button" --expr "getComputedStyle(this).backgroundColor"
```

### JS/CSS Injection

Inject code for prototyping:
```bash
# Inject JavaScript
dev-browser-go inject --script "document.body.style.backgroundColor = 'yellow'"

# Inject CSS
dev-browser-go inject --style "body { background-color: yellow; }"

# Inject from file
dev-browser-go inject --file ./fix.js
dev-browser-go inject --file ./patch.css
```

### Asset Snapshot

Save HTML with assets for offline review:
```bash
dev-browser-go asset-snapshot --path offline.html
```

Configure asset handling:
```bash
dev-browser-go asset-snapshot --path offline.html \
  --asset-types css,js \
  --max-depth 3 \
  --strip-scripts \
  --inline-threshold 20480
```

### Visual Diff / Regression Testing

Save a baseline:
```bash
dev-browser-go goto https://example.com
dev-browser-go save-baseline --path baseline.png --full-page
```

Compare current state:
```bash
dev-browser-go visual-diff --baseline baseline.png \
  --output diff.png \
  --tolerance 0.05 \
  --pixel-threshold 5
```

Element-level baseline:
```bash
dev-browser-go save-baseline --path button-baseline.png \
  --selector ".submit-button" \
  --padding-px 20
```

## Feature Deep Dive

### JavaScript Evaluation

`js-eval` lets you run arbitrary JavaScript in the page context and get the result back. This is useful for:

- **Debugging**: Check variable values, console state, or runtime conditions
- **Data extraction**: Query the DOM without snapshot overhead
- **Computed styles**: Get `getComputedStyle()` results for any element
- **Box model**: Read `getBoundingClientRect()` for layout debugging

**Common patterns:**
```bash
# Page state
dev-browser-go js-eval --expr "document.readyState"
dev-browser-go js-eval --expr "window.scrollY"

# Element inspection
dev-browser-go js-eval --selector ".card" --expr "this.getBoundingClientRect()"
dev-browser-go js-eval --selector ".button" --expr "this.classList.contains('active')"

# Data extraction
dev-browser-go js-eval --expr "Array.from(document.querySelectorAll('.item')).map(el => el.textContent)"
dev-browser-go js-eval --expr "localStorage.getItem('user')"

# Network/Performance
dev-browser-go js-eval --expr "performance.getEntriesByType('resource').length"
```

### JS/CSS Injection

`inject` allows rapid prototyping without rebuilding. You can:

- **Hot-reload styles**: Test CSS changes instantly
- **Patch JavaScript**: Apply quick fixes or overrides
- **Prototype features**: Try out new UI ideas
- **Debug visually**: Change colors, spacing, or visibility

**Quick iteration workflow:**
```bash
# 1. Try a style change
dev-browser-go inject --style ".header { background: red; }"

# 2. Take a screenshot to verify
dev-browser-go screenshot --output test.png

# 3. Iterate with different values
dev-browser-go inject --style ".header { background: blue; padding: 20px; }"

# 4. Save your changes to a file when happy
echo ".header { background: blue; padding: 20px; }" > fix.css
dev-browser-go inject --file fix.css
```

**JavaScript patching:**
```bash
# Override a function
dev-browser-go inject --script "window.fetch = () => console.log('fetch intercepted')"

# Add debugging
dev-browser-go inject --script "document.addEventListener('click', e => console.log(e.target))"

# Mock data
dev-browser-go inject --script "window.MOCK_API = true"
```

### Asset Snapshot

`asset-snapshot` creates a self-contained HTML file for offline sharing or review:

- **Asset discovery**: Finds CSS, JS, fonts, images up to configurable depth
- **Inlining**: Small assets (<10KB by default) are embedded directly
- **Script stripping**: Optional removal of `<script>` tags for security
- **Offline capable**: Works without network after saving

**Use cases:**
```bash
# Share page with design team
dev-browser-go asset-snapshot --path design-review.html

# Archive component for later review
dev-browser-go asset-snapshot --path component-backup.html --selector ".my-component"

# Create test fixture
dev-browser-go asset-snapshot --path test-fixture.html --strip-scripts --asset-types css
```

**Asset filtering:**
```bash
# Only CSS (no JS/images)
dev-browser-go asset-snapshot --path minimal.html --asset-types css

# Inline threshold (reserved for future inlining)
dev-browser-go asset-snapshot --path big.html --inline-threshold 51200

# Deeper scan
dev-browser-go asset-snapshot --path deep.html --max-depth 5
```

Note: asset snapshot preserves original asset URLs; assets are not fetched/embedded yet. Use `--no-include-assets` to skip asset discovery.

### Visual Diff / Regression Testing

`visual-diff` + `save-baseline` enables automated visual regression testing:

- **Pixel-level comparison**: Detect 1px changes
- **Tolerance settings**: Ignore minor color shifts
- **Region ignoring**: Exclude dynamic areas (timestamps, ads)
- **Pass/fail thresholds**: Set acceptable change limits

**Regression workflow:**
```bash
# 1. Save baseline after feature is complete
dev-browser-go goto https://example.com/new-feature
dev-browser-go save-baseline --path baselines/feature.png

# 2. In CI/CD or after changes, compare
dev-browser-go goto https://example.com/new-feature
dev-browser-go visual-diff --baseline baselines/feature.png --output diff.png

# 3. Exit with error if test fails
if ! dev-browser-go visual-diff --baseline baselines/feature.png --pixel-threshold 0; then
  echo "REGRESSION DETECTED!"
  exit 1
fi
```

**Component-level testing:**
```bash
# Test specific component
dev-browser-go save-baseline --path baselines/button.png \
  --selector ".submit-button" --padding-px 20

# Compare (ignoring dynamic content nearby)
dev-browser-go visual-diff --baseline baselines/button.png \
  --ignore 100,0,200,50 # Ignore region with timestamp
```

**Quick before/after diff (no baseline storage):**
```bash
dev-browser-go diff-images --after-wait-ms 1000 --threshold 5 --output json
dev-browser-go diff-images --before baseline.png --after latest.png --diff-path diff.png
```

**Tuning diff sensitivity:**
```bash
# Strict: only exact matches pass
dev-browser-go visual-diff --baseline test.png --tolerance 0 --pixel-threshold 0

# Lenient: allow minor color shifts
dev-browser-go visual-diff --baseline test.png --tolerance 0.05 --pixel-threshold 100
```

## Integration with AI Agents

Add to your project's agent docs (or use [SKILL.md](SKILL.md) directly):

```markdown
## Browser Automation

Use `dev-browser-go` CLI for browser tasks. Keeps context small via ref-based interaction.

Workflow:
1. `dev-browser-go goto <url>` - navigate
2. `dev-browser-go snapshot` - get interactive elements as refs (e1, e2, etc.)
3. `dev-browser-go click-ref <ref>` or `dev-browser-go fill-ref <ref> "text"` - interact
4. `dev-browser-go screenshot` - capture state if needed
```

Element-level capture:
```bash
dev-browser-go bounds ".vault-panel" --nth 1
dev-browser-go screenshot --selector ".vault-panel" --padding-px 10
```

Style capture:
```bash
dev-browser-go goto https://example.com
dev-browser-go style-capture --mode inline --max-nodes 1200
dev-browser-go style-capture --mode bundle --css-path styles.css --selector ".main"
```

Visual diff:
```bash
dev-browser-go goto https://example.com
dev-browser-go save-baseline --path baseline.png
dev-browser-go visual-diff --baseline baseline.png --output diff.png --pixel-threshold 5
```

Diff images (before/after):
```bash
dev-browser-go goto https://example.com
dev-browser-go diff-images --after-wait-ms 1000 --threshold 5 --output json
dev-browser-go diff-images --before baseline.png --after latest.png --diff-path diff.png
```

For detailed workflow examples, see [SKILL.md](SKILL.md).

## Integration with Codex

Codex can use the CLI directly via its shell access. Example prompt:

```
Use dev-browser-go to navigate to example.com and find all links on the page.

Available commands:
- dev-browser-go goto <url>
- dev-browser-go snapshot [--no-interactive-only] [--no-include-headings]
- dev-browser-go click-ref <ref>
- dev-browser-go fill-ref <ref> "text"
- dev-browser-go screenshot
- dev-browser-go style-capture
- dev-browser-go visual-diff
- dev-browser-go diff-images
- dev-browser-go save-baseline
- dev-browser-go js-eval
- dev-browser-go inject
- dev-browser-go asset-snapshot
- dev-browser-go press <key>
- dev-browser-go console [--since <id>] [--limit <n>] [--level <lvl> ...]
```

## Tools

- `goto <url>` - navigate
- `snapshot` - accessibility tree with refs
- `click-ref <ref>` - click element
- `fill-ref <ref> "text"` - fill input
- `press <key>` - keyboard input
- `screenshot` - save screenshot
- `style-capture` - capture computed styles (inline or bundled CSS)
- `visual-diff` - compare current screenshot against baseline
- `diff-images` - capture before/after screenshots and save diff image
- `save-baseline` - save current page state as baseline
- `js-eval` - evaluate JavaScript in page context
- `inject` - inject JavaScript or CSS into page
- `asset-snapshot` - save HTML with linked assets for offline review
- `bounds` - get element bounds (selector/ARIA)
- `console` - read page console logs (default levels: info,warning,error; repeatable `--level`)
- `save-html` - save page HTML
- `wait` - wait for page state
- `list-pages` - show open pages
- `close-page <name>` - close named page
- `call <tool>` - generic tool call with JSON args
- `actions` - batch tool calls from JSON
- `status` / `start` / `stop` - daemon management

## Versioning & Releases

- Simple SemVer tags (`v0.y.z` for fast moves; bump to `v1.0.0` once stable).
- GitHub Release on each tag with the single Go binary (`dev-browser-go`) and checksums.
- Nix flake outputs follow the tag; no extra artifacts.

## License

AGPL-3.0-or-later. See [LICENSE](LICENSE).

Vendored code from SawyerHood/dev-browser is MIT licensed. See [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
