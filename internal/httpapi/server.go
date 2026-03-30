package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"nyx/internal/auth"
	"nyx/internal/config"
	"nyx/internal/functions"
	"nyx/internal/observability"
	"nyx/internal/queue"
	"nyx/internal/store"
	"nyx/internal/version"
)

type Server struct {
	cfg      config.Config
	repo     store.Repository
	gateway  *functions.Gateway
	queue    queue.Transport
	logger   *slog.Logger
	metrics  *observability.Registry
	authn    auth.Authenticator
	inflight int64

	rateLimitRequests int
	rateLimitWindow   time.Duration
	rateLimitMu       sync.Mutex
	rateLimitBuckets  map[string]rateLimitBucket
}

type rateLimitBucket struct {
	count      int
	resetAt    time.Time
	lastAccess time.Time
}

type pageInfo struct {
	Limit     int    `json:"limit"`
	Returned  int    `json:"returned"`
	After     string `json:"after,omitempty"`
	NextAfter string `json:"next_after,omitempty"`
	HasMore   bool   `json:"has_more"`
}

type batchApprovalReviewRequest struct {
	ApprovalIDs []string `json:"approval_ids"`
	Approved    bool     `json:"approved"`
	Note        string   `json:"note"`
}

const (
	defaultHTTPRateLimitRequests = 120
	defaultHTTPRateLimitWindow   = time.Minute
	defaultListLimit             = 50
	maxListLimit                 = 200
)

type Option func(*Server)

func WithAuthenticator(authn auth.Authenticator) Option {
	return func(s *Server) {
		s.authn = authn
	}
}

func NewServer(cfg config.Config, repo store.Repository, gateway *functions.Gateway, transport queue.Transport, logger *slog.Logger, metrics *observability.Registry, opts ...Option) *Server {
	if logger == nil {
		logger = observability.NewLogger(observability.LoggerConfig{Service: cfg.ServiceName})
	}
	if metrics == nil {
		metrics = observability.NewRegistry()
	}
	server := &Server{
		cfg:               cfg,
		repo:              repo,
		gateway:           gateway,
		queue:             transport,
		logger:            logger,
		metrics:           metrics,
		rateLimitRequests: cfg.HTTPRateLimitRequests,
		rateLimitWindow:   cfg.HTTPRateLimitWindow,
		rateLimitBuckets:  make(map[string]rateLimitBucket),
	}
	if server.rateLimitRequests == 0 {
		server.rateLimitRequests = defaultHTTPRateLimitRequests
	}
	if server.rateLimitWindow == 0 {
		server.rateLimitWindow = defaultHTTPRateLimitWindow
	}
	for _, opt := range opts {
		if opt != nil {
			opt(server)
		}
	}
	return server
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/readyz", s.handleHealthz)
	mux.Handle("/metrics", s.metrics.Handler())
	mux.HandleFunc("/api/v1/functions", s.handleFunctions)
	mux.HandleFunc("/api/v1/architecture", s.handleArchitecture)
	mux.HandleFunc("/api/v1/approvals", s.handleApprovals)
	mux.HandleFunc("/api/v1/approvals/", s.handleApprovalRoutes)
	mux.HandleFunc("/api/v1/flows", s.handleFlows)
	mux.HandleFunc("/api/v1/flows/", s.handleFlowRoutes)
	mux.HandleFunc("/api/v1/workspaces", s.handleWorkspaces)
	mux.HandleFunc("/workspace", s.handleWorkspaceRoutes)
	mux.HandleFunc("/workspace/", s.handleWorkspaceRoutes)
	return s.withMaxBodySize(s.withJSON(s.withRequestLogging(s.withRequestContext(s.withRateLimit(mux)))))
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	status := http.StatusOK
	body := map[string]any{
		"status":            "ok",
		"service":           s.cfg.ServiceName,
		"version":           version.String(),
		"queue_mode":        s.queue.Mode(),
		"inflight_requests": atomic.LoadInt64(&s.inflight),
	}
	if err := s.repo.Ping(context.Background()); err != nil {
		status = http.StatusServiceUnavailable
		body["status"] = "degraded"
		body["error"] = err.Error()
	}
	writeJSON(w, status, body)
}

