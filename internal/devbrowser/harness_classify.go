package devbrowser

import "strings"

func firstNonEmptyLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}

// ClassifyViteOverlay is a simple, low-maintenance heuristic.
// Keep classes stable; add new ones as needed.
func ClassifyViteOverlay(text string) string {
	t := strings.ToLower(text)
	switch {
	case strings.Contains(t, "failed to resolve import"):
		return "missing-module"
	case strings.Contains(t, "cannot find module"):
		return "missing-module"
	case strings.Contains(t, "syntaxerror"):
		return "syntax-error"
	case strings.Contains(t, "unexpected token"):
		return "syntax-error"
	case strings.Contains(t, "typeerror"):
		return "type-error"
	case strings.Contains(t, "referenceerror"):
		return "reference-error"
	default:
		return "unknown"
	}
}

func ClassifyHarnessError(typ string, message string) string {
	t := strings.ToLower(typ)
	m := strings.ToLower(message)
	switch {
	case t == "unhandledrejection" && strings.Contains(m, "fetch"):
		return "unhandledrejection-fetch"
	case strings.Contains(m, "typeerror"):
		return "type-error"
	case strings.Contains(m, "referenceerror"):
		return "reference-error"
	case strings.Contains(m, "syntaxerror"):
		return "syntax-error"
	default:
		return "unknown"
	}
}
