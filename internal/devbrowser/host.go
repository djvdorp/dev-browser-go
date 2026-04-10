package devbrowser

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"
)

type PageEntry struct {
	Name     string
	TargetID string
	URL      string
	Title    string
}

type BrowserHost struct {
	profile  string
	headless bool
	cdpPort  int
	window   *WindowSize
	device   string

	mu       sync.Mutex
	pw       *playwright.Playwright
	context  playwright.BrowserContext
	ws       string
	registry map[string]pageHolder
	userData string
	logs     *consoleStore
	settings BrowserContextSettings
}

type pageHolder struct {
	page          playwright.Page
	targetID      string
	consoleHooked bool
}

func NewBrowserHost(profile string, headless bool, cdpPort int, window *WindowSize, device string) *BrowserHost {
	stateBase := filepath.Join(PlatformStateDir(), cacheSubdir, profile)
	settings := normalizeContextRequest(headless, window, device)
	return &BrowserHost{
		profile:  profile,
		headless: settings.Headless,
		cdpPort:  cdpPort,
		window:   cloneWindowSize(settings.Window),
		device:   settings.Device,
		registry: make(map[string]pageHolder),
		userData: filepath.Join(stateBase, "chromium-profile"),
		logs:     newConsoleStore(0),
		settings: settings,
	}
}

func (b *BrowserHost) WSEndpoint() (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.ws == "" {
		return "", errors.New("host not started")
	}
	return b.ws, nil
}

func (b *BrowserHost) Start() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.startLocked()
}

func (b *BrowserHost) Stop() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.stopLocked()
}

func (b *BrowserHost) stopLocked() {
	for name, holder := range b.registry {
		if holder.page != nil && !holder.page.IsClosed() {
			_ = holder.page.Close()
		}
		delete(b.registry, name)
	}

	if b.context != nil {
		_ = b.context.Close()
	}
	b.context = nil

	if b.pw != nil {
		_ = b.pw.Stop()
	}
	b.pw = nil
	b.ws = ""
	if b.logs != nil {
		b.logs.clearAll()
	}
}

func (b *BrowserHost) ContextSettings() BrowserContextSettings {
	b.mu.Lock()
	defer b.mu.Unlock()
	return cloneContextSettings(b.settings)
}

func (b *BrowserHost) PrimaryPageURL() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	if holder, ok := b.registry["main"]; ok && holder.page != nil && !holder.page.IsClosed() {
		return holder.page.URL()
	}
	names := make([]string, 0, len(b.registry))
	for name, holder := range b.registry {
		if holder.page != nil && !holder.page.IsClosed() {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		return b.registry[name].page.URL()
	}
	return ""
}

func (b *BrowserHost) Reconfigure(headless bool, window *WindowSize, device string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	requested := normalizeContextRequest(headless, window, device)
	if effectiveContextMatches(b.settings, requested) {
		return nil
	}

	restore := b.capturePagesLocked()
	b.stopLocked()

	b.headless = requested.Headless
	b.window = cloneWindowSize(requested.Window)
	b.device = requested.Device
	b.settings = requested

	if err := b.startLocked(); err != nil {
		return err
	}
	return b.restorePagesLocked(restore)
}

func (b *BrowserHost) ListPages() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	names := []string{}
	for name, holder := range b.registry {
		if holder.page != nil && !holder.page.IsClosed() {
			names = append(names, name)
		}
	}
	return names
}

func (b *BrowserHost) ClosePage(name string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	holder, ok := b.registry[name]
	if !ok {
		return false
	}
	if holder.page != nil && !holder.page.IsClosed() {
		_ = holder.page.Close()
	}
	delete(b.registry, name)
	if b.logs != nil {
		b.logs.clear(name)
	}
	return true
}

func (b *BrowserHost) GetOrCreatePage(name string) (PageEntry, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.context == nil {
		if err := b.startLocked(); err != nil {
			return PageEntry{}, err
		}
	}

	if holder, ok := b.registry[name]; ok && holder.page != nil && !holder.page.IsClosed() {
		identity := PageIdentity{}
		identity, err := b.describeHolderLocked(holder)
		if err == nil {
			holder.targetID = identity.TargetID
			b.registry[name] = holder
		} else {
			identity = PageIdentity{
				TargetID: holder.targetID,
				URL:      holder.page.URL(),
				Title:    safeTitle(holder.page),
			}
			if strings.TrimSpace(identity.TargetID) == "" {
				return PageEntry{}, fmt.Errorf("resolve target id for page %q: %w", name, err)
			}
		}
		if !holder.consoleHooked {
			b.attachConsoleLocked(name, holder.page)
		}
		return PageEntry{Name: name, TargetID: identity.TargetID, URL: identity.URL, Title: identity.Title}, nil
	}

	page, err := b.context.NewPage()
	if err != nil {
		return PageEntry{}, err
	}
	identity, err := describePageInContext(b.context, page)
	if err != nil {
		_ = page.Close()
		return PageEntry{}, err
	}
	b.registry[name] = pageHolder{page: page, targetID: identity.TargetID}
	b.attachConsoleLocked(name, page)
	return PageEntry{Name: name, TargetID: identity.TargetID, URL: identity.URL, Title: identity.Title}, nil
}