func (s *Server) handleFunctions(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"functions": s.gateway.Definitions()})
}

func (s *Server) handleArchitecture(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"name": "nyx-v2",
		"services": []string{
			"api",
			"orchestrator",
			"agent-runtime",
			"function-gateway",
			"executor-manager",
			"browser-service",
			"memory-service",
			"report-service",
		},
		"model": []string{
			"flows",
			"tasks",
			"subtasks",
			"actions",
			"artifacts",
			"memories",
			"findings",
			"agents",
			"executions",
		},
		"transport": map[string]string{
			"flow_dispatch":   s.queue.Mode(),
			"action_dispatch": s.queue.Mode(),
			"event_fanout":    s.queue.Mode(),
			"flow_events":     "postgres-sse",
			"dead_letter":     s.queue.Mode(),
		},
		"executor_mode":               s.cfg.ExecutorMode,
		"browser_mode":                s.browserMode(),
		"executor_network_mode":       s.executorNetworkMode(),
		"executor_network_name":       s.executorNetworkName(),
		"executor_net_raw_enabled":    s.cfg.ExecutorEnableNetRaw,
		"terminal_network_enabled":    s.terminalNetworkEnabled(),
		"risky_approval_required":     s.cfg.RequireRiskyApproval,
		"flow_max_concurrent_actions": s.cfg.FlowMaxConcurrentActions,
		"flow_min_action_interval_ms": s.cfg.FlowMinActionInterval.Milliseconds(),
		"browser_warning":             s.browserWarning(),
		"network_warning":             s.networkWarning(),
		"risk_warning":                s.riskWarning(),
		"version":                     version.String(),
	})
}

func (s *Server) browserMode() string {
	mode := strings.ToLower(strings.TrimSpace(s.cfg.BrowserMode))
	if mode == "" {
		return config.BrowserModeAuto
	}
	return mode
}

func (s *Server) browserWarning() string {
	if s.browserMode() == config.BrowserModeHTTP {
		return "Browser service is running in HTTP mode. Rendered screenshots and JavaScript execution require chromedp mode."
	}
	return ""
}

func (s *Server) executorNetworkMode() string {
	mode := strings.ToLower(strings.TrimSpace(s.cfg.ExecutorNetworkMode))
	if mode == "" {
		return config.NetworkModeNone
	}
	return mode
}

func (s *Server) executorNetworkName() string {
	if s.executorNetworkMode() != config.NetworkModeCustom {
		return ""
	}
	return strings.TrimSpace(s.cfg.ExecutorNetworkName)
}

func (s *Server) terminalNetworkEnabled() bool {
	switch s.executorNetworkMode() {
	case config.NetworkModeBridge, config.NetworkModeCustom:
		return true
	default:
		return false
	}
}

func (s *Server) networkWarning() string {
	if !s.terminalNetworkEnabled() {
		return ""
	}
	mode := s.executorNetworkMode()
	switch strings.ToLower(strings.TrimSpace(s.cfg.ExecutorMode)) {
	case config.ExecutorDocker:
		if mode == config.NetworkModeCustom && s.executorNetworkName() != "" {
			return fmt.Sprintf("Networked terminal execution is enabled in docker mode on network %q. Terminal scope checks still run before execution.", s.executorNetworkName())
		}
		return fmt.Sprintf("Networked terminal execution is enabled in docker mode with %s networking. Terminal scope checks still run before execution.", mode)
	case config.ExecutorAuto:
		if mode == config.NetworkModeCustom && s.executorNetworkName() != "" {
			return fmt.Sprintf("Auto executor mode may use networked docker workers on network %q. Terminal scope checks still run before execution.", s.executorNetworkName())
		}
		return fmt.Sprintf("Auto executor mode may use networked docker workers with %s networking. Terminal scope checks still run before execution.", mode)
	default:
		return ""
	}
}

