package azure

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/patryk-plonka/devops-ai-agents/internal/health"
	"github.com/patryk-plonka/devops-ai-agents/internal/runstore"
)

const advisorInitialRequest = "https://management.azure.com/subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft.Advisor/recommendations?api-version=2026-10-01"
const advisorNextRequest = advisorInitialRequest + "&$top=10&$skiptoken=advisor-token-canary%2B%2F%3D%23opaque"

func advisorFixture(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile("testdata/advisor-" + name + ".json")
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func newAdvisorCheckpoint(t *testing.T) (*health.Run, *runstore.Store) {
	t.Helper()
	run := health.NewRun("test", "test")
	run.Scope = &health.Scope{SubscriptionID: selectedID}
	query := health.AdvisorQuery(time.Date(2026, 11, 18, 8, 0, 0, 0, time.UTC))
	source := run.Sources[advisorRoute]
	source.Query = &query
	run.Sources[advisorRoute] = source
	store, err := runstore.New(t.TempDir(), &run)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return &run, store
}

func readAdvisorCheckpoint(t *testing.T, store *runstore.Store) health.Run {
	t.Helper()
	raw, err := os.ReadFile(store.Path())
	if err != nil {
		t.Fatal(err)
	}
	for _, excluded := range []string{"canary", otherID, "nextLink", "synthetic-test-token"} {
		if bytes.Contains(raw, []byte(excluded)) {
			t.Fatalf("checkpoint leaked %q", excluded)
		}
	}
	var saved health.Run
	if err := json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	return saved
}

func TestAdvisorLiteralFixturesAndRequests(t *testing.T) {
	run, store := newAdvisorCheckpoint(t)
	var debug bytes.Buffer
	ConfigureDebugLogging(true, &debug)
	defer ConfigureDebugLogging(false, nil)
	calls := 0
	c := collection(t, func(req *http.Request) (*http.Response, error) {
		calls++
		want := advisorInitialRequest
		if calls == 2 {
			want = advisorNextRequest
		}
		u, _ := url.Parse(want)
		if calls > 2 || req.Method != http.MethodGet || req.Body != nil && req.Body != http.NoBody ||
			req.URL.Scheme != u.Scheme || req.URL.Host != u.Host || req.URL.Path != u.Path ||
			req.URL.Query().Encode() != u.Query().Encode() || req.Header.Get("Authorization") != "Bearer synthetic-test-token" {
			t.Fatalf("unexpected Advisor request: %s %s", req.Method, req.URL)
		}
		deadline, ok := req.Context().Deadline()
		if !ok || time.Until(deadline) <= 0 || time.Until(deadline) > OperationTimeout {
			t.Fatal("expected a bounded page request")
		}
		if calls == 2 {
			saved := readAdvisorCheckpoint(t, store)
			source := saved.Sources[advisorRoute]
			if source.Pages != 1 || source.FirstPageAt == nil || source.LastPageAt == nil || !source.FirstPageAt.Equal(*source.LastPageAt) || len(saved.Evidence) != 2 {
				t.Fatalf("first Advisor page not committed before continuation: %+v", source)
			}
			return response(200, advisorFixture(t, "empty")), nil
		}
		return response(200, advisorFixture(t, "recommendations")), nil
	})
	code, err := health.Check(context.Background(), run, []health.NamedCollector{{Name: advisorRoute, Collect: c.Advisor}}, store.Save)
	saved := readAdvisorCheckpoint(t, store)
	source := saved.Sources[advisorRoute]
	if err != nil || code != 0 || calls != 2 || source.Status != "completed" || source.Pages != 2 ||
		source.RejectedRecords != 0 || source.FirstPageAt == nil || source.LastPageAt == nil || !source.LastPageAt.After(*source.FirstPageAt) ||
		len(saved.Evidence) != 2 || len(saved.Findings) != 1 {
		t.Fatalf("collection: code %d error %v calls %d source %+v", code, err, calls, source)
	}
	const target = "/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/rg-test/providers/Microsoft.Compute/availabilitySets/set-test"
	for _, e := range saved.Evidence {
		p := e.Advisor
		if p == nil || e.APIVersion != "2026-10-01" || e.Operation != "Recommendations.List" || e.SubscriptionID != selectedID || !e.CollectedAt.Equal(*source.FirstPageAt) {
			t.Fatalf("lost evidence provenance: %+v", e)
		}
		switch e.EntityID {
		case target + "/providers/Microsoft.Advisor/recommendations/suggestion-one":
			refreshed := time.Date(2026, 11, 17, 20, 30, 43, 321640800, time.UTC)
			updated := time.Date(2026, 11, 16, 15, 20, 43, 321640800, time.UTC)
			if p.Status != "New" || p.Impact != "Medium" || p.Category != "HighAvailability" || e.ResourceID != target ||
				p.TypeID != "" || p.LastRefreshed == nil || !p.LastRefreshed.Equal(refreshed) || p.LastUpdated == nil || !p.LastUpdated.Equal(updated) {
				t.Fatalf("lost properties for resource extension: %+v", p)
			}
		case "/subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft.Advisor/recommendations/suggestion-two":
			if p.Status != "Postponed" || p.Impact != "Low" || p.Category != "Cost" || p.TypeID != "44444444-4444-4444-4444-444444444444" ||
				p.LastRefreshed != nil || p.LastUpdated != nil || e.ResourceID != "" || p.AttributionLimitation != "unknown_resource_attribution" {
				t.Fatalf("unexpected optional properties: %+v", p)
			}
		default:
			t.Fatalf("unexpected recommendation ID %q", e.EntityID)
		}
	}
	if strings.Contains(debug.String(), "canary") || strings.Contains(debug.String(), "synthetic-test-token") {
		t.Fatal("private wire contents in debug output")
	}
}

func TestAdvisorEmptyEnumeration(t *testing.T) {
	for _, tc := range []struct{ name, body string }{
		{"literal fixture", advisorFixture(t, "empty")},
		{"null continuation", `{"value":[],"nextLink":null}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			run, store := newAdvisorCheckpoint(t)
			calls := 0
			c := collection(t, func(*http.Request) (*http.Response, error) {
				calls++
				return response(200, tc.body), nil
			})
			code, err := health.Check(context.Background(), run, []health.NamedCollector{{Name: advisorRoute, Collect: c.Advisor}}, store.Save)
			saved := readAdvisorCheckpoint(t, store)
			source := saved.Sources[advisorRoute]
			if err != nil || code != 0 || calls != 1 || source.Status != "completed" || source.Pages != 1 || len(saved.Evidence) != 0 ||
				source.FirstPageAt == nil || source.LastPageAt == nil || !source.FirstPageAt.Equal(*source.LastPageAt) {
				t.Fatalf("empty Advisor collection: code %d error %v calls %d source %+v", code, err, calls, source)
			}
		})
	}
}

func TestAdvisorDuplicateEnvelopeFieldsFailWithoutCheckpoint(t *testing.T) {
	for _, tc := range []struct{ name, body string }{
		{"malformed list first", `{"value":null,"value":[]}`},
		{"malformed continuation first", `{"value":[],"nextLink":{"url":"envelope-canary"},"nextLink":null}`},
		{"both lists well typed", `{"value":[{}],"value":[]}`},
		{"escaped duplicate key", `{"value":[],"nextLink":null,"next\u004cink":""}`},
		{"case aliases", `{"value":[],"nextLink":"","Value":null,"NextLink":{"bad":"envelope-canary"}}`},
		{"case alias without canonical link", `{"value":[],"NextLink":"envelope-canary"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			run, store := newAdvisorCheckpoint(t)
			calls := 0
			c := collection(t, func(*http.Request) (*http.Response, error) {
				calls++
				return response(200, tc.body), nil
			})
			code, err := health.Check(context.Background(), run, []health.NamedCollector{{Name: advisorRoute, Collect: c.Advisor}}, store.Save)
			source := readAdvisorCheckpoint(t, store).Sources[advisorRoute]
			if err != nil || code != 1 || calls != 1 || source.Status != "failed" || source.Reason != "invalid_response" ||
				source.Pages != 0 || source.FirstPageAt != nil || source.LastPageAt != nil {
				t.Fatalf("invalid Advisor envelope: code %d error %v calls %d source %+v", code, err, calls, source)
			}
		})
	}
}

func TestAdvisorDuplicateRecordFieldsKeepValidSiblings(t *testing.T) {
	const prefix = "/subscriptions/" + selectedID + "/providers/Microsoft.Advisor/recommendations/"
	const validID = prefix + "valid"
	for _, tc := range []struct{ name, record string }{
		{"workflow", `{"id":"` + prefix + `ambiguous-canary","properties":{"recommendationStatus":"Dismissed","recommendationStatus":"New"}}`},
		{"instance ID", `{"id":"/subscriptions/` + otherID + `/providers/Microsoft.Advisor/recommendations/foreign-canary","id":"` + prefix + `ambiguous-canary","properties":{"recommendationStatus":"New"}}`},
		{"properties", `{"id":"` + prefix + `ambiguous-canary","properties":{"impact":"High"},"properties":{"recommendationStatus":"New"}}`},
		{"resource metadata", `{"id":"` + prefix + `ambiguous-canary","properties":{"recommendationStatus":"New","resourceMetadata":{"resourceId":"first","resourceId":"second"}}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			run, store := newAdvisorCheckpoint(t)
			c := collection(t, func(*http.Request) (*http.Response, error) {
				return response(200, `{"value":[`+tc.record+`,{"id":"`+validID+`","properties":{"recommendationStatus":"New"}}]}`), nil
			})
			code, err := health.Check(context.Background(), run, []health.NamedCollector{{Name: advisorRoute, Collect: c.Advisor}}, store.Save)
			saved := readAdvisorCheckpoint(t, store)
			source := saved.Sources[advisorRoute]
			if err != nil || code != 3 || source.Status != "partial" || source.Reason != "invalid_records" ||
				source.Pages != 1 || source.RejectedRecords != 1 || len(saved.Evidence) != 1 ||
				saved.Evidence[0].EntityID != validID || len(saved.Findings) != 1 || saved.Findings[0].EntityID != strings.ToLower(validID) {
				t.Fatalf("mixed Advisor page: code %d error %v source %+v evidence %d findings %d", code, err, source, len(saved.Evidence), len(saved.Findings))
			}
		})
	}
}

func TestAdvisorLateDecodedPageKeepsCommittedEvidence(t *testing.T) {
	run, store := newAdvisorCheckpoint(t)
	const firstID = "/subscriptions/" + selectedID + "/providers/Microsoft.Advisor/recommendations/first"
	const lateID = "/subscriptions/" + selectedID + "/providers/Microsoft.Advisor/recommendations/late"
	type decodedPage struct {
		next *string
		id   string
	}
	fetches := 0
	collect := func(ctx context.Context, subscription string, emit func(health.Page) error) error {
		pager := runtime.NewPager(runtime.PagingHandler[decodedPage]{
			More: func(page decodedPage) bool { return page.next != nil },
			Fetcher: func(ctx context.Context, previous *decodedPage) (decodedPage, error) {
				fetches++
				if previous == nil {
					next := advisorNextRequest
					return decodedPage{next: &next, id: firstID}, nil
				}
				// Simulate decoding ending after a successful HTTP response and
				// after the page context has expired.
				<-ctx.Done()
				return decodedPage{id: lateID}, nil
			},
		})
		return pages(ctx, Collection{timeout: 500 * time.Millisecond}, subscription, advisorRoute, advisorInitialRequest, pager, func(page decodedPage, _ []health.RejectedRecordInput) (*string, error) {
			return page.next, emit(health.Page{Advisor: []health.AdvisorInput{{ID: page.id, Status: "New"}}})
		})
	}
	code, err := health.Check(context.Background(), run, []health.NamedCollector{{Name: advisorRoute, Collect: collect}}, store.Save)
	saved := readAdvisorCheckpoint(t, store)
	source := saved.Sources[advisorRoute]
	if err != nil || code != 3 || fetches != 2 || source.Status != "partial" || source.Reason != "timeout" || source.Pages != 1 ||
		source.FirstPageAt == nil || source.LastPageAt == nil || !source.FirstPageAt.Equal(*source.LastPageAt) ||
		len(saved.Evidence) != 1 || saved.Evidence[0].EntityID != firstID || len(saved.Findings) != 1 {
		t.Fatalf("late page affected the checkpoint: code %d error %v fetches %d source %+v evidence %d", code, err, fetches, source, len(saved.Evidence))
	}
}

func TestAdvisorMixedPageSurvivesLaterFailures(t *testing.T) {
	for _, tc := range []struct {
		name, reason string
		calls, pages int
	}{
		{"complete", "invalid_records", 2, 2},
		{"denied", "access_denied", 2, 1},
		{"throttled", "throttled", 5, 1},
		{"timeout", "timeout", 2, 1},
		{"malformed", "invalid_response", 2, 1},
		{"foreign link", "unsafe_continuation", 1, 1},
		{"repeated link", "repeated_continuation", 2, 2},
		{"page ceiling", "page_limit", 1, 1},
		{"redirect", "read_boundary_rejected", 2, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var payload struct {
				Value    []json.RawMessage `json:"value"`
				NextLink string            `json:"nextLink"`
			}
			if err := json.Unmarshal([]byte(advisorFixture(t, "recommendations")), &payload); err != nil {
				t.Fatal(err)
			}
			const badID = "/subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft.Advisor/recommendations/invalid-canary"
			badRecords := []string{
				`null`, `42`, `{"id":42}`,
				`{"id":"` + badID + `","properties":[]}`,
				`{"id":"` + badID + `","properties":{"recommendationStatus":42}}`,
				`{"id":"` + badID + `","properties":{"lastRefreshed":"timestamp-canary"}}`,
				`{"id":"` + badID + `","properties":{"resourceMetadata":{"resourceId":[]}}}`,
			}
			for _, record := range badRecords {
				payload.Value = append(payload.Value, json.RawMessage(record))
			}
			if tc.name == "foreign link" {
				payload.NextLink = strings.Replace(payload.NextLink, selectedID, otherID, 1)
			}
			first, err := json.Marshal(payload)
			if err != nil {
				t.Fatal(err)
			}
			run, store := newAdvisorCheckpoint(t)
			calls := 0
			c := collection(t, func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return response(200, string(first)), nil
				}
				if calls > tc.calls {
					t.Fatal("unexpected request after continuation or source failure")
				}
				saved := readAdvisorCheckpoint(t, store)
				source := saved.Sources[advisorRoute]
				if len(saved.Evidence) != 2 || len(saved.Findings) != 1 || source.Pages != 1 || source.RejectedRecords != len(badRecords) {
					t.Fatalf("mixed page not committed before next attempt: %+v", source)
				}
				switch tc.name {
				case "denied", "throttled":
					status := 403
					if tc.name == "throttled" {
						status = 429
					}
					return response(status, `{"error":{"message":"response-canary"}}`), nil
				case "timeout":
					<-req.Context().Done()
					return nil, req.Context().Err()
				case "malformed":
					return response(200, advisorFixture(t, "malformed")), nil
				case "repeated link":
					link := "https://management.azure.com/subscriptions/" + selectedID + "/providers/microsoft.advisor/recommendations?$skipToken=advisor-token-canary%2b%2f%3d%23opaque&api-version=2026-10-01&$top=25"
					return response(200, `{"value":[],"nextLink":`+quoteJSON(link)+`}`), nil
				case "redirect":
					res := response(302, "")
					res.Header.Set("Location", advisorNextRequest)
					return res, nil
				default:
					return response(200, advisorFixture(t, "empty")), nil
				}
			})
			if tc.name == "page ceiling" {
				c.pageLimit = 1
			}
			if tc.name == "timeout" {
				c.timeout = time.Second
			}
			code, err := health.Check(context.Background(), run, []health.NamedCollector{{Name: advisorRoute, Collect: c.Advisor}}, store.Save)
			saved := readAdvisorCheckpoint(t, store)
			source := saved.Sources[advisorRoute]
			if err != nil || code != 3 || calls != tc.calls || source.Status != "partial" || source.Reason != tc.reason ||
				source.Pages != tc.pages || source.RejectedRecords != len(badRecords) || len(saved.Evidence) != 2 || len(saved.Findings) != 1 {
				t.Fatalf("code %d err %v calls %d source %+v", code, err, calls, source)
			}
		})
	}
}
