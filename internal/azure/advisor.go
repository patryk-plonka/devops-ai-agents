package azure

import (
	"context"
	jsonv2 "encoding/json/v2"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/patryk-plonka/devops-ai-agents/internal/health"
)

const advisorListPath = "/providers/Microsoft.Advisor/recommendations"

type advisorPage struct {
	Value    []*advisorRecommendation `json:"value"`
	NextLink *string                  `json:"nextLink"`
}

// The record filter and pager share this projection of the cached list.
// Descriptions, actions and arbitrary metadata are outside the projection.
type advisorRecommendation struct {
	ID         string `json:"id"`
	Properties *struct {
		TypeID        string     `json:"recommendationTypeId"`
		Category      string     `json:"category"`
		Impact        string     `json:"impact"`
		Status        string     `json:"recommendationStatus"`
		LastRefreshed *time.Time `json:"lastRefreshed"`
		LastUpdated   *time.Time `json:"lastUpdated"`
		Resource      *struct {
			ID string `json:"resourceId"`
		} `json:"resourceMetadata"`
	} `json:"properties"`
}

// Use exact field names and reject duplicate members in each Advisor record.
func (r *advisorRecommendation) UnmarshalJSON(data []byte) error {
	type wireRecommendation advisorRecommendation
	return jsonv2.Unmarshal(data, (*wireRecommendation)(r))
}

func (c Collection) Advisor(ctx context.Context, subscription string, emit func(health.Page) error) error {
	options, credential, err := c.options(subscription, advisorRoute)
	if err != nil {
		return collectionFailure(err)
	}
	client, err := arm.NewClient("github.com/patryk-plonka/devops-ai-agents/internal/azure/advisor", "v1.0.0", credential, options)
	if err != nil {
		return fail("client_failure", "Cannot initialize Azure Advisor collection.", 1)
	}
	initial := client.Endpoint() + "/subscriptions/" + subscription + advisorListPath + "?api-version=" + health.AdvisorAPI
	pager := runtime.NewPager(runtime.PagingHandler[advisorPage]{
		More: func(page advisorPage) bool { return value(page.NextLink) != "" },
		Fetcher: func(ctx context.Context, page *advisorPage) (advisorPage, error) {
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
				return advisorPage{}, err
			}
			var result advisorPage
			if runtime.UnmarshalAsJSON(resp, &result) != nil {
				return advisorPage{}, fail("invalid_response", "Azure Advisor returned invalid JSON data.", 1)
			}
			return result, nil
		},
		Tracer: client.Tracer(),
	})
	return pages(ctx, c, subscription, advisorRoute, initial, pager, func(response advisorPage, rejected []health.RejectedRecordInput) (*string, error) {
		page := health.Page{CollectedAt: time.Now().UTC(), RejectedScopes: rejected}
		for _, recommendation := range response.Value {
			var input health.AdvisorInput
			if recommendation != nil {
				input.ID = recommendation.ID
				if p := recommendation.Properties; p != nil {
					input.TypeID, input.Category, input.Impact, input.Status = p.TypeID, p.Category, p.Impact, p.Status
					input.LastRefreshed, input.LastUpdated = p.LastRefreshed, p.LastUpdated
					if p.Resource != nil {
						input.TargetResourceID = p.Resource.ID
					}
				}
			}
			page.Advisor = append(page.Advisor, input)
		}
		return response.NextLink, emit(page)
	})
}
