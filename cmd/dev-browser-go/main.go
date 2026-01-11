package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	devbrowser "github.com/joshp123/dev-browser-go/internal/devbrowser"
)

type globals struct {
	profile  string
	headless bool
	output   string
	outPath  string
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) > 0 && args[0] == "--daemon" {
		return runDaemon(args[1:])
	}

	g, rest, err := parseGlobals(args)
	if err != nil {
		return err
	}
	if len(rest) == 0 {
		return errors.New("command required")
	}

	cmd := rest[0]
	rest = rest[1:]
	switch cmd {
	case "status":
		if devbrowser.IsDaemonHealthy(g.profile) {
			fmt.Printf("ok profile=%s url=%s\n", g.profile, devbrowser.DaemonBaseURL(g.profile))
			return nil
		}
		fmt.Printf("not running profile=%s\n", g.profile)
		return nil

	case "start":
		if err := devbrowser.StartDaemon(g.profile, g.headless); err != nil {
			return err
		}
		fmt.Printf("started profile=%s url=%s\n", g.profile, devbrowser.DaemonBaseURL(g.profile))
		return nil

	case "stop":
		stopped, err := devbrowser.StopDaemon(g.profile)
		if err != nil {
			return err
		}
		if stopped {
			fmt.Printf("stopped profile=%s\n", g.profile)
			return nil
		}
		fmt.Printf("not running profile=%s\n", g.profile)
		return nil

	case "list-pages":
		if err := devbrowser.StartDaemon(g.profile, g.headless); err != nil {
			return err
		}
		base := devbrowser.DaemonBaseURL(g.profile)
		if base == "" {
			return errors.New("daemon state missing after start")
		}
		data, err := devbrowser.HTTPJSON("GET", base+"/pages", nil, 3*time.Second)
		if err != nil {
			return err
		}
		out, err := devbrowser.WriteOutput(g.profile, g.output, map[string]any{"pages": data["pages"]}, g.outPath)
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil

	case "call":
		fs := flag.NewFlagSet("call", flag.ContinueOnError)
		argsJSON := fs.String("args", "{}", "")
		page := fs.String("page", "main", "")
		fs.SetOutput(io.Discard)
		if err := fs.Parse(rest); err != nil {
			return err
		}
		if fs.NArg() < 1 {
			return errors.New("tool name required")
		}
		tool := fs.Arg(0)
		argMap := map[string]interface{}{}
		if err := json.Unmarshal([]byte(*argsJSON), &argMap); err != nil {
			return errors.New("invalid JSON for --args")
		}
		return runWithPage(g, *page, tool, argMap)

	case "goto":
		fs := flag.NewFlagSet("goto", flag.ContinueOnError)
		pageName := fs.String("page", "main", "")
		waitUntil := fs.String("wait-until", "domcontentloaded", "")
		timeout := fs.Int("timeout-ms", 45_000, "")
		fs.SetOutput(io.Discard)
		if err := fs.Parse(rest); err != nil {
			return err
		}
		if fs.NArg() < 1 {
			return errors.New("url required")
		}
		urlVal := fs.Arg(0)
		return runWithPage(g, *pageName, "goto", map[string]interface{}{"url": urlVal, "wait_until": *waitUntil, "timeout_ms": *timeout})

	case "snapshot":
		fs := flag.NewFlagSet("snapshot", flag.ContinueOnError)
		pageName := fs.String("page", "main", "")
		engine := fs.String("engine", "simple", "")
		format := fs.String("format", "list", "")
		interactiveOnly := fs.Bool("interactive-only", true, "")
		includeHeadings := fs.Bool("include-headings", true, "")
		maxItems := fs.Int("max-items", 80, "")
		maxChars := fs.Int("max-chars", 8000, "")
		fs.SetOutput(io.Discard)
		if err := fs.Parse(rest); err != nil {
			return err
		}
		return runWithPage(g, *pageName, "snapshot", map[string]interface{}{
			"engine":           *engine,
			"format":           *format,
			"interactive_only": *interactiveOnly,
			"include_headings": *includeHeadings,
			"max_items":        *maxItems,
			"max_chars":        *maxChars,
		})

	case "click-ref":
		fs := flag.NewFlagSet("click-ref", flag.ContinueOnError)
		pageName := fs.String("page", "main", "")
		timeout := fs.Int("timeout-ms", 15_000, "")
		fs.SetOutput(io.Discard)
		if err := fs.Parse(rest); err != nil {
			return err
		}
		if fs.NArg() < 1 {
			return errors.New("ref required")
		}
		ref := fs.Arg(0)
		return runWithPage(g, *pageName, "click_ref", map[string]interface{}{"ref": ref, "timeout_ms": *timeout})

	case "fill-ref":
		fs := flag.NewFlagSet("fill-ref", flag.ContinueOnError)
		pageName := fs.String("page", "main", "")
		timeout := fs.Int("timeout-ms", 15_000, "")
		fs.SetOutput(io.Discard)
		if err := fs.Parse(rest); err != nil {
			return err
		}
		if fs.NArg() < 2 {
			return errors.New("ref and text required")
		}
		ref := fs.Arg(0)
		text := fs.Arg(1)
		return runWithPage(g, *pageName, "fill_ref", map[string]interface{}{"ref": ref, "text": text, "timeout_ms": *timeout})

	case "press":
		fs := flag.NewFlagSet("press", flag.ContinueOnError)
		pageName := fs.String("page", "main", "")
		fs.SetOutput(io.Discard)
		if err := fs.Parse(rest); err != nil {
			return err
		}
		if fs.NArg() < 1 {
			return errors.New("key required")
		}
		key := fs.Arg(0)
		return runWithPage(g, *pageName, "press", map[string]interface{}{"key": key})

	case "screenshot":
		fs := flag.NewFlagSet("screenshot", flag.ContinueOnError)
		pageName := fs.String("page", "main", "")
		pathArg := fs.String("path", "", "")
		fullPage := fs.Bool("full-page", true, "")
		annotate := fs.Bool("annotate-refs", false, "")
		crop := fs.String("crop", "", "")
		fs.SetOutput(io.Discard)
		if err := fs.Parse(rest); err != nil {
			return err
		}
		payload := map[string]interface{}{"path": *pathArg, "full_page": *fullPage, "annotate_refs": *annotate}
		if strings.TrimSpace(*crop) != "" {
			payload["crop"] = *crop
		}
		return runWithPage(g, *pageName, "screenshot", payload)

	case "save-html":
		fs := flag.NewFlagSet("save-html", flag.ContinueOnError)
		pageName := fs.String("page", "main", "")
		pathArg := fs.String("path", "", "")
		fs.SetOutput(io.Discard)
		if err := fs.Parse(rest); err != nil {
			return err
		}
		return runWithPage(g, *pageName, "save_html", map[string]interface{}{"path": *pathArg})

	case "actions":
		fs := flag.NewFlagSet("actions", flag.ContinueOnError)
		callsArg := fs.String("calls", "", "")
		pageName := fs.String("page", "main", "")
		fs.SetOutput(io.Discard)
		if err := fs.Parse(rest); err != nil {
			return err
		}
		raw := strings.TrimSpace(*callsArg)
		if raw == "" {
			b, err := io.ReadAll(os.Stdin)
			if err != nil {
				return err
			}
			raw = string(b)
		}
		var calls []map[string]interface{}
		if err := json.Unmarshal([]byte(raw), &calls); err != nil {
			return errors.New("invalid JSON for --calls/stdin")
		}
		ws, tid, err := devbrowser.EnsurePage(g.profile, g.headless, *pageName)
		if err != nil {
			return err
		}
		pw, browser, page, err := devbrowser.OpenPage(ws, tid)
		if err != nil {
			return err
		}
		defer browser.Close()
		defer pw.Stop()

		res, err := devbrowser.RunActions(page, calls, devbrowser.ArtifactDir(g.profile))
		if err != nil {
			return err
		}
		output := map[string]any{"results": res.Results}
		if res.Snapshot != "" {
			output["snapshot"] = res.Snapshot
		}
		out, err := devbrowser.WriteOutput(g.profile, g.output, output, g.outPath)
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil

	case "wait":
		fs := flag.NewFlagSet("wait", flag.ContinueOnError)
		pageName := fs.String("page", "main", "")
		strategy := fs.String("strategy", "playwright", "")
		stateVal := fs.String("state", "load", "")
		timeout := fs.Int("timeout-ms", 10_000, "")
		minWait := fs.Int("min-wait-ms", 0, "")
		fs.SetOutput(io.Discard)
		if err := fs.Parse(rest); err != nil {
			return err
		}
		return runWithPage(g, *pageName, "wait", map[string]interface{}{"strategy": *strategy, "state": *stateVal, "timeout_ms": *timeout, "min_wait_ms": *minWait})

	case "close-page":
		fs := flag.NewFlagSet("close-page", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		if err := fs.Parse(rest); err != nil {
			return err
		}
		if fs.NArg() < 1 {
			return errors.New("page name required")
		}
		name := fs.Arg(0)
		if err := devbrowser.StartDaemon(g.profile, g.headless); err != nil {
			return err
		}
		base := devbrowser.DaemonBaseURL(g.profile)
		if base == "" {
			return errors.New("daemon state missing after start")
		}
		encoded := url.PathEscape(name)
		data, err := devbrowser.HTTPJSON("DELETE", base+"/pages/"+encoded, nil, 5*time.Second)
		if err != nil {
			return err
		}
		if ok, _ := data["ok"].(bool); !ok {
			return fmt.Errorf("close failed: %v", data["error"])
		}
		out, err := devbrowser.WriteOutput(g.profile, g.output, map[string]any{"page": name, "closed": true}, g.outPath)
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
	}

	return fmt.Errorf("unknown command: %s", cmd)
}

