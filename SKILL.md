---
name: dev-browser-go
description: Browser automation with persistent sessions via CLI. Use when users ask to navigate websites, fill forms, take screenshots, test web apps, or automate browser workflows. Trigger phrases include "go to [url]", "click on", "fill out the form", "take a screenshot", "test the website", or any browser interaction request.
---

# Dev Browser Skill (CLI)

Browser automation that maintains page state via a persistent daemon. Uses ref-based interaction for token efficiency.

## When to Use

- Testing web apps during development
- Filling forms, clicking buttons
- Taking screenshots
- Scraping structured data
- Any browser automation task

## Quick Start

The daemon starts automatically on first command. Just run:

```bash
dev-browser-go goto https://example.com
dev-browser-go snapshot
dev-browser-go click-ref e3
```

Run `dev-browser-go --help` for full CLI reference.

## Output

```bash
dev-browser-go snapshot --output summary    # Default text output
dev-browser-go snapshot --output json       # JSON payload
dev-browser-go save-html                    # Save page HTML to default artifact path
dev-browser-go save-html --path page.html   # Save page HTML to specified path
```

Note: `--output` and `--out` are global serialization flags (controls format/destination of the
command result JSON). `--path` on `save-html` (and `asset-snapshot`, `save-baseline`) is the
artifact file path written by the tool itself.

## Core Workflow

1. **Navigate** to a URL
2. **Snapshot** to get interactive elements as refs (e1, e2, etc.)
3. **Interact** using refs (click, fill, press)
4. **Screenshot** if visual verification needed

### Example: Login Flow

```bash
# Navigate to login page
dev-browser-go goto https://github.com/login

# Get interactive elements
dev-browser-go snapshot
# Output:
# e1: textbox "Username or email address"
# e2: textbox "Password"
# e3: button "Sign in"

# Fill and submit
dev-browser-go fill-ref e1 "myusername"
dev-browser-go fill-ref e2 "mypassword"
dev-browser-go click-ref e3

# Verify result
dev-browser-go snapshot
```

## Commands Reference

### Navigation & Pages
```bash
dev-browser-go goto <url>                    # Navigate to URL
dev-browser-go goto <url> --page checkout    # Use named page
dev-browser-go list-pages                    # List open pages
dev-browser-go close-page <name>             # Close named page
```

### Inspection
```bash
dev-browser-go snapshot                      # Get refs for interactive elements
dev-browser-go snapshot --no-interactive-only  # Include all elements
dev-browser-go snapshot --engine aria        # Use ARIA engine (better for complex UIs)
dev-browser-go screenshot                    # Full-page screenshot
dev-browser-go screenshot --annotate-refs    # Overlay ref labels on screenshot
dev-browser-go screenshot --selector ".panel" --padding-px 10  # Element crop + padding
dev-browser-go screenshot --crop 0,0,800,600 # Crop region (max 2000x2000)
dev-browser-go bounds ".panel" --nth 1      # Element bounds (CSS or ARIA)
dev-browser-go save-html --path page.html    # Save page HTML
dev-browser-go style-capture --mode inline   # Inline computed styles
dev-browser-go style-capture --mode bundle --css-path styles.css --selector ".panel"
dev-browser-go js-eval --expr "document.title"  # Evaluate JavaScript
dev-browser-go js-eval --selector ".btn" --expr "this.textContent"
dev-browser-go asset-snapshot --path offline.html  # Save with assets
dev-browser-go save-baseline --path baseline.png   # Save visual baseline
dev-browser-go visual-diff --baseline baseline.png # Compare against baseline
dev-browser-go diff-images --after-wait-ms 1000 --threshold 5
```

### Interaction
```bash
dev-browser-go click-ref <ref>               # Click element by ref
dev-browser-go fill-ref <ref> "text"         # Fill input by ref
dev-browser-go press Enter                   # Press key
dev-browser-go press Tab                     # Navigate with Tab
dev-browser-go press Escape                  # Close modals
```

### Waiting
```bash
dev-browser-go wait                          # Wait for page load
dev-browser-go wait --state networkidle      # Wait for network idle
dev-browser-go wait --timeout-ms 5000        # Custom timeout
```

### JavaScript Evaluation
```bash
# Positional expression (shorthand)
dev-browser-go js-eval "document.title"

# Get page title
dev-browser-go js-eval --expr "document.title"

# Read computed styles
dev-browser-go js-eval --selector ".header" --expr "getComputedStyle(this).backgroundColor"

# Get element text
dev-browser-go js-eval --selector "h1" --expr "this.textContent"

# Check element state
dev-browser-go js-eval --selector ".button" --expr "this.disabled"

# Get box model
dev-browser-go js-eval --selector ".card" --expr "this.getBoundingClientRect()"

# Complex queries
dev-browser-go js-eval --expr "Array.from(document.querySelectorAll('a')).length"
```

