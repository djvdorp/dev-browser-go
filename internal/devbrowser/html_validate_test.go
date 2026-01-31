package devbrowser

import "testing"

func TestValidateHTML_DuplicateID(t *testing.T) {
	html := `<!doctype html><html><body><div id="a"></div><span id="a"></span></body></html>`
	findings, err := ValidateHTML(html)
	if err != nil {
		t.Fatalf("ValidateHTML error: %v", err)
	}
	found := false
	for _, f := range findings {
		if f.RuleID == "duplicate-id" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected duplicate-id finding; got %v", findings)
	}
}

func TestValidateHTML_ImgAlt(t *testing.T) {
	html := `<!doctype html><html><body><img src="x.png"></body></html>`
	findings, err := ValidateHTML(html)
	if err != nil {
		t.Fatalf("ValidateHTML error: %v", err)
	}
	found := false
	for _, f := range findings {
		if f.RuleID == "img-alt" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected img-alt finding")
	}
}

func TestValidateHTML_ControlName_LabelFor(t *testing.T) {
	html := `<!doctype html><html><body>
<label for="email">Email</label>
<input id="email" type="text" />
</body></html>`
	findings, err := ValidateHTML(html)
	if err != nil {
		t.Fatalf("ValidateHTML error: %v", err)
	}
	for _, f := range findings {
		if f.RuleID == "control-name" {
			t.Fatalf("did not expect control-name finding; got %v", f)
		}
	}
}

// Note: invalid nesting is "best-effort" and HTML parsing can auto-correct markup.
// We keep validation logic but only unit-test the checks that are stable under parsing.
