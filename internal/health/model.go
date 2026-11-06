// Package health defines the normalized local run record.
package health

import "time"

type Scope struct {
	SubscriptionID string `json:"subscription_id"`
	Name           string `json:"name"`
	TenantID       string `json:"tenant_id"`
	Cloud          string `json:"cloud"`
}

type Source struct {
	Status          string `json:"status"`
	Reason          string `json:"reason,omitempty"`
	Pages           int    `json:"pages"`
	RejectedRecords int    `json:"rejected_records"`
}

type Run struct {
	SchemaVersion  int               `json:"schema_version"`
	ID             string            `json:"run_id"`
	Revision       int               `json:"revision"`
	ToolVersion    string            `json:"tool_version,omitempty"`
	StartedAt      time.Time         `json:"started_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	EndedAt        *time.Time        `json:"ended_at,omitempty"`
	Outcome        string            `json:"outcome"`
	Coverage       string            `json:"coverage"`
	RequestedScope string            `json:"requested_scope"`
	Scope          *Scope            `json:"scope,omitempty"`
	Sources        map[string]Source `json:"sources"`
	Diagnostics    []string          `json:"diagnostics"`
	Redactions     int               `json:"redactions"`
	Resources      []Resource        `json:"resources"`
	Evidence       []Evidence        `json:"evidence"`
	Observations   []Observation     `json:"observations"`
	Findings       []Finding         `json:"findings"`
	Gaps           []Gap             `json:"coverage_gaps"`
}

func NewRun(requested, version string) Run {
	text, count := SafeText(requested)
	sources := make(map[string]Source)
	for _, name := range []string{"inventory", "resource_health"} {
		sources[name] = Source{Status: "not_checked", Reason: "not_attempted"}
	}
	for _, name := range []string{"advisor", "service_health", "azure_monitor"} {
		sources[name] = Source{Status: "not_checked", Reason: "not_implemented_in_s01"}
	}
	return Run{SchemaVersion: 1, ToolVersion: version, Outcome: "in_progress", Coverage: "incomplete", RequestedScope: text, Sources: sources, Diagnostics: []string{}, Redactions: count}
}
