package devbrowser

import (
	"sort"
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
				tm, ok := m["time_ms"].(float64)
				if !ok {
					// Skip entries without valid time_ms to avoid distorting timeline
					continue
				}
				timeMs := int64(tm)
				kind := "errorhook"
				events = append(events, DiagnoseEvent{Kind: kind, TimeMS: timeMs, Data: m})
			}
		}
		if arr, ok := harness["overlays"].([]interface{}); ok {
			for _, v := range arr {
				m, ok := v.(map[string]any)
				if !ok {
					continue
				}
				tm, ok := m["time_ms"].(float64)
				if !ok {
					// Skip entries without valid time_ms to avoid distorting timeline
					continue
				}
				events = append(events, DiagnoseEvent{Kind: "overlay", TimeMS: int64(tm), Data: m})
			}
		}
	}

	// Use stable sort with complete tie-breakers for deterministic output
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].TimeMS != events[j].TimeMS {
			return events[i].TimeMS < events[j].TimeMS
		}
		if events[i].Kind != events[j].Kind {
			return events[i].Kind < events[j].Kind
		}
		// Stable tie-breakers by kind
		return stableCompareData(events[i], events[j])
	})

	return events
}

// stableCompareData provides deterministic ordering for events with same TimeMS and Kind
func stableCompareData(a, b DiagnoseEvent) bool {
	switch a.Kind {
	case "console":
		// Compare by id first (guaranteed unique), then text
		idA, okA := a.Data["id"].(int)
		idB, okB := b.Data["id"].(int)
		if okA && okB && idA != idB {
			return idA < idB
		}
		return stringField(a.Data, "text") < stringField(b.Data, "text")
	case "network":
		// Compare by url+method+status
		urlA := stringField(a.Data, "url")
		urlB := stringField(b.Data, "url")
		if urlA != urlB {
			return urlA < urlB
		}
		methodA := stringField(a.Data, "method")
		methodB := stringField(b.Data, "method")
		if methodA != methodB {
			return methodA < methodB
		}
		statusA, _ := a.Data["status"].(int)
		statusB, _ := b.Data["status"].(int)
		return statusA < statusB
	case "errorhook":
		// Compare by type+message+stack
		typeA := stringField(a.Data, "type")
		typeB := stringField(b.Data, "type")
		if typeA != typeB {
			return typeA < typeB
		}
		msgA := stringField(a.Data, "message")
		msgB := stringField(b.Data, "message")
		if msgA != msgB {
			return msgA < msgB
		}
		return stringField(a.Data, "stack") < stringField(b.Data, "stack")
	case "overlay":
		// Compare by text
		return stringField(a.Data, "text") < stringField(b.Data, "text")
	default:
		return false
	}
}

func stringField(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
