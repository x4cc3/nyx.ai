package bootstrap

import (
	"nyx/internal/config"
	"nyx/internal/memvec"
	"nyx/internal/openai"
)

func ConfigureEmbeddings(cfg config.Config) {
	switch cfg.MemoryEmbeddingsMode {
	case "openai":
		memvec.Configure(openai.NewEmbeddingProvider(openai.EmbeddingConfig{
			APIKey:     cfg.OpenAIAPIKey,
			BaseURL:    cfg.OpenAIBaseURL,
			Model:      cfg.OpenAIEmbeddingModel,
			Dimensions: cfg.OpenAIEmbeddingDims,
		}))
	case "auto":
		if cfg.OpenAIAPIKey != "" {
			memvec.Configure(openai.NewEmbeddingProvider(openai.EmbeddingConfig{
				APIKey:     cfg.OpenAIAPIKey,
				BaseURL:    cfg.OpenAIBaseURL,
				Model:      cfg.OpenAIEmbeddingModel,
				Dimensions: cfg.OpenAIEmbeddingDims,
			}))
			return
		}
		memvec.Configure(memvec.NewHashProvider(cfg.OpenAIEmbeddingDims))
	default:
		memvec.Configure(memvec.NewHashProvider(cfg.OpenAIEmbeddingDims))
	}
}
