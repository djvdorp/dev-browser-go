package devbrowser

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type AssertRules struct {
	MaxConsole map[string]int   `json:"maxConsole,omitempty"`
	Network    *AssertNetwork   `json:"network,omitempty"`
	Selectors  []AssertSelector `json:"selectors,omitempty"`
	Perf       *AssertPerf      `json:"perf,omitempty"`
}

type AssertNetwork struct {
	MaxFailed int                `json:"maxFailed,omitempty"`
	MaxStatus *AssertStatusCount `json:"maxStatus,omitempty"`
}

type AssertStatusCount struct {
	Min   int `json:"min"`
	Count int `json:"count"`
}

type AssertSelector struct {
	Selector string `json:"selector"`
	Min      *int   `json:"min,omitempty"`
	Max      *int   `json:"max,omitempty"`
}

type AssertPerf struct {
	LCPMaxMs *float64 `json:"lcpMaxMs,omitempty"`
	CLSMax   *float64 `json:"clsMax,omitempty"`
}

type AssertFailedCheck struct {
	ID      string         `json:"id"`
	Message string         `json:"message"`
	Context map[string]any `json:"context,omitempty"`
}

type AssertResult struct {
	Passed       bool                `json:"passed"`
	FailedChecks []AssertFailedCheck `json:"failedChecks"`
	Context      map[string]any      `json:"context,omitempty"`
}

func ParseAssertRules(raw string) (*AssertRules, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New("--rules is required")
	}
	if strings.HasPrefix(raw, "@") {
		path := strings.TrimSpace(strings.TrimPrefix(raw, "@"))
		if path == "" {
			return nil, errors.New("--rules @path requires a non-empty path")
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		raw = string(b)
	}

	var rules AssertRules
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&rules); err != nil {
		return nil, err
	}
	if len(rules.Selectors) > 0 {
		for i := range rules.Selectors {
			rules.Selectors[i].Selector = strings.TrimSpace(rules.Selectors[i].Selector)
			if rules.Selectors[i].Selector == "" {
				return nil, fmt.Errorf("selectors[%d].selector is required", i)
			}
		}
	}
	return &rules, nil
}

