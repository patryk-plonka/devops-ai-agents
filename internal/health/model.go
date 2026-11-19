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
	Status              string       `json:"status"`
	Reason              string       `json:"reason,omitempty"`
	Pages               int          `json:"pages"`
	RejectedRecords     int          `json:"rejected_records"`
	ExcludedRecords     int          `json:"excluded_records,omitempty"`
	UnclassifiedRecords int          `json:"unclassified_records,omitempty"`
	Query               *SourceQuery `json:"query,omitempty"`
	FirstPageAt         *time.Time   `json:"first_page_at,omitempty"`
	LastPageAt          *time.Time   `json:"last_page_at,omitempty"`
}

// SourceQuery records fixed query policy, never continuation tokens or URLs.
type SourceQuery struct {
	ReferenceAt    time.Time  `json:"reference_at"`
	Cutoff         *time.Time `json:"cutoff,omitempty"`
	QueryStartTime string     `json:"query_start_time,omitempty"`
	TimeRange      string     `json:"time_range,omitempty"`
	Operation      string     `json:"operation,omitempty"`
	APIVersion     string     `json:"api_version,omitempty"`
	// RequestedAt records preparation at the pre-collection checkpoint.
	RequestedAt *time.Time `json:"requested_at,omitempty"`
}

func ServiceHealthQuery(reference time.Time) SourceQuery {
	reference = reference.UTC()
	cutoff := reference.Add(-30 * 24 * time.Hour)
	return SourceQuery{ReferenceAt: reference, Cutoff: &cutoff, QueryStartTime: cutoff.Format("1/2/2006")}
}

func MonitorQuery(reference time.Time) SourceQuery {
	return SourceQuery{ReferenceAt: reference.UTC(), TimeRange: "30d"}
}

func AdvisorQuery(reference time.Time) SourceQuery {
	return SourceQuery{ReferenceAt: reference.UTC(), Operation: AdvisorListOperation, APIVersion: AdvisorAPI}
}

type Run struct {
	SchemaVersion    int               `json:"schema_version"`
	ID               string            `json:"run_id"`
	Revision         int               `json:"revision"`
	ToolVersion      string            `json:"tool_version,omitempty"`
	StartedAt        time.Time         `json:"started_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
	EndedAt          *time.Time        `json:"ended_at,omitempty"`
	Outcome          string            `json:"outcome"`
	Coverage         string            `json:"coverage"`
	RequestedScope   string            `json:"requested_scope"`
	Scope            *Scope            `json:"scope,omitempty"`
	Sources          map[string]Source `json:"sources"`
	Diagnostics      []string          `json:"diagnostics"`
	Redactions       int               `json:"redactions"`
	Resources        []Resource        `json:"resources"`
	Evidence         []Evidence        `json:"evidence"`
	Observations     []Observation     `json:"observations"`
	Findings         []Finding         `json:"findings"`
	AdvisorInstances []AdvisorInstance `json:"advisor_instances,omitempty"`
	Gaps             []Gap             `json:"coverage_gaps"`
}

func NewRun(requested, version string) Run {
	text, count := SafeText(requested)
	sources := make(map[string]Source)
	for _, name := range []string{"inventory", "resource_health", "service_health", "azure_monitor", "advisor"} {
		sources[name] = Source{Status: "not_checked", Reason: "not_attempted"}
	}
	return Run{SchemaVersion: 3, ToolVersion: version, Outcome: "in_progress", Coverage: "incomplete", RequestedScope: text, Sources: sources, Diagnostics: []string{}, Redactions: count}
}
