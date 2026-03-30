package memvec

import (
	"context"
	"errors"
	"testing"
)

type stubProvider struct {
	name       string
	dimensions int
	vector     []float32
	err        error
}

func (s stubProvider) Name() string    { return s.name }
func (s stubProvider) Dimensions() int { return s.dimensions }
func (s stubProvider) Embed(context.Context, string) ([]float32, error) {
	if s.err != nil {
		return nil, s.err
	}
	return append([]float32(nil), s.vector...), nil
}

func TestPrepareUsesConfiguredProvider(t *testing.T) {
	original := CurrentProvider()
	t.Cleanup(func() { Configure(original) })
	Configure(stubProvider{name: "text-embedding-3-small", dimensions: 3, vector: []float32{1, 2, 3}})

	_, metadata, embedding := Prepare("observation", "hello world", nil)
	if metadata["embedding_model"] != "text-embedding-3-small" {
		t.Fatalf("unexpected model: %s", metadata["embedding_model"])
	}
	if len(embedding) != 3 || embedding[2] != 3 {
		t.Fatalf("unexpected embedding: %+v", embedding)
	}
}

func TestPrepareFallsBackToHashProvider(t *testing.T) {
	original := CurrentProvider()
	t.Cleanup(func() { Configure(original) })
	Configure(stubProvider{name: "text-embedding-3-small", dimensions: 8, err: errors.New("boom")})

	_, metadata, embedding := Prepare("observation", "hello world", nil)
	if metadata["embedding_fallback"] != "true" {
		t.Fatalf("expected fallback metadata, got %+v", metadata)
	}
	if metadata["embedding_model"] != "nyx-hash-1536" && metadata["embedding_model"] != "nyx-hash-8" {
		t.Fatalf("unexpected fallback model: %s", metadata["embedding_model"])
	}
	if len(embedding) != 1536 && len(embedding) != 8 {
		t.Fatalf("unexpected embedding length: %d", len(embedding))
	}
}
