package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

type RuntimeConfig struct {
	Mode           string
	Timeout        time.Duration
	ArtifactsRoot  string
	Headless       bool
	ExecutablePath string
}

type Service struct {
	cfg    RuntimeConfig
	client *http.Client
}

type Request struct {
	URL                string
	WaitSelector       string
	AuthHeader         string
	CookiesJSON        string
	LocalStorageJSON   string
	SessionStorageJSON string
	CaptureMode        string
}

type Result struct {
	Title          string
	Summary        string
	FinalURL       string
	Text           string
	HTML           string
	ScreenshotPath string
	SnapshotPath   string
	Mode           string
	AuthState      string
	StatusCode     int
}

type cookieSpec struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Domain   string `json:"domain"`
	Path     string `json:"path"`
	HTTPOnly bool   `json:"http_only"`
	Secure   bool   `json:"secure"`
}

const (
	captureModeFull       = "full"
	captureModeScreenshot = "screenshot"
)

func NewService() *Service {
	return NewServiceWithRuntime(RuntimeConfig{})
}

func NewServiceWithRuntime(cfg RuntimeConfig) *Service {
	if strings.TrimSpace(cfg.Mode) == "" {
		cfg.Mode = "auto"
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 20 * time.Second
	}
	if strings.TrimSpace(cfg.ArtifactsRoot) == "" {
		cfg.ArtifactsRoot = filepath.Join(os.TempDir(), "nyx-browser")
	}
	if strings.TrimSpace(cfg.ExecutablePath) == "" {
		cfg.ExecutablePath = detectBrowserExecutable()
	}
	return &Service{
		cfg: cfg,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

func (s *Service) Navigate(ctx context.Context, req Request) Result {
	if strings.EqualFold(strings.TrimSpace(s.cfg.Mode), "http") {
		return s.navigateHTTP(ctx, req, nil)
	}

	result, err := s.navigateChromedp(ctx, req)
	if err == nil {
		return result
	}
	if strings.EqualFold(strings.TrimSpace(s.cfg.Mode), "chromedp") {
		return Result{
			FinalURL: req.URL,
			Mode:     "chromedp",
			Summary:  "Chromedp navigation failed: " + err.Error(),
		}
	}
	return s.navigateHTTP(ctx, req, err)
}

func (s *Service) navigateChromedp(ctx context.Context, req Request) (Result, error) {
	artifactsDir, err := s.prepareArtifactsDir(req.URL)
	if err != nil {
		return Result{}, err
	}
	mode := normalizeCaptureMode(req.CaptureMode)

	allocOptions := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", s.cfg.Headless),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("hide-scrollbars", true),
		chromedp.Flag("mute-audio", true),
	)
	if strings.TrimSpace(s.cfg.ExecutablePath) != "" {
		allocOptions = append(allocOptions, chromedp.ExecPath(s.cfg.ExecutablePath))
	}

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, allocOptions...)
	defer cancelAlloc()

	browserCtx, cancelBrowser := chromedp.NewContext(allocCtx)
	defer cancelBrowser()

	runCtx, cancel := context.WithTimeout(browserCtx, s.cfg.Timeout)
	defer cancel()

	finalURL := req.URL
	title := ""
	bodyText := ""
	fullHTML := ""
	var screenshot []byte

	actions := []chromedp.Action{
		network.Enable(),
	}
	if strings.TrimSpace(req.AuthHeader) != "" {
		actions = append(actions, network.SetExtraHTTPHeaders(network.Headers{
			"Authorization": req.AuthHeader,
		}))
	}
	actions = append(actions, chromedp.ActionFunc(func(ctx context.Context) error {
		return applyCookies(ctx, req.URL, req.CookiesJSON)
	}))
	actions = append(actions, chromedp.Navigate(req.URL))
	if wait := strings.TrimSpace(req.WaitSelector); wait != "" {
		actions = append(actions, chromedp.WaitVisible(wait))
	}
	if strings.TrimSpace(req.LocalStorageJSON) != "" || strings.TrimSpace(req.SessionStorageJSON) != "" {
		actions = append(actions, chromedp.ActionFunc(func(ctx context.Context) error {
			if err := applyStorage(ctx, "localStorage", req.LocalStorageJSON); err != nil {
				return err
			}
			if err := applyStorage(ctx, "sessionStorage", req.SessionStorageJSON); err != nil {
				return err
			}
			return chromedp.Reload().Do(ctx)
		}))
	}
	actions = append(actions, chromedp.Location(&finalURL), chromedp.Title(&title))
	if mode == captureModeFull {
		actions = append(actions,
			chromedp.Text("body", &bodyText, chromedp.ByQuery, chromedp.NodeVisible),
			chromedp.OuterHTML("html", &fullHTML, chromedp.ByQuery),
		)
	}
	actions = append(actions, chromedp.FullScreenshot(&screenshot, 90))

	if err := chromedp.Run(runCtx, actions...); err != nil {
		return Result{}, err
	}

	screenshotPath := filepath.Join(artifactsDir, "screenshot.jpg")
	if len(screenshot) > 0 {
		if err := os.WriteFile(screenshotPath, screenshot, 0o644); err != nil {
			return Result{}, err
		}
	}
	htmlPath := ""
	if mode == captureModeFull {
		htmlPath = filepath.Join(artifactsDir, "page.html")
		if err := os.WriteFile(htmlPath, []byte(fullHTML), 0o644); err != nil {
			return Result{}, err
		}
	}

	authState := "anonymous"
	if strings.TrimSpace(req.AuthHeader) != "" || strings.TrimSpace(req.CookiesJSON) != "" || strings.TrimSpace(req.LocalStorageJSON) != "" || strings.TrimSpace(req.SessionStorageJSON) != "" {
		authState = "session-replayed"
	}
	summary := summarizeNavigation("chromedp", finalURL, title, bodyText, authState)
	if mode == captureModeScreenshot {
		summary = summarizeScreenshotCapture("chromedp", finalURL, title, authState)
	}

	return Result{
		Title:          title,
		Summary:        summary,
		FinalURL:       finalURL,
		Text:           strings.TrimSpace(bodyText),
		HTML:           fullHTML,
		ScreenshotPath: screenshotPath,
		SnapshotPath:   htmlPath,
		Mode:           "chromedp",
		AuthState:      authState,
		StatusCode:     200,
	}, nil
}

