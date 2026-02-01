package devbrowser

import "strings"

func containsAny(haystack string, needles []string) bool {
	haystack = strings.ToLower(haystack)
	for _, n := range needles {
		n = strings.ToLower(strings.TrimSpace(n))
		if n == "" {
			continue
		}
		if strings.Contains(haystack, n) {
			return true
		}
	}
	return false
}
