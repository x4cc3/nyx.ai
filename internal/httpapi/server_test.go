package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nyx/internal/auth"
	"nyx/internal/config"
	"nyx/internal/domain"
	"nyx/internal/executor"
	"nyx/internal/functions"
	"nyx/internal/queue"
	"nyx/internal/services/browser"
	"nyx/internal/services/memory"
	"nyx/internal/store"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()

	repo := store.NewMemoryStore()
	memoryService := memory.New(repo)
	browserService := browser.NewService()
	gateway := functions.NewGateway(browserService, memoryService, executor.NewLocalManagerWithRoot(t.TempDir()))
	return NewServer(config.Config{
		ListenAddr:          ":0",
		ServiceName:         "test",
		ExecutorMode:        "local",
		DefaultTenant:       "default",
		RequireFlowApproval: false,
	}, repo, gateway, queue.NewNoopTransport(), nil, nil)
}

type stubAuthenticator struct {
	identity auth.Identity
	err      error
}

func (s stubAuthenticator) Verify(_ context.Context, _ string) (auth.Identity, error) {
	if s.err != nil {
		return auth.Identity{}, s.err
	}
	return s.identity, nil
}

type failingFlowRunTransport struct {
	*queue.NoopTransport
	err error
}

func (t failingFlowRunTransport) PublishFlowRun(context.Context, queue.FlowRunMessage) error {
	return t.err
}

