package devbrowser

// TruncateStringRunes truncates a string to at most max runes.
//
// It returns the (possibly truncated) body, and whether truncation occurred.
func TruncateStringRunes(s string, max int) (body string, truncated bool) {
	if max <= 0 {
		return s, false
	}
	r := []rune(s)
	if len(r) <= max {
		return s, false
	}
	return string(r[:max]), true
}

// clampBody is kept for backwards compatibility.
// It also returns an encoding label for future-proofing.
func clampBody(s string, max int) (body string, truncated bool, encoding string) {
	encoding = "utf8"
	body, truncated = TruncateStringRunes(s, max)
	return body, truncated, encoding
}

func clampBytes(b []byte, max int) []byte {
	if max <= 0 || len(b) <= max {
		return b
	}
	return b[:max]
}
