package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"nyx/internal/dbgen"
	"nyx/internal/domain"
	"nyx/internal/ids"
	"nyx/internal/memvec"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgresStore struct {
	db *sql.DB
	q  *dbgen.Queries
}

func NewPostgresStore(databaseURL string) (*PostgresStore, error) {
	return NewPostgresStoreWithConfig(databaseURL, 10, 5, 30*time.Minute, 5*time.Minute)
}

func NewPostgresStoreWithConfig(databaseURL string, maxOpen, maxIdle int, connMaxLifetime, connMaxIdleTime time.Duration) (*PostgresStore, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(connMaxLifetime)
	db.SetConnMaxIdleTime(connMaxIdleTime)
	return &PostgresStore{db: db, q: dbgen.New(db)}, nil
}

func (s *PostgresStore) Init(ctx context.Context) error {
	return applyMigrations(ctx, s.db)
}

func (s *PostgresStore) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }
func (s *PostgresStore) Close() error                   { return s.db.Close() }

func (s *PostgresStore) CreateFlow(ctx context.Context, input domain.CreateFlowInput) (domain.Flow, error) {
	return s.CreateFlowForTenant(ctx, "default", input)
}

func (s *PostgresStore) CreateFlowForTenant(ctx context.Context, tenantID string, input domain.CreateFlowInput) (domain.Flow, error) {
	flow, err := scanFlow(s.db.QueryRowContext(ctx, `
		INSERT INTO flows (id, tenant_id, name, target, objective, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, tenant_id, name, target, objective, status, created_at, updated_at
	`, ids.New("flow"), normalizeTenant(tenantID), input.Name, input.Target, input.Objective, string(domain.StatusPending)))
	if err != nil {
		return domain.Flow{}, fmt.Errorf("create flow for tenant: %w", err)
	}
	_, _ = s.RecordEvent(ctx, flow.ID, domain.EventFlowCreated, "Flow created", map[string]any{"status": flow.Status})
	return flow, nil
}

func (s *PostgresStore) ListFlows(ctx context.Context) ([]domain.Flow, error) {
	return s.ListFlowsByTenant(ctx, "")
}

func (s *PostgresStore) ListFlowsByTenant(ctx context.Context, tenantID string) ([]domain.Flow, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if normalizeTenant(tenantID) == "" || strings.TrimSpace(tenantID) == "" {
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, tenant_id, name, target, objective, status, created_at, updated_at
			FROM flows
			ORDER BY created_at DESC
		`)
	} else {
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, tenant_id, name, target, objective, status, created_at, updated_at
			FROM flows
			WHERE tenant_id = $1
			ORDER BY created_at DESC
		`, normalizeTenant(tenantID))
	}
	if err != nil {
		return nil, fmt.Errorf("list flows by tenant: %w", err)
	}
	defer rows.Close()
	return scanFlows(rows)
}

func (s *PostgresStore) ListFlowsPageByTenant(ctx context.Context, tenantID, afterID string, limit int) ([]domain.Flow, string, bool, error) {
	if limit < 1 {
		limit = 1
	}
	useTenant := strings.TrimSpace(tenantID) != ""
	args := make([]any, 0, 4)
	conditions := make([]string, 0, 2)

	if useTenant {
		conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", len(args)+1))
		args = append(args, normalizeTenant(tenantID))
	}

	afterID = strings.TrimSpace(afterID)
	if afterID != "" {
		var (
			cursor domain.Flow
			err    error
		)
		if useTenant {
			cursor, err = s.GetFlowForTenant(ctx, tenantID, afterID)
		} else {
			cursor, err = s.GetFlow(ctx, afterID)
		}
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return nil, "", false, ErrInvalidPageCursor
			}
			return nil, "", false, fmt.Errorf("list flows page by tenant: %w", err)
		}
		conditions = append(conditions, fmt.Sprintf("(created_at < $%d OR (created_at = $%d AND id < $%d))", len(args)+1, len(args)+1, len(args)+2))
		args = append(args, cursor.CreatedAt, cursor.ID)
	}

	query := `
		SELECT id, tenant_id, name, target, objective, status, created_at, updated_at
		FROM flows
	`
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC, id DESC LIMIT $%d", len(args)+1)
	args = append(args, limit+1)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", false, fmt.Errorf("list flows page by tenant: %w", err)
	}
	defer rows.Close()

	flows, err := scanFlows(rows)
	if err != nil {
		return nil, "", false, fmt.Errorf("list flows page by tenant: %w", err)
	}
	hasMore := len(flows) > limit
	if hasMore {
		flows = flows[:limit]
	}
	nextAfter := ""
	if hasMore && len(flows) > 0 {
		nextAfter = flows[len(flows)-1].ID
	}
	return flows, nextAfter, hasMore, nil
}

