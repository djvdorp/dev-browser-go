package devbrowser

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	cacheSubdir = "dev-browser-go"
)

func NowMS() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func PlatformCacheDir() string {
	if dir := os.Getenv("XDG_CACHE_HOME"); strings.TrimSpace(dir) != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Caches")
	}
	return filepath.Join(home, ".cache")
}

func PlatformStateDir() string {
	if dir := os.Getenv("XDG_STATE_HOME"); strings.TrimSpace(dir) != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Application Support")
	}
	return filepath.Join(home, ".local", "state")
}

func ArtifactDir(profile string) string {
	return filepath.Join(PlatformCacheDir(), cacheSubdir, profile, "artifacts")
}

func StateDir(profile string) string {
	return filepath.Join(PlatformStateDir(), cacheSubdir, profile)
}

func StateFile(profile string) string {
	return filepath.Join(StateDir(profile), "daemon.json")
}

func SafeArtifactPath(artifactDir, pathArg, defaultName string) (string, error) {
	allowUnsafe := envTruthy("DEV_BROWSER_ALLOW_UNSAFE_PATHS")

	if strings.TrimSpace(pathArg) == "" {
		if err := os.MkdirAll(artifactDir, 0o755); err != nil {
			return "", err
		}
		return filepath.Join(artifactDir, defaultName), nil
	}

	expanded := os.ExpandEnv(pathArg)
	if strings.HasPrefix(expanded, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			expanded = filepath.Join(home, strings.TrimPrefix(expanded, "~"))
		}
	}

	resolved := expanded
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(artifactDir, resolved)
	}
	resolved, err := filepath.Abs(resolved)
	if err != nil {
		return "", err
	}

	if allowUnsafe {
		if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
			return "", err
		}
		return resolved, nil
	}

	artifactResolved, err := filepath.Abs(artifactDir)
	if err != nil {
		return "", err
	}
	allowed := artifactResolved == resolved || strings.HasPrefix(resolved, artifactResolved+string(os.PathSeparator))
	if !allowed {
		return "", fmt.Errorf("Refusing to write outside artifact dir: %s (allowed under %s). Set DEV_BROWSER_ALLOW_UNSAFE_PATHS=1 to override.", resolved, artifactResolved)
	}

	if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
		return "", err
	}
	return resolved, nil
}

func envTruthy(name string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func clampNonNegativeInt(value int) (int, error) {
	if value < 0 {
		return 0, errors.New("value must be non-negative")
	}
	return value, nil
}
