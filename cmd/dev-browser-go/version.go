package main

import (
	"fmt"
	"runtime/debug"
	"strings"
)

const baseVersion = "0.2.3"

// versionOverride is set via -ldflags for packaged builds that do not carry VCS metadata.
var versionOverride string

func buildVersion() string {
	if v := strings.TrimSpace(versionOverride); v != "" {
		return v
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return baseVersion
	}
	return versionFromBuildInfo(baseVersion, info)
}

func versionFromBuildInfo(base string, info *debug.BuildInfo) string {
	version := strings.TrimSpace(base)
	if info == nil {
		return version
	}

	revision, dirty := vcsRevision(info)
	if revision == "" {
		if mainVersion := strings.TrimSpace(info.Main.Version); mainVersion != "" && mainVersion != "(devel)" {
			return strings.TrimPrefix(mainVersion, "v")
		}
		return version
	}

	suffix := fmt.Sprintf("+g%s", shortRevision(revision))
	if dirty {
		suffix += ".dirty"
	}
	return version + suffix
}

func vcsRevision(info *debug.BuildInfo) (revision string, dirty bool) {
	if info == nil {
		return "", false
	}
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = strings.TrimSpace(setting.Value)
		case "vcs.modified":
			dirty = strings.TrimSpace(setting.Value) == "true"
		}
	}
	return revision, dirty
}

func shortRevision(revision string) string {
	revision = strings.TrimSpace(revision)
	if len(revision) <= 12 {
		return revision
	}
	return revision[:12]
}