func (s *PostgresStore) GetFlow(ctx context.Context, id string) (domain.Flow, error) {
	flow, err := scanFlow(s.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, name, target, objective, status, created_at, updated_at
		FROM flows
		WHERE id = $1
	`, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Flow{}, ErrNotFound
		}
		return domain.Flow{}, fmt.Errorf("get flow: %w", err)
	}
	return flow, nil
}

func (s *PostgresStore) GetFlowForTenant(ctx context.Context, tenantID, id string) (domain.Flow, error) {
	flow, err := scanFlow(s.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, name, target, objective, status, created_at, updated_at
		FROM flows
		WHERE id = $1 AND tenant_id = $2
	`, id, normalizeTenant(tenantID)))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Flow{}, ErrNotFound
		}
		return domain.Flow{}, fmt.Errorf("get flow for tenant: %w", err)
	}
	return flow, nil
}

func (s *PostgresStore) QueueFlow(ctx context.Context, id string) (domain.Flow, error) {
	flow, err := scanFlow(s.db.QueryRowContext(ctx, `
		UPDATE flows
		SET status = 'queued', updated_at = NOW()
		WHERE id = $1
		RETURNING id, tenant_id, name, target, objective, status, created_at, updated_at
	`, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Flow{}, ErrNotFound
		}
		return domain.Flow{}, fmt.Errorf("queue flow: %w", err)
	}
	return flow, nil
}

func (s *PostgresStore) QueueFlowForTenant(ctx context.Context, tenantID, id string) (domain.Flow, error) {
	if _, err := s.GetFlowForTenant(ctx, tenantID, id); err != nil {
		return domain.Flow{}, fmt.Errorf("queue flow for tenant: %w", err)
	}
	return s.QueueFlow(ctx, id)
}

func (s *PostgresStore) UpdateFlowStatus(ctx context.Context, id string, status domain.Status) (domain.Flow, error) {
	flow, err := scanFlow(s.db.QueryRowContext(ctx, `
		UPDATE flows
		SET status = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING id, tenant_id, name, target, objective, status, created_at, updated_at
	`, id, string(status)))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Flow{}, ErrNotFound
		}
		return domain.Flow{}, fmt.Errorf("update flow status: %w", err)
	}
	return flow, nil
}

func (s *PostgresStore) ClaimNextQueuedFlow(ctx context.Context) (domain.Flow, bool, error) {
	flow, err := scanFlow(s.db.QueryRowContext(ctx, `
		WITH next_flow AS (
			SELECT id
			FROM flows
			WHERE status = 'queued'
			ORDER BY created_at ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE flows AS f
		SET status = 'running', updated_at = NOW()
		FROM next_flow
		WHERE f.id = next_flow.id
		RETURNING f.id, f.tenant_id, f.name, f.target, f.objective, f.status, f.created_at, f.updated_at
	`))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Flow{}, false, nil
		}
		return domain.Flow{}, false, fmt.Errorf("claim next queued flow: %w", err)
	}
	return flow, true, nil
}

func (s *PostgresStore) CreateAgent(ctx context.Context, flowID, role, model string) (domain.Agent, error) {
	row, err := s.q.CreateAgent(ctx, dbgen.CreateAgentParams{
		ID:     ids.New("agent"),
		FlowID: flowID,
		Role:   role,
		Model:  model,
		Status: string(domain.StatusPending),
	})
	if err != nil {
		return domain.Agent{}, fmt.Errorf("create agent: %w", err)
	}
	return toDomainAgent(row), nil
}

