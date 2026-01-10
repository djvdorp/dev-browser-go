# dev-browser-mcp

Token-light browser automation via Playwright. CLI + MCP server + daemon.

Uses ref-based interaction: get a compact accessibility snapshot, then click/fill by ref ID. Keeps context small for LLM workflows.

## Install

Requires Python 3.11+ and Playwright browsers.

```bash
# Install playwright browsers (one-time)
playwright install chromium

# Run directly
python server.py        # MCP stdio server
python daemon.py        # HTTP daemon
python cli.py goto https://example.com
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

## MCP Tools

- `page` / `list_pages` / `close_page` - page management
- `goto` - navigate to URL
- `snapshot` - accessibility tree with refs
- `click_ref` / `fill_ref` - interact via refs
- `press` - keyboard input
- `screenshot` / `save_html` - save artifacts
- `actions` - batch multiple calls

## Acknowledgments

This project builds on [SawyerHood/dev-browser](https://github.com/SawyerHood/dev-browser), a Claude Skill for browser automation. The ARIA snapshot extraction logic is vendored from that project (MIT licensed). Thanks to Sawyer Hood for the original work.

## License

AGPL-3.0-or-later. See [LICENSE](LICENSE).

Vendored code from SawyerHood/dev-browser is MIT licensed. See [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
