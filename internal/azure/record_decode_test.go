package azure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/patryk-plonka/devops-ai-agents/internal/health"
	"github.com/patryk-plonka/devops-ai-agents/internal/runstore"
)

func TestMixedSchemaPagesRetainValidRecordsAndCheckpoint(t *testing.T) {
	for _, route := range []string{inventoryRoute, healthRoute} {
		t.Run(route, func(t *testing.T) {
			run := health.NewRun("test", "test")
			run.Scope = &health.Scope{SubscriptionID: selectedID}
			store, err := runstore.New(t.TempDir(), &run)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			calls := 0
			c := collection(t, func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 2 {
					b, e := os.ReadFile(store.Path())
					if e != nil {
						t.Fatal(e)
					}
					var saved health.Run
					if json.Unmarshal(b, &saved) != nil {
						t.Fatal("invalid checkpoint")
					}
					if len(saved.Evidence) != 1 || saved.Sources[route].RejectedRecords != 2 || saved.Sources[route].Pages != 1 {
						t.Fatalf("valid siblings not checkpointed before next page: %+v", saved)
					}
					return response(200, `{"value":[]}`), nil
				}
				if calls > 2 {
					t.Fatal("unexpected retry")
				}
				good := fmt.Sprintf(`{"id":%q,"name":"vm","tags":{"endpoint":"rediss://app:credential-canary@cache.example:6380"}}`, collectionVM)
				bad := `{"id":42,"name":"malformed-record-canary"}`
				if route == healthRoute {
					good = fmt.Sprintf(`{"id":%q,"properties":{"availabilityState":"Unavailable"}}`, collectionVM+"/providers/Microsoft.ResourceHealth/availabilityStatuses/current")
					bad = fmt.Sprintf(`{"id":%q,"properties":{"availabilityState":{"bad":"malformed-record-canary"}}}`, collectionVM+"/providers/Microsoft.ResourceHealth/availabilityStatuses/current")
				}
				return response(200, fmt.Sprintf(`{"value":[%s,%s,null],"nextLink":%q}`, good, bad, listURL(route, "next"))), nil
			})
			collect := c.Inventory
			if route == healthRoute {
				collect = c.ResourceHealth
			}
			complete := func(context.Context, string, func(health.Page) error) error { return nil }
			inventory, resourceHealth := complete, complete
			if route == inventoryRoute {
				inventory = collect
			} else {
				resourceHealth = collect
			}
			code, err := health.Check(context.Background(), &run, inventory, resourceHealth, store.Save)
			if err != nil || code != 3 || calls != 2 || run.Outcome != "partial" || len(run.Evidence) != 1 || run.Sources[route].RejectedRecords != 2 || run.Sources[route].Pages != 2 || run.Sources[route].Reason != "invalid_records" {
				t.Fatalf("lost valid siblings: code %d error %v calls %d run %+v", code, err, calls, run)
			}
			b, e := os.ReadFile(store.Path())
			if e != nil {
				t.Fatal(e)
			}
			if strings.Contains(string(b), "malformed-record-canary") || strings.Contains(string(b), "credential-canary") {
				t.Fatal("rejected/secret payload reached saved checkpoint")
			}
			if route == healthRoute && len(run.Findings) != 1 {
				t.Fatal("valid issue evidence lost")
			}
		})
	}
}

func TestMalformedPageEnvelopesFailWithoutRetry(t *testing.T) {
	for _, route := range []string{inventoryRoute, healthRoute} {
		for _, body := range []string{`{invalid`, `null`, `[]`, `{}`, `{"value":null}`, `{"value":{}}`, `{"value":[],"nextLink":{"bad":"envelope-canary"}}`} {
			t.Run(route+"/"+body, func(t *testing.T) {
				calls := 0
				c := collection(t, func(*http.Request) (*http.Response, error) { calls++; return response(200, body), nil })
				collect := c.Inventory
				if route == healthRoute {
					collect = c.ResourceHealth
				}
				err := collect(context.Background(), selectedID, func(health.Page) error { t.Fatal("malformed envelope was emitted"); return nil })
				var diagnostic interface{ Diagnostic() string }
				if err == nil || !errors.As(err, &diagnostic) || diagnostic.Diagnostic() != "invalid_response" || calls != 1 || strings.Contains(err.Error(), "envelope-canary") {
					t.Fatalf("invalid page outcome: %v calls %d", err, calls)
				}
			})
		}
	}
}

func TestMixedSchemaCheckpointSurvivesLaterRetriesAndUnsafeContinuation(t *testing.T) {
	for _, route := range []string{inventoryRoute, healthRoute} {
		for _, scenario := range []string{"throttled", "unsafe continuation"} {
			t.Run(route+"/"+scenario, func(t *testing.T) {
				calls := 0
				c := collection(t, func(*http.Request) (*http.Response, error) {
					calls++
					if calls > 1 {
						return response(429, `{"error":{"message":"response-canary"}}`), nil
					}
					good := fmt.Sprintf(`{"id":%q}`, collectionVM)
					bad := `{"id":42}`
					if route == healthRoute {
						good = fmt.Sprintf(`{"id":%q,"properties":{"availabilityState":"Unavailable"}}`, collectionVM+"/providers/Microsoft.ResourceHealth/availabilityStatuses/current")
						bad = `{"properties":{"availabilityState":42}}`
					}
					next := listURL(route, "next")
					if scenario == "unsafe continuation" {
						next = strings.Replace(next, selectedID, otherID, 1)
					}
					return response(200, fmt.Sprintf(`{"value":[%s,%s],"nextLink":%q}`, good, bad, next)), nil
				})
				run := health.NewRun("test", "test")
				run.Scope = &health.Scope{SubscriptionID: selectedID}
				store, err := runstore.New(t.TempDir(), &run)
				if err != nil {
					t.Fatal(err)
				}
				defer store.Close()
				collect := c.Inventory
				if route == healthRoute {
					collect = c.ResourceHealth
				}
				complete := func(context.Context, string, func(health.Page) error) error { return nil }
				inventory, resourceHealth := complete, complete
				if route == inventoryRoute {
					inventory = collect
				} else {
					resourceHealth = collect
				}
				code, err := health.Check(context.Background(), &run, inventory, resourceHealth, store.Save)
				expectedCalls, reason := 5, "throttled"
				if scenario == "unsafe continuation" {
					expectedCalls, reason = 1, "unsafe_continuation"
				}
				if err != nil || code != 3 || calls != expectedCalls || run.Sources[route].Reason != reason || run.Sources[route].RejectedRecords != 1 {
					t.Fatalf("outcome %d %v calls %d source %+v", code, err, calls, run.Sources[route])
				}
				b, e := os.ReadFile(store.Path())
				if e != nil {
					t.Fatal(e)
				}
				var saved health.Run
				if json.Unmarshal(b, &saved) != nil {
					t.Fatal("invalid snapshot")
				}
				if len(saved.Evidence) != 1 || saved.Sources[route].Pages != 1 || saved.Coverage != "incomplete" || strings.Contains(string(b), "response-canary") {
					t.Fatal("valid checkpoint lost")
				}
				foundRejected := false
				for _, gap := range saved.Gaps {
					if gap.Source == route && gap.Reason == "invalid_records" {
						foundRejected = true
					}
				}
				if !foundRejected {
					t.Fatal("later failure concealed rejected-record gap")
				}
			})
		}
	}
}
