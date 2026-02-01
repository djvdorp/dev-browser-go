package devbrowser

import "testing"

func TestContainsAny_EmptyHaystack(t *testing.T) {
	if containsAny("", []string{"foo", "bar"}) {
		t.Fatal("expected false for empty haystack")
	}
}

func TestContainsAny_EmptyNeedles(t *testing.T) {
	if containsAny("some text", []string{}) {
		t.Fatal("expected false for empty needles")
	}
}

func TestContainsAny_NeedlesWithWhitespace(t *testing.T) {
	if !containsAny("hello world", []string{"  world  "}) {
		t.Fatal("expected true for needle with whitespace that gets trimmed")
	}
}

func TestContainsAny_NeedlesWithEmptyStrings(t *testing.T) {
	if containsAny("hello", []string{"", "  ", "goodbye"}) {
		t.Fatal("expected false when no non-empty needles match")
	}
	if !containsAny("hello", []string{"", "  ", "hello"}) {
		t.Fatal("expected true when at least one non-empty needle matches")
	}
}

func TestContainsAny_CaseInsensitive(t *testing.T) {
	if !containsAny("Hello World", []string{"world"}) {
		t.Fatal("expected case-insensitive match")
	}
	if !containsAny("hello world", []string{"WORLD"}) {
		t.Fatal("expected case-insensitive match")
	}
}

func TestContainsAny_PartialMatch(t *testing.T) {
	if !containsAny("this is a test", []string{"is a"}) {
		t.Fatal("expected partial substring match")
	}
}

func TestContainsAny_NoMatch(t *testing.T) {
	if containsAny("hello world", []string{"foo", "bar", "baz"}) {
		t.Fatal("expected false when no needles match")
	}
}
