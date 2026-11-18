package azure

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const inventoryRoute = "inventory"
const healthRoute = "resource_health"
const serviceHealthRoute = "service_health"
const monitorRoute = "azure_monitor"

// ReadTransport validates each attempt before network access. Route narrows a
// collector to its own operation, including all continuation requests.
type ReadTransport struct {
	Subscription   string
	Route          string
	QueryStartTime string
	Base           http.RoundTripper
}

func allowedRequest(req *http.Request, subscription, route, queryStartTime string) bool {
	u := req.URL
	if u == nil || !uuid.MatchString(subscription) || req.Method != http.MethodGet ||
		u.Scheme != "https" || (u.Host != "management.azure.com" && (route != monitorRoute || u.Host != "management.azure.com:443")) || req.Host != "" && req.Host != u.Host ||
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
	case serviceHealthRoute:
		path += "/providers/Microsoft.ResourceHealth/events"
		version = "2025-05-01"
	case monitorRoute:
		path += "/providers/Microsoft.AlertsManagement/alerts"
		version = "2019-03-01"
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
	if route == serviceHealthRoute || route == monitorRoute {
		if req.Body != nil && req.Body != http.NoBody {
			return false
		}
		return allowedOperationalQuery(q, route, queryStartTime)
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
	if !allowedRequest(req, t.Subscription, t.Route, t.QueryStartTime) {
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

// Initial queries are fixed. Continuations may omit initial options, but cannot
// change them or add filters. Tokens are only consumed in memory.
func allowedOperationalQuery(q url.Values, route, queryStartTime string) bool {
	for _, values := range q {
		if len(values) != 1 {
			return false
		}
	}
	if route == serviceHealthRoute {
		date, err := time.Parse("1/2/2006", queryStartTime)
		if err != nil || date.Format("1/2/2006") != queryStartTime {
			return false
		}
		for key, values := range q {
			switch key {
			case "api-version":
			case "queryStartTime":
				if values[0] != queryStartTime {
					return false
				}
			case "$skipToken":
				if values[0] == "" {
					return false
				}
			default:
				return false
			}
		}
		return q.Has("$skipToken") || q.Get("queryStartTime") == queryStartTime
	}
	fixed := monitorOptions()
	for key, values := range q {
		switch key {
		case "api-version":
		case "ctoken":
			if values[0] == "" {
				return false
			}
		case "pageCount":
			count, err := strconv.ParseUint(values[0], 10, 16)
			if err != nil || count < 1 || count > 250 {
				return false
			}
		case "sortBy":
			if values[0] != "lastModifiedDateTime" {
				return false
			}
		case "sortOrder":
			if values[0] != "desc" {
				return false
			}
		default:
			if want, ok := fixed[key]; !ok || values[0] != want {
				return false
			}
		}
	}
	if q.Has("ctoken") {
		return true
	}
	if len(q) != len(fixed)+1 {
		return false
	}
	for key, want := range fixed {
		if q.Get(key) != want {
			return false
		}
	}
	return true
}

func monitorOptions() map[string]string {
	return map[string]string{"timeRange": "30d", "monitorCondition": "Fired", "includeContext": "false", "includeEgressConfig": "false", "pageCount": "100"}
}
