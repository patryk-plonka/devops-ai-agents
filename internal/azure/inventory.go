package azure

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v4"
	"github.com/patryk-plonka/devops-ai-agents/internal/health"
)

func value[T any](v *T) T {
	if v != nil {
		return *v
	}
	var zero T
	return zero
}

func (c Collection) Inventory(ctx context.Context, subscription string, emit func(health.Page) error) error {
	options, credential, err := c.options(subscription, inventoryRoute)
	if err != nil {
		return collectionFailure(err)
	}
	client, err := armresources.NewClient(subscription, credential, options)
	if err != nil {
		return fail("client_failure", "Cannot initialize inventory collection.", 1)
	}
	return pages(ctx, c, subscription, inventoryRoute, client.NewListPager(nil), func(response armresources.ClientListResponse) (*string, error) {
		if response.Value == nil {
			return nil, fail("invalid_response", "Inventory response has no resource list.", 1)
		}
		page := health.Page{CollectedAt: time.Now().UTC()}
		for _, resource := range response.Value {
			if resource == nil {
				page.Resources = append(page.Resources, health.ResourceInput{})
				continue
			}
			page.Resources = append(page.Resources, health.ResourceInput{ID: value(resource.ID), Name: value(resource.Name), Type: value(resource.Type), Location: value(resource.Location), Tags: resource.Tags})
		}
		return response.NextLink, emit(page)
	})
}
