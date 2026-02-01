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

func TestWatchStamp_IgnoresCommonBuildDirectories(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	nmDir := filepath.Join(dir, "node_modules")
	distDir := filepath.Join(dir, "dist")
	buildDir := filepath.Join(dir, "build")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(nmDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Put a file in each ignored dir.
	_ = os.WriteFile(filepath.Join(gitDir, "x"), []byte("x"), 0o644)
	_ = os.WriteFile(filepath.Join(nmDir, "y"), []byte("y"), 0o644)
	_ = os.WriteFile(filepath.Join(distDir, "d"), []byte("d"), 0o644)
	_ = os.WriteFile(filepath.Join(buildDir, "b"), []byte("b"), 0o644)

	// And one normal file.
	normalFile := filepath.Join(dir, "z")
	_ = os.WriteFile(normalFile, []byte("z"), 0o644)

	s1 := watchStamp([]string{dir})
	if s1 == 0 {
		t.Fatalf("expected non-zero stamp")
	}

	// Ensure modtime advances.
	time.Sleep(20 * time.Millisecond)

	// Modify files in ignored dirs - stamp should not change.
	_ = os.WriteFile(filepath.Join(gitDir, "x"), []byte("x2"), 0o644)
	_ = os.WriteFile(filepath.Join(nmDir, "y"), []byte("y2"), 0o644)
	_ = os.WriteFile(filepath.Join(distDir, "d"), []byte("d2"), 0o644)
	_ = os.WriteFile(filepath.Join(buildDir, "b"), []byte("b2"), 0o644)

	s2 := watchStamp([]string{dir})
	if s2 != s1 {
		t.Fatalf("expected stamp unchanged after modifying ignored dirs; before=%d after=%d", s1, s2)
	}

	// Ensure modtime advances.
	time.Sleep(20 * time.Millisecond)

	// Modify normal file - stamp should change.
	_ = os.WriteFile(normalFile, []byte("z2"), 0o644)

	s3 := watchStamp([]string{dir})
	if s3 <= s2 {
		t.Fatalf("expected stamp to increase after modifying normal file; before=%d after=%d", s2, s3)
	}
}