func (s *PostgresStore) UpdateAgentStatus(ctx context.Context, id string, status domain.Status) error {
	if err := s.q.UpdateAgentStatus(ctx, dbgen.UpdateAgentStatusParams{
		ID:     id,
		Status: string(status),
	}); err != nil {
		return fmt.Errorf("update agent status: %w", err)
	}
	return nil
}

func (s *PostgresStore) CompleteAgent(ctx context.Context, id string, status domain.Status) error {
	return s.UpdateAgentStatus(ctx, id, status)
}

func (s *PostgresStore) CreateTask(ctx context.Context, flowID, name, description, role string) (domain.Task, error) {
	row, err := s.q.CreateTask(ctx, dbgen.CreateTaskParams{
		ID:          ids.New("task"),
		FlowID:      flowID,
		Name:        name,
		Description: description,
		AgentRole:   role,
		Status:      string(domain.StatusPending),
	})
	if err != nil {
		return domain.Task{}, fmt.Errorf("create task: %w", err)
	}
	return toDomainTask(row), nil
}

func (s *PostgresStore) UpdateTaskStatus(ctx context.Context, id string, status domain.Status) error {
	if err := s.q.UpdateTaskStatus(ctx, dbgen.UpdateTaskStatusParams{
		ID:     id,
		Status: string(status),
	}); err != nil {
		return fmt.Errorf("update task status: %w", err)
	}
	return nil
}

func (s *PostgresStore) CompleteTask(ctx context.Context, id string, status domain.Status) error {
	return s.UpdateTaskStatus(ctx, id, status)
}

func (s *PostgresStore) CreateSubtask(ctx context.Context, flowID, taskID, name, description, role string) (domain.Subtask, error) {
	row, err := s.q.CreateSubtask(ctx, dbgen.CreateSubtaskParams{
		ID:          ids.New("subtask"),
		FlowID:      flowID,
		TaskID:      taskID,
		Name:        name,
		Description: description,
		AgentRole:   role,
		Status:      string(domain.StatusPending),
	})
	if err != nil {
		return domain.Subtask{}, fmt.Errorf("create subtask: %w", err)
	}
	return toDomainSubtask(row), nil
}

func (s *PostgresStore) UpdateSubtaskStatus(ctx context.Context, id string, status domain.Status) error {
	if err := s.q.UpdateSubtaskStatus(ctx, dbgen.UpdateSubtaskStatusParams{
		ID:     id,
		Status: string(status),
	}); err != nil {
		return fmt.Errorf("update subtask status: %w", err)
	}
	return nil
}

func (s *PostgresStore) CompleteSubtask(ctx context.Context, id string, status domain.Status) error {
	return s.UpdateSubtaskStatus(ctx, id, status)
}

func (s *PostgresStore) CreateAction(ctx context.Context, flowID, taskID, subtaskID, role, functionName, executionMode string, input map[string]string) (domain.Action, error) {
	row, err := s.q.CreateAction(ctx, dbgen.CreateActionParams{
		ID:            ids.New("action"),
		FlowID:        flowID,
		TaskID:        taskID,
		SubtaskID:     subtaskID,
		AgentRole:     role,
		FunctionName:  functionName,
		Input:         toJSON(input),
		Status:        string(domain.StatusRunning),
		ExecutionMode: executionMode,
	})
	if err != nil {
		return domain.Action{}, fmt.Errorf("create action: %w", err)
	}
	return toDomainAction(row), nil
}

func (s *PostgresStore) CompleteAction(ctx context.Context, id string, status domain.Status, output map[string]string) (domain.Action, error) {
	row, err := s.q.CompleteAction(ctx, dbgen.CompleteActionParams{
		ID:     id,
		Status: string(status),
		Output: toJSON(output),
	})
	if err != nil {
		return domain.Action{}, fmt.Errorf("complete action: %w", err)
	}
	return toDomainAction(row), nil
}

