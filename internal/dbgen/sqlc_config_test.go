package dbgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSQLCConfigReferencesExistingPaths(t *testing.T) {
	root := filepath.Join("..", "..")
	configPath := filepath.Join(root, "sqlc.yaml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read sqlc config: %v", err)
	}

	text := string(content)
	schemaPaths := yamlListValue(text, "schema:")
	queryPath := yamlStringValue(text, "queries:")
	if len(schemaPaths) == 0 {
		t.Fatal("schema path missing from sqlc.yaml")
	}
	if queryPath == "" {
		t.Fatal("query path missing from sqlc.yaml")
	}

	for _, schemaPath := range schemaPaths {
		if _, err := os.Stat(filepath.Join(root, schemaPath)); err != nil {
			t.Fatalf("schema path does not exist: %s: %v", schemaPath, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, queryPath)); err != nil {
		t.Fatalf("query path does not exist: %s: %v", queryPath, err)
	}
}

func yamlStringValue(text, key string) string {
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, key) {
			continue
		}
		value := strings.TrimSpace(strings.TrimPrefix(trimmed, key))
		return strings.Trim(value, "\"")
	}
	return ""
}

func yamlListValue(text, key string) []string {
	lines := strings.Split(text, "\n")
	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, key) {
			continue
		}
		inline := strings.TrimSpace(strings.TrimPrefix(trimmed, key))
		if inline != "" {
			inline = strings.Trim(inline, "[]")
			if inline == "" {
				return nil
			}
			raw := strings.Split(inline, ",")
			values := make([]string, 0, len(raw))
			for _, item := range raw {
				item = strings.Trim(strings.TrimSpace(item), "\"")
				if item != "" {
					values = append(values, item)
				}
			}
			return values
		}

		values := make([]string, 0)
		for _, child := range lines[idx+1:] {
			childTrimmed := strings.TrimSpace(child)
			if childTrimmed == "" {
				continue
			}
			if !strings.HasPrefix(childTrimmed, "- ") {
				break
			}
			value := strings.Trim(strings.TrimSpace(strings.TrimPrefix(childTrimmed, "- ")), "\"")
			if value != "" {
				values = append(values, value)
			}
		}
		return values
	}
	return nil
}
