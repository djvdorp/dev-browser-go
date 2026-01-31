package devbrowser

import "testing"

func TestDiffDomSnapshots(t *testing.T) {
	before := &DomSnapshot{Items: []map[string]interface{}{
		{"role": "button", "name": "Save", "heading": "", "disabled": false},
		{"role": "textbox", "name": "Email", "heading": "Form", "disabled": false},
	}}
	after := &DomSnapshot{Items: []map[string]interface{}{
		{"role": "button", "name": "Save", "heading": "", "disabled": true}, // changed
		{"role": "link", "name": "Home", "heading": "", "disabled": false},  // added
	}}

	d := DiffDomSnapshots(before, after)
	if d.AddedCount != 1 {
		t.Fatalf("expected 1 added, got %d", d.AddedCount)
	}
	if d.RemovedCount != 1 {
		t.Fatalf("expected 1 removed, got %d", d.RemovedCount)
	}
	if d.ChangedCount != 1 {
		t.Fatalf("expected 1 changed, got %d", d.ChangedCount)
	}
	if len(d.Changed) != 1 || len(d.Changed[0].Fields) == 0 {
		t.Fatalf("expected changed fields")
	}
}