func TestCreateAndFetchFlow(t *testing.T) {
	server := newTestServer(t)
	handler := server.Handler()

	body, err := json.Marshal(domain.CreateFlowInput{
		Name:      "Acme",
		Target:    "https://app.example.com",
		Objective: "Rebuild NYX",
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/flows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}

	var flow domain.Flow
	if err := json.Unmarshal(rec.Body.Bytes(), &flow); err != nil {
		t.Fatalf("decode flow: %v", err)
	}
	if flow.ID == "" {
		t.Fatal("expected flow id to be set")
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/flows/"+flow.ID, nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected detail status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestQueueFlow(t *testing.T) {
	server := newTestServer(t)
	handler := server.Handler()

	body, _ := json.Marshal(domain.CreateFlowInput{
		Name:      "Acme",
		Target:    "https://app.example.com",
		Objective: "Queue the flow",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/flows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var flow domain.Flow
	if err := json.Unmarshal(rec.Body.Bytes(), &flow); err != nil {
		t.Fatalf("decode flow: %v", err)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/flows/"+flow.ID+"/start", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, rec.Code)
	}
}

func TestQueueFlowRollbackOnPublishFailure(t *testing.T) {
	repo := store.NewMemoryStore()
	memoryService := memory.New(repo)
	browserService := browser.NewService()
	gateway := functions.NewGateway(browserService, memoryService, executor.NewLocalManagerWithRoot(t.TempDir()))
	server := NewServer(config.Config{
		ListenAddr:          ":0",
		ServiceName:         "test",
		ExecutorMode:        "local",
		DefaultTenant:       "default",
		RequireFlowApproval: false,
	}, repo, gateway, failingFlowRunTransport{
		NoopTransport: queue.NewNoopTransport(),
		err:           errors.New("broker unavailable"),
	}, nil, nil)
	handler := server.Handler()

	body, _ := json.Marshal(domain.CreateFlowInput{
		Name:      "Acme",
		Target:    "https://app.example.com",
		Objective: "Queue the flow",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/flows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var flow domain.Flow
	if err := json.Unmarshal(rec.Body.Bytes(), &flow); err != nil {
		t.Fatalf("decode flow: %v", err)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/flows/"+flow.ID+"/start", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}

	updatedFlow, err := repo.GetFlow(context.Background(), flow.ID)
	if err != nil {
		t.Fatalf("get flow: %v", err)
	}
	if updatedFlow.Status != domain.StatusPending {
		t.Fatalf("expected pending flow after dispatch failure, got %s", updatedFlow.Status)
	}
}

func TestListFunctions(t *testing.T) {
	server := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/functions", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var payload struct {
		Functions []domain.FunctionDef `json:"functions"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode functions response: %v", err)
	}
	if len(payload.Functions) == 0 {
		t.Fatal("expected functions to be returned")
	}
	var foundTerminalExec bool
	var foundBrowserScreenshot bool
	for _, def := range payload.Functions {
		if def.Name == "terminal_exec" {
			foundTerminalExec = true
			if !def.RequiresPentestImage {
				t.Fatal("expected terminal_exec to expose pentest-image metadata")
			}
			if def.Category != "environment" {
				t.Fatalf("unexpected category for terminal_exec: %q", def.Category)
			}
		}
		if def.Name == "browser_screenshot" {
			foundBrowserScreenshot = true
			if def.Category != "search_network" {
				t.Fatalf("unexpected category for browser_screenshot: %q", def.Category)
			}
		}
	}
	if !foundTerminalExec {
		t.Fatal("expected terminal_exec in the API function catalog")
	}
	if !foundBrowserScreenshot {
		t.Fatal("expected browser_screenshot in the API function catalog")
	}
}

func TestTenantIsolationForFlows(t *testing.T) {
	server := newTestServer(t)
	handler := server.Handler()

	for _, tc := range []struct {
		tenant string
		name   string
	}{
		{tenant: "alpha", name: "Alpha flow"},
		{tenant: "bravo", name: "Bravo flow"},
	} {
		body, _ := json.Marshal(domain.CreateFlowInput{
			Name:      tc.name,
			Target:    "https://app.example.com",
			Objective: "Tenant isolation",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/flows", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-NYX-Tenant", tc.tenant)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create flow for %s: expected 201, got %d", tc.tenant, rec.Code)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/flows", nil)
	req.Header.Set("X-NYX-Tenant", "alpha")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var payload struct {
		Flows []domain.Flow `json:"flows"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode flows: %v", err)
	}
	if len(payload.Flows) != 1 || payload.Flows[0].TenantID != "alpha" {
		t.Fatalf("expected only alpha flows, got %+v", payload.Flows)
	}
}

func TestCreateFlowValidationReturnsFieldErrors(t *testing.T) {
	server := newTestServer(t)
	handler := server.Handler()

	body, _ := json.Marshal(domain.CreateFlowInput{
		Name:      "",
		Target:    "not-a-url",
		Objective: "",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/flows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	var payload struct {
		Error struct {
			Code        string            `json:"code"`
			FieldErrors map[string]string `json:"field_errors"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode error payload: %v", err)
	}
	if payload.Error.Code != "invalid_flow" {
		t.Fatalf("unexpected error code: %q", payload.Error.Code)
	}
	if payload.Error.FieldErrors["target"] == "" || payload.Error.FieldErrors["objective"] == "" {
		t.Fatalf("expected field errors, got %+v", payload.Error.FieldErrors)
	}
}

func TestListFlowsPagination(t *testing.T) {
	server := newTestServer(t)
	handler := server.Handler()

	for _, name := range []string{"flow-a", "flow-b", "flow-c"} {
		body, _ := json.Marshal(domain.CreateFlowInput{
			Name:      name,
			Target:    "https://app.example.com",
			Objective: "pagination coverage",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/flows", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create flow %s: expected 201, got %d", name, rec.Code)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/flows?limit=2", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var payload struct {
		Flows    []domain.Flow `json:"flows"`
		PageInfo pageInfo      `json:"page_info"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Flows) != 2 {
		t.Fatalf("expected 2 flows, got %d", len(payload.Flows))
	}
	if !payload.PageInfo.HasMore || payload.PageInfo.NextAfter == "" {
		t.Fatalf("expected page_info to expose next cursor, got %+v", payload.PageInfo)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/flows?limit=2&after="+payload.PageInfo.NextAfter, nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected second page 200, got %d", rec.Code)
	}

	var second struct {
		Flows []domain.Flow `json:"flows"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &second); err != nil {
		t.Fatalf("decode second page: %v", err)
	}
	if len(second.Flows) != 1 {
		t.Fatalf("expected 1 flow on second page, got %d", len(second.Flows))
	}
}

func TestBatchWorkspaces(t *testing.T) {
	server := newTestServer(t)
	handler := server.Handler()

	flowIDs := make([]string, 0, 2)
	for _, name := range []string{"batch-a", "batch-b"} {
		body, _ := json.Marshal(domain.CreateFlowInput{
			Name:      name,
			Target:    "https://app.example.com",
			Objective: "batch workspace coverage",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/flows", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create flow %s: expected 201, got %d", name, rec.Code)
		}

		var flow domain.Flow
		if err := json.Unmarshal(rec.Body.Bytes(), &flow); err != nil {
			t.Fatalf("decode flow %s: %v", name, err)
		}
		flowIDs = append(flowIDs, flow.ID)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces?flow_ids="+flowIDs[0]+","+flowIDs[1], nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var payload struct {
		Workspaces []domain.Workspace `json:"workspaces"`
		Errors     map[string]string  `json:"errors"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode workspaces: %v", err)
	}
	if len(payload.Workspaces) != 2 {
		t.Fatalf("expected 2 workspaces, got %d", len(payload.Workspaces))
	}
	if len(payload.Errors) != 0 {
		t.Fatalf("expected no workspace errors, got %+v", payload.Errors)
	}
}

func TestFlowStartApprovalAndReview(t *testing.T) {
	repo := store.NewMemoryStore()
	memoryService := memory.New(repo)
	browserService := browser.NewService()
	gateway := functions.NewGateway(browserService, memoryService, executor.NewLocalManagerWithRoot(t.TempDir()))
	server := NewServer(config.Config{
		ListenAddr:          ":0",
		ServiceName:         "test",
		ExecutorMode:        "local",
		DefaultTenant:       "default",
		RequireFlowApproval: true,
	}, repo, gateway, queue.NewNoopTransport(), nil, nil)
	handler := server.Handler()

	body, _ := json.Marshal(domain.CreateFlowInput{
		Name:      "Approval flow",
		Target:    "https://app.example.com",
		Objective: "Needs approval",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/flows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-NYX-Tenant", "alpha")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var flow domain.Flow
	if err := json.Unmarshal(rec.Body.Bytes(), &flow); err != nil {
		t.Fatalf("decode flow: %v", err)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/flows/"+flow.ID+"/start", nil)
	req.Header.Set("X-NYX-Tenant", "alpha")
	req.Header.Set("X-NYX-Operator", "alice")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rec.Code)
	}

	var startPayload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &startPayload); err != nil {
		t.Fatalf("decode approval payload: %v", err)
	}
	approvalID, _ := startPayload["approval_id"].(string)
	if approvalID == "" {
		t.Fatal("expected approval id")
	}

	reviewBody, _ := json.Marshal(domain.ApprovalReviewInput{Approved: true, Note: "safe to run"})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/approvals/"+approvalID+"/review", bytes.NewReader(reviewBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-NYX-Tenant", "alpha")
	req.Header.Set("X-NYX-Operator", "reviewer-1")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	updatedFlow, err := repo.GetFlow(context.Background(), flow.ID)
	if err != nil {
		t.Fatalf("get flow: %v", err)
	}
	if updatedFlow.Status != domain.StatusQueued {
		t.Fatalf("expected queued flow after approval, got %s", updatedFlow.Status)
	}
}

func TestBatchApprovalReview(t *testing.T) {
	repo := store.NewMemoryStore()
	memoryService := memory.New(repo)
	browserService := browser.NewService()
	gateway := functions.NewGateway(browserService, memoryService, executor.NewLocalManagerWithRoot(t.TempDir()))
	server := NewServer(config.Config{
		ListenAddr:          ":0",
		ServiceName:         "test",
		ExecutorMode:        "local",
		DefaultTenant:       "default",
		RequireFlowApproval: true,
	}, repo, gateway, queue.NewNoopTransport(), nil, nil)
	handler := server.Handler()

	var approvalIDs []string
	for _, name := range []string{"alpha", "bravo"} {
		body, _ := json.Marshal(domain.CreateFlowInput{
			Name:      name,
			Target:    "https://app.example.com",
			Objective: "Needs approval",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/flows", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-NYX-Tenant", "alpha")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		var flow domain.Flow
		if err := json.Unmarshal(rec.Body.Bytes(), &flow); err != nil {
			t.Fatalf("decode flow: %v", err)
		}

		req = httptest.NewRequest(http.MethodPost, "/api/v1/flows/"+flow.ID+"/start", nil)
		req.Header.Set("X-NYX-Tenant", "alpha")
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		var payload map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode start payload: %v", err)
		}
		approvalID, _ := payload["approval_id"].(string)
		approvalIDs = append(approvalIDs, approvalID)
	}

	body, _ := json.Marshal(batchApprovalReviewRequest{
		ApprovalIDs: approvalIDs,
		Approved:    true,
		Note:        "bulk approve",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/approvals/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-NYX-Tenant", "alpha")
	req.Header.Set("X-NYX-Operator", "reviewer-1")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var payload struct {
		Reviewed  int               `json:"reviewed"`
		Approvals []domain.Approval `json:"approvals"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode batch payload: %v", err)
	}
	if payload.Reviewed != 2 || len(payload.Approvals) != 2 {
		t.Fatalf("unexpected batch response: %+v", payload)
	}
	for _, approval := range payload.Approvals {
		if approval.Status != "approved" {
			t.Fatalf("expected approved status, got %q", approval.Status)
		}
	}
}

func TestCancelFlowEndpoint(t *testing.T) {
	server := newTestServer(t)
	handler := server.Handler()

	body, _ := json.Marshal(domain.CreateFlowInput{
		Name:      "Cancelable flow",
		Target:    "https://app.example.com",
		Objective: "cancel coverage",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/flows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var flow domain.Flow
	if err := json.Unmarshal(rec.Body.Bytes(), &flow); err != nil {
		t.Fatalf("decode flow: %v", err)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/flows/"+flow.ID+"/cancel", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/flows/"+flow.ID, nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var detail domain.FlowDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode flow detail: %v", err)
	}
	if detail.Flow.Status != domain.StatusCancelled {
		t.Fatalf("expected cancelled flow, got %q", detail.Flow.Status)
	}
}

func TestRateLimitRejectsBurstRequests(t *testing.T) {
	repo := store.NewMemoryStore()
	memoryService := memory.New(repo)
	browserService := browser.NewService()
	gateway := functions.NewGateway(browserService, memoryService, executor.NewLocalManagerWithRoot(t.TempDir()))
	server := NewServer(config.Config{
		ListenAddr:            ":0",
		ServiceName:           "test",
		ExecutorMode:          "local",
		DefaultTenant:         "default",
		RequireFlowApproval:   false,
		HTTPRateLimitRequests: 1,
		HTTPRateLimitWindow:   time.Hour,
	}, repo, gateway, queue.NewNoopTransport(), nil, nil)

	handler := server.Handler()
	first := httptest.NewRequest(http.MethodGet, "/api/v1/functions", nil)
	first.Header.Set("X-NYX-Operator", "alice")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, first)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected first request 200, got %d", rec.Code)
	}

	second := httptest.NewRequest(http.MethodGet, "/api/v1/functions", nil)
	second.Header.Set("X-NYX-Operator", "alice")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, second)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}
}

func TestWorkspaceEndpoints(t *testing.T) {
	repo := store.NewMemoryStore()
	memoryService := memory.New(repo)
	browserService := browser.NewService()
	gateway := functions.NewGateway(browserService, memoryService, executor.NewLocalManagerWithRoot(t.TempDir()))
	server := NewServer(config.Config{
		ListenAddr:           ":0",
		ServiceName:          "test",
		ExecutorMode:         "docker",
		BrowserMode:          "http",
		ExecutorNetworkMode:  "custom",
		ExecutorNetworkName:  "nyx-targets",
		ExecutorEnableNetRaw: true,
		DefaultTenant:        "default",
		RequireFlowApproval:  false,
	}, repo, gateway, queue.NewNoopTransport(), nil, nil)
	handler := server.Handler()

	body, _ := json.Marshal(domain.CreateFlowInput{
		Name:      "Workspace flow",
		Target:    "https://app.example.com",
		Objective: "Inspect workspace surface",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/flows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var flow domain.Flow
	if err := json.Unmarshal(rec.Body.Bytes(), &flow); err != nil {
		t.Fatalf("decode flow: %v", err)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/flows/"+flow.ID+"/workspace", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected workspace json 200, got %d", rec.Code)
	}
	var workspace domain.Workspace
	if err := json.Unmarshal(rec.Body.Bytes(), &workspace); err != nil {
		t.Fatalf("decode workspace: %v", err)
	}
	if !workspace.TerminalNetworkEnabled {
		t.Fatal("expected terminal network warning state to be exposed")
	}
	if workspace.ExecutorNetworkMode != "custom" {
		t.Fatalf("expected custom executor network mode, got %q", workspace.ExecutorNetworkMode)
	}
	if workspace.BrowserMode != "http" {
		t.Fatalf("expected browser mode http, got %q", workspace.BrowserMode)
	}
	if workspace.ExecutorNetworkName != "nyx-targets" {
		t.Fatalf("expected custom network name, got %q", workspace.ExecutorNetworkName)
	}
	if workspace.BrowserWarning == "" {
		t.Fatal("expected operator-visible browser warning")
	}
	if workspace.NetworkWarning == "" {
		t.Fatal("expected operator-visible network warning")
	}

	req = httptest.NewRequest(http.MethodGet, "/workspace/flows/"+flow.ID, nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected workspace html 200, got %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("Workspace flow")) {
		t.Fatal("expected workspace html to include flow name")
	}
}

func TestWorkspaceIncludesExecutionMetadata(t *testing.T) {
	repo := store.NewMemoryStore()
	memoryService := memory.New(repo)
	browserService := browser.NewService()
	gateway := functions.NewGateway(browserService, memoryService, executor.NewLocalManagerWithRoot(t.TempDir()))
	server := NewServer(config.Config{
		ListenAddr:          ":0",
		ServiceName:         "test",
		ExecutorMode:        "docker",
		DefaultTenant:       "default",
		RequireFlowApproval: false,
	}, repo, gateway, queue.NewNoopTransport(), nil, nil)
	handler := server.Handler()

	body, _ := json.Marshal(domain.CreateFlowInput{
		Name:      "Execution workspace flow",
		Target:    "https://app.example.com",
		Objective: "Inspect execution metadata",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/flows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var flow domain.Flow
	if err := json.Unmarshal(rec.Body.Bytes(), &flow); err != nil {
		t.Fatalf("decode flow: %v", err)
	}

	action, err := repo.CreateAction(context.Background(), flow.ID, "", "", "operator", "terminal_exec", "ephemeral", map[string]string{
		"command": "curl https://app.example.com/healthz",
	})
	if err != nil {
		t.Fatalf("create action: %v", err)
	}
	execution, err := repo.CreateExecution(context.Background(), flow.ID, action.ID, "terminal", "docker", map[string]string{
		"command":      "curl https://app.example.com/healthz",
		"image":        "nyx-executor-pentest:latest",
		"network_mode": "custom",
		"network_name": "nyx-targets",
	})
	if err != nil {
		t.Fatalf("create execution: %v", err)
	}
	if err := repo.CompleteExecution(context.Background(), execution.ID, domain.StatusCompleted, "terminal", "docker", map[string]string{
		"command":        "curl https://app.example.com/healthz",
		"image":          "nyx-executor-pentest:latest",
		"network_mode":   "custom",
		"network_name":   "nyx-targets",
		"evidence_paths": "/tmp/evidence.txt",
	}); err != nil {
		t.Fatalf("complete execution: %v", err)
	}
	if _, err := repo.AddArtifact(context.Background(), flow.ID, action.ID, "execution", "terminal_exec-trace", "command: curl https://app.example.com/healthz", map[string]string{
		"command":        "curl https://app.example.com/healthz",
		"image":          "nyx-executor-pentest:latest",
		"network_mode":   "custom",
		"evidence_paths": "/tmp/evidence.txt",
	}); err != nil {
		t.Fatalf("add artifact: %v", err)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/flows/"+flow.ID+"/workspace", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected workspace json 200, got %d", rec.Code)
	}
	var workspace domain.Workspace
	if err := json.Unmarshal(rec.Body.Bytes(), &workspace); err != nil {
		t.Fatalf("decode workspace: %v", err)
	}
	if len(workspace.Executions) != 1 {
		t.Fatalf("expected 1 execution, got %d", len(workspace.Executions))
	}
	if workspace.Executions[0].Metadata["image"] != "nyx-executor-pentest:latest" {
		t.Fatalf("expected execution image metadata, got %+v", workspace.Executions[0].Metadata)
	}
	if workspace.Executions[0].Metadata["command"] != "curl https://app.example.com/healthz" {
		t.Fatalf("expected execution command metadata, got %+v", workspace.Executions[0].Metadata)
	}

	req = httptest.NewRequest(http.MethodGet, "/workspace/flows/"+flow.ID, nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected workspace html 200, got %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("nyx-executor-pentest:latest")) {
		t.Fatal("expected workspace html to include execution image metadata")
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("curl https://app.example.com/healthz")) {
		t.Fatal("expected workspace html to include execution command metadata")
	}
}

func TestArchitectureIncludesExecutorNetworkStatus(t *testing.T) {
	server := NewServer(config.Config{
		ListenAddr:          ":0",
		ServiceName:         "test",
		ExecutorMode:        "auto",
		BrowserMode:         "http",
		ExecutorNetworkMode: "bridge",
		DefaultTenant:       "default",
	}, store.NewMemoryStore(), functions.NewGateway(browser.NewService(), memory.New(store.NewMemoryStore()), executor.NewLocalManagerWithRoot(t.TempDir())), queue.NewNoopTransport(), nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/architecture", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode architecture payload: %v", err)
	}
	if payload["executor_network_mode"] != "bridge" {
		t.Fatalf("expected bridge network mode, got %v", payload["executor_network_mode"])
	}
	if payload["terminal_network_enabled"] != true {
		t.Fatalf("expected terminal network enabled, got %v", payload["terminal_network_enabled"])
	}
	if payload["browser_mode"] != "http" {
		t.Fatalf("expected http browser mode, got %v", payload["browser_mode"])
	}
	if payload["browser_warning"] == "" {
		t.Fatal("expected browser warning in architecture payload")
	}
	if payload["network_warning"] == "" {
		t.Fatal("expected network warning in architecture payload")
	}
}

func TestAPIKeyAuth(t *testing.T) {
	repo := store.NewMemoryStore()
	memoryService := memory.New(repo)
	browserService := browser.NewService()
	gateway := functions.NewGateway(browserService, memoryService, executor.NewLocalManagerWithRoot(t.TempDir()))
	server := NewServer(config.Config{
		ListenAddr:          ":0",
		ServiceName:         "test",
		ExecutorMode:        "local",
		DefaultTenant:       "default",
		RequireFlowApproval: false,
		APIKey:              "secret-123",
	}, repo, gateway, queue.NewNoopTransport(), nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/functions", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without API key, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/functions", nil)
	req.Header.Set("X-NYX-API-Key", "secret-123")
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with API key, got %d", rec.Code)
	}
}

func TestFlowReportExport(t *testing.T) {
	server := newTestServer(t)
	handler := server.Handler()

	body, _ := json.Marshal(domain.CreateFlowInput{
		Name:      "Report flow",
		Target:    "https://app.example.com",
		Objective: "Generate exports",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/flows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var flow domain.Flow
	if err := json.Unmarshal(rec.Body.Bytes(), &flow); err != nil {
		t.Fatalf("decode flow: %v", err)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/flows/"+flow.ID+"/report?format=markdown", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected markdown report 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct == "" || ct[:13] != "text/markdown" {
		t.Fatalf("expected markdown content type, got %q", ct)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/flows/"+flow.ID+"/report?format=pdf", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected pdf report 200, got %d", rec.Code)
	}
	if got := rec.Body.Bytes(); len(got) < 5 || string(got[:5]) != "%PDF-" {
		t.Fatal("expected pdf export to start with %PDF-")
	}
}

func TestSupabaseAuthRequiresBearer(t *testing.T) {
	repo := store.NewMemoryStore()
	memoryService := memory.New(repo)
	browserService := browser.NewService()
	gateway := functions.NewGateway(browserService, memoryService, executor.NewLocalManagerWithRoot(t.TempDir()))
	server := NewServer(config.Config{
		ListenAddr:          ":0",
		ServiceName:         "test",
		ExecutorMode:        "local",
		DefaultTenant:       "default",
		SupabaseURL:         "https://nyx.supabase.co",
		SupabaseJWTAudience: "authenticated",
	}, repo, gateway, queue.NewNoopTransport(), nil, nil, WithAuthenticator(stubAuthenticator{
		identity: auth.Identity{Subject: "user-1", Email: "operator@nyx.ai"},
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/functions", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without bearer token, got %d", rec.Code)
	}
}

func TestSupabaseAuthAcceptsBearerAndClaims(t *testing.T) {
	repo := store.NewMemoryStore()
	memoryService := memory.New(repo)
	browserService := browser.NewService()
	gateway := functions.NewGateway(browserService, memoryService, executor.NewLocalManagerWithRoot(t.TempDir()))
	server := NewServer(config.Config{
		ListenAddr:          ":0",
		ServiceName:         "test",
		ExecutorMode:        "local",
		DefaultTenant:       "default",
		RequireFlowApproval: true,
		SupabaseURL:         "https://nyx.supabase.co",
		SupabaseJWTAudience: "authenticated",
	}, repo, gateway, queue.NewNoopTransport(), nil, nil, WithAuthenticator(stubAuthenticator{
		identity: auth.Identity{
			Subject:  "user-1",
			Email:    "operator@nyx.ai",
			TenantID: "alpha",
		},
	}))
	handler := server.Handler()

	body, _ := json.Marshal(domain.CreateFlowInput{
		Name:      "Auth flow",
		Target:    "https://app.example.com",
		Objective: "Verify bearer auth",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/flows", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer good-token")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-NYX-Tenant", "spoofed")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	var flow domain.Flow
	if err := json.Unmarshal(rec.Body.Bytes(), &flow); err != nil {
		t.Fatalf("decode flow: %v", err)
	}
	if flow.TenantID != "alpha" {
		t.Fatalf("expected tenant from token claim, got %q", flow.TenantID)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/flows/"+flow.ID+"/start", nil)
	req.Header.Set("Authorization", "Bearer good-token")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rec.Code)
	}

	approvals, err := repo.ListApprovalsByTenant(context.Background(), "alpha")
	if err != nil {
		t.Fatalf("list approvals: %v", err)
	}
	if len(approvals) != 1 {
		t.Fatalf("expected one approval, got %d", len(approvals))
	}
	if approvals[0].RequestedBy != "operator@nyx.ai" {
		t.Fatalf("expected operator email from token, got %q", approvals[0].RequestedBy)
	}
}

func TestSupabaseAuthAllowsAPIKeyFallback(t *testing.T) {
	repo := store.NewMemoryStore()
	memoryService := memory.New(repo)
	browserService := browser.NewService()
	gateway := functions.NewGateway(browserService, memoryService, executor.NewLocalManagerWithRoot(t.TempDir()))
	server := NewServer(config.Config{
		ListenAddr:          ":0",
		ServiceName:         "test",
		ExecutorMode:        "local",
		DefaultTenant:       "default",
		SupabaseURL:         "https://nyx.supabase.co",
		SupabaseJWTAudience: "authenticated",
		APIKey:              "secret-123",
	}, repo, gateway, queue.NewNoopTransport(), nil, nil, WithAuthenticator(stubAuthenticator{
		err: errors.New("should not verify when api key is used"),
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/functions", nil)
	req.Header.Set("X-NYX-API-Key", "secret-123")
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with api key fallback, got %d", rec.Code)
	}
}
