package devbrowser

import (
	"encoding/base64"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"
)

type NetworkMonitorOptions struct {
	WaitStrategy string
	WaitState    string
	TimeoutMs    int
	MinWaitMs    int

	MaxEntries     int
	IncludeBodies  bool
	MaxBodyBytes   int
	URLContains    string
	MethodEquals   string
	TypeEquals     string
	StatusEquals   int
	StatusMin      int
	StatusMax      int
	OnlyFailed     bool
	IncludeHeaders bool
}

type NetworkEntry struct {
	URL      string `json:"url"`
	Method   string `json:"method"`
	Type     string `json:"type"`
	Started  int64  `json:"started_ms"`
	Finished int64  `json:"finished_ms"`

	Status int    `json:"status"`
	OK     bool   `json:"ok"`
	Error  string `json:"error,omitempty"`

	RequestHeaders  map[string]string `json:"request_headers,omitempty"`
	ResponseHeaders map[string]string `json:"response_headers,omitempty"`

	RequestBody  string `json:"request_body,omitempty"`
	ResponseBody string `json:"response_body,omitempty"`
	BodyEncoding string `json:"body_encoding,omitempty"` // "utf8" or "base64"
	Truncated    bool   `json:"truncated,omitempty"`
}

type NetworkSummary struct {
	Entries []NetworkEntry `json:"entries"`

	Total      int  `json:"total"`
	Matched    int  `json:"matched"`
	Truncated  bool `json:"truncated"`
	FailedOnly bool `json:"failed_only"`
}