func (s *PostgresStore) CreateExecution(ctx context.Context, flowID, actionID, profile, runtime string, metadata map[string]string) (domain.Execution, error) {
	row, err := s.q.CreateExecution(ctx, dbgen.CreateExecutionParams{
		ID:       ids.New("exec"),
		FlowID:   flowID,
		ActionID: actionID,
		Profile:  profile,
		Runtime:  runtime,
		Metadata: toJSON(metadata),
		Status:   string(domain.StatusRunning),
	})
	if err != nil {
		return domain.Execution{}, fmt.Errorf("create execution: %w", err)
	}
	return toDomainExecution(createExecutionRow(row)), nil
}

func (s *PostgresStore) CompleteExecution(ctx context.Context, id string, status domain.Status, profile, runtime string, metadata map[string]string) error {
	if err := s.q.CompleteExecution(ctx, dbgen.CompleteExecutionParams{
		ID:       id,
		Profile:  profile,
		Runtime:  runtime,
		Metadata: toJSON(metadata),
		Status:   string(status),
	}); err != nil {
		return fmt.Errorf("complete execution: %w", err)
	}
	return nil
}

func (s *PostgresStore) AddArtifact(ctx context.Context, flowID, actionID, kind, name, content string, metadata map[string]string) (domain.Artifact, error) {
	row, err := s.q.AddArtifact(ctx, dbgen.AddArtifactParams{
		ID:       ids.New("artifact"),
		FlowID:   flowID,
		ActionID: actionID,
		Kind:     kind,
		Name:     name,
		Content:  content,
		Metadata: toJSON(metadata),
	})
	if err != nil {
		return domain.Artifact{}, fmt.Errorf("add artifact: %w", err)
	}
	return toDomainArtifact(row), nil
}

func (s *PostgresStore) AddMemory(ctx context.Context, flowID, actionID, kind, content string, metadata map[string]string) (domain.Memory, error) {
	content, metadata, embedding := memvec.Prepare(kind, content, metadata)
	row := struct {
		ID        string
		FlowID    string
		ActionID  string
		Kind      string
		Content   string
		Metadata  []byte
		CreatedAt time.Time
	}{}
	err := s.db.QueryRowContext(ctx, fmt.Sprintf(`
		INSERT INTO memories (id, flow_id, action_id, kind, content, metadata, embedding, embedding_model, retention_policy)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, CAST($7 AS vector(%d)), $8, $9)
		RETURNING id, flow_id, action_id, kind, content, metadata, created_at
	`, memvec.Dimensions()), ids.New("memory"), flowID, actionID, kind, content, toJSON(metadata), memvec.VectorLiteral(embedding), metadata["embedding_model"], metadata["retention_policy"]).Scan(
		&row.ID,
		&row.FlowID,
		&row.ActionID,
		&row.Kind,
		&row.Content,
		&row.Metadata,
		&row.CreatedAt,
	)
	if err != nil {
		return domain.Memory{}, fmt.Errorf("add memory: %w", err)
	}
	return domain.Memory{
		ID:        row.ID,
		FlowID:    row.FlowID,
		ActionID:  row.ActionID,
		Kind:      row.Kind,
		Content:   row.Content,
		Metadata:  toStringMap(row.Metadata),
		CreatedAt: row.CreatedAt,
	}, nil
}

func (s *PostgresStore) AddFinding(ctx context.Context, flowID, title, severity, description string) (domain.Finding, error) {
	row, err := s.q.AddFinding(ctx, dbgen.AddFindingParams{
		ID:          ids.New("finding"),
		FlowID:      flowID,
		Title:       title,
		Severity:    severity,
		Description: description,
	})
	if err != nil {
		return domain.Finding{}, fmt.Errorf("add finding: %w", err)
	}
	return toDomainFinding(row), nil
}

