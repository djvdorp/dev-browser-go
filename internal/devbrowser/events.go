package devbrowser

import (
	"sort"
	"strings"
)

func BuildDiagnoseEvents(console []ConsoleEntry, network []NetworkEntry, harness map[string]any) []DiagnoseEvent {
	events := make([]DiagnoseEvent, 0, len(console)+len(network))

	for _, c := range console {
		data := map[string]any{
			"type": c.Type,
			"text": c.Text,
			"url":  c.URL,
			"line": c.Line,
			"col":  c.Column,
			"id":   c.ID,
		}
		events = append(events, DiagnoseEvent{Kind: "console", TimeMS: c.TimeMS, Data: data})
	}

	for _, n := range network {
		data := map[string]any{
			"url":    n.URL,
			"method": n.Method,
			"type":   n.Type,
			"status": n.Status,
			"ok":     n.OK,
			"error":  n.Error,
		}
		events = append(events, DiagnoseEvent{Kind: "network", TimeMS: n.Started, Data: data})
	}

	if harness != nil {
		if arr, ok := harness["errors"].([]interface{}); ok {
			for _, v := range arr {
				m, ok := v.(map[string]any)
				if !ok {
					continue
				}
				tm, _ := m["time_ms"].(float64)
				timeMs := int64(tm)
				kind := "errorhook"
				if t, _ := m["type"].(string); strings.TrimSpace(t) != "" {
					// keep the type inside data; kind stays stable
				}
				events = append(events, DiagnoseEvent{Kind: kind, TimeMS: timeMs, Data: m})
			}
		}
		if arr, ok := harness["overlays"].([]interface{}); ok {
			for _, v := range arr {
				m, ok := v.(map[string]any)
				if !ok {
					continue
				}
				tm, _ := m["time_ms"].(float64)
				events = append(events, DiagnoseEvent{Kind: "overlay", TimeMS: int64(tm), Data: m})
			}
		}
	}

	sort.Slice(events, func(i, j int) bool {
		if events[i].TimeMS == events[j].TimeMS {
			if events[i].Kind == events[j].Kind {
				return stringify(events[i].Data) < stringify(events[j].Data)
			}
			return events[i].Kind < events[j].Kind
		}
		return events[i].TimeMS < events[j].TimeMS
	})

	return events
}

func stringify(m map[string]any) string {
	if m == nil {
		return ""
	}
	if v, ok := m["url"].(string); ok {
		return v
	}
	if v, ok := m["text"].(string); ok {
		return v
	}
	return ""
}
