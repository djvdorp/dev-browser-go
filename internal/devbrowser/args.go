package devbrowser

import (
	"fmt"
	"strings"
)

func optionalStringSlice(args map[string]interface{}, key string) ([]string, error) {
	raw, ok := args[key]
	if !ok || raw == nil {
		return nil, nil
	}
	switch v := raw.(type) {
	case []string:
		return trimStringSlice(v), nil
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
				continue
			}
			return nil, fmt.Errorf("expected string array '%s'", key)
		}
		return trimStringSlice(out), nil
	case string:
		if strings.TrimSpace(v) == "" {
			return nil, nil
		}
		parts := strings.Split(v, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		return trimStringSlice(parts), nil
	default:
		return nil, fmt.Errorf("expected string array '%s'", key)
	}
}

func trimStringSlice(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func optionalStringAllowEmpty(args map[string]interface{}, key string, def string) (string, error) {
	raw, ok := args[key]
	if !ok || raw == nil {
		return def, nil
	}
	str, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("expected string '%s'", key)
	}
	return str, nil
}