func (s *PostgresStore) FlowDetail(ctx context.Context, id string) (domain.FlowDetail, error) {
	flow, err := s.GetFlow(ctx, id)
	if err != nil {
		return domain.FlowDetail{}, fmt.Errorf("flow detail: %w", err)
	}
	detail := domain.FlowDetail{Flow: flow}
	taskRows, err := s.q.ListTasksByFlow(ctx, id)
	if err != nil {
		return domain.FlowDetail{}, fmt.Errorf("flow detail: %w", err)
	}
	detail.Tasks = toDomainTasks(taskRows)
	subtaskRows, err := s.q.ListSubtasksByFlow(ctx, id)
	if err != nil {
		return domain.FlowDetail{}, fmt.Errorf("flow detail: %w", err)
	}
	detail.Subtasks = toDomainSubtasks(subtaskRows)
	actionRows, err := s.q.ListActionsByFlow(ctx, id)
	if err != nil {
		return domain.FlowDetail{}, fmt.Errorf("flow detail: %w", err)
	}
	detail.Actions = toDomainActions(actionRows)
	artifactRows, err := s.q.ListArtifactsByFlow(ctx, id)
	if err != nil {
		return domain.FlowDetail{}, fmt.Errorf("flow detail: %w", err)
	}
	detail.Artifacts = toDomainArtifacts(artifactRows)
	memoryRows, err := s.q.ListMemoriesByFlow(ctx, id)
	if err != nil {
		return domain.FlowDetail{}, fmt.Errorf("flow detail: %w", err)
	}
	detail.Memories = toDomainMemories(memoryRows)
	findingRows, err := s.q.ListFindingsByFlow(ctx, id)
	if err != nil {
		return domain.FlowDetail{}, fmt.Errorf("flow detail: %w", err)
	}
	detail.Findings = toDomainFindings(findingRows)
	agentRows, err := s.q.ListAgentsByFlow(ctx, id)
	if err != nil {
		return domain.FlowDetail{}, fmt.Errorf("flow detail: %w", err)
	}
	detail.Agents = toDomainAgents(agentRows)
	executionRows, err := s.q.ListExecutionsByFlow(ctx, id)
	if err != nil {
		return domain.FlowDetail{}, fmt.Errorf("flow detail: %w", err)
	}
	detail.Executions = toDomainExecutions(executionRows)
	return detail, nil
}

func (s *PostgresStore) FlowDetailForTenant(ctx context.Context, tenantID, id string) (domain.FlowDetail, error) {
	flow, err := s.GetFlowForTenant(ctx, tenantID, id)
	if err != nil {
		return domain.FlowDetail{}, fmt.Errorf("flow detail for tenant: %w", err)
	}
	detail, err := s.FlowDetail(ctx, id)
	if err != nil {
		return domain.FlowDetail{}, fmt.Errorf("flow detail for tenant: %w", err)
	}
	detail.Flow = flow
	return detail, nil
}