func (s *Server) riskWarning() string {
	parts := make([]string, 0, 2)
	if s.cfg.RequireRiskyApproval {
		parts = append(parts, "High-risk terminal actions require operator approval before execution: raw socket scans, high-rate scans, and intrusive exploit tools.")
	}
	parts = append(parts, fmt.Sprintf("Per-flow execution is limited to %d concurrent action(s) with at least %dms between executor starts.", s.cfg.FlowMaxConcurrentActions, s.cfg.FlowMinActionInterval.Milliseconds()))
	return strings.Join(parts, " ")
}

func parseListLimit(raw string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return defaultListLimit, nil
	}
	limit, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("limit must be an integer")
	}
	if limit < 1 || limit > maxListLimit {
		return 0, fmt.Errorf("limit must be between 1 and %d", maxListLimit)
	}
	return limit, nil
}

func dedupeTrimmed(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func newPageInfo(limit int, after string, returned int, nextAfter string, hasMore bool) pageInfo {
	return pageInfo{
		Limit:     limit,
		Returned:  returned,
		After:     after,
		NextAfter: nextAfter,
		HasMore:   hasMore,
	}
}

func requestedIDs(values []string) []string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			parts = append(parts, part)
		}
	}
	return dedupeTrimmed(parts)
}

func (s *Server) withMaxBodySize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
			r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) withJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := s.cfg.CORSAllowedOrigins
		if origin == "" {
			origin = "*"
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-NYX-API-Key, X-NYX-Tenant, X-NYX-Operator")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) withRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/metrics" || s.rateLimitRequests <= 0 || s.rateLimitWindow <= 0 {
			next.ServeHTTP(w, r)
			return
		}

		now := time.Now()
		key := strings.Join([]string{
			currentTenant(r, s.cfg.DefaultTenant),
			currentOperator(r),
			r.Method,
			metricPath(r.URL.Path),
		}, ":")

		s.rateLimitMu.Lock()
		bucket := s.rateLimitBuckets[key]
		if bucket.resetAt.IsZero() || now.After(bucket.resetAt) {
			bucket = rateLimitBucket{resetAt: now.Add(s.rateLimitWindow)}
		}
		bucket.count++
		bucket.lastAccess = now
		s.rateLimitBuckets[key] = bucket
		// Evict stale buckets older than 10 minutes
		if len(s.rateLimitBuckets) > 100 {
			for k, b := range s.rateLimitBuckets {
				if now.Sub(b.lastAccess) > 10*time.Minute {
					delete(s.rateLimitBuckets, k)
				}
			}
		}
		remaining := s.rateLimitRequests - bucket.count
		retryAfter := int(time.Until(bucket.resetAt).Seconds())
		s.rateLimitMu.Unlock()

		if remaining < 0 {
			if retryAfter < 1 {
				retryAfter = 1
			}
			w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
			writeError(w, http.StatusTooManyRequests, "rate_limit_exceeded", "API rate limit exceeded")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) withRequestContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" || r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}
		apiKeyValid := s.cfg.APIKey != "" && r.Header.Get("X-NYX-API-Key") == s.cfg.APIKey

		var identity *auth.Identity
		if s.authn != nil {
			token := bearerToken(r.Header.Get("Authorization"))
			if token == "" && !apiKeyValid {
				writeError(w, http.StatusUnauthorized, "unauthorized", "Missing bearer token")
				return
			}
			if token != "" {
				verified, err := s.authn.Verify(r.Context(), token)
				if err != nil {
					writeError(w, http.StatusUnauthorized, "invalid_token", "Missing or invalid bearer token")
					return
				}
				identity = &verified
			}
		} else if s.cfg.APIKey != "" && !apiKeyValid {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid API key")
			return
		}

		tenantID := tenantForRequest(r, s.cfg.DefaultTenant, identity)
		operator := operatorForRequest(r, identity)
		ctx, traceID := observability.EnsureTrace(r.Context())
		ctx = context.WithValue(ctx, tenantContextKey{}, tenantID)
		ctx = context.WithValue(ctx, operatorContextKey{}, operator)
		if identity != nil {
			ctx = context.WithValue(ctx, identityContextKey{}, *identity)
		}
		w.Header().Set("X-NYX-Trace-ID", traceID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) withRequestLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		atomic.AddInt64(&s.inflight, 1)
		defer atomic.AddInt64(&s.inflight, -1)

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		traceID := observability.TraceID(r.Context())
		s.metrics.IncCounter("nyx_http_requests_total", map[string]string{
			"method": r.Method,
			"path":   metricPath(r.URL.Path),
			"status": fmt.Sprintf("%d", rec.status),
		}, 1)
		s.metrics.SetGauge("nyx_http_inflight_requests", nil, float64(atomic.LoadInt64(&s.inflight)))
		s.logger.Info("http request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration_ms", time.Since(started).Milliseconds(),
			"tenant_id", currentTenant(r, s.cfg.DefaultTenant),
			"trace_id", traceID,
		)
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		slog.Default().Warn("failed to encode JSON response", "status", status, "error", err)
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeErrorWithFields(w, status, code, message, nil)
}

