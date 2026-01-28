package devbrowser

import (
	"strings"
	"testing"
)

func TestOptionalFloat(t *testing.T) {
	tests := []struct {
		name string
		args map[string]interface{}
		key  string
		def  float64
		want float64
		err  bool
	}{
		{
			name: "returns default when missing",
			args: map[string]interface{}{},
			key:  "value",
			def:  1.5,
			want: 1.5,
			err:  false,
		},
		{
			name: "returns float64",
			args: map[string]interface{}{"value": 2.5},
			key:  "value",
			def:  1.5,
			want: 2.5,
			err:  false,
		},
		{
			name: "returns int as float",
			args: map[string]interface{}{"value": 3},
			key:  "value",
			def:  1.5,
			want: 3.0,
			err:  false,
		},
		{
			name: "returns float32 as float64",
			args: map[string]interface{}{"value": float32(2.5)},
			key:  "value",
			def:  1.5,
			want: 2.5,
			err:  false,
		},
		{
			name: "errors on invalid type",
			args: map[string]interface{}{"value": "not a number"},
			key:  "value",
			def:  1.5,
			want: 0,
			err:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := optionalFloat(tt.args, tt.key, tt.def)
			if (err != nil) != tt.err {
				t.Errorf("optionalFloat() error = %v, wantErr %v", err, tt.err)
				return
			}
			if got != tt.want {
				t.Errorf("optionalFloat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCountInlined(t *testing.T) {
	assets := []map[string]interface{}{
		{"url": "a.css", "size": 5000},
		{"url": "b.js", "size": 15000},
		{"url": "c.css", "size": 8000},
		{"url": "d.jpg", "size": 20000},
	}

	count := countInlined(assets, 10240)
	if count != 2 {
		t.Errorf("countInlined() = %d, want 2", count)
	}
}

func TestCountLinked(t *testing.T) {
	assets := []map[string]interface{}{
		{"url": "a.css", "size": 5000},
		{"url": "b.js", "size": 15000},
		{"url": "c.css", "size": 8000},
		{"url": "d.jpg", "size": 20000},
	}

	count := countLinked(assets, 10240)
	if count != 2 {
		t.Errorf("countLinked() = %d, want 2", count)
	}
}

func TestRemoveScripts(t *testing.T) {
	input := `<div><script>alert("x")</script><p>ok</p></div>`
	got := removeScripts(input)
	if !strings.Contains(got, "<!-- <script") {
		t.Fatalf("expected script tag to be commented")
	}
	if !strings.Contains(got, "</script> -->") {
		t.Fatalf("expected script closing tag to be commented")
	}
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