func (s *PostgresStore) SearchMemories(ctx context.Context, flowID, query string) ([]domain.Memory, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		rows, err := s.q.SearchMemories(ctx, dbgen.SearchMemoriesParams{
			FlowID: flowID,
			Query:  "%%",
		})
		if err != nil {
			return nil, fmt.Errorf("search memories: %w", err)
		}
		return toSearchDomainMemories(rows), nil
	}

	vector := memvec.VectorLiteral(memvec.Embed(query))
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, flow_id, action_id, kind, content, metadata, created_at
		FROM memories
		WHERE flow_id = $1
		ORDER BY
			CASE WHEN content ILIKE '%%' || $3 || '%%' THEN 0 ELSE 1 END ASC,
			CASE WHEN embedding IS NULL THEN 1.0 ELSE embedding <=> CAST($2 AS vector(%d)) END ASC,
			created_at DESC
		LIMIT 8
	`, memvec.Dimensions()), flowID, vector, query)
	if err != nil {
		return nil, fmt.Errorf("search memories: %w", err)
	}
	defer rows.Close()

	out := make([]domain.Memory, 0)
	for rows.Next() {
		var memory domain.Memory
		var rawMetadata []byte
		if err := rows.Scan(&memory.ID, &memory.FlowID, &memory.ActionID, &memory.Kind, &memory.Content, &rawMetadata, &memory.CreatedAt); err != nil {
			return nil, fmt.Errorf("search memories: %w", err)
		}
		memory.Metadata = toStringMap(rawMetadata)
		out = append(out, memory)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("search memories: %w", err)
	}
	return out, nil
}

func (s *PostgresStore) CreateApproval(ctx context.Context, flowID, tenantID, kind, requestedBy, reason string, payload map[string]string) (domain.Approval, error) {
	approval, err := scanApproval(s.db.QueryRowContext(ctx, `
		INSERT INTO approvals (id, flow_id, tenant_id, kind, status, requested_by, reason, payload)
		VALUES ($1, $2, $3, $4, 'pending', $5, $6, $7::jsonb)
		RETURNING id, flow_id, tenant_id, kind, status, requested_by, reviewed_by, review_note, reason, payload, created_at, reviewed_at
	`, ids.New("approval"), flowID, normalizeTenant(tenantID), kind, blankToDefault(strings.TrimSpace(requestedBy), "anonymous"), reason, toJSON(payload)))
	if err != nil {
		return domain.Approval{}, fmt.Errorf("create approval: %w", err)
	}
	return approval, nil
}

func (s *PostgresStore) GetApproval(ctx context.Context, id string) (domain.Approval, error) {
	approval, err := scanApproval(s.db.QueryRowContext(ctx, `
		SELECT id, flow_id, tenant_id, kind, status, requested_by, reviewed_by, review_note, reason, payload, created_at, reviewed_at
		FROM approvals
		WHERE id = $1
	`, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Approval{}, ErrNotFound
		}
		return domain.Approval{}, fmt.Errorf("get approval: %w", err)
	}
	return approval, nil
}

func (s *PostgresStore) ListApprovalsByTenant(ctx context.Context, tenantID string) ([]domain.Approval, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, flow_id, tenant_id, kind, status, requested_by, reviewed_by, review_note, reason, payload, created_at, reviewed_at
		FROM approvals
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`, normalizeTenant(tenantID))
	if err != nil {
		return nil, fmt.Errorf("list approvals by tenant: %w", err)
	}
	defer rows.Close()
	return scanApprovals(rows)
}

func (s *PostgresStore) ListApprovalsPageByTenant(ctx context.Context, tenantID, afterID string, limit int) ([]domain.Approval, string, bool, error) {
	if limit < 1 {
		limit = 1
	}
	tenantID = normalizeTenant(tenantID)
	args := []any{tenantID}
	conditions := []string{"tenant_id = $1"}

	afterID = strings.TrimSpace(afterID)
	if afterID != "" {
		cursor, err := s.GetApproval(ctx, afterID)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return nil, "", false, ErrInvalidPageCursor
			}
			return nil, "", false, fmt.Errorf("list approvals page by tenant: %w", err)
		}
		if cursor.TenantID != tenantID {
			return nil, "", false, ErrInvalidPageCursor
		}
		conditions = append(conditions, fmt.Sprintf("(created_at < $%d OR (created_at = $%d AND id < $%d))", len(args)+1, len(args)+1, len(args)+2))
		args = append(args, cursor.CreatedAt, cursor.ID)
	}

	query := `
		SELECT id, flow_id, tenant_id, kind, status, requested_by, reviewed_by, review_note, reason, payload, created_at, reviewed_at
		FROM approvals
		WHERE ` + strings.Join(conditions, " AND ")
	query += fmt.Sprintf(" ORDER BY created_at DESC, id DESC LIMIT $%d", len(args)+1)
	args = append(args, limit+1)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", false, fmt.Errorf("list approvals page by tenant: %w", err)
	}
	defer rows.Close()

	approvals, err := scanApprovals(rows)
	if err != nil {
		return nil, "", false, fmt.Errorf("list approvals page by tenant: %w", err)
	}
	hasMore := len(approvals) > limit
	if hasMore {
		approvals = approvals[:limit]
	}
	nextAfter := ""
	if hasMore && len(approvals) > 0 {
		nextAfter = approvals[len(approvals)-1].ID
	}
	return approvals, nextAfter, hasMore, nil
}

