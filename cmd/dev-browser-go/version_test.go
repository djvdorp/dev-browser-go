package main

import (
	"runtime/debug"
	"testing"
)

func TestVersionFromBuildInfoWithoutBuildInfo(t *testing.T) {
	if got := versionFromBuildInfo(baseVersion, nil); got != baseVersion {
		t.Fatalf("versionFromBuildInfo(..., nil) = %q, want %q", got, baseVersion)
	}
}

func TestVersionFromBuildInfoUsesModuleVersion(t *testing.T) {
	info := &debug.BuildInfo{
		Main: debug.Module{
			Version: "v1.2.3",
		},
	}
	if got := versionFromBuildInfo(baseVersion, info); got != "1.2.3" {
		t.Fatalf("versionFromBuildInfo() = %q, want %q", got, "1.2.3")
	}
}

func TestVersionFromBuildInfoPrefersBaseVersionWhenRevisionPresent(t *testing.T) {
	info := &debug.BuildInfo{
		Main: debug.Module{
			Version: "v1.2.3",
		},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "a38dae31234567890abcdef1234567890abcdef"},
		},
	}
	if got := versionFromBuildInfo(baseVersion, info); got != "0.2.2+ga38dae312345" {
		t.Fatalf("versionFromBuildInfo() = %q, want %q", got, "0.2.2+ga38dae312345")
	}
}

func TestVersionFromBuildInfoAddsRevision(t *testing.T) {
	info := &debug.BuildInfo{
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "a38dae31234567890abcdef1234567890abcdef"},
		},
	}
	if got := versionFromBuildInfo(baseVersion, info); got != "0.2.2+ga38dae312345" {
		t.Fatalf("versionFromBuildInfo() = %q, want %q", got, "0.2.2+ga38dae312345")
	}
}

func TestVersionFromBuildInfoAddsDirtyMarker(t *testing.T) {
	info := &debug.BuildInfo{
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "a38dae31234567890abcdef1234567890abcdef"},
			{Key: "vcs.modified", Value: "true"},
		},
	}
	if got := versionFromBuildInfo(baseVersion, info); got != "0.2.2+ga38dae312345.dirty" {
		t.Fatalf("versionFromBuildInfo() = %q, want %q", got, "0.2.2+ga38dae312345.dirty")
	}
}
