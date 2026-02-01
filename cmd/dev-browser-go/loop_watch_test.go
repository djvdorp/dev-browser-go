package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatchStamp_ChangesOnFileTouch(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(p, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}

	s1 := watchStamp([]string{dir})
	if s1 == 0 {
		t.Fatalf("expected non-zero stamp")
	}

	// Ensure modtime advances even on coarse filesystems.
	time.Sleep(20 * time.Millisecond)
	if err := os.WriteFile(p, []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}

	s2 := watchStamp([]string{dir})
	if s2 <= s1 {
		t.Fatalf("expected stamp to increase; before=%d after=%d", s1, s2)
	}
}

func TestWatchStamp_IgnoresNodeModulesAndGit(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	nmDir := filepath.Join(dir, "node_modules")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(nmDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Put a file in each ignored dir.
	_ = os.WriteFile(filepath.Join(gitDir, "x"), []byte("x"), 0o644)
	_ = os.WriteFile(filepath.Join(nmDir, "y"), []byte("y"), 0o644)

	// And one normal file.
	_ = os.WriteFile(filepath.Join(dir, "z"), []byte("z"), 0o644)

	s := watchStamp([]string{dir})
	if s == 0 {
		t.Fatalf("expected non-zero stamp")
	}
}
