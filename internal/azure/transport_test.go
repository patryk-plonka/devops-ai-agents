package azure

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func response(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
}

func TestTransportRejectsUnsafeRequestsBeforeSending(t *testing.T) {
	baseURL := "https://management.azure.com/subscriptions/" + selectedID + "?api-version=2022-12-01"
	calls := 0
	transport := ReadTransport{Subscription: selectedID, Base: roundTrip(func(*http.Request) (*http.Response, error) { calls++; return response(200, `{}`), nil })}
	for _, tc := range []struct{ method, url string }{
		{"GET", baseURL}, {"POST", baseURL}, {"PUT", baseURL}, {"PATCH", baseURL}, {"DELETE", baseURL},
		{"GET", strings.Replace(baseURL, selectedID, otherID, 1)},
		{"GET", strings.Replace(baseURL, "https:", "http:", 1)},
		{"GET", strings.Replace(baseURL, "management.azure.com", "evil.example", 1)},
		{"GET", strings.Replace(baseURL, "management.azure.com", "management.azure.com:443", 1)},
		{"GET", strings.Replace(baseURL, "/subscriptions/", "/subscriptions/%2e%2e/", 1)},
		{"GET", strings.Replace(baseURL, "?", "/providers/Microsoft.Compute/register?", 1)},
		{"GET", baseURL + "&extra=secret"}, {"GET", baseURL + "#fragment"}, {"GET", baseURL + "&api-version=2022-12-01"},
	} {
		t.Run(tc.method+tc.url, func(t *testing.T) {
			before := calls
			r, err := http.NewRequest(tc.method, tc.url, nil)
			if err != nil {
				t.Fatal(err)
			}
			res, err := transport.Do(r)
			if res != nil {
				_ = res.Body.Close()
			}
			allowed := tc.method == "GET" && tc.url == baseURL
			if allowed && (err != nil || calls != before+1) {
				t.Fatalf("allowed request failed: %v", err)
			}
			if !allowed && (err == nil || calls != before) {
				t.Fatal("unsafe request reached transport")
			}
		})
	}
}

func TestTransportNeverFollowsRedirects(t *testing.T) {
	calls := 0
	tr := ReadTransport{Subscription: selectedID, Base: roundTrip(func(*http.Request) (*http.Response, error) {
		calls++
		r := response(302, "")
		r.Header.Set("Location", "https://evil.example/token")
		return r, nil
	})}
	r, _ := http.NewRequest("GET", "https://management.azure.com/subscriptions/"+selectedID+"?api-version=2022-12-01", nil)
	if _, err := tr.Do(r); err == nil || calls != 1 {
		t.Fatalf("redirect followed: calls %d, err %v", calls, err)
	}
}

func TestCollectionTransportAndContinuationBoundaries(t *testing.T) {
	for _, route := range []string{inventoryRoute, healthRoute} {
		t.Run(route, func(t *testing.T) {
			calls := 0
			tr := ReadTransport{Subscription: selectedID, Route: route, Base: roundTrip(func(*http.Request) (*http.Response, error) { calls++; return response(200, `{}`), nil })}
			valid := listURL(route, "opaque%2Ftoken")
			other := inventoryRoute
			if route == inventoryRoute {
				other = healthRoute
			}
			for _, tc := range []struct {
				method, url string
				allowed     bool
			}{
				{"GET", listURL(route, ""), true}, {"GET", valid, true},
				{"POST", valid, false}, {"PUT", valid, false}, {"PATCH", valid, false}, {"DELETE", valid, false},
				{"GET", strings.Replace(valid, selectedID, otherID, 1), false},
				{"GET", listURL(other, "page"), false},
				{"GET", strings.Replace(valid, "https:", "http:", 1), false},
				{"GET", strings.Replace(valid, "management.azure.com", "evil.example", 1), false},
				{"GET", strings.Replace(valid, "management.azure.com", "user:secret@management.azure.com", 1), false},
				{"GET", strings.Replace(valid, "management.azure.com", "management.azure.com:443", 1), false},
				{"GET", strings.Replace(valid, "/subscriptions/", "/%73ubscriptions/", 1), false},
				{"GET", strings.Replace(valid, "/subscriptions/", "/subscriptions/../", 1), false},
				{"GET", valid + "#fragment", false},
				{"GET", valid + "&api-version=2025-04-01", false},
				{"GET", valid + "&$filter=omit-resources", false},
				{"GET", valid + "&$expand=recommendedactions", false},
				{"GET", valid + "&token=canary", false},
				{"GET", valid + "&broken=%zz", false},
				{"GET", strings.Replace(valid, "2025-", "2024-", 1), false},
				{"GET", strings.Replace(valid, "?", "/providers/Microsoft.Compute/register?", 1), false},
			} {
				before := calls
				req, err := http.NewRequest(tc.method, tc.url, nil)
				if err != nil {
					t.Fatal(err)
				}
				res, err := tr.Do(req)
				if res != nil {
					_ = res.Body.Close()
				}
				if tc.allowed && (err != nil || calls != before+1) {
					t.Errorf("valid request failed %s %v", tc.url, err)
				}
				if !tc.allowed && (err == nil || calls != before) {
					t.Errorf("unsafe request sent %s", tc.url)
				}
				if tc.method == "GET" {
					_, continuationErr := continuation(tc.url, selectedID, route)
					if (continuationErr == nil) != tc.allowed {
						t.Errorf("continuation validation disagrees %s %v", tc.url, continuationErr)
					}
				}
			}
			tr.Base = roundTrip(func(*http.Request) (*http.Response, error) {
				calls++
				res := response(307, "redirect-body-canary")
				res.Header.Set("Location", listURL(other, "escape"))
				return res, nil
			})
			before := calls
			req, _ := http.NewRequest("GET", valid, nil)
			if res, err := tr.Do(req); res != nil || err == nil || calls != before+1 {
				t.Fatal("redirect escaped boundary")
			}
		})
	}
}

