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
```

### Inspection
```bash
dev-browser-go snapshot                      # Get refs for interactive elements
dev-browser-go snapshot --no-interactive-only  # Include all elements
dev-browser-go snapshot --engine aria        # Use ARIA engine (better for complex UIs)
DEV_BROWSER_WINDOW_SIZE=7680x2160 dev-browser-go screenshot   # Full-page at ultrawide viewport (will be downscaled by models)
dev-browser-go screenshot --annotate-refs    # Overlay ref labels on screenshot
dev-browser-go screenshot --crop 0,0,2000,2000 # Crop region (hard clamp 2000x2000; models degrade badly above this)
dev-browser-go save-html                     # Save page HTML
```

### Interaction
```bash
dev-browser-go click-ref <ref>               # Click element by ref
dev-browser-go fill-ref <ref> "text"         # Fill input by ref
dev-browser-go press Enter                   # Press key
dev-browser-go press Tab                     # Navigate with Tab
dev-browser-go press Escape                  # Close modals
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
For CI or background tasks:
```bash
dev-browser-go stop
dev-browser-go start --headless
dev-browser-go goto https://example.com
```

### Debugging
If something isn't working:
```bash
dev-browser-go screenshot  # See current state
dev-browser-go snapshot --no-interactive-only  # See all elements
# Force viewport size (default 2500x1920, env overrides)
DEV_BROWSER_WINDOW_SIZE=7680x2160 dev-browser-go snapshot
```

## Comparison with SawyerHood/dev-browser-go

This is a Python/CLI rewrite of [SawyerHood/dev-browser-go](https://github.com/SawyerHood/dev-browser-go). Same ref-based model, different interface:

| This (CLI) | Sawyer's (Skill) |
|------------|------------------|
| Shell commands | TypeScript scripts |
| `dev-browser-go` CLI | `npx tsx` heredocs |
| Python + Playwright | Node + Playwright |
| Simpler, lower context | More features (extension mode) |

Consider Sawyer's original if you want Chrome extension mode (control existing logged-in browser).
