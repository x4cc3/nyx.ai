package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Mode              string
	WebMode           string
	DeepMode          string
	ExploitMode       string
	CodeMode          string
	BaseURL           string
	Timeout           time.Duration
	ResultLimit       int
	UserAgent         string
	TavilyAPIKey      string
	TavilyBaseURL     string
	PerplexityAPIKey  string
	PerplexityBaseURL string
	PerplexityModel   string
	SploitusBaseURL   string
	HTTPClient        *http.Client
}

type Service struct {
	cfg    Config
	client *http.Client
}

type Request struct {
	Target    string
	Objective string
	Query     string
	Kind      string
}

type Result struct {
	Query   string
	Source  string
	Summary string
	Results []Item
}

type Item struct {
	Title   string
	URL     string
	Snippet string
}

const (
	KindWeb     = "web"
	KindDeep    = "deep"
	KindExploit = "exploit"
	KindCode    = "code"
)

var (
	duckResultPattern = regexp.MustCompile(`(?is)<a[^>]+class="[^"]*result__a[^"]*"[^>]+href="([^"]+)"[^>]*>(.*?)</a>`)
	tagPattern        = regexp.MustCompile(`(?s)<[^>]+>`)
	spacePattern      = regexp.MustCompile(`\s+`)
)

func NewService() *Service {
	return NewServiceWithConfig(Config{})
}

func NewServiceWithConfig(cfg Config) *Service {
	if strings.TrimSpace(cfg.Mode) == "" {
		cfg.Mode = "duckduckgo"
	}
	if strings.TrimSpace(cfg.WebMode) == "" {
		cfg.WebMode = cfg.Mode
	}
	if strings.TrimSpace(cfg.DeepMode) == "" {
		cfg.DeepMode = "auto"
	}
	if strings.TrimSpace(cfg.ExploitMode) == "" {
		cfg.ExploitMode = "auto"
	}
	if strings.TrimSpace(cfg.CodeMode) == "" {
		cfg.CodeMode = "auto"
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 12 * time.Second
	}
	if cfg.ResultLimit < 1 {
		cfg.ResultLimit = 5
	}
	if strings.TrimSpace(cfg.UserAgent) == "" {
		cfg.UserAgent = "NYX/2 autonomous-search"
	}
	if strings.TrimSpace(cfg.TavilyBaseURL) == "" {
		cfg.TavilyBaseURL = "https://api.tavily.com/search"
	}
	if strings.TrimSpace(cfg.PerplexityBaseURL) == "" {
		cfg.PerplexityBaseURL = "https://api.perplexity.ai/chat/completions"
	}
	if strings.TrimSpace(cfg.PerplexityModel) == "" {
		cfg.PerplexityModel = "sonar-pro"
	}
	if strings.TrimSpace(cfg.SploitusBaseURL) == "" {
		cfg.SploitusBaseURL = "https://sploitus.com"
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: cfg.Timeout}
	}
	return &Service{cfg: cfg, client: client}
}

func (s *Service) Search(ctx context.Context, req Request) (Result, error) {
	query := composeQuery(req)
	if query == "" {
		return Result{}, fmt.Errorf("search query is required")
	}
	mode := s.resolveMode(req.Kind)
	switch mode {
	case "disabled":
		return Result{}, fmt.Errorf("search provider is disabled")
	case "duckduckgo":
		return s.searchDuckDuckGo(ctx, query)
	case "searxng":
		return s.searchSearxNG(ctx, query)
	case "tavily":
		return s.searchTavily(ctx, query, req.Kind)
	case "perplexity":
		return s.searchPerplexity(ctx, query)
	case "sploitus":
		return s.searchSploitus(ctx, query)
	default:
		return Result{}, fmt.Errorf("unsupported search mode %q", mode)
	}
}

func composeQuery(req Request) string {
	if query := strings.TrimSpace(req.Query); query != "" {
		return query
	}
	parts := make([]string, 0, 3)
	if target := strings.TrimSpace(req.Target); target != "" {
		parts = append(parts, target)
	}
	if objective := strings.TrimSpace(req.Objective); objective != "" {
		parts = append(parts, objective)
	}
	parts = append(parts, "security reconnaissance")
	return strings.Join(parts, " ")
}

func (s *Service) resolveMode(kind string) string {
	fallback := strings.ToLower(strings.TrimSpace(s.cfg.Mode))
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case KindDeep:
		return resolveSearchMode(strings.ToLower(strings.TrimSpace(s.cfg.DeepMode)), func() string {
			if strings.TrimSpace(s.cfg.PerplexityAPIKey) != "" {
				return "perplexity"
			}
			if strings.TrimSpace(s.cfg.TavilyAPIKey) != "" {
				return "tavily"
			}
			if strings.ToLower(strings.TrimSpace(s.cfg.WebMode)) == "searxng" {
				return "searxng"
			}
			return strings.ToLower(strings.TrimSpace(s.cfg.WebMode))
		}, fallback)
	case KindExploit:
		return resolveSearchMode(strings.ToLower(strings.TrimSpace(s.cfg.ExploitMode)), func() string {
			return "sploitus"
		}, fallback)
	case KindCode:
		return resolveSearchMode(strings.ToLower(strings.TrimSpace(s.cfg.CodeMode)), func() string {
			if strings.ToLower(strings.TrimSpace(s.cfg.WebMode)) == "searxng" {
				return "searxng"
			}
			return strings.ToLower(strings.TrimSpace(s.cfg.WebMode))
		}, fallback)
	default:
		return resolveSearchMode(strings.ToLower(strings.TrimSpace(s.cfg.WebMode)), func() string {
			return fallback
		}, fallback)
	}
}

