package search

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestDuckDuckGoSearchParsesResults(t *testing.T) {
	service := NewServiceWithConfig(Config{
		Mode: "duckduckgo",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if !strings.Contains(req.URL.RawQuery, "q=acme") {
					t.Fatalf("expected query in request, got %s", req.URL.RawQuery)
				}
				body := `
<html><body>
  <a class="result__a" href="//duckduckgo.com/l/?uddg=https%3A%2F%2Fexample.com%2Flogin">Acme Login Portal</a>
  <a class="result__a" href="https://docs.example.com">Acme Developer Docs</a>
</body></html>`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(body)),
					Header:     make(http.Header),
					Request:    req,
				}, nil
			}),
		},
	})

	result, err := service.Search(context.Background(), Request{Query: "acme"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if result.Source != "duckduckgo-html" {
		t.Fatalf("unexpected source: %s", result.Source)
	}
	if len(result.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result.Results))
	}
	if result.Results[0].URL != "https://example.com/login" {
		t.Fatalf("unexpected url: %s", result.Results[0].URL)
	}
}

func TestSearchDeepUsesPerplexityProvider(t *testing.T) {
	var authHeader string
	service := NewServiceWithConfig(Config{
		Mode:              "duckduckgo",
		DeepMode:          "perplexity",
		PerplexityAPIKey:  "perplexity-key",
		PerplexityBaseURL: "https://perplexity.example.test/chat/completions",
		PerplexityModel:   "sonar-pro",
		ResultLimit:       3,
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				authHeader = req.Header.Get("Authorization")
				if req.URL.String() != "https://perplexity.example.test/chat/completions" {
					t.Fatalf("unexpected url: %s", req.URL.String())
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"search_results":[{"title":"Acme KB","url":"https://kb.example.com/auth","snippet":"Auth internals"}],
						"choices":[{"message":{"content":"Deep answer with cited context."}}]
					}`)),
					Header:  make(http.Header),
					Request: req,
				}, nil
			}),
		},
	})

	result, err := service.Search(context.Background(), Request{Kind: KindDeep, Query: "acme auth flow"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if authHeader != "Bearer perplexity-key" {
		t.Fatalf("unexpected auth header: %q", authHeader)
	}
	if result.Source != "perplexity" {
		t.Fatalf("unexpected source: %s", result.Source)
	}
	if result.Summary != "Deep answer with cited context." {
		t.Fatalf("unexpected summary: %q", result.Summary)
	}
	if len(result.Results) != 1 || result.Results[0].URL != "https://kb.example.com/auth" {
		t.Fatalf("unexpected results: %+v", result.Results)
	}
}

func TestSearchWebUsesTavilyProvider(t *testing.T) {
	var requestBody map[string]any
	service := NewServiceWithConfig(Config{
		Mode:          "duckduckgo",
		WebMode:       "tavily",
		TavilyAPIKey:  "tavily-key",
		TavilyBaseURL: "https://tavily.example.test/search",
		ResultLimit:   4,
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if err := json.NewDecoder(req.Body).Decode(&requestBody); err != nil {
					t.Fatalf("decode request: %v", err)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"answer":"Tavily summary",
						"results":[{"title":"Acme Docs","url":"https://docs.example.com","content":"API auth guide"}]
					}`)),
					Header:  make(http.Header),
					Request: req,
				}, nil
			}),
		},
	})

	result, err := service.Search(context.Background(), Request{Kind: KindWeb, Query: "acme docs"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if result.Source != "tavily" {
		t.Fatalf("unexpected source: %s", result.Source)
	}
	if requestBody["query"] != "acme docs" {
		t.Fatalf("unexpected query payload: %+v", requestBody)
	}
	if requestBody["search_depth"] != "basic" {
		t.Fatalf("unexpected search depth: %+v", requestBody)
	}
}

func TestSearchExploitUsesSploitusProvider(t *testing.T) {
	var capturedQuery string
	service := NewServiceWithConfig(Config{
		Mode:            "duckduckgo",
		ExploitMode:     "sploitus",
		SploitusBaseURL: "https://sploitus.example.test",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				capturedQuery = req.URL.Query().Get("q")
				body := `<html><body><a class="result__a" href="https://sploitus.example.test/exploit?id=42">PoC</a></body></html>`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(body)),
					Header:     make(http.Header),
					Request:    req,
				}, nil
			}),
		},
	})

	result, err := service.Search(context.Background(), Request{Kind: KindExploit, Query: "acme auth bypass"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if !strings.Contains(capturedQuery, "site:https://sploitus.example.test/exploit acme auth bypass") {
		t.Fatalf("unexpected query: %q", capturedQuery)
	}
	if result.Source != "sploitus" {
		t.Fatalf("unexpected source: %s", result.Source)
	}
	if result.Query != capturedQuery {
		t.Fatalf("expected returned query to match provider query, got %q want %q", result.Query, capturedQuery)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
