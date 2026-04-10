package devbrowser

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type DaemonState struct {
	PID        int                    `json:"pid"`
	Host       string                 `json:"host"`
	Port       int                    `json:"port"`
	Profile    string                 `json:"profile"`
	CDPPort    int                    `json:"cdpPort"`
	WSEndpoint string                 `json:"wsEndpoint"`
	Version    string                 `json:"version"`
	Context    BrowserContextSettings `json:"context"`
	PageURL    string                 `json:"pageURL,omitempty"`
}

type PageSessionInfo struct {
	WSEndpoint string
	PageIdentity
}

func ReadState(profile string) (*DaemonState, error) {
	path := StateFile(profile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var state DaemonState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func DaemonBaseURL(profile string) string {
	state, err := ReadState(profile)
	if err != nil || state == nil {
		return ""
	}
	if state.Host == "" || state.Port == 0 {
		return ""
	}
	return fmt.Sprintf("http://%s:%d", state.Host, state.Port)
}

func HTTPJSON(method string, url string, body map[string]any, timeout time.Duration) (map[string]any, error) {
	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

func IsDaemonHealthy(profile string) bool {
	health, err := ReadDaemonHealth(profile)
	return err == nil && health != nil && health.OK
}

func EnsureDaemon(profile string, headless bool, window *WindowSize, device string) (DaemonStartResult, error) {
	if window != nil && strings.TrimSpace(device) != "" {
		return DaemonStartResult{}, errors.New("use either --window-size/--window-scale or --device")
	}

	requested := normalizeContextRequest(headless, window, device)
	if health, err := ReadDaemonHealth(profile); err == nil && health != nil && health.OK {
		baseURL := fmt.Sprintf("http://%s:%d", health.Host, health.Port)
		if strings.TrimSpace(health.Version) == "" || strings.TrimSpace(health.Version) != DaemonVersion() {
			if _, err := StopDaemon(profile); err != nil {
				return DaemonStartResult{}, fmt.Errorf("failed to stop existing dev-browser daemon (profile=%s): %w", profile, err)
			}
		} else if effectiveContextMatches(health.Context, requested) {
			return DaemonStartResult{
				Action:     DaemonActionReused,
				Context:    health.Context,
				BaseURL:    baseURL,
				Profile:    profile,
				PageURL:    health.PageURL,
				WSEndpoint: health.WSEndpoint,
			}, nil
		} else {
			data, err := HTTPJSON(http.MethodPost, baseURL+"/reconfigure", map[string]any{
				"headless": requested.Headless,
				"window":   requested.Window,
				"device":   requested.Device,
			}, 90*time.Second)
			if err != nil {
				return DaemonStartResult{}, err
			}
			updated := decodeDaemonHealthMap(data)
			if !updated.OK {
				return DaemonStartResult{}, fmt.Errorf("reconfigure daemon: response not ok")
			}
			return DaemonStartResult{
				Action:     DaemonActionReconfigured,
				Reason:     describeContextDiff(health.Context, requested),
				Context:    updated.Context,
				BaseURL:    baseURL,
				Profile:    profile,
				PageURL:    updated.PageURL,
				WSEndpoint: updated.WSEndpoint,
			}, nil
		}
	}

	dir := StateDir(profile)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return DaemonStartResult{}, err
	}
	logPath := filepath.Join(dir, "daemon.log")

	exe, err := os.Executable()
	if err != nil {
		return DaemonStartResult{}, err
	}

	args := []string{"--daemon", "--profile", profile}
	if requested.Headless {
		args = append(args, "--headless")
	}
	if requested.Window != nil {
		args = append(args, "--window-size", fmt.Sprintf("%dx%d", requested.Window.Width, requested.Window.Height))
	}
	if strings.TrimSpace(requested.Device) != "" {
		args = append(args, "--device", requested.Device)
	}

	cmd := exec.Command(exe, args...)
	configureDaemonProcess(cmd)
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return DaemonStartResult{}, err
	}
	defer logFile.Close()
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Stdin = nil
	if err := cmd.Start(); err != nil {
		return DaemonStartResult{}, err
	}

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if health, err := ReadDaemonHealth(profile); err == nil && health != nil && health.OK {
			return DaemonStartResult{
				Action:     DaemonActionStarted,
				Context:    health.Context,
				BaseURL:    fmt.Sprintf("http://%s:%d", health.Host, health.Port),
				Profile:    profile,
				PageURL:    health.PageURL,
				WSEndpoint: health.WSEndpoint,
			}, nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return DaemonStartResult{}, fmt.Errorf("timed out waiting for dev-browser daemon (profile=%s). See %s", profile, logPath)
}

func StartDaemon(profile string, headless bool, window *WindowSize, device string) error {
	_, err := EnsureDaemon(profile, headless, window, device)
	return err
}

func StopDaemon(profile string) (bool, error) {
	state, err := ReadState(profile)
	base := DaemonBaseURL(profile)
	if state == nil || base == "" || err != nil {
		return false, nil
	}

	_, _ = HTTPJSON(http.MethodPost, base+"/shutdown", map[string]any{}, 3*time.Second)

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if !IsDaemonHealthy(profile) {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	if IsDaemonHealthy(profile) && state.PID > 0 {
		_ = syscall.Kill(state.PID, syscall.SIGTERM)
		time.Sleep(200 * time.Millisecond)
	}

	_ = os.Remove(StateFile(profile))
	return true, nil
}

func EnsurePage(profile string, headless bool, page string, window *WindowSize, device string) (string, string, error) {
	info, err := EnsurePageInfo(profile, headless, page, window, device)
	if err != nil {
		return "", "", err
	}
	return info.WSEndpoint, info.TargetID, nil
}

func EnsurePageInfo(profile string, headless bool, page string, window *WindowSize, device string) (PageSessionInfo, error) {
	result, err := EnsureDaemon(profile, headless, window, device)
	if err != nil {
		return PageSessionInfo{}, err
	}
	base := result.BaseURL
	if strings.TrimSpace(base) == "" {
		base = DaemonBaseURL(profile)
	}
	if base == "" {
		return PageSessionInfo{}, errors.New("daemon state missing after start")
	}
	data, err := HTTPJSON(http.MethodPost, base+"/pages", map[string]any{"name": page}, 10*time.Second)
	if err != nil {
		return PageSessionInfo{}, err
	}
	ws, _ := data["wsEndpoint"].(string)
	tid, _ := data["targetId"].(string)
	if strings.TrimSpace(ws) == "" {
		return PageSessionInfo{}, errors.New("daemon did not return wsEndpoint")
	}
	if strings.TrimSpace(tid) == "" {
		return PageSessionInfo{}, errors.New("daemon did not return targetId")
	}
	info := PageSessionInfo{
		WSEndpoint: ws,
		PageIdentity: PageIdentity{
			TargetID: tid,
		},
	}
	if url, _ := data["url"].(string); strings.TrimSpace(url) != "" {
		info.URL = url
	}
	if title, _ := data["title"].(string); strings.TrimSpace(title) != "" {
		info.Title = title
	}
	return info, nil
}

func ReadDaemonHealth(profile string) (*DaemonHealth, error) {
	base := DaemonBaseURL(profile)
	if base == "" {
		return nil, nil
	}
	data, err := HTTPJSON(http.MethodGet, base+"/health", nil, 1500*time.Millisecond)
	if err != nil {
		return nil, err
	}
	health := decodeDaemonHealthMap(data)
	if !health.OK {
		return nil, nil
	}
	return &health, nil
}

func decodeDaemonHealthMap(data map[string]any) DaemonHealth {
	health := DaemonHealth{}
	if ok, _ := data["ok"].(bool); ok {
		health.OK = true
	}
	health.PID = intValue(data["pid"])
	health.Host, _ = data["host"].(string)
	health.Port = intValue(data["port"])
	health.Profile, _ = data["profile"].(string)
	health.CDPPort = intValue(data["cdpPort"])
	health.WSEndpoint, _ = data["wsEndpoint"].(string)
	health.Version, _ = data["version"].(string)
	health.PageURL, _ = data["pageURL"].(string)
	if raw, ok := data["context"].(map[string]any); ok {
		health.Context = decodeBrowserContextSettings(raw)
	}
	return health
}

func decodeBrowserContextSettings(data map[string]any) BrowserContextSettings {
	settings := BrowserContextSettings{}
	if headless, ok := data["headless"].(bool); ok {
		settings.Headless = headless
	}
	settings.Device, _ = data["device"].(string)
	if raw, ok := data["window"].(map[string]any); ok {
		settings.Window = decodeWindowSizeMap(raw)
	}
	if raw, ok := data["viewport"].(map[string]any); ok {
		settings.Viewport = decodeWindowSizeMap(raw)
	}
	if raw, ok := data["screen"].(map[string]any); ok {
		settings.Screen = decodeWindowSizeMap(raw)
	}
	switch scale := data["deviceScaleFactor"].(type) {
	case float64:
		settings.DeviceScaleFactor = scale
	case int:
		settings.DeviceScaleFactor = float64(scale)
	}
	if mobile, ok := data["isMobile"].(bool); ok {
		settings.IsMobile = mobile
	}
	if touch, ok := data["hasTouch"].(bool); ok {
		settings.HasTouch = touch
	}
	settings.UserAgent, _ = data["userAgent"].(string)
	return settings
}

func decodeWindowSizeMap(data map[string]any) *WindowSize {
	width := intValue(data["width"])
	height := intValue(data["height"])
	if width <= 0 || height <= 0 {
		return nil
	}
	return &WindowSize{Width: width, Height: height}
}

func intValue(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	default:
		return 0
	}
}

func WriteOutput(profile string, mode string, result any, outPath string) (string, error) {
	switch mode {
	case "json":
		enc, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return "", err
		}
		return string(enc), nil
	case "html":
		if m, ok := result.(map[string]any); ok {
			if html, ok := m["html"].(string); ok {
				return html, nil
			}
		}
		return "", errors.New("html output not available")
	case "summary":
		switch v := result.(type) {
		case map[string]any:
			if snap, ok := v["snapshot"].(string); ok {
				return snap, nil
			}
			if path, ok := v["path"].(string); ok {
				return path, nil
			}
		case *DiagnoseReport:
			if v != nil {
				return v.Snapshot.YAML, nil
			}
		case DiagnoseReport:
			return v.Snapshot.YAML, nil
		case AssertResult:
			enc, _ := json.Marshal(v)
			return string(enc), nil
		case HTMLValidateReport:
			enc, _ := json.Marshal(v)
			return string(enc), nil
		}
		enc, err := json.Marshal(result)
		if err != nil {
			return "", err
		}
		return string(enc), nil
	case "path":
		path, err := SafeArtifactPath(ArtifactDir(profile), outPath, fmt.Sprintf("cli-%d.json", NowMS()))
		if err != nil {
			return "", err
		}
		enc, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(path, enc, 0o644); err != nil {
			return "", err
		}
		return path, nil
	default:
		return "", fmt.Errorf("unknown output mode: %s", mode)
	}
}
