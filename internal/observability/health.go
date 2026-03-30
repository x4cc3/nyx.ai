package observability

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type CheckFunc func(context.Context) error

type Health struct {
	Service string
	Checks  map[string]CheckFunc
}

func (h Health) ReadyHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status, body := h.run(r.Context())
		writeHealthJSON(w, status, body)
	})
}

func (h Health) run(ctx context.Context) (int, map[string]any) {
	status := http.StatusOK
	body := map[string]any{
		"status":  "ok",
		"service": h.Service,
		"checks":  map[string]string{},
	}
	checks := body["checks"].(map[string]string)
	for name, fn := range h.Checks {
		checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		err := fn(checkCtx)
		cancel()
		if err != nil {
			status = http.StatusServiceUnavailable
			body["status"] = "degraded"
			checks[name] = err.Error()
			continue
		}
		checks[name] = "ok"
	}
	return status, body
}

func StartServer(ctx context.Context, addr string, logger Logger, health Health, metrics *Registry) *http.Server {
	if addr == "" {
		return nil
	}
	mux := http.NewServeMux()
	mux.Handle("/healthz", health.ReadyHandler())
	mux.Handle("/readyz", health.ReadyHandler())
	if metrics != nil {
		mux.Handle("/metrics", metrics.Handler())
	}
	server := &http.Server{Addr: addr, Handler: mux}
	go func() {
		logger.Info("observability server starting", "addr", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("observability server failed", "error", err.Error())
		}
	}()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	return server
}

type Logger interface {
	Info(string, ...any)
	Error(string, ...any)
}

func writeHealthJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
