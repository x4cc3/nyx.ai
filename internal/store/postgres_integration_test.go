package store

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"

	"nyx/internal/domain"
)

func integrationDatabaseURL(t *testing.T) string {
	t.Helper()
	if value := os.Getenv("NYX_TEST_DATABASE_URL"); value != "" {
		return value
	}
	t.Skip("NYX_TEST_DATABASE_URL is not set")
	return ""
}

func resetIntegrationDatabase(t *testing.T, databaseURL string) {
	t.Helper()

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open integration database: %v", err)
	}
	defer func() { _ = db.Close() }()

	if _, err := db.Exec(`
TRUNCATE TABLE
  approvals,
  events,
  executions,
  findings,
  memories,
  artifacts,
  actions,
  subtasks,
  tasks,
  agents,
  flows
RESTART IDENTITY CASCADE
`); err != nil {
		t.Fatalf("reset integration database: %v", err)
	}
}

func TestPostgresStoreCRUDAndTenantIsolation(t *testing.T) {
	databaseURL := integrationDatabaseURL(t)
	store, err := NewPostgresStore(databaseURL)
	if err != nil {
		t.Fatalf("NewPostgresStore: %v", err)
	}
	defer func() { _ = store.Close() }()

	if err := store.Init(context.Background()); err != nil {
		t.Fatalf("Init: %v", err)
	}
	resetIntegrationDatabase(t, databaseURL)

	flow, err := store.CreateFlowForTenant(context.Background(), "tenant-a", domain.CreateFlowInput{
		Name:      "Authenticated review",
		Target:    "https://app.example.com",
		Objective: "exercise postgres CRUD",
	})
	if err != nil {
		t.Fatalf("CreateFlowForTenant: %v", err)
	}
	if flow.TenantID != "tenant-a" {
		t.Fatalf("unexpected tenant id: %q", flow.TenantID)
	}

	task, err := store.CreateTask(context.Background(), flow.ID, "Recon", "Enumerate application", "planner")
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	subtask, err := store.CreateSubtask(context.Background(), flow.ID, task.ID, "Navigate", "Navigate to target", "browser")
	if err != nil {
		t.Fatalf("CreateSubtask: %v", err)
	}

	action, err := store.CreateAction(context.Background(), flow.ID, task.ID, subtask.ID, "browser", "navigate", "docker", map[string]string{"url": "https://app.example.com"})
	if err != nil {
		t.Fatalf("CreateAction: %v", err)
	}

	if _, err := store.CreateApproval(context.Background(), flow.ID, "tenant-a", "flow.start", "alice", "operator review", map[string]string{"flow_id": flow.ID}); err != nil {
		t.Fatalf("CreateApproval: %v", err)
	}
	if _, err := store.AddMemory(context.Background(), flow.ID, action.ID, "observation", "Captured login page", map[string]string{"function_name": "browser"}); err != nil {
		t.Fatalf("AddMemory: %v", err)
	}

	detail, err := store.FlowDetailForTenant(context.Background(), "tenant-a", flow.ID)
	if err != nil {
		t.Fatalf("FlowDetailForTenant: %v", err)
	}
	if len(detail.Tasks) != 1 || len(detail.Memories) != 1 {
		t.Fatalf("unexpected detail payload: %+v", detail)
	}

	if _, err := store.FlowDetailForTenant(context.Background(), "tenant-b", flow.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected tenant isolation error, got %v", err)
	}

	memories, err := store.SearchMemories(context.Background(), flow.ID, "login")
	if err != nil {
		t.Fatalf("SearchMemories: %v", err)
	}
	if len(memories) == 0 {
		t.Fatal("expected search results")
	}
}