func (s *Service) navigateHTTP(ctx context.Context, req Request, fallbackErr error) Result {
	artifactsDir, _ := s.prepareArtifactsDir(req.URL)
	captureMode := normalizeCaptureMode(req.CaptureMode)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, req.URL, nil)
	if err != nil {
		return Result{
			FinalURL: req.URL,
			Mode:     "http",
			Summary:  "HTTP browser fallback failed: " + err.Error(),
		}
	}
	if strings.TrimSpace(req.AuthHeader) != "" {
		httpReq.Header.Set("Authorization", req.AuthHeader)
	}
	if cookies, parseErr := parseCookies(req.CookiesJSON); parseErr == nil {
		for _, cookie := range cookies {
			httpReq.AddCookie(&http.Cookie{
				Name:     cookie.Name,
				Value:    cookie.Value,
				Path:     defaultCookiePath(cookie.Path),
				Domain:   cookie.Domain,
				HttpOnly: cookie.HTTPOnly,
				Secure:   cookie.Secure,
			})
		}
	}

	resp, err := s.client.Do(httpReq)
	if err != nil {
		summary := "HTTP browser fallback failed: " + err.Error()
		if fallbackErr != nil {
			summary = "Chromedp unavailable; HTTP fallback also failed: " + fallbackErr.Error() + "; " + err.Error()
		}
		return Result{
			FinalURL: req.URL,
			Mode:     "http",
			Summary:  summary,
		}
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	html := string(body)
	title := extractTitle(html)
	text := summarizeText(html)
	htmlPath := ""
	if artifactsDir != "" && captureMode == captureModeFull {
		htmlPath = filepath.Join(artifactsDir, "page.html")
		_ = os.WriteFile(htmlPath, body, 0o644)
	}

	authState := "anonymous"
	if strings.TrimSpace(req.AuthHeader) != "" || strings.TrimSpace(req.CookiesJSON) != "" {
		authState = "session-replayed"
	}
	mode := "http"
	if fallbackErr != nil {
		mode = "http-fallback"
	}
	summary := summarizeNavigation(mode, resp.Request.URL.String(), title, text, authState)
	if captureMode == captureModeScreenshot {
		summary = summarizeScreenshotFallback(mode, resp.Request.URL.String(), title, authState)
		text = ""
		html = ""
	}

	return Result{
		Title:        title,
		Summary:      summary,
		FinalURL:     resp.Request.URL.String(),
		Text:         text,
		HTML:         html,
		SnapshotPath: htmlPath,
		Mode:         mode,
		AuthState:    authState,
		StatusCode:   resp.StatusCode,
	}
}

