package domain

import (
	"strings"
	"time"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusQueued    Status = "queued"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

type Flow struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Name      string    `json:"name"`
	Target    string    `json:"target"`
	Objective string    `json:"objective"`
	Status    Status    `json:"status"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Task struct {
	ID          string    `json:"id"`
	FlowID      string    `json:"flow_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      Status    `json:"status"`
	AgentRole   string    `json:"agent_role"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Subtask struct {
	ID          string    `json:"id"`
	TaskID      string    `json:"task_id"`
	FlowID      string    `json:"flow_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      Status    `json:"status"`
	AgentRole   string    `json:"agent_role"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Action struct {
	ID            string            `json:"id"`
	FlowID        string            `json:"flow_id"`
	TaskID        string            `json:"task_id"`
	SubtaskID     string            `json:"subtask_id"`
	AgentRole     string            `json:"agent_role"`
	FunctionName  string            `json:"function_name"`
	Input         map[string]string `json:"input"`
	Output        map[string]string `json:"output"`
	Status        Status            `json:"status"`
	ExecutionMode string            `json:"execution_mode"`
	RetryCount    int               `json:"retry_count"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

type Artifact struct {
	ID        string            `json:"id"`
	FlowID    string            `json:"flow_id"`
	ActionID  string            `json:"action_id"`
	Kind      string            `json:"kind"`
	Name      string            `json:"name"`
	Content   string            `json:"content"`
	Metadata  map[string]string `json:"metadata"`
	TTL       *time.Duration    `json:"ttl,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

type Memory struct {
	ID        string            `json:"id"`
	FlowID    string            `json:"flow_id"`
	ActionID  string            `json:"action_id"`
	Kind      string            `json:"kind"`
	Content   string            `json:"content"`
	Metadata  map[string]string `json:"metadata"`
	CreatedAt time.Time         `json:"created_at"`
}

const (
	MemoryNamespaceAll                = "all"
	MemoryNamespaceTargetObservations = "target_observations"
	MemoryNamespaceExploitReferences  = "exploit_references"
	MemoryNamespaceReferenceMaterials = "reference_materials"
	MemoryNamespaceOperatorNotes      = "operator_notes"
)

func NormalizeMemoryNamespace(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", MemoryNamespaceAll:
		return ""
	case MemoryNamespaceTargetObservations:
		return MemoryNamespaceTargetObservations
	case MemoryNamespaceExploitReferences:
		return MemoryNamespaceExploitReferences
	case MemoryNamespaceReferenceMaterials:
		return MemoryNamespaceReferenceMaterials
	case MemoryNamespaceOperatorNotes:
		return MemoryNamespaceOperatorNotes
	default:
		return ""
	}
}

func MemoryNamespace(kind string, metadata map[string]string) string {
	if metadata != nil {
		if namespace := NormalizeMemoryNamespace(metadata["namespace"]); namespace != "" {
			return namespace
		}
	}
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "exploit_reference":
		return MemoryNamespaceExploitReferences
	case "reference_material", "reference":
		return MemoryNamespaceReferenceMaterials
	case "operator_note":
		return MemoryNamespaceOperatorNotes
	default:
		return MemoryNamespaceTargetObservations
	}
}

type Finding struct {
	ID          string    `json:"id"`
	FlowID      string    `json:"flow_id"`
	Title       string    `json:"title"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type Agent struct {
	ID        string    `json:"id"`
	FlowID    string    `json:"flow_id"`
	Role      string    `json:"role"`
	Model     string    `json:"model"`
	Status    Status    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Execution struct {
	ID          string            `json:"id"`
	FlowID      string            `json:"flow_id"`
	ActionID    string            `json:"action_id"`
	Profile     string            `json:"profile"`
	Runtime     string            `json:"runtime"`
	Metadata    map[string]string `json:"metadata"`
	Status      Status            `json:"status"`
	StartedAt   time.Time         `json:"started_at"`
	CompletedAt time.Time         `json:"completed_at"`
}

type FlowDetail struct {
	Flow       Flow        `json:"flow"`
	Tasks      []Task      `json:"tasks"`
	Subtasks   []Subtask   `json:"subtasks"`
	Actions    []Action    `json:"actions"`
	Artifacts  []Artifact  `json:"artifacts"`
	Memories   []Memory    `json:"memories"`
	Findings   []Finding   `json:"findings"`
	Agents     []Agent     `json:"agents"`
	Executions []Execution `json:"executions"`
}

type Approval struct {
	ID          string            `json:"id"`
	FlowID      string            `json:"flow_id"`
	TenantID    string            `json:"tenant_id"`
	Kind        string            `json:"kind"`
	Status      string            `json:"status"`
	RequestedBy string            `json:"requested_by"`
	ReviewedBy  string            `json:"reviewed_by"`
	ReviewNote  string            `json:"review_note"`
	Reason      string            `json:"reason"`
	Payload     map[string]string `json:"payload"`
	ExpiresAt   *time.Time        `json:"expires_at,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	ReviewedAt  time.Time         `json:"reviewed_at"`
}

type Workspace struct {
	Flow                     Flow          `json:"flow"`
	Tasks                    []Task        `json:"tasks"`
	Subtasks                 []Subtask     `json:"subtasks"`
	Actions                  []Action      `json:"actions"`
	Artifacts                []Artifact    `json:"artifacts"`
	Memories                 []Memory      `json:"memories"`
	Findings                 []Finding     `json:"findings"`
	Agents                   []Agent       `json:"agents"`
	Executions               []Execution   `json:"executions"`
	Approvals                []Approval    `json:"approvals"`
	Functions                []FunctionDef `json:"functions"`
	TenantID                 string        `json:"tenant_id"`
	QueueMode                string        `json:"queue_mode"`
	ExecutorMode             string        `json:"executor_mode"`
	BrowserMode              string        `json:"browser_mode"`
	ExecutorNetworkMode      string        `json:"executor_network_mode"`
	ExecutorNetworkName      string        `json:"executor_network_name"`
	ExecutorNetRawEnabled    bool          `json:"executor_net_raw_enabled"`
	TerminalNetworkEnabled   bool          `json:"terminal_network_enabled"`
	RiskyApprovalRequired    bool          `json:"risky_approval_required"`
	FlowMaxConcurrentActions int           `json:"flow_max_concurrent_actions"`
	FlowMinActionIntervalMS  int64         `json:"flow_min_action_interval_ms"`
	BrowserWarning           string        `json:"browser_warning"`
	NetworkWarning           string        `json:"network_warning"`
	RiskWarning              string        `json:"risk_warning"`
	NeedsReview              bool          `json:"needs_review"`
}

type Event struct {
	ID        string         `json:"id"`
	Sequence  int64          `json:"sequence"`
	FlowID    string         `json:"flow_id"`
	Type      string         `json:"type"`
	Message   string         `json:"message"`
	Payload   map[string]any `json:"payload"`
	CreatedAt time.Time      `json:"created_at"`
}

type CreateFlowInput struct {
	Name      string `json:"name"`
	Target    string `json:"target"`
	Objective string `json:"objective"`
}

type ApprovalReviewInput struct {
	Approved bool   `json:"approved"`
	Note     string `json:"note"`
}

type FunctionInputField struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

type FunctionDef struct {
	Name                 string               `json:"name"`
	Description          string               `json:"description"`
	Profile              string               `json:"profile"`
	Category             string               `json:"category,omitempty"`
	SafetyProfile        string               `json:"safety_profile,omitempty"`
	RequiresNetwork      bool                 `json:"requires_network,omitempty"`
	RequiresPentestImage bool                 `json:"requires_pentest_image,omitempty"`
	InputSchema          []FunctionInputField `json:"input_schema,omitempty"`
}
