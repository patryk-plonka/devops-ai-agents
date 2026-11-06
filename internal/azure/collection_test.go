package azure

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/patryk-plonka/devops-ai-agents/internal/health"
)

const collectionVM = "/subscriptions/" + selectedID + "/resourceGroups/g/providers/Microsoft.Compute/virtualMachines/vm"

func listURL(route, token string) string {
	path, version := "/resources", "2025-04-01"
	if route == healthRoute {
		path = "/providers/Microsoft.ResourceHealth/availabilityStatuses"
		version = "2025-05-01"
	}
	result := "https://management.azure.com/subscriptions/" + selectedID + path + "?api-version=" + version
	if token != "" {
		key := "$skiptoken"
		if route == healthRoute {
			key = "$skipToken"
		}
		result += "&" + key + "=" + token
	}
	return result
}
func collection(t *testing.T, transport roundTrip) Collection {
	t.Helper()
	return Collection{Base: transport, retryDelay: -1, Credential: func(id string) (azcore.TokenCredential, error) {
		if id != selectedID {
			t.Fatal("wrong credential subscription")
		}
		return fakeCredential{}, nil
	}}
}
func TestNativePagersCheckpointAndWhitelist(t *testing.T) {
	for _, route := range []string{inventoryRoute, healthRoute} {
		t.Run(route, func(t *testing.T) {
			emitted, calls := 0, 0
			c := collection(t, func(req *http.Request) (*http.Response, error) {
				calls++
				if calls > 2 {
					t.Fatal("unexpected read")
				}
				if calls == 2 && emitted != 1 {
					t.Fatal("request before preceding checkpoint")
				}
				deadline, ok := req.Context().Deadline()
				if !ok || time.Until(deadline) > OperationTimeout || time.Until(deadline) <= 0 {
					t.Fatal("missing per-page deadline")
				}
				want := listURL(route, "")
				if calls == 2 {
					want = listURL(route, "next")
				}
				wantURL, _ := url.Parse(want)
				if req.URL.Path != wantURL.Path || req.URL.Query().Encode() != wantURL.Query().Encode() {
					t.Fatalf("native API route %s want %s", req.URL, want)
				}
				record := fmt.Sprintf(`{"id":%q,"name":"vm","type":"Microsoft.Compute/virtualMachines","location":"eastus","tags":{"env":"dev","password":"tag-secret-canary"},"properties":{"password":"properties-canary"},"identity":{"principalId":"identity-canary"}}`, collectionVM)
				if route == healthRoute {
					record = fmt.Sprintf(`{"id":%q,"properties":{"availabilityState":null,"healthEventId":"event-123","reportedTime":"2026-11-06T10:00:00Z","occuredTime":"2026-11-06T09:00:00Z","summary":"prose-canary","recommendedActions":[{"action":"action-canary"}],"serviceImpactingEvents":[{"correlationId":"event-456"}]}}`, collectionVM+"/providers/Microsoft.ResourceHealth/availabilityStatuses/current")
				}
				body := `{"value":[` + record + `]}`
				if calls == 1 {
					body = fmt.Sprintf(`{"value":[%s],"nextLink":%q}`, record, listURL(route, "next"))
				}
				return response(200, body), nil
			})
			run := health.NewRun("test", "test")
			run.Scope = &health.Scope{SubscriptionID: selectedID}
			emit := func(p health.Page) error { emitted++; run.NormalizePage(route, p); return nil }
			collect := c.Inventory
			if route == healthRoute {
				collect = c.ResourceHealth
			}
			if err := collect(context.Background(), selectedID, emit); err != nil {
				t.Fatal(err)
			}
			if calls != 2 || emitted != 2 || len(run.Evidence) != 1 {
				t.Fatalf("pages %d emitted %d evidence %d", calls, emitted, len(run.Evidence))
			}
			e := run.Evidence[0]
			if route == inventoryRoute {
				if e.APIVersion != health.InventoryAPI || e.Resource.Tags["password"] != "[REDACTED]" || e.Resource.Tags["env"] != "dev" {
					t.Fatal("unsafe inventory mapping")
				}
			} else {
				if e.Kind != "event" || e.State != "undetermined" || e.ReportedAt == nil || e.OccurredAt == nil || len(e.EventIDs) != 1 || e.EventIDs[0] != "event-456" {
					t.Fatalf("lost structured event %+v", e)
				}
			}
		})
	}
}

