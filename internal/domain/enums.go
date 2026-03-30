package domain

// ArtifactKind identifies the category of an artifact.
type ArtifactKind = string

const (
	ArtifactKindLog        ArtifactKind = "log"
	ArtifactKindApproval   ArtifactKind = "approval"
	ArtifactKindExecution  ArtifactKind = "execution"
	ArtifactKindScreenshot ArtifactKind = "screenshot"
	ArtifactKindSnapshot   ArtifactKind = "snapshot"
	ArtifactKindStdout     ArtifactKind = "stdout"
	ArtifactKindStderr     ArtifactKind = "stderr"
	ArtifactKindPlan       ArtifactKind = "plan"
)

// MemoryKind identifies the category of a memory entry.
type MemoryKind = string

const (
	MemoryKindExploitReference  MemoryKind = "exploit_reference"
	MemoryKindReferenceMaterial MemoryKind = "reference_material"
	MemoryKindOperatorNote      MemoryKind = "operator_note"
	MemoryKindPlanSummary       MemoryKind = "plan_summary"
)
