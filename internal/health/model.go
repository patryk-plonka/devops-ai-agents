// Package health defines the normalized local run record.
package health

import (
	"fmt"
	"time"
)

type Scope struct {
	SubscriptionID string `json:"subscription_id"`
	Name           string `json:"name"`
	TenantID       string `json:"tenant_id"`
	Cloud          string `json:"cloud"`
}

type Source struct {
	Status              string            `json:"status"`
	Reason              string            `json:"reason,omitempty"`
	Pages               int               `json:"pages"`
	RejectedRecords     int               `json:"rejected_records"`
	ExcludedRecords     int               `json:"excluded_records,omitempty"`
	UnclassifiedRecords int               `json:"unclassified_records,omitempty"`
	Query               *SourceQuery      `json:"query,omitempty"`
	FirstPageAt         *time.Time        `json:"first_page_at,omitempty"`
	LastPageAt          *time.Time        `json:"last_page_at,omitempty"`
	EnumerationComplete bool              `json:"enumeration_complete,omitempty"`
	RejectedScopes      []ScopedRejection `json:"rejected_scopes,omitempty"`
}

// ScopedRejection records only verified attribution for a discarded row.
type ScopedRejection struct {
	ResourceID string `json:"resource_id"`
	EntityID   string `json:"entity_id,omitempty"`
}

func (s Source) finishedWithRejections() bool {
	return s.EnumerationComplete && s.Status == "partial" && s.Reason == "invalid_records" && s.RejectedRecords > 0
}

// SourceQuery records fixed query policy, never continuation tokens or URLs.
type SourceQuery struct {
	ReferenceAt      time.Time  `json:"reference_at"`
	Cutoff           *time.Time `json:"cutoff,omitempty"`
	QueryStartTime   string     `json:"query_start_time,omitempty"`
	TimeRange        string     `json:"time_range,omitempty"`
	Operation        string     `json:"operation,omitempty"`
	APIVersion       string     `json:"api_version,omitempty"`
	SubscriptionID   string     `json:"subscription_id,omitempty"`
	MonitorCondition string     `json:"monitor_condition,omitempty"`
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
	SchemaVersion       int                  `json:"schema_version"`
	ID                  string               `json:"run_id"`
	Revision            int                  `json:"revision"`
	ToolVersion         string               `json:"tool_version,omitempty"`
	StartedAt           time.Time            `json:"started_at"`
	UpdatedAt           time.Time            `json:"updated_at"`
	EndedAt             *time.Time           `json:"ended_at,omitempty"`
	Outcome             string               `json:"outcome"`
	Coverage            string               `json:"coverage"`
	RequestedScope      string               `json:"requested_scope"`
	Scope               *Scope               `json:"scope,omitempty"`
	Sources             map[string]Source    `json:"sources"`
	Diagnostics         []string             `json:"diagnostics"`
	Redactions          int                  `json:"redactions"`
	Resources           []Resource           `json:"resources"`
	Evidence            []Evidence           `json:"evidence"`
	Observations        []Observation        `json:"observations"`
	Findings            []Finding            `json:"findings"`
	AdvisorInstances    []AdvisorInstance    `json:"advisor_instances,omitempty"`
	Gaps                []Gap                `json:"coverage_gaps"`
	ResourceAssessments []ResourceAssessment `json:"resource_assessments,omitempty"`
	AssessmentSummary   *AssessmentSummary   `json:"assessment_summary,omitempty"`
}

const AssessmentSchemaVersion = 4

type AssessmentSummary struct {
	OK      int `json:"ok"`
	Finding int `json:"finding"`
	Unknown int `json:"unknown"`
}

// ResourceCounts formats the disjoint counts shared by CLI and Markdown.
func (s AssessmentSummary) ResourceCounts() string {
	noun := "resources"
	if s.OK == 1 {
		noun = "resource"
	}
	return fmt.Sprintf("%d %s OK, %d with findings, %d unknown", s.OK, noun, s.Finding, s.Unknown)
}

type ResourceAssessment struct {
	ResourceID           string            `json:"resource_id"`
	Disposition          string            `json:"disposition"`
	Checks               []AssessmentCheck `json:"checks"`
	BlockingReasons      []Gap             `json:"blocking_reasons,omitempty"`
	FindingIDs           []string          `json:"finding_ids,omitempty"`
	HistoricalFindingIDs []string          `json:"historical_finding_ids,omitempty"`
	PostponedEvidenceIDs []string          `json:"postponed_evidence_ids,omitempty"`
}

// SourceRef identifies the named entry in this snapshot's sources object.
type AssessmentCheck struct {
	SourceRef   string   `json:"source_ref"`
	EvidenceIDs []string `json:"evidence_ids,omitempty"`
}

func NewRun(requested, version string) Run {
	text, count := SafeText(requested)
	sources := make(map[string]Source)
	for _, name := range []string{"inventory", "resource_health", "service_health", "azure_monitor", "advisor"} {
		sources[name] = Source{Status: "not_checked", Reason: "not_attempted"}
	}
	return Run{SchemaVersion: AssessmentSchemaVersion, ToolVersion: version, Outcome: "in_progress", Coverage: "incomplete", RequestedScope: text, Sources: sources, Diagnostics: []string{}, Redactions: count}
}
