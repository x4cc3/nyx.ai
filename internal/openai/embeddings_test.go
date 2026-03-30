package openai

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestEmbeddingProviderParsesResponse(t *testing.T) {
	provider := NewEmbeddingProvider(EmbeddingConfig{
		APIKey:     "test-key",
		BaseURL:    "https://example.test/v1",
		Model:      "text-embedding-3-small",
		Dimensions: 3,
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(`{"data":[{"embedding":[0.1,0.2,0.3]}]}`)),
					Request:    req,
				}, nil
			}),
		},
	})

	embedding, err := provider.Embed(context.Background(), "login page")
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	if len(embedding) != 3 || embedding[2] != 0.3 {
		t.Fatalf("unexpected embedding: %+v", embedding)
	}
}
