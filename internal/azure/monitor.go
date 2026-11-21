package azure

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/patryk-plonka/devops-ai-agents/internal/health"
)

// Private projections deliberately omit context, egress configuration, rule
// links, descriptions and personal identifiers, even if a service returns them.
type monitorPage struct {
	Value    []*monitorAlert `json:"value"`
	NextLink *string         `json:"nextLink"`
}
type monitorAlert struct {
	ID         string `json:"id"`
	Properties *struct {
		Essentials *monitorEssentials `json:"essentials"`
	} `json:"properties"`
}
type monitorEssentials struct {
	TargetResource                   string     `json:"targetResource"`
	SignalType                       string     `json:"signalType"`
	MonitorCondition                 string     `json:"monitorCondition"`
	AlertState                       string     `json:"alertState"`
	Severity                         string     `json:"severity"`
	StartDateTime                    *time.Time `json:"startDateTime"`
	LastModifiedDateTime             *time.Time `json:"lastModifiedDateTime"`
	LastModifiedTimeByMonitorService *time.Time `json:"lastModifiedTimeByMonitorService"`
	ActionStatus                     *struct {
		IsSuppressed *bool `json:"isSuppressed"`
	} `json:"actionStatus"`
}

func (c Collection) AzureMonitor(ctx context.Context, subscription string, emit func(health.Page) error) error {
	options, credential, err := c.options(subscription, monitorRoute)
	if err != nil {
		return collectionFailure(err)
	}
	// Telemetry requires a semantic adapter version, independent of CLI builds.
	client, err := arm.NewClient("github.com/patryk-plonka/devops-ai-agents/internal/azure/monitor", "v1.0.0", credential, options)
	if err != nil {
		return fail("client_failure", "Cannot initialize Azure Monitor collection.", 1)
	}
	q := url.Values{"api-version": {"2019-03-01"}}
	for key, value := range monitorOptions() {
		q.Set(key, value)
	}
	initial := client.Endpoint() + "/subscriptions/" + subscription + "/providers/Microsoft.AlertsManagement/alerts?" + q.Encode()
	pager := runtime.NewPager(runtime.PagingHandler[monitorPage]{
		More: func(page monitorPage) bool { return page.NextLink != nil && *page.NextLink != "" },
		Fetcher: func(ctx context.Context, page *monitorPage) (monitorPage, error) {
			next := ""
			if page != nil {
				next = value(page.NextLink)
			}
			resp, err := runtime.FetcherForNextLink(ctx, client.Pipeline(), next, func(ctx context.Context) (*policy.Request, error) {
				req, err := runtime.NewRequest(ctx, http.MethodGet, initial)
				if err == nil {
					req.Raw().Header.Set("Accept", "application/json")
				}
				return req, err
			}, nil)
			if err != nil {
				return monitorPage{}, err
			}
			var result monitorPage
			if runtime.UnmarshalAsJSON(resp, &result) != nil {
				return monitorPage{}, fail("invalid_response", "Azure Monitor returned invalid JSON data.", 1)
			}
			return result, nil
		},
		Tracer: client.Tracer(),
	})
	return pages(ctx, c, subscription, monitorRoute, initial, pager, func(response monitorPage, rejected []health.RejectedRecordInput) (*string, error) {
		if response.Value == nil {
			return nil, fail("invalid_response", "Azure Monitor response has no alert list.", 1)
		}
		page := health.Page{CollectedAt: time.Now().UTC(), RejectedScopes: rejected}
		for _, alert := range response.Value {
			input := health.MonitorInput{}
			if alert != nil {
				input.ID = alert.ID
				if alert.Properties != nil && alert.Properties.Essentials != nil {
					p := alert.Properties.Essentials
					input.PropertiesPresent = true
					input.TargetResourceID, input.SignalType, input.Condition = p.TargetResource, p.SignalType, p.MonitorCondition
					input.WorkflowState, input.Severity = p.AlertState, p.Severity
					input.StartTime, input.LastModifiedTime, input.LastModifiedTimeByMonitorService = p.StartDateTime, p.LastModifiedDateTime, p.LastModifiedTimeByMonitorService
					if p.ActionStatus != nil {
						input.IsSuppressed = p.ActionStatus.IsSuppressed
					}
				}
			}
			page.Monitor = append(page.Monitor, input)
		}
		return response.NextLink, emit(page)
	})
}