func (s *PostgresStore) ListApprovalsByFlow(ctx context.Context, flowID string) ([]domain.Approval, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, flow_id, tenant_id, kind, status, requested_by, reviewed_by, review_note, reason, payload, created_at, reviewed_at
		FROM approvals
		WHERE flow_id = $1
		ORDER BY created_at ASC
	`, flowID)
	if err != nil {
		return nil, fmt.Errorf("list approvals by flow: %w", err)
	}
	defer rows.Close()
	return scanApprovals(rows)
}

func (s *PostgresStore) ReviewApproval(ctx context.Context, id string, approved bool, reviewedBy, note string) (domain.Approval, error) {
	status := "rejected"
	if approved {
		status = "approved"
	}
	approval, err := scanApproval(s.db.QueryRowContext(ctx, `
		UPDATE approvals
		SET status = $2, reviewed_by = $3, review_note = $4, reviewed_at = NOW()
		WHERE id = $1
		RETURNING id, flow_id, tenant_id, kind, status, requested_by, reviewed_by, review_note, reason, payload, created_at, reviewed_at
	`, id, status, blankToDefault(strings.TrimSpace(reviewedBy), "anonymous"), strings.TrimSpace(note)))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Approval{}, ErrNotFound
		}
		return domain.Approval{}, fmt.Errorf("review approval: %w", err)
	}
	return approval, nil
}

func (s *PostgresStore) RecordEvent(ctx context.Context, flowID, eventType, message string, payload map[string]any) (domain.Event, error) {
	row, err := s.q.RecordEvent(ctx, dbgen.RecordEventParams{
		ID:        ids.New("evt"),
		FlowID:    flowID,
		EventType: eventType,
		Message:   message,
		Payload:   toJSON(payload),
	})
	if err != nil {
		return domain.Event{}, fmt.Errorf("record event: %w", err)
	}
	return toDomainEvent(row), nil
}

func scanFlow(scanner interface{ Scan(...any) error }) (domain.Flow, error) {
	var flow domain.Flow
	err := scanner.Scan(
		&flow.ID,
		&flow.TenantID,
		&flow.Name,
		&flow.Target,
		&flow.Objective,
		&flow.Status,
		&flow.CreatedAt,
		&flow.UpdatedAt,
	)
	return flow, err
}

func scanFlows(rows *sql.Rows) ([]domain.Flow, error) {
	out := make([]domain.Flow, 0)
	for rows.Next() {
		flow, err := scanFlow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan flow row: %w", err)
		}
		out = append(out, flow)
	}
	return out, rows.Err()
}

func scanApproval(scanner interface{ Scan(...any) error }) (domain.Approval, error) {
	var approval domain.Approval
	var payload []byte
	var reviewedAt sql.NullTime
	err := scanner.Scan(
		&approval.ID,
		&approval.FlowID,
		&approval.TenantID,
		&approval.Kind,
		&approval.Status,
		&approval.RequestedBy,
		&approval.ReviewedBy,
		&approval.ReviewNote,
		&approval.Reason,
		&payload,
		&approval.CreatedAt,
		&reviewedAt,
	)
	if err != nil {
		return domain.Approval{}, fmt.Errorf("scan approval: %w", err)
	}
	approval.Payload = toStringMap(payload)
	approval.ReviewedAt = nullTime(reviewedAt)
	return approval, nil
}

func scanApprovals(rows *sql.Rows) ([]domain.Approval, error) {
	out := make([]domain.Approval, 0)
	for rows.Next() {
		approval, err := scanApproval(rows)
		if err != nil {
			return nil, fmt.Errorf("scan approval row: %w", err)
		}
		out = append(out, approval)
	}
	return out, rows.Err()
}

func blankToDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func (s *PostgresStore) ListEvents(ctx context.Context, flowID string, afterSequence int64) ([]domain.Event, error) {
	rows, err := s.q.ListEvents(ctx, dbgen.ListEventsParams{
		FlowID: flowID,
		Seq:    afterSequence,
	})
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	return toDomainEvents(rows), nil
}
