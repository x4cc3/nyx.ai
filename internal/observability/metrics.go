package observability

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
)

type Registry struct {
	mu       sync.RWMutex
	counters map[string]float64
	gauges   map[string]float64
}

func NewRegistry() *Registry {
	return &Registry{
		counters: make(map[string]float64),
		gauges:   make(map[string]float64),
	}
}

func (r *Registry) IncCounter(name string, labels map[string]string, delta float64) {
	if delta == 0 {
		delta = 1
	}
	key := metricKey(name, labels)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.counters[key] += delta
}

func (r *Registry) SetGauge(name string, labels map[string]string, value float64) {
	key := metricKey(name, labels)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.gauges[key] = value
}

func (r *Registry) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = fmt.Fprint(w, r.Render())
	})
}

func (r *Registry) Render() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	keys := make([]string, 0, len(r.counters)+len(r.gauges))
	for key := range r.counters {
		keys = append(keys, key)
	}
	for key := range r.gauges {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	seen := make(map[string]struct{})
	var b strings.Builder
	for _, key := range keys {
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if value, ok := r.counters[key]; ok {
			fmt.Fprintf(&b, "%s %g\n", key, value)
		}
		if value, ok := r.gauges[key]; ok {
			fmt.Fprintf(&b, "%s %g\n", key, value)
		}
	}
	return b.String()
}

func metricKey(name string, labels map[string]string) string {
	if len(labels) == 0 {
		return sanitizeMetric(name)
	}
	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf(`%s="%s"`, sanitizeMetric(key), escapeLabel(labels[key])))
	}
	return sanitizeMetric(name) + "{" + strings.Join(parts, ",") + "}"
}

func sanitizeMetric(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, ".", "_")
	name = strings.ReplaceAll(name, "/", "_")
	if name == "" {
		return "nyx_metric"
	}
	return name
}

func escapeLabel(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return value
}
