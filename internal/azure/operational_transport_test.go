package azure

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/patryk-plonka/devops-ai-agents/internal/health"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestOperationalRequestPolicies(t *testing.T) {
	for _, route := range []string{serviceHealthRoute, monitorRoute} {
		t.Run(route, func(t *testing.T) {
			initial, next := listURL(route, ""), listURL(route, "opaque%2B%2F%3D%23token-canary")
			cases := []struct {
				name, link string
				allowed    bool
			}{
				{"initial", initial, true}, {"continuation", next, true},
				{"wrong subscription", strings.Replace(next, selectedID, otherID, 1), false},
				{"wrong host", strings.Replace(next, "management.azure.com", "evil.example", 1), false},
				{"HTTP", strings.Replace(next, "https:", "http:", 1), false},
				{"userinfo", strings.Replace(next, "management.azure.com", "user:token-canary@management.azure.com", 1), false},
				{"port 444", strings.Replace(next, "management.azure.com", "management.azure.com:444", 1), false},
				{"encoded path", strings.Replace(next, "/subscriptions/", "/%73ubscriptions/", 1), false},
				{"dot path", strings.Replace(next, "/subscriptions/", "/subscriptions/../", 1), false},
				{"fragment", next + "#token-canary", false},
				{"duplicate version", next + "&api-version=2019-03-01", false},
				{"filter", next + "&$filter=token-canary", false}, {"expand", next + "&$expand=token-canary", false},
				{"other operation", listURL(inventoryRoute, "next"), false},
				{"global events", "https://management.azure.com/providers/Microsoft.ResourceHealth/events?api-version=2025-05-01&$skipToken=next", false},
				{"tenant events", "https://management.azure.com/providers/Microsoft.ResourceHealth/events?api-version=2025-05-01", false},
				{"registration", strings.Replace(next, "?", "/register?", 1), false},
				{"unparseable query", next + "&broken=%zz", false},
			}
			if route == serviceHealthRoute {
				cases = append(cases, []struct {
					name, link string
					allowed    bool
				}{
					{"matching date", next + "&queryStartTime=10%2F16%2F2026", true},
					{"changed date", next + "&queryStartTime=10/17/2026", false},
					{"duplicate date", next + "&queryStartTime=10/16/2026&queryStartTime=10/16/2026", false},
					{"empty date", next + "&queryStartTime=", false},
					{"duplicate token", next + "&$skipToken=other", false},
					{"empty token", listURL(route, "") + "&$skipToken=", false},
					{"undocumented token alias", strings.Replace(next, "$skipToken", "$skiptoken", 1), false},
					{"top", next + "&$top=100", false},
					{"default port", strings.Replace(next, "management.azure.com", "management.azure.com:443", 1), false},
					{"wrong version", strings.Replace(next, "2025-05-01", "2024-02-01", 1), false},
					{"missing date and token", strings.Replace(initial, "&queryStartTime=10/16/2026", "", 1), false},
				}...)
			} else {
				cases = append(cases, []struct {
					name, link string
					allowed    bool
				}{
					{"default port", strings.Replace(next, "management.azure.com", "management.azure.com:443", 1), true},
					{"repeat fixed options", next + "&timeRange=30d&monitorCondition=Fired&includeContext=false&includeEgressConfig=false&pageCount=100", true},
					{"minimum page", next + "&pageCount=1", true}, {"maximum page", next + "&pageCount=250", true},
					{"default sort", next + "&sortBy=lastModifiedDateTime&sortOrder=desc", true},
					{"changed window", next + "&timeRange=1d", false}, {"context", next + "&includeContext=true", false},
					{"egress", next + "&includeEgressConfig=true", false}, {"changed condition", next + "&monitorCondition=Resolved", false},
					{"custom window", next + "&customTimeRange=token-canary", false}, {"signal filter", next + "&signalType=Metric", false},
					{"workflow filter", next + "&alertState=New", false}, {"severity filter", next + "&severity=Sev0", false},
					{"target filter", next + "&targetResource=token-canary", false}, {"select", next + "&select=signalType", false},
					{"sort ascending", next + "&sortOrder=asc", false}, {"sort by severity", next + "&sortBy=severity", false},
					{"preview version", strings.Replace(next, "2019-03-01", "2018-05-05-preview", 1), false},
					{"sample foreign subscription", strings.Replace(next, selectedID, "1e3ff1c0-771a-4119-a03b-be82a51e232d", 1), false},
					{"duplicate token", next + "&ctoken=other", false}, {"empty token", initial + "&ctoken=", false},
					{"skip token", next + "&$skiptoken=next", false},
					{"duplicate option", next + "&timeRange=30d&timeRange=30d", false},
					{"duplicate page", next + "&pageCount=100&pageCount=100", false},
					{"duplicate sort", next + "&sortOrder=desc&sortOrder=desc", false},
					{"missing initial flags", strings.Replace(initial, "&includeContext=false", "", 1), false},
					{"changed initial size", strings.Replace(initial, "pageCount=100", "pageCount=25", 1), false},
					{"continuation without token", strings.Split(next, "&ctoken=")[0], false},
				}...)
				for _, size := range []string{"0", "251", "-1", "1.5", "", "text", "65536"} {
					cases = append(cases, struct {
						name, link string
						allowed    bool
					}{"invalid size " + size, next + "&pageCount=" + size, false})
				}
			}
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					calls := 0
					tr := ReadTransport{Subscription: selectedID, Route: route, QueryStartTime: "10/16/2026", Base: roundTrip(func(*http.Request) (*http.Response, error) { calls++; return response(200, `{}`), nil })}
					req, err := http.NewRequest(http.MethodGet, tc.link, nil)
					if err != nil {
						t.Fatal(err)
					}
					res, err := tr.Do(req)
					if res != nil {
						_ = res.Body.Close()
					}
					_, continuationErr := continuation(tc.link, selectedID, route, "10/16/2026")
					if tc.allowed && (err != nil || continuationErr != nil || calls != 1) {
						t.Fatalf("valid shape rejected %v %v", err, continuationErr)
					}
					if !tc.allowed && (err == nil || continuationErr == nil || calls != 0) {
						t.Fatal("unsafe shape reached transport")
					}
					if err != nil && strings.Contains(err.Error(), "canary") {
						t.Fatal("raw request leaked in error")
					}
				})
			}
			for _, method := range []string{"POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"} {
				tr := ReadTransport{Subscription: selectedID, Route: route, QueryStartTime: "10/16/2026", Base: roundTrip(func(*http.Request) (*http.Response, error) { t.Fatal("non-GET reached transport"); return nil, nil })}
				req, _ := http.NewRequest(method, next, nil)
				if _, err := tr.Do(req); err == nil {
					t.Fatalf("accepted %s", method)
				}
			}
			for _, change := range []string{"body", "host", "opaque", "force query", "raw path", "cancelled"} {
				tr := ReadTransport{Subscription: selectedID, Route: route, QueryStartTime: "10/16/2026", Base: roundTrip(func(*http.Request) (*http.Response, error) {
					t.Fatal("unsafe request reached transport")
					return nil, nil
				})}
				req, _ := http.NewRequest("GET", next, nil)
				switch change {
				case "body":
					req, _ = http.NewRequest("GET", next, strings.NewReader("payload-canary"))
				case "host":
					req.Host = "evil.example"
				case "opaque":
					req.URL.Opaque = "opaque"
				case "force query":
					req.URL.ForceQuery = true
				case "raw path":
					req.URL.RawPath = req.URL.Path
				case "cancelled":
					ctx, cancel := context.WithCancel(context.Background())
					cancel()
					req = req.WithContext(ctx)
				}
				if _, err := tr.Do(req); err == nil {
					t.Fatalf("accepted %s", change)
				}
			}
		})
	}
}

