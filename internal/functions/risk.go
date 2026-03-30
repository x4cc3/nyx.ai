package functions

import (
	"errors"
	"sort"
	"strconv"
	"strings"
)

var ErrApprovalRequired = errors.New("operator approval required")

const approvalRequiredOutputKey = "approval_required"

const (
	RiskCategoryRawSocket = "raw_socket_scan"
	RiskCategoryHighRate  = "high_rate_scan"
	RiskCategoryIntrusive = "intrusive_exploit_tool"
)

type TerminalRiskAssessment struct {
	Categories []string
}

func (a TerminalRiskAssessment) RequiresApproval() bool {
	return len(a.Categories) > 0
}

func (a TerminalRiskAssessment) CategoryList() string {
	return strings.Join(a.Categories, ",")
}

func (a TerminalRiskAssessment) Summary() string {
	if len(a.Categories) == 0 {
		return ""
	}
	labels := make([]string, 0, len(a.Categories))
	for _, category := range a.Categories {
		switch category {
		case RiskCategoryRawSocket:
			labels = append(labels, "raw socket scans")
		case RiskCategoryHighRate:
			labels = append(labels, "high-rate scans")
		case RiskCategoryIntrusive:
			labels = append(labels, "intrusive exploit tools")
		default:
			labels = append(labels, category)
		}
	}
	return strings.Join(labels, ", ")
}

func IsApprovalRequired(err error) bool {
	return errors.Is(err, ErrApprovalRequired)
}

func ApprovalRequiredOutput(summary string, assessment TerminalRiskAssessment) map[string]string {
	output := map[string]string{
		approvalRequiredOutputKey: "true",
		"summary":                 strings.TrimSpace(summary),
	}
	if categories := assessment.CategoryList(); categories != "" {
		output["risk_categories"] = categories
	}
	return output
}

func ApprovalRequired(result CallResult) bool {
	return strings.EqualFold(strings.TrimSpace(result.Output[approvalRequiredOutputKey]), "true") || IsApprovalRequired(result.Err)
}

func AssessTerminalRisk(functionName string, input map[string]string) TerminalRiskAssessment {
	switch strings.TrimSpace(functionName) {
	case "terminal", "terminal_exec":
	default:
		return TerminalRiskAssessment{}
	}

	command := strings.ToLower(strings.TrimSpace(input["command"]))
	if command == "" {
		return TerminalRiskAssessment{}
	}

	categories := make(map[string]struct{}, 3)
	tokens := strings.Fields(command)

	if requiresRawSocketInput(input) || usesRawSocketTool(tokens, command) {
		categories[RiskCategoryRawSocket] = struct{}{}
	}
	if isHighRateCommand(tokens, command) {
		categories[RiskCategoryHighRate] = struct{}{}
	}
	if usesIntrusiveTool(tokens, command) {
		categories[RiskCategoryIntrusive] = struct{}{}
	}

	if len(categories) == 0 {
		return TerminalRiskAssessment{}
	}
	out := make([]string, 0, len(categories))
	for category := range categories {
		out = append(out, category)
	}
	sort.Strings(out)
	return TerminalRiskAssessment{Categories: out}
}

func requiresRawSocketInput(input map[string]string) bool {
	for _, key := range []string{"requires_raw_socket", "enable_net_raw"} {
		if value, err := strconv.ParseBool(strings.TrimSpace(input[key])); err == nil && value {
			return true
		}
	}
	return false
}

func usesRawSocketTool(tokens []string, command string) bool {
	for _, tool := range []string{"masscan", "hping", "hping3", "nping", "arp-scan", "tcpdump"} {
		if hasToken(tokens, tool) {
			return true
		}
	}
	if hasToken(tokens, "naabu") {
		return true
	}
	if hasToken(tokens, "nmap") {
		for _, marker := range []string{"-ss", "-su", "-o", "-a", "--privileged"} {
			if strings.Contains(command, marker) {
				return true
			}
		}
	}
	return false
}

func isHighRateCommand(tokens []string, command string) bool {
	if hasToken(tokens, "masscan") {
		return true
	}
	if hasToken(tokens, "naabu") && numericFlagAtLeast(tokens, []string{"-rate", "--rate"}, 500) {
		return true
	}
	if commandUsesConcurrency(tokens, command, []string{"ffuf", "gobuster", "feroxbuster", "dirsearch", "httpx", "nuclei"}, 40) {
		return true
	}
	if numericFlagAtLeast(tokens, []string{"-rl", "-rate", "--rate", "--rate-limit"}, 100) {
		return true
	}
	return false
}

func commandUsesConcurrency(tokens []string, command string, tools []string, threshold int) bool {
	for _, tool := range tools {
		if !hasToken(tokens, tool) {
			continue
		}
		if numericFlagAtLeast(tokens, []string{"-t", "-threads", "--threads", "-c", "--concurrency", "--bulk-size"}, threshold) {
			return true
		}
		if strings.Contains(command, "--stream") && tool == "ffuf" {
			return true
		}
	}
	return false
}

func usesIntrusiveTool(tokens []string, command string) bool {
	for _, tool := range []string{"sqlmap", "nikto", "nuclei", "wpscan", "hydra", "commix", "msfconsole"} {
		if hasToken(tokens, tool) {
			return true
		}
	}
	return strings.Contains(command, "metasploit")
}

func hasToken(tokens []string, wanted string) bool {
	for _, token := range tokens {
		if strings.TrimSpace(token) == wanted {
			return true
		}
	}
	return false
}

func numericFlagAtLeast(tokens []string, flags []string, threshold int) bool {
	if threshold <= 0 {
		return false
	}
	for idx, token := range tokens {
		for _, flag := range flags {
			if token == flag && idx+1 < len(tokens) {
				if value, err := strconv.Atoi(strings.TrimSpace(tokens[idx+1])); err == nil && value >= threshold {
					return true
				}
			}
			prefix := flag + "="
			if strings.HasPrefix(token, prefix) {
				if value, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(token, prefix))); err == nil && value >= threshold {
					return true
				}
			}
		}
	}
	return false
}