func writeErrorWithFields(w http.ResponseWriter, status int, code, message string, fieldErrors map[string]string) {
	payload := map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	}
	if len(fieldErrors) > 0 {
		payload["error"].(map[string]any)["field_errors"] = fieldErrors
	}
	writeJSON(w, status, map[string]any{
		"error": payload["error"],
	})
}

func writeSSE(w http.ResponseWriter, event string, payload any) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		slog.Default().Warn("failed to marshal SSE payload", "event", event, "error", err)
		return
	}
	fmt.Fprintf(w, "event: %s\n", event)
	fmt.Fprintf(w, "data: %s\n\n", bytes)
}

type tenantContextKey struct{}
type operatorContextKey struct{}
type identityContextKey struct{}

func bearerToken(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}
	if !strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return ""
	}
	return strings.TrimSpace(header[len("Bearer "):])
}

func tenantForRequest(r *http.Request, fallback string, identity *auth.Identity) string {
	if identity != nil && strings.TrimSpace(identity.TenantID) != "" {
		return strings.TrimSpace(identity.TenantID)
	}
	return currentTenant(r, fallback)
}

func operatorForRequest(r *http.Request, identity *auth.Identity) string {
	if identity != nil {
		if strings.TrimSpace(identity.Email) != "" {
			return strings.TrimSpace(identity.Email)
		}
		if strings.TrimSpace(identity.Subject) != "" {
			return strings.TrimSpace(identity.Subject)
		}
	}
	return currentOperator(r)
}

func currentTenant(r *http.Request, fallback string) string {
	if value, ok := r.Context().Value(tenantContextKey{}).(string); ok && strings.TrimSpace(value) != "" {
		return value
	}
	value := strings.TrimSpace(r.Header.Get("X-NYX-Tenant"))
	if value == "" {
		value = strings.TrimSpace(fallback)
	}
	if value == "" {
		value = "default"
	}
	return value
}

func currentOperator(r *http.Request) string {
	if value, ok := r.Context().Value(operatorContextKey{}).(string); ok && strings.TrimSpace(value) != "" {
		return value
	}
	value := strings.TrimSpace(r.Header.Get("X-NYX-Operator"))
	if value == "" {
		value = "anonymous"
	}
	return value
}

// responseRecorder is a minimal ResponseWriter that discards output.
// Used when dispatching flows from approval handlers where we need to call
// dispatchFlow but return a different HTTP response.
type responseRecorder struct {
	header http.Header
}

func (r *responseRecorder) Header() http.Header {
	return r.header
}

func (r *responseRecorder) Write([]byte) (int, error) {
	return 0, nil
}

func (r *responseRecorder) WriteHeader(int) {}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func metricPath(path string) string {
	for _, prefix := range []string{"/api/v1/flows/", "/api/v1/approvals/", "/workspace/flows/"} {
		if strings.HasPrefix(path, prefix) {
			return prefix + ":id"
		}
	}
	return path
}