func TestLaterPageFailuresPreserveFirstPage(t *testing.T) {
	for _, route := range []string{inventoryRoute, healthRoute} {
		for _, scenario := range []string{"denied", "unauthorized", "throttled", "timeout", "unsafe host", "wrong subscription", "other operation", "other version", "extra query", "repeat", "initial repeat", "limit", "redirect", "malformed", "missing list", "registration", "unsupported"} {
			t.Run(route+"/"+scenario, func(t *testing.T) {
				calls, emitted := 0, 0
				c := collection(t, func(req *http.Request) (*http.Response, error) {
					calls++
					if calls == 1 {
						next := listURL(route, "next")
						switch scenario {
						case "unsafe host":
							next = strings.Replace(next, "management.azure.com", "evil.example", 1)
						case "wrong subscription":
							next = strings.Replace(next, selectedID, otherID, 1)
						case "other operation":
							other := inventoryRoute
							if route == inventoryRoute {
								other = healthRoute
							}
							next = listURL(other, "next")
						case "other version":
							next = strings.Replace(next, "2025-", "2024-", 1)
						case "initial repeat":
							next = listURL(route, "")
						case "extra query":
							next += "&$filter=omit-resources"
						}
						return response(200, fmt.Sprintf(`{"value":[],"nextLink":%q}`, next)), nil
					}
					if emitted != 1 {
						t.Fatal("lost first checkpoint")
					}
					switch scenario {
					case "denied":
						return response(403, `{"error":{"code":"AuthorizationFailed","message":"body-canary"}}`), nil
					case "unauthorized":
						return response(401, `{}`), nil
					case "throttled":
						return response(429, `{}`), nil
					case "timeout":
						return nil, context.DeadlineExceeded
					case "repeat":
						return response(200, fmt.Sprintf(`{"value":[],"nextLink":%q}`, listURL(route, "next"))), nil
					case "redirect":
						r := response(302, `redirect-canary`)
						r.Header.Set("Location", "https://evil.example")
						return r, nil
					case "malformed":
						return response(200, `{invalid`), nil
					case "missing list":
						return response(200, `{}`), nil
					case "unsupported":
						return response(400, `{"error":{"code":"UnsupportedResourceType","message":"body-canary"}}`), nil
					case "registration":
						return response(409, `{"error":{"code":"MissingSubscriptionRegistration","message":"body-canary"}}`), nil
					}
					t.Fatal("unsafe continuation reached network")
					return nil, nil
				})
				if scenario == "limit" {
					c.pageLimit = 1
				}
				collect := c.Inventory
				if route == healthRoute {
					collect = c.ResourceHealth
				}
				err := collect(context.Background(), selectedID, func(health.Page) error { emitted++; return nil })
				if err == nil || strings.Contains(err.Error(), "canary") {
					t.Fatalf("unsafe/missing error %v", err)
				}
				reason := "collection_failed"
				switch scenario {
				case "denied":
					reason = "access_denied"
				case "unauthorized":
					reason = "login_required"
				case "throttled":
					reason = "throttled"
				case "timeout":
					reason = "timeout"
				case "unsafe host", "wrong subscription", "other operation", "other version", "extra query":
					reason = "unsafe_continuation"
				case "repeat", "initial repeat":
					reason = "repeated_continuation"
				case "limit":
					reason = "page_limit"
				case "redirect":
					reason = "read_boundary_rejected"
				case "missing list", "malformed":
					reason = "invalid_response"
				case "unsupported":
					reason = "unsupported"
				}
				var diagnostic interface{ Diagnostic() string }
				if !errors.As(err, &diagnostic) || diagnostic.Diagnostic() != reason {
					t.Fatalf("failure category got %v want %s", err, reason)
				}
				expectedCalls, expectedPages := 2, 1
				switch scenario {
				case "unsafe host", "wrong subscription", "other operation", "other version", "extra query", "initial repeat", "limit":
					expectedCalls = 1
				case "throttled", "timeout":
					expectedCalls = 5 // initial page + one attempt and three retries
				case "repeat":
					expectedPages = 2
				}
				if calls != expectedCalls || emitted != expectedPages {
					t.Fatalf("calls %d want %d pages %d want %d err %v", calls, expectedCalls, emitted, expectedPages, err)
				}
			})
		}
	}
}
func TestCollectorDeadlineAndCheckpointFailure(t *testing.T) {
	for _, route := range []string{inventoryRoute, healthRoute} {
		t.Run(route, func(t *testing.T) {
			calls := 0
			c := collection(t, func(*http.Request) (*http.Response, error) {
				calls++
				return response(200, fmt.Sprintf(`{"value":[],"nextLink":%q}`, listURL(route, "next"))), nil
			})
			collect := c.Inventory
			if route == healthRoute {
				collect = c.ResourceHealth
			}
			sentinel := errors.New("write-canary")
			if err := collect(context.Background(), selectedID, func(health.Page) error { return sentinel }); !errors.Is(err, sentinel) || calls != 1 {
				t.Fatal("write failure allowed another page")
			}
			c.timeout = -time.Second
			calls = 0
			collect = c.Inventory
			if route == healthRoute {
				collect = c.ResourceHealth
			}
			err := collect(context.Background(), selectedID, func(health.Page) error { t.Fatal("expired page emitted"); return nil })
			if err == nil || calls != 0 {
				t.Fatalf("expired context accessed ARM: %d %v", calls, err)
			}
		})
	}
}

func TestCollectorCancellationStopsBeforeNextRequest(t *testing.T) {
	for _, route := range []string{inventoryRoute, healthRoute} {
		t.Run(route, func(t *testing.T) {
			calls := 0
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			c := collection(t, func(*http.Request) (*http.Response, error) {
				calls++
				return response(200, fmt.Sprintf(`{"value":[],"nextLink":%q}`, listURL(route, "next"))), nil
			})
			collect := c.Inventory
			if route == healthRoute {
				collect = c.ResourceHealth
			}
			pages := 0
			err := collect(ctx, selectedID, func(health.Page) error { pages++; cancel(); return nil })
			if !errors.Is(err, context.Canceled) || calls != 1 || pages != 1 {
				t.Fatalf("reads after cancellation: calls %d pages %d error %v", calls, pages, err)
			}
		})
	}
}
