package azure

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcehealth/armresourcehealth"
	"github.com/patryk-plonka/devops-ai-agents/internal/health"
)

func (c Collection) ResourceHealth(ctx context.Context, subscription string, emit func(health.Page) error) error {
	options, credential, err := c.options(subscription, healthRoute)
	if err != nil {
		return collectionFailure(err)
	}
	client, err := armresourcehealth.NewAvailabilityStatusesClient(subscription, credential, options)
	if err != nil {
		return fail("client_failure", "Cannot initialize Resource Health collection.", 1)
	}
	return pages(ctx, c, subscription, healthRoute, client.NewListBySubscriptionIDPager(nil), func(response armresourcehealth.AvailabilityStatusesClientListBySubscriptionIDResponse) (*string, error) {
		if response.Value == nil {
			return nil, fail("invalid_response", "Resource Health response has no observation list.", 1)
		}
		page := health.Page{CollectedAt: time.Now().UTC()}
		for _, status := range response.Value {
			input := health.HealthInput{}
			if status != nil {
				input.StatusID = value(status.ID)
				if p := status.Properties; p != nil {
					input.PropertiesPresent = true
					if p.AvailabilityState != nil {
						s := string(*p.AvailabilityState)
						input.State = &s
					}
					input.EventID = value(p.HealthEventID)
					input.EventPresent = value(p.HealthEventType) != "" || value(p.HealthEventCause) != "" || value(p.HealthEventCategory) != "" || len(p.ServiceImpactingEvents) > 0
					input.ReportedAt = p.ReportedTime
					input.OccurredAt = p.OccurredTime
					for _, event := range p.ServiceImpactingEvents {
						if event != nil {
							input.EventIDs = append(input.EventIDs, value(event.CorrelationID))
						}
					}
				}
			}
			page.Health = append(page.Health, input)
		}
		return response.NextLink, emit(page)
	})
}
