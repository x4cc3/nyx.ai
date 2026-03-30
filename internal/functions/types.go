package functions

import (
	"context"

	"nyx/internal/domain"
)

type Handler func(context.Context, string, string, map[string]string) CallResult

type ToolSpec struct {
	Definition domain.FunctionDef
	Handler    Handler
}
