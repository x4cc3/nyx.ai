package memory

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"nyx/internal/domain"
	"nyx/internal/store"
)

type Service struct {
	repo store.MemoryReadWriter
}

func New(repo store.MemoryReadWriter) *Service {
	return &Service{repo: repo}
}

func (s *Service) Search(ctx context.Context, flowID, query string) []domain.Memory {
	return s.SearchNamespace(ctx, flowID, query, "")
}

func (s *Service) SearchNamespace(ctx context.Context, flowID, query, namespace string) []domain.Memory {
	results, err := s.repo.SearchMemories(ctx, flowID, strings.TrimSpace(query))
	if err != nil {
		return nil
	}
	namespace = domain.NormalizeMemoryNamespace(namespace)
	if namespace != "" {
		filtered := make([]domain.Memory, 0, len(results))
		for _, item := range results {
			if domain.MemoryNamespace(item.Kind, item.Metadata) == namespace {
				filtered = append(filtered, item)
			}
		}
		results = filtered
	}
	if len(results) > 5 {
		results = results[:5]
	}
	return results
}

func (s *Service) StoreTargetObservation(ctx context.Context, flowID, actionID, content string, metadata map[string]string) (domain.Memory, error) {
	return s.store(ctx, flowID, actionID, "observation", domain.MemoryNamespaceTargetObservations, content, metadata)
}

func (s *Service) StoreExploitReference(ctx context.Context, flowID, actionID, content string, metadata map[string]string) (domain.Memory, error) {
	return s.store(ctx, flowID, actionID, domain.MemoryKindExploitReference, domain.MemoryNamespaceExploitReferences, content, metadata)
}

func (s *Service) StoreReferenceMaterial(ctx context.Context, flowID, actionID, content string, metadata map[string]string) (domain.Memory, error) {
	return s.store(ctx, flowID, actionID, domain.MemoryKindReferenceMaterial, domain.MemoryNamespaceReferenceMaterials, content, metadata)
}

func (s *Service) StoreOperatorNote(ctx context.Context, flowID, actionID, content string, metadata map[string]string) (domain.Memory, error) {
	return s.store(ctx, flowID, actionID, domain.MemoryKindOperatorNote, domain.MemoryNamespaceOperatorNotes, content, metadata)
}

func (s *Service) StoreActionResult(ctx context.Context, flowID, actionID, role, functionName string, input, output map[string]string) []domain.Memory {
	specs := memorySpecsFromActionResult(role, functionName, input, output)
	if len(specs) == 0 {
		return nil
	}
	stored := make([]domain.Memory, 0, len(specs))
	for _, spec := range specs {
		item, err := s.store(ctx, flowID, actionID, spec.Kind, spec.Namespace, spec.Content, spec.Metadata)
		if err == nil {
			stored = append(stored, item)
		}
	}
	return stored
}

type memorySpec struct {
	Kind      string
	Namespace string
	Content   string
	Metadata  map[string]string
}

func (s *Service) store(ctx context.Context, flowID, actionID, kind, namespace, content string, metadata map[string]string) (domain.Memory, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return domain.Memory{}, nil
	}
	next := cloneMetadata(metadata)
	next["namespace"] = domain.MemoryNamespace(kind, map[string]string{"namespace": namespace})
	return s.repo.AddMemory(ctx, flowID, actionID, kind, content, next)
}

func memorySpecsFromActionResult(role, functionName string, input, output map[string]string) []memorySpec {
	metadata := map[string]string{
		"agent_role":       strings.TrimSpace(role),
		"source_function":  strings.TrimSpace(functionName),
		"preferred_output": strings.TrimSpace(input["preferred_output"]),
	}
	specs := make([]memorySpec, 0, 2)

	switch functionName {
	case "search_exploits":
		if content := summarizeSearchOutput(output, "Exploit research"); content != "" {
			next := cloneMetadata(metadata)
			copyIfPresent(next, output, "source")
			copyIfPresent(next, input, "query")
			specs = append(specs, memorySpec{
				Kind:      domain.MemoryKindExploitReference,
				Namespace: domain.MemoryNamespaceExploitReferences,
				Content:   content,
				Metadata:  next,
			})
		}
	case "search_code":
		if content := summarizeSearchOutput(output, "Reference material"); content != "" {
			next := cloneMetadata(metadata)
			copyIfPresent(next, output, "source")
			copyIfPresent(next, input, "query")
			specs = append(specs, memorySpec{
				Kind:      domain.MemoryKindReferenceMaterial,
				Namespace: domain.MemoryNamespaceReferenceMaterials,
				Content:   content,
				Metadata:  next,
			})
		}
	case "search_web", "search_deep":
		if content := summarizeSearchOutput(output, "Target research"); content != "" {
			next := cloneMetadata(metadata)
			copyIfPresent(next, output, "source")
			copyIfPresent(next, input, "query")
			specs = append(specs, memorySpec{
				Kind:      "observation",
				Namespace: domain.MemoryNamespaceTargetObservations,
				Content:   content,
				Metadata:  next,
			})
		}
	case "browser", "browser_html", "browser_markdown", "browser_links", "browser_screenshot":
		if content := summarizeBrowserOutput(output); content != "" {
			next := cloneMetadata(metadata)
			copyIfPresent(next, output, "final_url")
			copyIfPresent(next, output, "status_code")
			copyIfPresent(next, output, "auth_state")
			specs = append(specs, memorySpec{
				Kind:      "observation",
				Namespace: domain.MemoryNamespaceTargetObservations,
				Content:   content,
				Metadata:  next,
			})
		}
	case "file", "file_write":
		if content := summarizeOperatorNote(input, output); content != "" {
			next := cloneMetadata(metadata)
			copyIfPresent(next, input, "path")
			copyIfPresent(next, output, "file_path")
			specs = append(specs, memorySpec{
				Kind:      domain.MemoryKindOperatorNote,
				Namespace: domain.MemoryNamespaceOperatorNotes,
				Content:   content,
				Metadata:  next,
			})
		}
	case "search_memory", "done", "ask":
		return nil
	default:
		if content := summarizeObservation(output["summary"]); content != "" {
			next := cloneMetadata(metadata)
			copyIfPresent(next, input, "target")
			specs = append(specs, memorySpec{
				Kind:      "observation",
				Namespace: domain.MemoryNamespaceTargetObservations,
				Content:   content,
				Metadata:  next,
			})
		}
	}

	return specs
}

