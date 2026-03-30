//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"nyx/internal/config"
	"nyx/internal/domain"
	"nyx/internal/executor"
	"nyx/internal/functions"
	"nyx/internal/httpapi"
	"nyx/internal/ids"
	"nyx/internal/orchestrator"
	"nyx/internal/queue"
	"nyx/internal/services/browser"
	"nyx/internal/services/memory"
	"nyx/internal/store"
)

func TestFlowLifecycleWithPostgresAndNATS(t *testing.T) {
	databaseURL := os.Getenv("NYX_IT_DATABASE_URL")
	natsURL := os.Getenv("NYX_IT_NATS_URL")
	if databaseURL == "" || natsURL == "" {
		t.Skip("NYX_IT_DATABASE_URL and NYX_IT_NATS_URL are required for integration tests")
	}

	suffix := ids.New("it")
	baseCfg := config.Config{
		ServiceName:           "nyx-it-" + suffix,
		DatabaseURL:           databaseURL,
		NATSURL:               natsURL,
		FlowStream:            "NYX_IT_FLOW_" + suffix,
		FlowSubject:           "nyx.it.flows." + suffix,
		FlowConsumer:          "nyx-it-orchestrator-" + suffix,
		ActionStream:          "NYX_IT_ACTION_" + suffix,
		ActionSubject:         "nyx.it.actions." + suffix,
		ActionConsumer:        "nyx-it-executor-" + suffix,
		ActionResultStream:    "NYX_IT_ACTION_RESULT_" + suffix,
		ActionResultSubject:   "nyx.it.actions.result." + suffix,
		EventStream:           "NYX_IT_EVENTS_" + suffix,
		EventSubject:          "nyx.it.events." + suffix,
		DLQStream:             "NYX_IT_DLQ_" + suffix,
		DLQSubject:            "nyx.it.dlq." + suffix,
		QueueMaxDeliver:       2,
		ActionResultTimeout:   10 * time.Second,
		ExecutorMode:          "local",
		ExecutorWorkspaceRoot: t.TempDir(),
		BrowserMode:           "http",
		BrowserArtifactsRoot:  t.TempDir(),
		DefaultTenant:         "integration",
		RequireFlowApproval:   false,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	repo, err := store.OpenRepository(ctx, baseCfg.DatabaseURL)
	if err != nil {
		t.Fatalf("open repository: %v", err)
	}
	defer repo.Close()

	apiTransport, err := queue.OpenTransport(ctx, baseCfg)
	if err != nil {
		t.Fatalf("open api transport: %v", err)
	}
	defer apiTransport.Close()

	execTransport, err := queue.OpenTransport(ctx, withService(baseCfg, "nyx-it-executor-"+suffix))
	if err != nil {
		t.Fatalf("open executor transport: %v", err)
	}
	defer execTransport.Close()

	orchestratorTransport, err := queue.OpenTransport(ctx, withService(baseCfg, "nyx-it-orchestrator-"+suffix))
	if err != nil {
		t.Fatalf("open orchestrator transport: %v", err)
	}
	defer orchestratorTransport.Close()

	memoryService := memory.New(repo)
	browserService := browser.NewServiceWithRuntime(browser.RuntimeConfig{
		Mode:          "http",
		ArtifactsRoot: t.TempDir(),
	})
	gateway := functions.NewGateway(browserService, memoryService, executor.NewLocalManagerWithRoot(t.TempDir()))

	go func() {
		_ = execTransport.ConsumeActionRequests(ctx, func(execCtx context.Context, req queue.ActionRequestMessage) (queue.ActionResultMessage, error) {
			result := gateway.Call(execCtx, req.FlowID, req.ActionID, req.FunctionName, req.Input)
			msg := queue.ActionResultMessage{
				FlowID:       req.FlowID,
				ActionID:     req.ActionID,
				FunctionName: req.FunctionName,
				Profile:      result.Profile,
				Runtime:      result.Runtime,
				Output:       result.Output,
			}
			if result.Err != nil {
				msg.Error = result.Err.Error()
			}
			return msg, nil
		})
	}()

	engine := orchestrator.New(repo, gateway, 25*time.Millisecond, 10*time.Second, orchestratorTransport, nil, nil, false)
	go func() {
		err := engine.RunForever(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Errorf("orchestrator run: %v", err)
		}
	}()

	server := httpapi.NewServer(baseCfg, repo, gateway, apiTransport, nil, nil)
	handler := server.Handler()

	body, _ := json.Marshal(domain.CreateFlowInput{
		Name:      "Integration flow",
		Target:    "integration-target",
		Objective: "Exercise Postgres + NATS + orchestrator",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/flows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-NYX-Tenant", "integration")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected create 201, got %d", rec.Code)
	}

	var flow domain.Flow
	if err := json.Unmarshal(rec.Body.Bytes(), &flow); err != nil {
		t.Fatalf("decode flow: %v", err)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/flows/"+flow.ID+"/start", nil)
	req.Header.Set("X-NYX-Tenant", "integration")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected start 202, got %d", rec.Code)
	}

	timeout := time.After(20 * time.Second)
	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-timeout:
			t.Fatal("timed out waiting for flow completion")
		case <-tick.C:
			current, err := repo.GetFlow(flow.ID)
			if err != nil {
				t.Fatalf("get flow: %v", err)
			}
			if current.Status == domain.StatusCompleted {
				detail, err := repo.FlowDetail(flow.ID)
				if err != nil {
					t.Fatalf("flow detail: %v", err)
				}
				if len(detail.Actions) == 0 || len(detail.Artifacts) == 0 {
					t.Fatalf("expected executed actions and artifacts, got %+v", detail)
				}
				return
			}
		}
	}
}

func withService(cfg config.Config, name string) config.Config {
	cfg.ServiceName = name
	return cfg
}