func resolveSearchMode(mode string, auto func() string, fallback string) string {
	switch mode {
	case "", "auto":
		return auto()
	default:
		return mode
	}
}

func (s *Service) searchDuckDuckGo(ctx context.Context, query string) (Result, error) {
	base := strings.TrimSpace(s.cfg.BaseURL)
	if base == "" {
		base = "https://html.duckduckgo.com/html/"
	}
	u, err := url.Parse(base)
	if err != nil {
		return Result{}, err
	}
	q := u.Query()
	q.Set("q", query)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("User-Agent", s.cfg.UserAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Result{}, err
	}
	if resp.StatusCode >= 300 {
		return Result{}, fmt.Errorf("search provider failed with %d", resp.StatusCode)
	}

	results := parseDuckDuckGoResults(string(body), s.cfg.ResultLimit)
	if len(results) == 0 {
		return Result{}, fmt.Errorf("search returned no parsable results")
	}
	return Result{
		Query:   query,
		Source:  "duckduckgo-html",
		Summary: summarizeResults(query, results),
		Results: results,
	}, nil
}

func (s *Service) searchSearxNG(ctx context.Context, query string) (Result, error) {
	base := strings.TrimSpace(s.cfg.BaseURL)
	if base == "" {
		return Result{}, fmt.Errorf("SEARCH_BASE_URL is required for searxng mode")
	}
	u, err := url.Parse(base)
	if err != nil {
		return Result{}, err
	}
	if !strings.HasSuffix(u.Path, "/search") {
		u.Path = strings.TrimRight(u.Path, "/") + "/search"
	}
	q := u.Query()
	q.Set("q", query)
	q.Set("format", "json")
	q.Set("language", "en-US")
	q.Set("safesearch", "0")
	q.Set("categories", "general")
	q.Set("limit", strconv.Itoa(s.cfg.ResultLimit))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("User-Agent", s.cfg.UserAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Result{}, err
	}
	if resp.StatusCode >= 300 {
		return Result{}, fmt.Errorf("search provider failed with %d", resp.StatusCode)
	}

	var payload struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Result{}, err
	}

	items := make([]Item, 0, min(len(payload.Results), s.cfg.ResultLimit))
	for _, item := range payload.Results {
		if strings.TrimSpace(item.Title) == "" || strings.TrimSpace(item.URL) == "" {
			continue
		}
		items = append(items, Item{
			Title:   compact(item.Title),
			URL:     strings.TrimSpace(item.URL),
			Snippet: compact(item.Content),
		})
		if len(items) == s.cfg.ResultLimit {
			break
		}
	}
	if len(items) == 0 {
		return Result{}, fmt.Errorf("search returned no usable results")
	}
	return Result{
		Query:   query,
		Source:  "searxng",
		Summary: summarizeResults(query, items),
		Results: items,
	}, nil
}

func (s *Service) searchTavily(ctx context.Context, query, kind string) (Result, error) {
	if strings.TrimSpace(s.cfg.TavilyAPIKey) == "" {
		return Result{}, fmt.Errorf("tavily search requires TAVILY_API_KEY")
	}
	payload := map[string]any{
		"query":               query,
		"topic":               "general",
		"search_depth":        "basic",
		"max_results":         s.cfg.ResultLimit,
		"include_answer":      false,
		"include_raw_content": false,
	}
	if kind == KindDeep {
		payload["search_depth"] = "advanced"
		payload["include_answer"] = true
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return Result{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.TavilyBaseURL, bytes.NewReader(body))
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.TavilyAPIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", s.cfg.UserAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return Result{}, err
	}
	if resp.StatusCode >= 300 {
		return Result{}, fmt.Errorf("search provider failed with %d", resp.StatusCode)
	}

	var parsed struct {
		Answer  string `json:"answer"`
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"results"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return Result{}, err
	}

	items := make([]Item, 0, min(len(parsed.Results), s.cfg.ResultLimit))
	for _, item := range parsed.Results {
		if strings.TrimSpace(item.Title) == "" || strings.TrimSpace(item.URL) == "" {
			continue
		}
		items = append(items, Item{
			Title:   compact(item.Title),
			URL:     strings.TrimSpace(item.URL),
			Snippet: compact(item.Content),
		})
		if len(items) == s.cfg.ResultLimit {
			break
		}
	}
	if len(items) == 0 {
		return Result{}, fmt.Errorf("search returned no usable results")
	}
	summary := summarizeResults(query, items)
	if strings.TrimSpace(parsed.Answer) != "" {
		summary = compact(parsed.Answer)
	}
	return Result{
		Query:   query,
		Source:  "tavily",
		Summary: summary,
		Results: items,
	}, nil
}