func summarizeSearchOutput(output map[string]string, prefix string) string {
	summary := summarizeObservation(output["summary"])
	if summary == "" {
		return ""
	}
	lines := []string{fmt.Sprintf("%s: %s", prefix, summary)}
	if source := strings.TrimSpace(output["source"]); source != "" {
		lines = append(lines, "Source: "+source)
	}
	references := collectSearchReferences(output, 3)
	if len(references) > 0 {
		lines = append(lines, "References: "+strings.Join(references, " | "))
	}
	return strings.Join(lines, "\n")
}

func summarizeBrowserOutput(output map[string]string) string {
	parts := make([]string, 0, 5)
	if title := strings.TrimSpace(output["title"]); title != "" {
		parts = append(parts, "Title: "+title)
	}
	if finalURL := strings.TrimSpace(output["final_url"]); finalURL != "" {
		parts = append(parts, "URL: "+finalURL)
	}
	if summary := summarizeObservation(output["summary"]); summary != "" {
		parts = append(parts, "Observation: "+summary)
	}
	if mode := strings.TrimSpace(output["mode"]); mode != "" {
		parts = append(parts, "Mode: "+mode)
	}
	if authState := strings.TrimSpace(output["auth_state"]); authState != "" {
		parts = append(parts, "Auth: "+authState)
	}
	return strings.Join(parts, "\n")
}

func summarizeOperatorNote(input, output map[string]string) string {
	path := strings.TrimSpace(input["path"])
	if path == "" {
		path = strings.TrimSpace(output["file_path"])
	}
	content := strings.TrimSpace(input["content"])
	if content == "" {
		content = summarizeObservation(output["summary"])
	}
	if path == "" && content == "" {
		return ""
	}
	if content == "" {
		return "Operator note written to " + path
	}
	content = summarizeObservation(content)
	if path == "" {
		return "Operator note: " + content
	}
	return fmt.Sprintf("Operator note at %s: %s", path, content)
}

func summarizeObservation(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) > 320 {
		value = value[:319] + "…"
	}
	return value
}

func collectSearchReferences(output map[string]string, limit int) []string {
	indices := make([]int, 0)
	for key := range output {
		if strings.HasPrefix(key, "result_") && strings.HasSuffix(key, "_url") {
			raw := strings.TrimSuffix(strings.TrimPrefix(key, "result_"), "_url")
			idx, err := strconv.Atoi(raw)
			if err == nil {
				indices = append(indices, idx)
			}
		}
	}
	sort.Ints(indices)
	refs := make([]string, 0, min(limit, len(indices)))
	for _, idx := range indices {
		title := strings.TrimSpace(output[fmt.Sprintf("result_%d_title", idx)])
		link := strings.TrimSpace(output[fmt.Sprintf("result_%d_url", idx)])
		if link == "" {
			continue
		}
		if title == "" {
			refs = append(refs, link)
		} else {
			refs = append(refs, fmt.Sprintf("%s (%s)", title, link))
		}
		if len(refs) == limit {
			break
		}
	}
	return refs
}

func copyIfPresent(dst, src map[string]string, key string) {
	if src == nil {
		return
	}
	if value := strings.TrimSpace(src[key]); value != "" {
		dst[key] = value
	}
}

func cloneMetadata(metadata map[string]string) map[string]string {
	if len(metadata) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(metadata))
	for key, value := range metadata {
		out[key] = value
	}
	return out
}