func TestOperationalContinuationCanonicalization(t *testing.T) {
	for _, route := range []string{serviceHealthRoute, monitorRoute} {
		original := listURL(route, "opaque%2B%2F%3D%23token")
		variant := original
		if route == serviceHealthRoute {
			variant += "&queryStartTime=10%2F16%2F2026"
		} else {
			variant = strings.Replace(variant, "management.azure.com", "management.azure.com:443", 1) + "&timeRange=30d&pageCount=250&sortBy=lastModifiedDateTime&sortOrder=desc"
		}
		u, _ := url.Parse(variant)
		u.Path = strings.ToUpper(u.Path)
		u.RawQuery = u.Query().Encode()
		variant = u.String()
		a, err := continuation(original, selectedID, route, "10/16/2026")
		if err != nil {
			t.Fatal(err)
		}
		b, err := continuation(variant, selectedID, route, "10/16/2026")
		if err != nil || a != b {
			t.Fatalf("equivalent links differ: %v", err)
		}
		calls, pages := 0, 0
		c := collection(t, func(*http.Request) (*http.Response, error) {
			calls++
			next := original
			if calls > 1 {
				next = variant
			}
			return response(200, `{"value":[],"nextLink":`+quoteJSON(next)+`}`), nil
		})
		err = collectorFor(c, route)(context.Background(), selectedID, func(healthPage health.Page) error { pages++; return nil })
		var diagnostic interface{ Diagnostic() string }
		if !errors.As(err, &diagnostic) || diagnostic.Diagnostic() != "repeated_continuation" || calls != 2 || pages != 2 {
			t.Fatalf("repeated equivalent link followed: calls %d pages %d error %v", calls, pages, err)
		}
	}
}

func quoteJSON(s string) string { b, _ := json.Marshal(s); return string(b) }
