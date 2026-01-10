# dev-browser-mcp

Token-light browser automation via Playwright. **CLI-first** design for LLM agent workflows.

Uses ref-based interaction: get a compact accessibility snapshot, then click/fill by ref ID. Keeps context small.

## Why CLI over MCP?

MCP adds overhead: extra process, stdio piping, JSON-RPC framing, connection management. For browser automation, that's a lot of indirection when you can just call a CLI.

The CLI approach:
- **Lower latency** - direct subprocess, no protocol overhead
- **Easier debugging** - run commands yourself, see exactly what happens
- **Simpler integration** - any agent that can shell out works
- **Persistent sessions** - daemon keeps browser alive between calls

The MCP server exists if you need it, but the CLI + daemon is the recommended path.

## Install

Requires Python 3.11+ and Playwright browsers.

```bash
# Install playwright browsers (one-time)
playwright install chromium

# Run CLI directly
python cli.py goto https://example.com
python cli.py snapshot
python cli.py click-ref e3
```

Or via Nix (see overlay example in source).

## CLI Usage

```bash
dev-browser goto https://example.com
dev-browser snapshot                    # get refs
dev-browser click-ref e3                # click ref e3
dev-browser fill-ref e5 "search query"  # fill input
dev-browser screenshot
dev-browser press Enter
```

The daemon starts automatically on first command and keeps the browser session alive.

## Integration with Claude Code

Add to your project's `CLAUDE.md`:

```markdown
## Browser Automation

Use `dev-browser` CLI for browser tasks. Keeps context small via ref-based interaction.

Workflow:
1. `dev-browser goto <url>` - navigate
2. `dev-browser snapshot` - get interactive elements as refs (e1, e2, etc.)
3. `dev-browser click-ref <ref>` or `dev-browser fill-ref <ref> "text"` - interact
4. `dev-browser screenshot` - capture state if needed

Example:
\`\`\`bash
dev-browser goto https://github.com/login
dev-browser snapshot
# Output: e1: textbox "Username" | e2: textbox "Password" | e3: button "Sign in"
dev-browser fill-ref e1 "myuser"
dev-browser fill-ref e2 "mypass"
dev-browser click-ref e3
\`\`\`
```

## Integration with Codex

Codex can use the CLI directly via its shell access. Example prompt:

```
Use dev-browser to navigate to example.com and find all links on the page.

Available commands:
- dev-browser goto <url>
- dev-browser snapshot [--interactive-only / --no-interactive-only]
- dev-browser click-ref <ref>
- dev-browser fill-ref <ref> "text"
- dev-browser screenshot
- dev-browser press <key>
```

## Tools

CLI commands (recommended):
- `goto <url>` - navigate
- `snapshot` - accessibility tree with refs
- `click-ref <ref>` - click element
- `fill-ref <ref> "text"` - fill input
- `press <key>` - keyboard input
- `screenshot` - save screenshot
- `save-html` - save page HTML
- `list-pages` - show open pages
- `status` / `start` / `stop` - daemon management

MCP tools (if you must):
- `page` / `list_pages` / `close_page`
- `goto` / `snapshot` / `click_ref` / `fill_ref` / `press`
- `screenshot` / `save_html`
- `actions` - batch calls

## Acknowledgments

This project builds on [SawyerHood/dev-browser](https://github.com/SawyerHood/dev-browser), a Claude Skill for browser automation. The ARIA snapshot extraction logic is vendored from that project (MIT licensed). Thanks to Sawyer Hood for the original work.

## License

AGPL-3.0-or-later. See [LICENSE](LICENSE).

Vendored code from SawyerHood/dev-browser is MIT licensed. See [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
