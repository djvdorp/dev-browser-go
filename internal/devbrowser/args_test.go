package devbrowser

import "testing"

func TestOptionalStringSlice(t *testing.T) {
	args := map[string]interface{}{}
	if got, err := optionalStringSlice(args, "properties"); err != nil || got != nil {
		t.Fatalf("expected nil, got %v err=%v", got, err)
	}

	args["properties"] = []interface{}{"color", "font-size"}
	got, err := optionalStringSlice(args, "properties")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0] != "color" || got[1] != "font-size" {
		t.Fatalf("unexpected values: %v", got)
	}

	args["properties"] = "color, font-size, ,"
	got, err = optionalStringSlice(args, "properties")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0] != "color" || got[1] != "font-size" {
		t.Fatalf("unexpected values: %v", got)
	}

	args["properties"] = 123
	if _, err := optionalStringSlice(args, "properties"); err == nil {
		t.Fatalf("expected error for non-string input")
	}
}

func TestOptionalStringAllowEmpty(t *testing.T) {
	args := map[string]interface{}{}
	if got, err := optionalStringAllowEmpty(args, "path", "default"); err != nil || got != "default" {
		t.Fatalf("expected default, got %q err=%v", got, err)
	}

	args["path"] = ""
	got, err := optionalStringAllowEmpty(args, "path", "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}

	args["path"] = 123
	if _, err := optionalStringAllowEmpty(args, "path", "default"); err == nil {
		t.Fatalf("expected error for non-string path")
	}
}
