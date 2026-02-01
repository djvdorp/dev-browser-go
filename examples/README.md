# Examples

## React dev harness (Vite default)

Assumed URLs:
- React dev server: `http://localhost:5173`
- Go backend API: `http://localhost:8080`

### Diagnose

```bash
dev-browser-go diagnose --url http://localhost:5173 --output json
```

### Assert (gating)

```bash
dev-browser-go assert --url http://localhost:5173 --rules @examples/assert-react-dev.json --output json
```

Notes:
- This ruleset assumes Vite’s default `#root` exists.
- Adjust selectors and perf budgets per project.