### JS/CSS Injection (Prototyping)
```bash
# Inject JavaScript
dev-browser-go inject --script "document.body.style.backgroundColor = 'yellow'"

# Inject CSS
dev-browser-go inject --style "body { background-color: yellow; } .header { color: red; }"

# Inject from file
dev-browser-go inject --file ./fix.js
dev-browser-go inject --file ./patch.css

# Prototype without rebuild - quick feedback loop!
dev-browser-go inject --style ".button { background: blue; }" && dev-browser-go screenshot
```

### Asset Snapshot (Offline Review)
```bash
# Save with all assets
dev-browser-go asset-snapshot --path offline.html

# Customize asset handling
dev-browser-go asset-snapshot --path offline.html \
  --asset-types css,js \
  --max-depth 3 \
  --strip-scripts \
  --inline-threshold 20480

# Share with team for offline review
```

Note: asset snapshot keeps original asset URLs; assets are not fetched/embedded yet. Use `--no-include-assets` to skip discovery.

### Visual Diff / Regression Testing
```bash
# Save baseline
dev-browser-go goto https://example.com
dev-browser-go save-baseline --path baseline.png

# After changes, compare
dev-browser-go visual-diff --baseline baseline.png \
  --output diff.png \
  --tolerance 0.05 \
  --pixel-threshold 5

# Quick before/after diff (no baseline storage)
dev-browser-go diff-images --after-wait-ms 1000 --threshold 5 --output json
dev-browser-go diff-images --before baseline.png --after latest.png --diff-path diff.png

# Element-level regression testing
dev-browser-go save-baseline --path button-baseline.png \
  --selector ".submit-button" --padding-px 20

# Later, check for regressions
dev-browser-go visual-diff --baseline button-baseline.png --pixel-threshold 2
```

### Batch Actions
```bash
# Execute multiple actions in one call
echo '[{"tool":"click_ref","args":{"ref":"e1"}},{"tool":"press","args":{"key":"Enter"}}]' | dev-browser-go actions
```

### Daemon Management
```bash
dev-browser-go status                        # Check daemon status
dev-browser-go stop                          # Stop daemon (closes browser)
dev-browser-go start --headless              # Start in headless mode
```

## Interpreting Snapshots

Snapshot output looks like:
```
e1: textbox "Search" [placeholder: "Type to search..."]
e2: button "Submit" [disabled]
e3: link "Home" [/url: /home]
e4: checkbox "Remember me" [checked]
e5: combobox "Country" [expanded]
```

- `eN` - Element reference for interaction
- `[disabled]`, `[checked]`, `[expanded]` - Element states
- `[placeholder: ...]`, `[/url: ...]` - Element properties

## Tips

### Small Steps
Run one action at a time, check output, then proceed. Don't chain multiple actions blindly.

### Use Named Pages
For multi-page workflows, use `--page` to keep contexts separate:
```bash
dev-browser-go goto https://app.com/settings --page settings
dev-browser-go goto https://app.com/profile --page profile
dev-browser-go snapshot --page settings  # Back to settings
```

### Headless Mode
Default is headless. To disable:
```bash
dev-browser-go start --headed
# or
HEADLESS=0 dev-browser-go goto https://example.com
```

### Viewport Size
Default is ultrawide 7680x2160. Adjust with flags:
```bash
dev-browser-go goto https://example.com --window-scale 0.75  # 5760x1620 (0.75x default)
# or
dev-browser-go goto https://example.com --window-size 3840x1080
```
Env default:
```bash
DEV_BROWSER_WINDOW_SIZE=412x915 dev-browser-go goto https://example.com
```

### Device Emulation
Use Playwright device profiles for UA + DPR + touch + viewport/screen:
```bash
dev-browser-go devices
dev-browser-go --device "Galaxy S20 Ultra" goto https://example.com
```
Do not combine `--device` with `--window-size` or `--window-scale`.
Note: device/viewport flags apply when the daemon starts. Stop the daemon to switch.

### Element Screenshots
For component-level captures, use CSS selectors from your codebase:
```bash
dev-browser-go bounds ".vault-panel"                        # Verify selector + check dimensions
dev-browser-go screenshot --selector ".vault-panel" --padding-px 8
```

Tips:
- Use class selectors from your source code — you know your component names
- Check `bounds` first to verify dimensions match expected area
- Small padding (8-16px) for tight crops

### Debugging
If something isn't working:
```bash
dev-browser-go screenshot                    # See current state
dev-browser-go snapshot --no-interactive-only  # See all elements
```

## See Also

- `dev-browser-go --help` for full CLI reference
- [README.md](README.md) for installation and architecture
