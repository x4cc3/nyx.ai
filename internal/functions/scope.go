package functions

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
)

var (
	urlCandidatePattern  = regexp.MustCompile(`https?://[^\s"'<>]+`)
	hostCandidatePattern = regexp.MustCompile(`\b(?:[a-zA-Z0-9-]+\.)+[a-zA-Z]{2,}\b|\b(?:\d{1,3}\.){3}\d{1,3}\b`)
	hostPortPattern      = regexp.MustCompile(`\b(?:[a-zA-Z0-9-]+\.)+[a-zA-Z]{2,}:\d{1,5}\b|\b(?:\d{1,3}\.){3}\d{1,3}:\d{1,5}\b`)
)

type targetScope struct {
	host       string
	rootDomain string
}

func deriveTargetScope(raw string) (targetScope, error) {
	host := parseHost(raw)
	if host == "" {
		return targetScope{}, fmt.Errorf("target scope requires a URL or hostname, got %q", strings.TrimSpace(raw))
	}
	return targetScope{
		host:       host,
		rootDomain: registrableDomain(host),
	}, nil
}

func validateBrowserScope(target, rawURL string) error {
	if strings.TrimSpace(rawURL) == "" {
		return nil
	}
	if strings.TrimSpace(target) == "" {
		return fmt.Errorf("browser actions require an in-scope target")
	}

	scope, err := deriveTargetScope(target)
	if err != nil {
		return err
	}
	host := parseHost(rawURL)
	if host == "" {
		return fmt.Errorf("browser URL %q is not a valid target URL", strings.TrimSpace(rawURL))
	}
	if !scope.allowsHost(host) {
		return fmt.Errorf("browser target %q is outside the allowed scope for %q", strings.TrimSpace(rawURL), strings.TrimSpace(target))
	}
	return nil
}

func validateCommandScope(target, command string) error {
	command = strings.TrimSpace(command)
	if command == "" {
		return nil
	}
	if strings.TrimSpace(target) == "" {
		return fmt.Errorf("terminal actions require an in-scope target")
	}

	scope, err := deriveTargetScope(target)
	if err != nil {
		return err
	}

	candidates := collectRemoteHosts(command)
	for _, candidate := range candidates {
		if scope.allowsHost(candidate) {
			continue
		}
		return fmt.Errorf("terminal command references out-of-scope host %q for target %q", candidate, strings.TrimSpace(target))
	}
	return nil
}

func (s targetScope) allowsHost(raw string) bool {
	host := parseHost(raw)
	if host == "" {
		return false
	}
	if s.host == host {
		return true
	}
	if ip := net.ParseIP(s.host); ip != nil {
		return false
	}
	if s.rootDomain != "" && registrableDomain(host) == s.rootDomain {
		return true
	}
	return strings.HasSuffix(host, "."+s.host)
}

func collectRemoteHosts(command string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0)

	for _, match := range urlCandidatePattern.FindAllString(command, -1) {
		if host := parseHost(match); host != "" {
			if _, ok := seen[host]; !ok {
				seen[host] = struct{}{}
				out = append(out, host)
			}
		}
	}

	for _, match := range hostCandidatePattern.FindAllString(command, -1) {
		host := parseHost(match)
		if host == "" || shouldIgnoreHostCandidate(host) {
			continue
		}
		if _, ok := seen[host]; ok {
			continue
		}
		seen[host] = struct{}{}
		out = append(out, host)
	}

	for _, match := range hostPortPattern.FindAllString(command, -1) {
		host := parseHost(match)
		if host == "" || shouldIgnoreHostCandidate(host) {
			continue
		}
		if _, ok := seen[host]; ok {
			continue
		}
		seen[host] = struct{}{}
		out = append(out, host)
	}
	return out
}

func shouldIgnoreHostCandidate(host string) bool {
	if host == "" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback() || ip.IsPrivate()
	}
	switch host {
	case "localhost":
		return true
	}
	for _, suffix := range []string{
		".sh", ".txt", ".md", ".json", ".yaml", ".yml", ".xml", ".html", ".js", ".ts", ".py", ".go", ".rs", ".php", ".jpg", ".jpeg", ".png", ".csv", ".log",
	} {
		if strings.HasSuffix(host, suffix) {
			return true
		}
	}
	return false
}

func parseHost(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	if strings.Contains(raw, "://") {
		parsed, err := url.Parse(raw)
		if err != nil {
			return ""
		}
		return strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	}

	trimmed := strings.Trim(raw, `"'()[]{}<>.,`)
	if trimmed == "" {
		return ""
	}
	if parsed := net.ParseIP(trimmed); parsed != nil {
		return parsed.String()
	}
	if host, _, err := net.SplitHostPort(trimmed); err == nil {
		if parsed := net.ParseIP(host); parsed != nil {
			return parsed.String()
		}
		if strings.Count(host, ".") > 0 {
			return strings.ToLower(strings.TrimSpace(host))
		}
	}
	if strings.Contains(trimmed, "/") {
		return ""
	}
	if strings.Count(trimmed, ".") == 0 {
		return ""
	}
	host := trimmed
	if parsed, err := url.Parse("https://" + trimmed); err == nil {
		host = parsed.Hostname()
	}
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return ""
	}
	return host
}

func registrableDomain(host string) string {
	host = parseHost(host)
	if host == "" {
		return ""
	}
	if net.ParseIP(host) != nil {
		return host
	}
	labels := strings.Split(host, ".")
	if len(labels) < 2 {
		return host
	}
	if len(labels) >= 3 && isCompoundSuffix(labels[len(labels)-2], labels[len(labels)-1]) {
		return strings.Join(labels[len(labels)-3:], ".")
	}
	return strings.Join(labels[len(labels)-2:], ".")
}

func isCompoundSuffix(secondLevel, topLevel string) bool {
	if len(topLevel) != 2 {
		return false
	}
	switch secondLevel {
	case "ac", "co", "com", "edu", "gov", "net", "org":
		return true
	default:
		return false
	}
}