func (s *Service) searchPerplexity(ctx context.Context, query string) (Result, error) {
	if strings.TrimSpace(s.cfg.PerplexityAPIKey) == "" {
		return Result{}, fmt.Errorf("perplexity search requires PERPLEXITY_API_KEY")
	}
	payload := map[string]any{
		"model": s.cfg.PerplexityModel,
		"messages": []map[string]string{
			{"role": "user", "content": query},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return Result{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.PerplexityBaseURL, bytes.NewReader(body))
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.PerplexityAPIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", s.cfg.UserAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return Result{}, err
	}
	if resp.StatusCode >= 300 {
		return Result{}, fmt.Errorf("search provider failed with %d", resp.StatusCode)
	}

	var parsed struct {
		Citations     []string `json:"citations"`
		SearchResults []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Snippet string `json:"snippet"`
		} `json:"search_results"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return Result{}, err
	}

	items := make([]Item, 0, s.cfg.ResultLimit)
	for _, item := range parsed.SearchResults {
		if strings.TrimSpace(item.URL) == "" {
			continue
		}
		title := compact(item.Title)
		if title == "" {
			title = hostFromURL(item.URL)
		}
		items = append(items, Item{
			Title:   title,
			URL:     strings.TrimSpace(item.URL),
			Snippet: compact(item.Snippet),
		})
		if len(items) == s.cfg.ResultLimit {
			break
		}
	}
	if len(items) == 0 {
		for _, citation := range parsed.Citations {
			if strings.TrimSpace(citation) == "" {
				continue
			}
			items = append(items, Item{
				Title: hostFromURL(citation),
				URL:   strings.TrimSpace(citation),
			})
			if len(items) == s.cfg.ResultLimit {
				break
			}
		}
	}
	if len(items) == 0 {
		return Result{}, fmt.Errorf("search returned no usable results")
	}
	summary := summarizeResults(query, items)
	if len(parsed.Choices) > 0 && strings.TrimSpace(parsed.Choices[0].Message.Content) != "" {
		summary = compact(parsed.Choices[0].Message.Content)
	}
	return Result{
		Query:   query,
		Source:  "perplexity",
		Summary: summary,
		Results: items,
	}, nil
}

func (s *Service) searchSploitus(ctx context.Context, query string) (Result, error) {
	base := strings.TrimSpace(s.cfg.BaseURL)
	if base == "" {
		base = "https://html.duckduckgo.com/html/"
	}
	searchQuery := fmt.Sprintf("site:%s/exploit %s", strings.TrimRight(strings.TrimSpace(s.cfg.SploitusBaseURL), "/"), query)
	u, err := url.Parse(base)
	if err != nil {
		return Result{}, err
	}
	q := u.Query()
	q.Set("q", searchQuery)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("User-Agent", s.cfg.UserAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Result{}, err
	}
	if resp.StatusCode >= 300 {
		return Result{}, fmt.Errorf("search provider failed with %d", resp.StatusCode)
	}

	results := parseDuckDuckGoResults(string(body), s.cfg.ResultLimit)
	if len(results) == 0 {
		return Result{}, fmt.Errorf("search returned no parsable results")
	}
	return Result{
		Query:   searchQuery,
		Source:  "sploitus",
		Summary: summarizeResults(searchQuery, results),
		Results: results,
	}, nil
}

func parseDuckDuckGoResults(body string, limit int) []Item {
	matches := duckResultPattern.FindAllStringSubmatch(body, limit)
	results := make([]Item, 0, len(matches))
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		rawURL := strings.TrimSpace(html.UnescapeString(match[1]))
		title := compact(stripTags(match[2]))
		if title == "" || rawURL == "" {
			continue
		}
		results = append(results, Item{
			Title: title,
			URL:   resolveDuckDuckGoURL(rawURL),
		})
	}
	return results
}

func resolveDuckDuckGoURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if uddg := u.Query().Get("uddg"); uddg != "" {
		if decoded, err := url.QueryUnescape(uddg); err == nil && decoded != "" {
			return decoded
		}
	}
	return raw
}

func summarizeResults(query string, results []Item) string {
	parts := make([]string, 0, min(3, len(results))+1)
	parts = append(parts, "Search query: "+query)
	for _, item := range results[:min(3, len(results))] {
		parts = append(parts, fmt.Sprintf("%s (%s)", item.Title, item.URL))
	}
	return strings.Join(parts, " | ")
}

func stripTags(value string) string {
	return tagPattern.ReplaceAllString(value, " ")
}

func compact(value string) string {
	value = html.UnescapeString(value)
	value = spacePattern.ReplaceAllString(value, " ")
	return strings.TrimSpace(value)
}

func hostFromURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Hostname() == "" {
		return strings.TrimSpace(raw)
	}
	return parsed.Hostname()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
