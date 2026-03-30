package memvec

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"math"
	"strings"
	"sync"
)

const DefaultDimension = 1536

type Provider interface {
	Name() string
	Dimensions() int
	Embed(context.Context, string) ([]float32, error)
}

var (
	providerMu sync.RWMutex
	provider   Provider = NewHashProvider(DefaultDimension)
)

type HashProvider struct {
	dimension int
}

func NewHashProvider(dimension int) *HashProvider {
	if dimension < 16 {
		dimension = DefaultDimension
	}
	return &HashProvider{dimension: dimension}
}

func (p *HashProvider) Name() string {
	return fmt.Sprintf("nyx-hash-%d", p.dimension)
}

func (p *HashProvider) Dimensions() int {
	return p.dimension
}

func (p *HashProvider) Embed(_ context.Context, text string) ([]float32, error) {
	return hashEmbed(text, p.dimension), nil
}

func Configure(next Provider) {
	providerMu.Lock()
	defer providerMu.Unlock()
	if next == nil {
		provider = NewHashProvider(DefaultDimension)
		return
	}
	provider = next
}

func CurrentProvider() Provider {
	providerMu.RLock()
	defer providerMu.RUnlock()
	return provider
}

func Dimensions() int {
	return CurrentProvider().Dimensions()
}

func Prepare(kind, content string, metadata map[string]string) (string, map[string]string, []float32) {
	clean := NormalizeContent(content)
	out := cloneMap(metadata)
	out["retention_policy"] = RetentionPolicy(kind)
	out["content_hash"] = ContentHash(clean)

	current := CurrentProvider()
	embedding, err := current.Embed(context.Background(), clean)
	if err != nil {
		fallback := NewHashProvider(current.Dimensions())
		embedding, _ = fallback.Embed(context.Background(), clean)
		out["embedding_model"] = fallback.Name()
		out["embedding_fallback"] = "true"
		out["embedding_error"] = truncate(err.Error(), 240)
		return clean, out, embedding
	}
	out["embedding_model"] = current.Name()
	return clean, out, embedding
}

func NormalizeContent(content string) string {
	content = strings.Join(strings.Fields(strings.TrimSpace(content)), " ")
	if len(content) > 4000 {
		content = content[:4000]
	}
	return content
}

func RetentionPolicy(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "evidence", "finding", "browser-snapshot", "exploit_reference", "reference_material", "operator_note":
		return "long"
	case "prompt", "transient":
		return "short"
	default:
		return "standard"
	}
}

func ContentHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:8])
}

func Embed(text string) []float32 {
	current := CurrentProvider()
	embedding, err := current.Embed(context.Background(), NormalizeContent(text))
	if err == nil {
		return embedding
	}
	fallback := NewHashProvider(current.Dimensions())
	embedding, _ = fallback.Embed(context.Background(), NormalizeContent(text))
	return embedding
}

func VectorLiteral(vec []float32) string {
	parts := make([]string, 0, len(vec))
	for _, value := range vec {
		parts = append(parts, fmt.Sprintf("%.6f", value))
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func Similarity(a, b []float32) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	var dot float64
	for i := 0; i < n; i++ {
		dot += float64(a[i] * b[i])
	}
	return dot
}

func hashEmbed(text string, dimension int) []float32 {
	vec := make([]float32, dimension)
	tokens := strings.Fields(strings.ToLower(text))
	if len(tokens) == 0 {
		return vec
	}
	for _, token := range tokens {
		token = strings.Trim(token, " \t\r\n.,:;!?()[]{}<>\"'`")
		if token == "" {
			continue
		}
		hasher := fnv.New32a()
		_, _ = hasher.Write([]byte(token))
		value := hasher.Sum32()
		index := int(value % uint32(dimension))
		sign := float32(1)
		if value&(1<<31) != 0 {
			sign = -1
		}
		vec[index] += sign
	}
	normalize(vec)
	return vec
}

func normalize(vec []float32) {
	var sum float64
	for _, value := range vec {
		sum += float64(value * value)
	}
	if sum == 0 {
		return
	}
	scale := float32(1 / math.Sqrt(sum))
	for i := range vec {
		vec[i] *= scale
	}
}

func cloneMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func truncate(value string, maxLen int) string {
	if len(value) <= maxLen {
		return value
	}
	return value[:maxLen]
}
