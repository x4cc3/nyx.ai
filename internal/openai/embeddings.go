package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type EmbeddingConfig struct {
	APIKey     string
	BaseURL    string
	Model      string
	Dimensions int
	HTTPClient *http.Client
}

type EmbeddingProvider struct {
	apiKey     string
	baseURL    string
	model      string
	dimensions int
	httpClient *http.Client
}

func NewEmbeddingProvider(cfg EmbeddingConfig) *EmbeddingProvider {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = "text-embedding-3-small"
	}
	dimensions := cfg.Dimensions
	if dimensions < 16 {
		dimensions = 1536
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	return &EmbeddingProvider{
		apiKey:     cfg.APIKey,
		baseURL:    baseURL,
		model:      model,
		dimensions: dimensions,
		httpClient: client,
	}
}

func (p *EmbeddingProvider) Name() string {
	return p.model
}

func (p *EmbeddingProvider) Dimensions() int {
	return p.dimensions
}

func (p *EmbeddingProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	if strings.TrimSpace(p.apiKey) == "" {
		return nil, fmt.Errorf("missing OPENAI_API_KEY")
	}
	payload := map[string]any{
		"input":           text,
		"model":           p.model,
		"dimensions":      p.dimensions,
		"encoding_format": "float",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("openai embeddings api failed with %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	var parsed struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Data) == 0 || len(parsed.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("openai embeddings response did not include an embedding")
	}
	return parsed.Data[0].Embedding, nil
}
