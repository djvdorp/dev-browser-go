package devbrowser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// RunOptions captures cross-command run metadata for harness-friendly reports.
//
// A run typically writes artifacts under:
//
//	<artifactRoot>/run-<timestamp>-<uuid8>/
//
// Timestamp is always treated as UTC for path stability.
// RunID is a UUID (string) for uniqueness.
//
// Most callers should use NewRunContextFromProfile.
//
// This is intentionally minimal to keep CLI commands thin.
type RunOptions struct {
	Profile string

	// ArtifactRoot is the profile-scoped artifact directory root.
	// If empty, callers may still use RunContext for IDs/timestamps.
	ArtifactRoot string

	RunID     string
	Timestamp time.Time
}

type RunContext struct {
	RunOptions
}

func NewRunID() string {
	// UUID is readable, stable, and avoids collisions.
	return uuid.NewString()
}

// NewDiagnoseRunID is kept for backwards compatibility.
// Prefer NewRunID.
func NewDiagnoseRunID() string {
	return NewRunID()
}

func NewRunContext(opts RunOptions) RunContext {
	if strings.TrimSpace(opts.Profile) == "" {
		opts.Profile = "default"
	}
	if opts.Timestamp.IsZero() {
		opts.Timestamp = time.Now()
	}
	if strings.TrimSpace(opts.RunID) == "" {
		opts.RunID = NewRunID()
	}
	if strings.TrimSpace(opts.ArtifactRoot) == "" && strings.TrimSpace(opts.Profile) != "" {
		opts.ArtifactRoot = ArtifactDir(opts.Profile)
	}
	return RunContext{RunOptions: opts}
}

func NewRunContextFromProfile(profile string) RunContext {
	return NewRunContext(RunOptions{Profile: profile})
}

func (c RunContext) DefaultRunDir() string {
	if strings.TrimSpace(c.ArtifactRoot) == "" {
		return ""
	}
	return DefaultRunArtifactDir(c.ArtifactRoot, c.RunID, c.Timestamp)
}

// ResolveRunDir resolves --artifact-dir style inputs.
//
// If dirArg is empty, it returns the default per-run directory.
// If dirArg is relative, it is resolved under ArtifactRoot.
//
// It returns an empty string if ArtifactRoot is not set.
func (c RunContext) ResolveRunDir(dirArg string) (string, error) {
	if strings.TrimSpace(c.ArtifactRoot) == "" {
		return "", nil
	}
	if strings.TrimSpace(dirArg) == "" {
		return c.DefaultRunDir(), nil
	}
	// When user passes a relative path, treat it as relative to artifact root.
	return SafeArtifactPath(c.ArtifactRoot, dirArg, "")
}

func (c RunContext) EnsureDir(dir string) error {
	if strings.TrimSpace(dir) == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

// DefaultRunArtifactDir returns a per-run subdirectory under root.
//
// Format: run-<timestamp>-<uuid8>/
func DefaultRunArtifactDir(root, runID string, ts time.Time) string {
	stamp := ts.UTC().Format("20060102T150405Z")
	short := runID
	if len(short) > 8 {
		short = short[:8]
	}
	return filepath.Join(root, fmt.Sprintf("run-%s-%s", stamp, short))
}