func runWithPage(g globals, pageName string, tool string, args map[string]interface{}) error {
	ws, tid, err := devbrowser.EnsurePage(g.profile, g.headless, pageName)
	if err != nil {
		return err
	}
	pw, browser, page, err := devbrowser.OpenPage(ws, tid)
	if err != nil {
		return err
	}
	defer browser.Close()
	defer pw.Stop()

	res, err := devbrowser.RunCall(page, tool, args, devbrowser.ArtifactDir(g.profile))
	if err != nil {
		return err
	}
	out, err := devbrowser.WriteOutput(g.profile, g.output, res, g.outPath)
	if err != nil {
		return err
	}
	fmt.Println(out)
	return nil
}

func runDaemon(args []string) error {
	profile := getenvDefault("DEV_BROWSER_PROFILE", "default")
	host := getenvDefault("DEV_BROWSER_HOST", "127.0.0.1")
	port := getenvInt("DEV_BROWSER_PORT", 0)
	cdpPort := getenvInt("DEV_BROWSER_CDP_PORT", 0)
	headless := envTruthy("HEADLESS")
	stateFile := getenvDefault("DEV_BROWSER_STATE_FILE", "")

	fs := flag.NewFlagSet("dev-browser-go-daemon", flag.ContinueOnError)
	fs.StringVar(&profile, "profile", profile, "")
	fs.StringVar(&host, "host", host, "")
	fs.IntVar(&port, "port", port, "")
	fs.IntVar(&cdpPort, "cdp-port", cdpPort, "")
	fs.BoolVar(&headless, "headless", headless, "")
	fs.StringVar(&stateFile, "state-file", stateFile, "")
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return err
	}

	logger := log.New(os.Stderr, "", log.LstdFlags)
	return devbrowser.ServeDaemon(devbrowser.DaemonOptions{Profile: profile, Host: host, Port: port, CDPPort: cdpPort, Headless: headless, StateFile: stateFile, Logger: logger})
}

func parseGlobals(args []string) (globals, []string, error) {
	g := globals{
		profile:  getenvDefault("DEV_BROWSER_PROFILE", "default"),
		headless: envTruthy("HEADLESS"),
		output:   "summary",
		outPath:  "",
	}

	remaining := []string{}
	i := 0
	for i < len(args) {
		a := args[i]
		switch a {
		case "--profile":
			if i+1 >= len(args) {
				return g, nil, errors.New("--profile requires value")
			}
			g.profile = args[i+1]
			i += 2
		case "--headless":
			g.headless = true
			i++
		case "--output":
			if i+1 >= len(args) {
				return g, nil, errors.New("--output requires value")
			}
			g.output = args[i+1]
			i += 2
		case "--out":
			if i+1 >= len(args) {
				return g, nil, errors.New("--out requires value")
			}
			g.outPath = args[i+1]
			i += 2
		default:
			remaining = args[i:]
			i = len(args)
		}
	}

	if g.output != "summary" && g.output != "json" && g.output != "path" {
		return g, nil, errors.New("--output must be summary|json|path")
	}

	return g, remaining, nil
}

func getenvDefault(name, def string) string {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return def
	}
	return v
}

func getenvInt(name string, def int) int {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func envTruthy(name string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}