func TestResourceHealthServiceContinuationShape(t *testing.T) {
	// Independent service-shaped fixture: the live API returns lowercase
	// $skiptoken and $top even though synthetic SDK-style links used $skipToken.
	valid := "https://management.azure.com/subscriptions/" + selectedID + "/providers/Microsoft.ResourceHealth/availabilityStatuses?api-version=2025-05-01&$skiptoken=synthetic-next&$top=10"
	for _, tc := range []struct {
		name, link string
		allowed    bool
	}{
		{"service pagination", valid, true},
		{"camel-case token", strings.Replace(valid, "$skiptoken", "$skipToken", 1), true},
		{"encoded keys", strings.ReplaceAll(valid, "$", "%24"), true},
		{"both token aliases", valid + "&$skipToken=other", false},
		{"duplicate token", valid + "&$skiptoken=other", false},
		{"empty token", strings.Replace(valid, "synthetic-next", "", 1), false},
		{"duplicate page size", valid + "&$top=20", false},
		{"zero page size", strings.Replace(valid, "$top=10", "$top=0", 1), false},
		{"negative page size", strings.Replace(valid, "$top=10", "$top=-1", 1), false},
		{"empty page size", strings.Replace(valid, "$top=10", "$top=", 1), false},
		{"fractional page size", strings.Replace(valid, "$top=10", "$top=1.5", 1), false},
		{"invalid page size", strings.Replace(valid, "$top=10", "$top=omit", 1), false},
		{"oversized page size", strings.Replace(valid, "$top=10", "$top=2147483648", 1), false},
		{"filter", valid + "&$filter=omit-resources", false},
		{"expand", valid + "&$expand=recommendedactions", false},
		{"wrong host", strings.Replace(valid, "management.azure.com", "evil.example", 1), false},
		{"wrong subscription", strings.Replace(valid, selectedID, otherID, 1), false},
		{"wrong API version", strings.Replace(valid, "2025-05-01", "2024-02-01", 1), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			tr := ReadTransport{Subscription: selectedID, Route: healthRoute, Base: roundTrip(func(*http.Request) (*http.Response, error) {
				calls++
				return response(200, `{}`), nil
			})}
			_, validationErr := continuation(tc.link, selectedID, healthRoute)
			req, err := http.NewRequest(http.MethodGet, tc.link, nil)
			if err != nil {
				t.Fatal(err)
			}
			res, sendErr := tr.Do(req)
			if res != nil {
				_ = res.Body.Close()
			}
			if tc.allowed && (validationErr != nil || sendErr != nil || calls != 1) {
				t.Fatalf("valid service pagination rejected: %v %v", validationErr, sendErr)
			}
			if !tc.allowed && (validationErr == nil || sendErr == nil || calls != 0) {
				t.Fatal("unsafe pagination reached the wire")
			}
		})
	}
	lower, err := continuation(valid, selectedID, healthRoute)
	if err != nil {
		t.Fatal(err)
	}
	upper, err := continuation(strings.Replace(valid, "$skiptoken", "$skipToken", 1), selectedID, healthRoute)
	if err != nil || lower != upper {
		t.Fatal("token alias bypasses repeated-continuation detection")
	}
	inventory := "https://management.azure.com/subscriptions/" + selectedID + "/resources?api-version=2025-04-01&$skiptoken=synthetic-next&$top=10"
	if _, err := continuation(inventory, selectedID, inventoryRoute); err == nil {
		t.Fatal("Resource Health page-size allowance leaked into inventory")
	}
}
