package executor

import "strings"

const (
	toolsetGeneral = "general"
	toolsetPentest = "pentest"
)

func resolveToolset(input map[string]string) string {
	if value := normalizeToolset(input["toolset"]); value != "" {
		return value
	}
	if shouldUsePentestToolset(input) {
		return toolsetPentest
	}
	return toolsetGeneral
}

func normalizeToolset(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case toolsetGeneral:
		return toolsetGeneral
	case toolsetPentest:
		return toolsetPentest
	default:
		return ""
	}
}

func shouldUsePentestToolset(input map[string]string) bool {
	if len(input) == 0 {
		return false
	}
	combined := strings.ToLower(strings.Join([]string{
		input["goal"],
		input["objective"],
		input["task_name"],
		input["subtask_name"],
	}, " "))
	for _, marker := range []string{
		"pentest",
		"penetration test",
		"security assessment",
		"security testing",
		"vulnerability",
		"recon",
		"reconnaissance",
		"enumeration",
		"exploit",
		"nuclei",
		"subfinder",
		"httpx",
		"katana",
		"sqlmap",
		"ffuf",
	} {
		if strings.Contains(combined, marker) {
			return true
		}
	}
	return false
}