func (b *BrowserHost) startLocked() error {
	if b.context != nil {
		return nil
	}

	if err := playwright.Install(&playwright.RunOptions{Browsers: []string{"chromium"}}); err != nil {
		return fmt.Errorf("install playwright: %w", err)
	}

	if err := os.MkdirAll(b.userData, 0o755); err != nil {
		return err
	}

	pw, err := playwright.Run()
	if err != nil {
		return fmt.Errorf("start playwright: %w", err)
	}

	deviceName, device, err := resolveDeviceProfile(pw, b.device)
	if err != nil {
		pw.Stop()
		return err
	}

	window := b.window
	if device != nil {
		if deviceWindow := deviceWindowSize(device); deviceWindow != nil {
			window = deviceWindow
		} else {
			window = nil
		}
	}
	opts := playwright.BrowserTypeLaunchPersistentContextOptions{
		AcceptDownloads:   playwright.Bool(true),
		Headless:          playwright.Bool(b.headless),
		IgnoreHttpsErrors: playwright.Bool(true),
		Args:              ChromiumLaunchArgs(b.cdpPort, window),
	}
	if device != nil {
		applyDeviceDescriptor(&opts, device)
	} else if window != nil {
		opts.Viewport = &playwright.Size{Width: window.Width, Height: window.Height}
		opts.Screen = &playwright.Size{Width: window.Width, Height: window.Height}
	}

	context, err := pw.Chromium.LaunchPersistentContext(b.userData, opts)
	if err != nil {
		pw.Stop()
		return fmt.Errorf("launch context: %w", err)
	}
	context.SetDefaultTimeout(15_000)
	if err := InstallHarnessInit(context); err != nil {
		context.Close()
		pw.Stop()
		return fmt.Errorf("install harness init: %w", err)
	}

	ws, err := waitForWSEndpoint(b.cdpPort, 10*time.Second)
	if err != nil {
		context.Close()
		pw.Stop()
		return err
	}

	pages := context.Pages()
	if len(pages) == 0 {
		p, err := context.NewPage()
		if err != nil {
			context.Close()
			pw.Stop()
			return err
		}
		pages = append(pages, p)
	}

	mainPage := pages[0]
	tid, err := resolveTargetID(context, mainPage)
	if err != nil {
		context.Close()
		pw.Stop()
		return err
	}

	b.pw = pw
	b.context = context
	b.ws = ws
	b.settings = browserContextSettingsFromLaunch(b.headless, deviceName, window, &opts)
	b.registry["main"] = pageHolder{page: mainPage, targetID: tid}
	b.attachConsoleLocked("main", mainPage)

	for _, pg := range pages[1:] {
		_ = pg.Close()
	}
	return nil
}

type pageRestoreState struct {
	Name string
	URL  string
}

