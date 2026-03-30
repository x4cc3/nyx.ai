package functions

import (
	"context"
	"strconv"

	"nyx/internal/domain"
	"nyx/internal/executor"
	"nyx/internal/services/browser"
	"nyx/internal/services/memory"
	"nyx/internal/services/search"
)

type Gateway struct {
	browser *browser.Service
	memory  *memory.Service
	search  *search.Service
	exec    executor.Manager
	defs    []domain.FunctionDef
	tools   map[string]ToolSpec
}

type Option func(*Gateway)

func WithSearchService(service *search.Service) Option {
	return func(g *Gateway) {
		if service != nil {
			g.search = service
		}
	}
}

type CallResult struct {
	Output  map[string]string
	Profile string
	Runtime string
	Err     error
}

func NewGateway(browserService *browser.Service, memoryService *memory.Service, execManager executor.Manager, opts ...Option) *Gateway {
	gateway := &Gateway{
		browser: browserService,
		memory:  memoryService,
		search:  search.NewService(),
		exec:    execManager,
	}
	for _, opt := range opts {
		opt(gateway)
	}
	specs := gateway.buildRegistry()
	gateway.tools = make(map[string]ToolSpec, len(specs))
	gateway.defs = make([]domain.FunctionDef, 0, len(specs))
	for _, spec := range specs {
		gateway.tools[spec.Definition.Name] = spec
		gateway.defs = append(gateway.defs, spec.Definition)
	}
	return gateway
}

func (g *Gateway) Definitions() []domain.FunctionDef {
	out := make([]domain.FunctionDef, len(g.defs))
	copy(out, g.defs)
	return out
}

func (g *Gateway) Call(ctx context.Context, flowID, actionID, name string, input map[string]string) CallResult {
	spec, ok := g.tools[name]
	if !ok || spec.Handler == nil {
		return CallResult{
			Profile: "unknown",
			Runtime: "go-function-gateway",
			Output: map[string]string{
				"summary": "Unknown function: " + name,
			},
		}
	}
	return spec.Handler(ctx, flowID, actionID, input)
}

func executorOutput(result executor.Result) map[string]string {
	output := map[string]string{
		"summary":     result.Summary,
		"stdout":      result.Stdout,
		"stderr":      result.Stderr,
		"workspace":   result.Workspace,
		"exit_code":   strconv.Itoa(result.ExitCode),
		"duration_ms": strconv.FormatInt(result.Duration.Milliseconds(), 10),
		"attempts":    strconv.Itoa(result.Attempts),
	}
	for key, value := range result.Metadata {
		output[key] = value
	}
	return output
}
