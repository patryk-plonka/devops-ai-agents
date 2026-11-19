package health

import (
	"context"
	"time"
)

const InventoryAPI = "2025-04-01"
const ResourceHealthAPI = "2025-05-01"
const ServiceHealthAPI = "2025-05-01"
const MonitorAPI = "2019-03-01"

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
	Resources     []ResourceInput
	Health        []HealthInput
	ServiceHealth []ServiceHealthInput
	Monitor       []MonitorInput
	Advisor       []AdvisorInput
	CollectedAt   time.Time
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
	ID             string              `json:"id"`
	Source         string              `json:"source"`
	Operation      string              `json:"operation"`
	APIVersion     string              `json:"api_version"`
	ResourceID     string              `json:"resource_id,omitempty"`
	EntityID       string              `json:"entity_id,omitempty"`
	SubscriptionID string              `json:"subscription_id,omitempty"`
	ServiceHealth  *ServiceHealthFacts `json:"service_health,omitempty"`
	Monitor        *MonitorFacts       `json:"monitor,omitempty"`
	Advisor        *AdvisorFacts       `json:"advisor,omitempty"`
	Disposition    string              `json:"disposition,omitempty"`
	Limitations    []string            `json:"limitations,omitempty"`
	StatusID       string              `json:"status_id,omitempty"`
	StatusRole     string              `json:"status_role,omitempty"`
	EventID        string              `json:"event_id,omitempty"`
	EventIDs       []string            `json:"event_ids,omitempty"`
	Kind           string              `json:"kind"`
	State          string              `json:"state,omitempty"`
	CollectedAt    time.Time           `json:"collected_at"`
	ReportedAt     *time.Time          `json:"source_reported_at,omitempty"`
	OccurredAt     *time.Time          `json:"source_occurred_at,omitempty"`
	Resource       *ResourceFacts      `json:"resource,omitempty"`
}
type Observation struct {
	ResourceID  string   `json:"resource_id"`
	State       string   `json:"state"`
	InInventory bool     `json:"in_inventory"`
	EvidenceIDs []string `json:"evidence_ids"`
}
type Finding struct {
	ID             string   `json:"id"`
	ResourceID     string   `json:"resource_id,omitempty"`
	Source         string   `json:"source,omitempty"`
	EntityID       string   `json:"entity_id,omitempty"`
	SubscriptionID string   `json:"subscription_id,omitempty"`
	Category       string   `json:"category"`
	Title          string   `json:"title"`
	EvidenceIDs    []string `json:"evidence_ids"`
}
type Gap struct {
	Source      string   `json:"source"`
	EntityID    string   `json:"entity_id,omitempty"`
	ResourceID  string   `json:"resource_id,omitempty"`
	Reason      string   `json:"reason"`
	EvidenceIDs []string `json:"evidence_ids,omitempty"`
}