func (s *Service) prepareArtifactsDir(rawURL string) (string, error) {
	target, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	host := target.Hostname()
	if host == "" {
		host = "unknown"
	}
	dir := filepath.Join(s.cfg.ArtifactsRoot, sanitizePathSegment(host), time.Now().UTC().Format("20060102T150405Z"))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func applyCookies(ctx context.Context, rawURL, raw string) error {
	cookies, err := parseCookies(raw)
	if err != nil || len(cookies) == 0 {
		return err
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	for _, cookie := range cookies {
		setter := network.SetCookie(cookie.Name, cookie.Value).
			WithDomain(defaultCookieDomain(cookie.Domain, parsed.Hostname())).
			WithPath(defaultCookiePath(cookie.Path)).
			WithHTTPOnly(cookie.HTTPOnly).
			WithSecure(cookie.Secure)
		if err := setter.Do(ctx); err != nil {
			return err
		}
	}
	return nil
}

func applyStorage(ctx context.Context, storageName, raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var items map[string]string
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return err
	}
	for key, value := range items {
		script := fmt.Sprintf(`window.%s.setItem(%q, %q)`, storageName, key, value)
		if err := chromedp.Evaluate(script, nil).Do(ctx); err != nil {
			return err
		}
	}
	return nil
}

func parseCookies(raw string) ([]cookieSpec, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var cookies []cookieSpec
	if err := json.Unmarshal([]byte(raw), &cookies); err != nil {
		return nil, err
	}
	return cookies, nil
}

func summarizeNavigation(mode, finalURL, title, text, authState string) string {
	if title == "" {
		title = "untitled page"
	}
	summary := fmt.Sprintf("Browser %s captured %s with title %q (%s).", mode, finalURL, title, authState)
	if excerpt := strings.TrimSpace(text); excerpt != "" {
		if len(excerpt) > 160 {
			excerpt = excerpt[:160] + "..."
		}
		summary += " Body excerpt: " + excerpt
	}
	return summary
}

func summarizeScreenshotCapture(mode, finalURL, title, authState string) string {
	if title == "" {
		title = "untitled page"
	}
	return fmt.Sprintf("Browser %s captured a screenshot for %s with title %q (%s).", mode, finalURL, title, authState)
}

func summarizeScreenshotFallback(mode, finalURL, title, authState string) string {
	return summarizeScreenshotCapture(mode, finalURL, title, authState) + " Screenshot-only capture is degraded because HTTP mode cannot render images."
}

func extractTitle(html string) string {
	re := regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	matches := re.FindStringSubmatch(html)
	if len(matches) < 2 {
		return "Target snapshot captured"
	}
	return strings.TrimSpace(stripTags(matches[1]))
}

func summarizeText(html string) string {
	text := stripTags(html)
	text = strings.Join(strings.Fields(text), " ")
	if len(text) > 2400 {
		text = text[:2400]
	}
	return text
}

func stripTags(input string) string {
	re := regexp.MustCompile(`(?s)<[^>]+>`)
	return re.ReplaceAllString(input, " ")
}

func sanitizePathSegment(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return "unknown"
	}
	var b strings.Builder
	for _, r := range input {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return strings.Trim(b.String(), "._-")
}

func defaultCookieDomain(raw, fallback string) string {
	if strings.TrimSpace(raw) != "" {
		return raw
	}
	return fallback
}

func defaultCookiePath(raw string) string {
	if strings.TrimSpace(raw) != "" {
		return raw
	}
	return "/"
}

func normalizeCaptureMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case captureModeScreenshot:
		return captureModeScreenshot
	default:
		return captureModeFull
	}
}

func detectBrowserExecutable() string {
	for _, candidate := range []string{
		"chromium",
		"chromium-browser",
		"google-chrome",
		"google-chrome-stable",
	} {
		if path, err := exec.LookPath(candidate); err == nil {
			return path
		}
	}
	return ""
}
