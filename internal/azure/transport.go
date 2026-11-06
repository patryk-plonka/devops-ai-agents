package azure

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const inventoryRoute = "inventory"
const healthRoute = "resource_health"

// ReadTransport validates each attempt before network access. Route narrows a
// collector to its own operation, including all continuation requests.
type ReadTransport struct {
	Subscription string
	Route        string
	Base         http.RoundTripper
}

func allowedRequest(req *http.Request, subscription, route string) bool {
	u := req.URL
	if u == nil || !uuid.MatchString(subscription) || req.Method != http.MethodGet ||
		u.Scheme != "https" || u.Host != "management.azure.com" || req.Host != "" && req.Host != u.Host ||
		u.User != nil || u.Fragment != "" || u.Opaque != "" || u.RawPath != "" || u.ForceQuery {
		return false
	}
	path, version := "/subscriptions/"+subscription, "2022-12-01"
	switch route {
	case inventoryRoute:
		path += "/resources"
		version = "2025-04-01"
	case healthRoute:
		path += "/providers/Microsoft.ResourceHealth/availabilityStatuses"
		version = "2025-05-01"
	case "":
	default:
		return false
	}
	if !strings.EqualFold(u.Path, path) {
		return false
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil || len(q["api-version"]) != 1 || q.Get("api-version") != version {
		return false
	}
	skipTokens := 0
	for key, values := range q {
		if key == "api-version" {
			continue
		}
		// Resource Health's service-generated continuations include a page size
		// and lowercase token key; older links also use $skipToken.
		if route == healthRoute && key == "$top" {
			if len(values) != 1 {
				return false
			}
			top, err := strconv.ParseUint(values[0], 10, 31)
			if err != nil || top == 0 {
				return false
			}
			continue
		}
		isToken := key == "$skiptoken" || route == healthRoute && key == "$skipToken"
		if route == "" || !isToken || len(values) != 1 || values[0] == "" {
			return false
		}
		skipTokens++
	}
	return skipTokens <= 1 // Reject ambiguous links supplying both token aliases.
}
func (t ReadTransport) Do(req *http.Request) (*http.Response, error) {
	if err := req.Context().Err(); err != nil {
		return nil, err
	}
	if !allowedRequest(req, t.Subscription, t.Route) {
		return nil, &boundaryError{"read_boundary_rejected"}
	}
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	client := http.Client{Transport: base, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, context.DeadlineExceeded
		}
		if errors.Is(err, context.Canceled) {
			return nil, context.Canceled
		}
		return nil, errors.New("ARM transport request failed")
	}
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		_ = resp.Body.Close()
		return nil, &boundaryError{"read_boundary_rejected"}
	}
	return resp, nil
}
