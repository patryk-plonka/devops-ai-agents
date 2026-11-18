package azure

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcehealth/armresourcehealth"
	"github.com/patryk-plonka/devops-ai-agents/internal/health"
)

func (c Collection) ServiceHealth(ctx context.Context, subscription string, emit func(health.Page) error) error {
	// Capture and validate the run policy before credentials or reads. Never derive
	// a moving window from wall-clock time while traversing pages.
	query := c.ServiceHealthQuery
	expected := health.ServiceHealthQuery(query.ReferenceAt)
	if query.ReferenceAt.IsZero() || query.Cutoff == nil || !query.Cutoff.Equal(*expected.Cutoff) || query.QueryStartTime != expected.QueryStartTime {
		return fail("invalid_query_policy", "Service Health requires a fixed run query policy.", 1)
	}
	options, credential, err := c.options(subscription, serviceHealthRoute)
	if err != nil {
		return collectionFailure(err)
	}
	client, err := armresourcehealth.NewEventsClient(subscription, credential, options)
	if err != nil {
		return fail("client_failure", "Cannot initialize Service Health collection.", 1)
	}
	date := query.QueryStartTime
	initial := "https://management.azure.com/subscriptions/" + subscription + "/providers/Microsoft.ResourceHealth/events?api-version=2025-05-01&queryStartTime=" + date
	pager := client.NewListBySubscriptionIDPager(&armresourcehealth.EventsClientListBySubscriptionIDOptions{QueryStartTime: &date})
	return pages(ctx, c, subscription, serviceHealthRoute, initial, pager, func(response armresourcehealth.EventsClientListBySubscriptionIDResponse) (*string, error) {
		if response.Value == nil {
			return nil, fail("invalid_response", "Service Health response has no event list.", 1)
		}
		page := health.Page{CollectedAt: time.Now().UTC()}
		for _, event := range response.Value {
			input := health.ServiceHealthInput{}
			if event != nil {
				input.ID = value(event.ID)
				if p := event.Properties; p != nil {
					input.PropertiesPresent = true
					input.EventType, input.Status = string(value(p.EventType)), string(value(p.Status))
					input.Level, input.EventSource = string(value(p.EventLevel)), string(value(p.EventSource))
					input.LastUpdateTime, input.ImpactStartTime, input.ImpactMitigationTime = p.LastUpdateTime, p.ImpactStartTime, p.ImpactMitigationTime
					input.Sensitive = p.IsEventSensitive
					for _, impact := range p.Impact {
						accepted := health.ServiceImpactInput{}
						if impact != nil {
							accepted.Service = value(impact.ImpactedService)
							for _, region := range impact.ImpactedRegions {
								facts := health.ServiceRegionInput{}
								if region != nil {
									facts.Name, facts.Status, facts.LastUpdateTime = value(region.ImpactedRegion), string(value(region.Status)), region.LastUpdateTime
									for _, id := range region.ImpactedSubscriptions {
										facts.ImpactedSubscriptions = append(facts.ImpactedSubscriptions, value(id))
									}
								}
								accepted.Regions = append(accepted.Regions, facts)
							}
						}
						input.Impact = append(input.Impact, accepted)
					}
				}
			}
			page.ServiceHealth = append(page.ServiceHealth, input)
		}
		return response.NextLink, emit(page)
	})
}
