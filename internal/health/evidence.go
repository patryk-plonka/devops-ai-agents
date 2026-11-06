package health

import (
	"context"
	"time"
)

const InventoryAPI = "2025-04-01"
const ResourceHealthAPI = "2025-05-01"

// Inputs whitelist source facts. SDK objects and arbitrary descriptions never
// cross this boundary. NormalizePage filters these facts before checkpointing.
type ResourceInput struct {
	ID, Name, Type, Location string
	Tags                     map[string]*string
}
type HealthInput struct {
	StatusID          string
	PropertiesPresent bool
	State             *string
	EventID           string
	EventIDs          []string
	EventPresent      bool
	ReportedAt        *time.Time
	OccurredAt        *time.Time
}
type Page struct {
	Resources   []ResourceInput
	Health      []HealthInput
	CollectedAt time.Time
}
type Collector func(context.Context, string, func(Page) error) error

type ResourceFacts struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Type          string            `json:"type"`
	ResourceGroup string            `json:"resource_group,omitempty"`
	Location      string            `json:"location,omitempty"`
	Tags          map[string]string `json:"tags"`
}
type Resource struct {
	ResourceFacts
	EvidenceIDs []string `json:"evidence_ids"`
}
type Evidence struct {
	ID          string         `json:"id"`
	Source      string         `json:"source"`
	Operation   string         `json:"operation"`
	APIVersion  string         `json:"api_version"`
	ResourceID  string         `json:"resource_id"`
	StatusID    string         `json:"status_id,omitempty"`
	StatusRole  string         `json:"status_role,omitempty"`
	EventID     string         `json:"event_id,omitempty"`
	EventIDs    []string       `json:"event_ids,omitempty"`
	Kind        string         `json:"kind"`
	State       string         `json:"state,omitempty"`
	CollectedAt time.Time      `json:"collected_at"`
	ReportedAt  *time.Time     `json:"source_reported_at,omitempty"`
	OccurredAt  *time.Time     `json:"source_occurred_at,omitempty"`
	Resource    *ResourceFacts `json:"resource,omitempty"`
}
type Observation struct {
	ResourceID  string   `json:"resource_id"`
	State       string   `json:"state"`
	InInventory bool     `json:"in_inventory"`
	EvidenceIDs []string `json:"evidence_ids"`
}
type Finding struct {
	ID          string   `json:"id"`
	ResourceID  string   `json:"resource_id"`
	Category    string   `json:"category"`
	Title       string   `json:"title"`
	EvidenceIDs []string `json:"evidence_ids"`
}
type Gap struct {
	Source      string   `json:"source"`
	ResourceID  string   `json:"resource_id,omitempty"`
	Reason      string   `json:"reason"`
	EvidenceIDs []string `json:"evidence_ids,omitempty"`
}
