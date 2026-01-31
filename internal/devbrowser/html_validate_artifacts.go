package devbrowser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

func WriteHTMLValidateArtifacts(dir string, report HTMLValidateReport, mode ArtifactMode) (string, error) {
	if mode == ArtifactModeNone || strings.TrimSpace(dir) == "" {
		return "", nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "html-validate.json")
	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	b = append(b, '\n')
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return "", err
	}
	return path, nil
}