func (b *BrowserHost) capturePagesLocked() []pageRestoreState {
	names := make([]string, 0, len(b.registry))
	for name, holder := range b.registry {
		if holder.page != nil && !holder.page.IsClosed() {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	restore := make([]pageRestoreState, 0, len(names))
	for _, name := range names {
		restore = append(restore, pageRestoreState{
			Name: name,
			URL:  b.registry[name].page.URL(),
		})
	}
	return restore
}

func (b *BrowserHost) restorePagesLocked(pages []pageRestoreState) error {
	if len(pages) == 0 {
		return nil
	}

	mainHolder, hasMain := b.registry["main"]
	restoredMain := false
	for _, state := range pages {
		if state.Name != "main" {
			continue
		}
		if hasMain && mainHolder.page != nil && !mainHolder.page.IsClosed() {
			if err := navigatePageForRestore(mainHolder.page, state.URL); err != nil {
				return fmt.Errorf("restore page %q: %w", state.Name, err)
			}
			identity, err := describePageInContext(b.context, mainHolder.page)
			if err != nil {
				return fmt.Errorf("restore page %q: %w", state.Name, err)
			}
			b.registry["main"] = pageHolder{page: mainHolder.page, targetID: identity.TargetID}
			b.attachConsoleLocked("main", mainHolder.page)
			restoredMain = true
		}
		break
	}

	for _, state := range pages {
		if state.Name == "main" {
			continue
		}
		page, err := b.context.NewPage()
		if err != nil {
			return fmt.Errorf("restore page %q: %w", state.Name, err)
		}
		if err := navigatePageForRestore(page, state.URL); err != nil {
			_ = page.Close()
			return fmt.Errorf("restore page %q: %w", state.Name, err)
		}
		identity, err := describePageInContext(b.context, page)
		if err != nil {
			_ = page.Close()
			return fmt.Errorf("restore page %q: %w", state.Name, err)
		}
		b.registry[state.Name] = pageHolder{page: page, targetID: identity.TargetID}
		b.attachConsoleLocked(state.Name, page)
	}

	if !restoredMain {
		if main, ok := b.registry["main"]; ok && main.page != nil && !main.page.IsClosed() {
			if len(pages) == 1 && pages[0].Name != "main" {
				_ = main.page.Close()
				delete(b.registry, "main")
			}
		}
	}

	return nil
}

func navigatePageForRestore(page playwright.Page, rawURL string) error {
	url := strings.TrimSpace(rawURL)
	if url == "" || url == "about:blank" {
		return nil
	}
	_, err := page.Goto(url, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		Timeout:   playwright.Float(45_000),
	})
	return err
}

func cloneContextSettings(src BrowserContextSettings) BrowserContextSettings {
	src.Window = cloneWindowSize(src.Window)
	src.Viewport = cloneWindowSize(src.Viewport)
	src.Screen = cloneWindowSize(src.Screen)
	return src
}

func browserContextSettingsFromLaunch(headless bool, device string, window *WindowSize, opts *playwright.BrowserTypeLaunchPersistentContextOptions) BrowserContextSettings {
	settings := BrowserContextSettings{
		Headless: headless,
		Device:   strings.TrimSpace(device),
		Window:   cloneWindowSize(window),
	}
	if opts == nil {
		return settings
	}
	if opts.Viewport != nil {
		settings.Viewport = &WindowSize{Width: opts.Viewport.Width, Height: opts.Viewport.Height}
	}
	if opts.Screen != nil {
		settings.Screen = &WindowSize{Width: opts.Screen.Width, Height: opts.Screen.Height}
	}
	if opts.DeviceScaleFactor != nil {
		settings.DeviceScaleFactor = *opts.DeviceScaleFactor
	}
	if opts.IsMobile != nil {
		settings.IsMobile = *opts.IsMobile
	}
	if opts.HasTouch != nil {
		settings.HasTouch = *opts.HasTouch
	}
	if opts.UserAgent != nil {
		settings.UserAgent = *opts.UserAgent
	}
	return settings
}

func (b *BrowserHost) attachConsoleLocked(name string, page playwright.Page) {
	holder, ok := b.registry[name]
	if !ok {
		// Page must be in registry before attaching console
		return
	}
	if holder.consoleHooked {
		return
	}
	// Ensure harness init is installed for this page/document.
	EnsureHarnessOnPage(page)

	page.OnConsole(func(msg playwright.ConsoleMessage) {
		if b.logs != nil {
			b.logs.append(name, msg)
		}
	})
	page.OnPageError(func(err error) {
		if b.logs != nil {
			b.logs.appendPageError(name, err)
		}
	})
	holder.page = page
	holder.consoleHooked = true
	b.registry[name] = holder
}

func (b *BrowserHost) ConsoleLogs(name string, since int64, limit int) ([]ConsoleEntry, int64, error) {
	if since < 0 {
		return nil, 0, errors.New("since must be >= 0")
	}
	if limit < 0 {
		return nil, 0, errors.New("limit must be >= 0")
	}
	b.mu.Lock()
	holder, ok := b.registry[name]
	pageOk := ok && holder.page != nil && !holder.page.IsClosed()
	b.mu.Unlock()
	if !pageOk {
		return nil, 0, errors.New("page not found")
	}
	if b.logs == nil {
		return nil, 0, nil
	}
	entries, lastID := b.logs.list(name, since, limit)
	return entries, lastID, nil
}

func (b *BrowserHost) describeHolderLocked(holder pageHolder) (PageIdentity, error) {
	if holder.page == nil {
		return PageIdentity{}, errors.New("page is nil")
	}
	if holder.page.IsClosed() {
		return PageIdentity{}, errors.New("page is closed")
	}
	return describePageInContext(b.context, holder.page)
}

func resolveTargetID(context playwright.BrowserContext, page playwright.Page) (string, error) {
	session, err := context.NewCDPSession(page)
	if err != nil {
		return "", err
	}
	infoRaw, err := session.Send("Target.getTargetInfo", map[string]interface{}{})
	_ = session.Detach()
	if err != nil {
		return "", err
	}
	infoMap, ok := infoRaw.(map[string]interface{})
	if !ok {
		return "", errors.New("unexpected target info")
	}
	ti, _ := infoMap["targetInfo"].(map[string]interface{})
	tid, _ := ti["targetId"].(string)
	if tid == "" {
		return "", errors.New("targetId missing")
	}
	return tid, nil
}

func waitForWSEndpoint(port int, timeout time.Duration) (string, error) {
	url := fmt.Sprintf("http://127.0.0.1:%d/json/version", port)
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			var data struct {
				WSEndpoint string `json:"webSocketDebuggerUrl"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&data); err == nil {
				_ = resp.Body.Close()
				if strings.TrimSpace(data.WSEndpoint) != "" {
					return data.WSEndpoint, nil
				}
			}
			_ = resp.Body.Close()
		} else if err != nil {
			lastErr = err
		}
		time.Sleep(200 * time.Millisecond)
	}
	if lastErr != nil {
		return "", fmt.Errorf("wait ws endpoint: %w", lastErr)
	}
	return "", fmt.Errorf("timed out waiting for Chromium CDP endpoint at %s", url)
}