func CollectNetwork(page playwright.Page, opts NetworkMonitorOptions) (NetworkSummary, error) {
	// Defaults
	if strings.TrimSpace(opts.WaitStrategy) == "" {
		opts.WaitStrategy = "playwright"
	}
	if strings.TrimSpace(opts.WaitState) == "" {
		opts.WaitState = "networkidle"
	}
	if opts.TimeoutMs <= 0 {
		opts.TimeoutMs = 45_000
	}
	if opts.MaxEntries <= 0 {
		opts.MaxEntries = 200
	}
	if opts.MaxBodyBytes <= 0 {
		opts.MaxBodyBytes = 64 * 1024
	}

	start := time.Now()
	startMS := func() int64 { return time.Since(start).Milliseconds() }

	mu := sync.Mutex{}
	// Use URL+method+startTime as a best-effort key.
	entries := map[string]*NetworkEntry{}
	orderedKeys := []string{}
	truncated := false

	mkKey := func(req playwright.Request) string {
		if req == nil {
			return ""
		}
		return fmt.Sprintf("%s|%s|%d", req.Method(), req.URL(), startMS())
	}

	// Attach listeners.
	page.OnRequest(func(req playwright.Request) {
		if req == nil {
			return
		}
		e := &NetworkEntry{
			URL:     req.URL(),
			Method:  req.Method(),
			Type:    safeString(req.ResourceType()),
			Started: startMS(),
			OK:      false,
		}
		if opts.IncludeHeaders {
			e.RequestHeaders = req.Headers()
		}
		if opts.IncludeBodies {
			if pd, err := req.PostData(); err == nil && strings.TrimSpace(pd) != "" {
				e.RequestBody, e.Truncated, e.BodyEncoding = clampBody(pd, opts.MaxBodyBytes)
			}
		}

		mu.Lock()
		defer mu.Unlock()
		if len(orderedKeys) >= opts.MaxEntries {
			truncated = true
			return
		}
		key := mkKey(req)
		entries[key] = e
		orderedKeys = append(orderedKeys, key)
	})

	page.OnResponse(func(resp playwright.Response) {
		if resp == nil {
			return
		}
		req := resp.Request()
		mu.Lock()
		defer mu.Unlock()
		// Try to find an existing entry by scanning backwards for same URL+method.
		key := ""
		for i := len(orderedKeys) - 1; i >= 0; i-- {
			k := orderedKeys[i]
			e := entries[k]
			if e != nil && req != nil && e.URL == req.URL() && e.Method == req.Method() && e.Status == 0 {
				key = k
				break
			}
		}
		if key == "" {
			// Not found (race / exceeded max / listener late). Ignore.
			return
		}
		e := entries[key]

		status := resp.Status()
		e.Status = status
		e.OK = status >= 200 && status < 400
		e.Finished = startMS()
		if opts.IncludeHeaders {
			e.ResponseHeaders = resp.Headers()
		}
		if opts.IncludeBodies {
			if body, err := resp.Body(); err == nil && len(body) > 0 {
				content := string(body)
				// If it looks binary, base64 it.
				if looksBinary(body) {
					e.ResponseBody = base64.StdEncoding.EncodeToString(clampBytes(body, opts.MaxBodyBytes))
					e.BodyEncoding = "base64"
					e.Truncated = len(body) > opts.MaxBodyBytes
				} else {
					e.ResponseBody, e.Truncated, e.BodyEncoding = clampBody(content, opts.MaxBodyBytes)
				}
			}
		}
	})

	page.OnRequestFailed(func(req playwright.Request) {
		if req == nil {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		// find latest matching entry
		var e *NetworkEntry
		for i := len(orderedKeys) - 1; i >= 0; i-- {
			k := orderedKeys[i]
			cand := entries[k]
			if cand != nil && cand.URL == req.URL() && cand.Method == req.Method() && cand.Status == 0 {
				e = cand
				break
			}
		}
		if e == nil {
			return
		}
		e.OK = false
		e.Finished = startMS()
		if f := req.Failure(); f != nil {
			e.Error = f.Error()
		} else {
			e.Error = "request failed"
		}
	})

	// Wait until stable.
	_, err := waitWithStrategy(page, opts.WaitStrategy, opts.WaitState, opts.TimeoutMs, opts.MinWaitMs)
	if err != nil {
		// still return what we captured
	}

	mu.Lock()
	keys := append([]string{}, orderedKeys...)
	mu.Unlock()

	out := make([]NetworkEntry, 0, len(keys))
	for _, k := range keys {
		mu.Lock()
		e := entries[k]
		mu.Unlock()
		if e == nil {
			continue
		}
		if matchNetwork(*e, opts) {
			out = append(out, *e)
		}
	}

	// Stable sort output by start time.
	sort.Slice(out, func(i, j int) bool {
		if out[i].Started == out[j].Started {
			return out[i].URL < out[j].URL
		}
		return out[i].Started < out[j].Started
	})

	return NetworkSummary{
		Entries:    out,
		Total:      len(keys),
		Matched:    len(out),
		Truncated:  truncated,
		FailedOnly: opts.OnlyFailed,
	}, nil
}

func matchNetwork(e NetworkEntry, opts NetworkMonitorOptions) bool {
	if opts.OnlyFailed && e.OK {
		return false
	}
	if strings.TrimSpace(opts.URLContains) != "" && !strings.Contains(e.URL, opts.URLContains) {
		return false
	}
	if strings.TrimSpace(opts.MethodEquals) != "" && strings.ToUpper(e.Method) != strings.ToUpper(opts.MethodEquals) {
		return false
	}
	if strings.TrimSpace(opts.TypeEquals) != "" && strings.ToLower(e.Type) != strings.ToLower(opts.TypeEquals) {
		return false
	}
	if opts.StatusEquals != 0 && e.Status != opts.StatusEquals {
		return false
	}
	if opts.StatusMin != 0 && e.Status < opts.StatusMin {
		return false
	}
	if opts.StatusMax != 0 && e.Status > opts.StatusMax {
		return false
	}
	return true
}

func looksBinary(b []byte) bool {
	// Heuristic: NUL byte or too many control chars.
	if len(b) == 0 {
		return false
	}
	ctrl := 0
	for _, c := range b {
		if c == 0 {
			return true
		}
		if c < 9 || (c > 13 && c < 32) {
			ctrl++
		}
	}
	return float64(ctrl)/float64(len(b)) > 0.2
}

func safeString(v interface{}) string {
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

func waitWithStrategy(page playwright.Page, strategy, state string, timeoutMs, minWaitMs int) (RunResult, error) {
	// Implement a minimal version by using Playwright's wait.
	// (We keep the signature compatible with runner's wait options, but do not use the perf strategy here.)
	// We prefer "networkidle" when requested.
	if minWaitMs > 0 {
		page.WaitForTimeout(float64(minWaitMs))
	}
	allowedStates := map[string]bool{"load": true, "domcontentloaded": true, "networkidle": true, "commit": true}
	if !allowedStates[strings.ToLower(state)] {
		state = "networkidle"
	}
	var loadState *playwright.LoadState
	switch strings.ToLower(state) {
	case "domcontentloaded", "commit":
		loadState = playwright.LoadStateDomcontentloaded
	case "networkidle":
		loadState = playwright.LoadStateNetworkidle
	default:
		loadState = playwright.LoadStateLoad
	}
	err := page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{State: loadState, Timeout: playwright.Float(float64(timeoutMs))})
	if err != nil {
		return nil, err
	}
	return RunResult{"ok": true}, nil
}