// EvaluateAssert evaluates rules against a DiagnoseReport and any selector counts already collected.
// This function is pure logic and should remain unit-testable.
func EvaluateAssert(report *DiagnoseReport, rules *AssertRules, selectorCounts map[string]int, perfOverride map[string]any) AssertResult {
	res := AssertResult{Passed: true, FailedChecks: []AssertFailedCheck{}}
	if report == nil || rules == nil {
		res.Passed = false
		res.FailedChecks = append(res.FailedChecks, AssertFailedCheck{ID: "invalid-input", Message: "missing report or rules"})
		return res
	}

	// Context subset (stable fields only).
	ctx := map[string]any{
		"console": map[string]any{"counts": report.Console.Counts},
		"network": map[string]any{"matched": report.Network.Matched, "total": report.Network.Total},
	}
	perf := report.Perf
	if perfOverride != nil {
		perf = perfOverride
	}
	if perf != nil {
		ctx["perf"] = bestEffortPerfSummary(perf)
	}
	res.Context = ctx

	// maxConsole.
	if len(rules.MaxConsole) > 0 {
		// deterministic check order by level.
		levels := make([]string, 0, len(rules.MaxConsole))
		for k := range rules.MaxConsole {
			levels = append(levels, k)
		}
		sort.Strings(levels)
		for _, level := range levels {
			max := rules.MaxConsole[level]
			count := 0
			switch strings.ToLower(level) {
			case "error", "errors":
				count = report.Console.Counts.Error
			case "warning", "warn", "warnings":
				count = report.Console.Counts.Warning
			case "info":
				count = report.Console.Counts.Info
			default:
				res.Passed = false
				res.FailedChecks = append(res.FailedChecks, AssertFailedCheck{ID: "rules.maxConsole", Message: fmt.Sprintf("unknown console level '%s'", level)})
				continue
			}
			if count > max {
				res.Passed = false
				res.FailedChecks = append(res.FailedChecks, AssertFailedCheck{
					ID:      "console.max",
					Message: fmt.Sprintf("console %s count %d > max %d", strings.ToLower(level), count, max),
					Context: map[string]any{"level": strings.ToLower(level), "count": count, "max": max},
				})
			}
		}
	}

	// network rules.
	if rules.Network != nil {
		{
			failed := 0
			for _, e := range report.Network.Entries {
				if !e.OK || strings.TrimSpace(e.Error) != "" {
					failed++
				}
			}
			if failed > rules.Network.MaxFailed {
				res.Passed = false
				res.FailedChecks = append(res.FailedChecks, AssertFailedCheck{ID: "network.maxFailed", Message: fmt.Sprintf("failed requests %d > max %d", failed, rules.Network.MaxFailed), Context: map[string]any{"failed": failed, "max": rules.Network.MaxFailed}})
			}
		}
		if rules.Network.MaxStatus != nil {
			min := rules.Network.MaxStatus.Min
			countMax := rules.Network.MaxStatus.Count
			count := 0
			for _, e := range report.Network.Entries {
				if e.Status >= min {
					count++
				}
			}
			if count > countMax {
				res.Passed = false
				res.FailedChecks = append(res.FailedChecks, AssertFailedCheck{ID: "network.maxStatus", Message: fmt.Sprintf("responses with status >= %d: %d > max %d", min, count, countMax), Context: map[string]any{"min": min, "count": count, "max": countMax}})
			}
		}
	}

	// selector checks.
	for _, sel := range rules.Selectors {
		count := selectorCounts[sel.Selector]
		if sel.Min != nil && count < *sel.Min {
			res.Passed = false
			res.FailedChecks = append(res.FailedChecks, AssertFailedCheck{ID: "selectors.min", Message: fmt.Sprintf("selector %q count %d < min %d", sel.Selector, count, *sel.Min), Context: map[string]any{"selector": sel.Selector, "count": count, "min": *sel.Min}})
		}
		if sel.Max != nil && count > *sel.Max {
			res.Passed = false
			res.FailedChecks = append(res.FailedChecks, AssertFailedCheck{ID: "selectors.max", Message: fmt.Sprintf("selector %q count %d > max %d", sel.Selector, count, *sel.Max), Context: map[string]any{"selector": sel.Selector, "count": count, "max": *sel.Max}})
		}
	}

	// perf checks.
	if rules.Perf != nil {
		if rules.Perf.LCPMaxMs != nil {
			if lcp, ok := extractFloat(perf, "cwv", "lcp"); ok {
				if lcp > *rules.Perf.LCPMaxMs {
					res.Passed = false
					res.FailedChecks = append(res.FailedChecks, AssertFailedCheck{ID: "perf.lcpMaxMs", Message: fmt.Sprintf("LCP %.1fms > max %.1fms", lcp, *rules.Perf.LCPMaxMs), Context: map[string]any{"lcpMs": lcp, "maxMs": *rules.Perf.LCPMaxMs}})
				}
			}
		}
		if rules.Perf.CLSMax != nil {
			if cls, ok := extractFloat(perf, "cwv", "cls"); ok {
				if cls > *rules.Perf.CLSMax {
					res.Passed = false
					res.FailedChecks = append(res.FailedChecks, AssertFailedCheck{ID: "perf.clsMax", Message: fmt.Sprintf("CLS %.3f > max %.3f", cls, *rules.Perf.CLSMax), Context: map[string]any{"cls": cls, "max": *rules.Perf.CLSMax}})
				}
			}
		}
	}

	// deterministic ordering for failed checks.
	sort.Slice(res.FailedChecks, func(i, j int) bool {
		if res.FailedChecks[i].ID == res.FailedChecks[j].ID {
			return res.FailedChecks[i].Message < res.FailedChecks[j].Message
		}
		return res.FailedChecks[i].ID < res.FailedChecks[j].ID
	})

	return res
}

func extractFloat(m map[string]any, keys ...string) (float64, bool) {
	cur := any(m)
	for _, k := range keys {
		mm, ok := cur.(map[string]any)
		if !ok {
			return 0, false
		}
		cur, ok = mm[k]
		if !ok {
			return 0, false
		}
	}
	switch v := cur.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	default:
		return 0, false
	}
}

func bestEffortPerfSummary(perf map[string]any) map[string]any {
	out := map[string]any{}
	if lcp, ok := extractFloat(perf, "cwv", "lcp"); ok {
		out["lcpMs"] = lcp
	}
	if cls, ok := extractFloat(perf, "cwv", "cls"); ok {
		out["cls"] = cls
	}
	if fps, ok := extractFloat(perf, "fps", "fps"); ok {
		out["fps"] = fps
	}
	return out
}

func WriteAssertArtifacts(dir string, result AssertResult, mode ArtifactMode) (string, error) {
	if mode == ArtifactModeNone || strings.TrimSpace(dir) == "" {
		return "", nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "assert.json")
	b, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}
	b = append(b, '\n')
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return "", err
	}
	return path, nil
}
