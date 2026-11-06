package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/patryk-plonka/devops-ai-agents/internal/azure"
	"github.com/patryk-plonka/devops-ai-agents/internal/health"
	"github.com/patryk-plonka/devops-ai-agents/internal/report"
	"github.com/patryk-plonka/devops-ai-agents/internal/runstore"
)

type armFixture func(*http.Request) (*http.Response, error)

func (f armFixture) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func armResponse(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
}

// Use the actual session discovery and AzureCLICredential subprocess paths. The
// only external replacement is a strict synthetic az and an in-memory ARM wire.
// No production endpoint/token flag or real Azure profile is involved.
func integrationApp(t *testing.T, wire armFixture) (App, *bytes.Buffer, *bytes.Buffer, string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("synthetic az executable uses POSIX shell")
	}
	a, out, diagnostics, _ := testApp(t)
	bin := t.TempDir()
	script, err := os.ReadFile("testdata/fake-az.sh")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "az"), script, 0700); err != nil {
		t.Fatal(err)
	}
	log := filepath.Join(bin, "commands")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("AZURE_CONFIG_DIR", bin)
	t.Setenv("FAKE_AZ_LOG", log)
	t.Setenv("FAKE_AZ_MODE", "")
	t.Setenv("AZURE_SDK_GO_LOGGING", "all")
	// Anything escaping the injected wire must fail locally, even if a future
	// adapter accidentally falls back to the global transport.
	prior := http.DefaultTransport
	http.DefaultTransport = armFixture(func(*http.Request) (*http.Response, error) {
		t.Error("attempted live network access")
		return nil, errors.New("network disabled in integration test")
	})
	t.Cleanup(func() { http.DefaultTransport = prior })
	base := armFixture(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Host != "management.azure.com" || !strings.HasPrefix(req.URL.Path, "/subscriptions/"+first) {
			t.Fatal("mutation or scope escape reached the wire")
		}
		if req.Header.Get("Authorization") != "Bearer synthetic-token-canary" {
			t.Fatal("credential did not come from the synthetic CLI")
		}
		if _, ok := req.Context().Deadline(); !ok {
			t.Fatal("unbounded ARM operation")
		}
		return wire(req)
	})
	a.Discover = nil
	a.Validate = (azure.Validator{Base: base}).Validate
	c := azure.Collection{Base: base}
	a.Inventory, a.ResourceHealth = c.Inventory, c.ResourceHealth
	a.Version = "synthetic-integration"
	return a, out, diagnostics, log
}

func integrationContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
	// Keep all three retries, with controlled delays and no wall-clock waits.
	return policy.WithRetryOptions(ctx, policy.RetryOptions{MaxRetries: 3, RetryDelay: time.Nanosecond, MaxRetryDelay: time.Nanosecond})
}

func subscriptionResponse() string {
	return fmt.Sprintf(`{"id":"/subscriptions/%s","subscriptionId":%q,"tenantId":%q}`, first, first, tenant)
}

func inventoryRecord(i int) string {
	return fmt.Sprintf(`{"id":%q,"name":%q,"type":"Microsoft.Compute/virtualMachines","location":"eastus","tags":{"team":"consulting","password":"tag-secret-canary","display":"<script>alert(1)</script>|[link](https://bad.example)"},"properties":{"secret":"property-canary"}}`, integrationResource(i), fmt.Sprintf("vm-%03d", i))
}

func integrationResource(i int) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/synthetic/providers/Microsoft.Compute/virtualMachines/vm-%03d", first, i)
}

func healthRecord(i int) string {
	properties, status := `{"availabilityState":"Available"}`, "current"
	switch i {
	case 0, 1:
		properties = `{"availabilityState":"Unavailable"}`
	case 2, 3, 4:
		properties = `{"availabilityState":"Degraded"}`
	case 5:
		properties = `{"availabilityState":null,"healthEventId":"synthetic-event"}`
	case 6:
		properties = `{"availabilityState":"Unknown"}`
	case 7:
		properties, status = `{"availabilityState":"Unavailable"}`, "2026-11-05T10:00:00Z"
	case 8:
		properties = `{}`
	case 9:
		properties = `{"availabilityState":"FutureState"}`
	}
	properties = strings.TrimSuffix(properties, "}")
	if properties != "{" {
		properties += ","
	}
	properties += `"reportedTime":"2026-11-06T10:00:00Z","occuredTime":"2026-11-06T09:00:00Z","summary":"source-prose-canary"}`
	return fmt.Sprintf(`{"id":%q,"properties":%s,"summary":"source-prose-canary"}`, integrationResource(i)+"/providers/Microsoft.ResourceHealth/availabilityStatuses/"+status, properties)
}

func integrationList(route string, records []string, next string) string {
	body := `{"value":[` + strings.Join(records, ",") + `]`
	if next != "" {
		path, version := "/resources", health.InventoryAPI
		if route == "resource_health" {
			path, version = "/providers/Microsoft.ResourceHealth/availabilityStatuses", health.ResourceHealthAPI
		}
		link := "https://management.azure.com/subscriptions/" + first + path + "?api-version=" + version + "&$skiptoken=" + next
		if route == "resource_health" {
			link += "&$top=120"
		}
		body += fmt.Sprintf(`,"nextLink":%q`, link)
	}
	return body + "}"
}

func requestSource(req *http.Request) string {
	if strings.HasSuffix(req.URL.Path, "/resources") {
		return "inventory"
	}
	if strings.HasSuffix(req.URL.Path, "/providers/Microsoft.ResourceHealth/availabilityStatuses") {
		return "resource_health"
	}
	return "subscription"
}

func assertIntegrationArtifacts(t *testing.T, a App, out, diagnostics *bytes.Buffer) health.Run {
	t.Helper()
	r := assertPublished(t, a.Directory, strings.TrimSpace(out.String()))
	b, err := os.ReadFile(strings.TrimSpace(out.String()))
	if err != nil {
		t.Fatal(err)
	}
	text := html.UnescapeString(string(b))
	snapshotBytes, err := os.ReadFile(filepath.Join(filepath.Dir(strings.TrimSpace(out.String())), "run.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(snapshotBytes)+text+diagnostics.String(), "canary") {
		t.Fatal("sensitive source/profile/token/error escaped into saved output")
	}
	if r.Coverage != "incomplete" || !strings.Contains(text, "No resource is classified overall OK") {
		t.Fatal("overall coverage claim missing or misleading")
	}
	for _, source := range []string{"advisor", "service_health", "azure_monitor"} {
		if r.Sources[source].Status != "not_checked" || r.Sources[source].Reason != "not_implemented_in_s01" || !strings.Contains(text, source) {
			t.Fatal("unimplemented source presented as covered")
		}
	}
	ids := map[string]bool{}
	for _, e := range r.Evidence {
		if ids[e.ID] || !strings.Contains(text, e.ID) {
			t.Fatal("duplicate or omitted evidence")
		}
		if e.Source == "resource_health" && (e.ReportedAt == nil || e.OccurredAt == nil || e.APIVersion != health.ResourceHealthAPI || !strings.Contains(text, "2026-11-06T10:00:00Z")) {
			t.Fatal("source timestamps or API version lost in the full flow")
		}
		ids[e.ID] = true
	}
	check := func(refs []string) {
		for _, id := range refs {
			if !ids[id] {
				t.Fatal("unresolvable saved evidence reference")
			}
		}
	}
	for _, resource := range r.Resources {
		check(resource.EvidenceIDs)
		if !strings.Contains(text, resource.ID) || resource.Tags["team"] != "consulting" || resource.Tags["password"] != "[REDACTED]" {
			t.Fatal("inventory enumeration or tag filtering lost")
		}
	}
	for _, observation := range r.Observations {
		check(observation.EvidenceIDs)
	}
	for _, finding := range r.Findings {
		if len(finding.EvidenceIDs) == 0 {
			t.Fatal("unsupported finding")
		}
		check(finding.EvidenceIDs)
	}
	for _, gap := range r.Gaps {
		check(gap.EvidenceIDs)
	}
	if strings.Contains(string(b), "<script>") || strings.Contains(string(b), "[link](https://bad.example)") {
		t.Fatal("source text became active Markdown/HTML")
	}
	return r
}

func assertSyntheticLogin(t *testing.T, log string, tokens bool) {
	t.Helper()
	b, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) < 2 || lines[0] != "cloud show --output json --only-show-errors" || lines[1] != "account list --all --output json --only-show-errors" {
		t.Fatal("unexpected CLI discovery")
	}
	if tokens != (len(lines) > 2) {
		t.Fatal("unexpected token acquisition")
	}
	for _, line := range lines[2:] {
		if line != "account get-access-token -o json --resource https://management.core.windows.net/ --subscription "+first {
			t.Fatalf("wrong subscription/tenant binding or CLI mutation: %s", line)
		}
	}
}

func TestFullFlowScaleByIDAndName(t *testing.T) {
	var priorID string
	for _, requested := range []string{first, "sYnThEtIc GuEsT"} {
		t.Run(requested, func(t *testing.T) {
			counts := map[string]int{}
			var a App
			a, out, diagnostics, log := integrationApp(t, func(req *http.Request) (*http.Response, error) {
				source := requestSource(req)
				counts[source]++
				if source == "subscription" {
					if snapshot(t, a.Directory).Revision != 1 {
						t.Fatal("validation preceded initial storage")
					}
					return armResponse(200, subscriptionResponse()), nil
				}
				page := counts[source] - 1
				if page > 2 {
					t.Fatal("unbounded enumeration")
				}
				checkpoint := snapshot(t, a.Directory)
				wantEvidence := page * 120
				if source == "resource_health" {
					wantEvidence += 360
				}
				if len(checkpoint.Evidence) != wantEvidence || checkpoint.Sources[source].Pages != page {
					t.Fatal("next request before saved page checkpoint")
				}
				if page > 0 && req.URL.Query().Get("$skiptoken") != fmt.Sprint(page) {
					t.Fatal("native pager lost continuation")
				}
				records := []string{}
				limit := (page + 1) * 120
				if source == "resource_health" && limit == 360 {
					limit = 359 // One inventory resource has no health evidence.
				}
				for i := page * 120; i < limit; i++ {
					record := inventoryRecord(i)
					if source == "resource_health" {
						record = healthRecord(i)
					}
					records = append(records, record)
				}
				next := ""
				if page < 2 {
					next = fmt.Sprint(page + 1)
				}
				return armResponse(200, integrationList(source, records, next)), nil
			})
			if code := a.Run(integrationContext(t), []string{"health", "check", "--subscription", requested}); code != 0 {
				t.Fatalf("exit %d: %s", code, diagnostics)
			}
			r := assertIntegrationArtifacts(t, a, out, diagnostics)
			if r.Outcome != "completed" || r.Scope.SubscriptionID != first || r.Scope.TenantID != tenant || len(r.Resources) != 360 || len(r.Evidence) != 719 || len(r.Findings) != 6 || len(r.Observations) != 360 || counts["subscription"] != 1 || counts["inventory"] != 3 || counts["resource_health"] != 3 {
				t.Fatalf("wrong full-flow counts: resources=%d evidence=%d findings=%d observations=%d requests=%v", len(r.Resources), len(r.Evidence), len(r.Findings), len(r.Observations), counts)
			}
			if r.ID == priorID {
				t.Fatal("reused run directory")
			}
			priorID = r.ID
			states := map[string]string{}
			for _, o := range r.Observations {
				states[o.ResourceID] = o.State
			}
			for i, want := range map[int]string{0: "Unavailable", 2: "Degraded", 5: "undetermined", 6: "Unknown", 7: "undetermined", 8: "undetermined", 9: "undetermined", 10: "Available", 359: "not_checked"} {
				if states[integrationResource(i)] != want {
					t.Fatalf("resource %d: got %s want %s", i, states[integrationResource(i)], want)
				}
			}
			assertSyntheticLogin(t, log, true)
		})
	}
}

func TestFullFlowScopeAndIndependentFailures(t *testing.T) {
	for _, scenario := range []string{"subscription denied", "wrong scope", "registration", "missing session", "expired login", "inventory denied", "health denied", "both denied", "empty", "missing health", "orphan health"} {
		t.Run(scenario, func(t *testing.T) {
			counts := map[string]int{}
			a, out, diagnostics, log := integrationApp(t, func(req *http.Request) (*http.Response, error) {
				source := requestSource(req)
				counts[source]++
				if source == "subscription" {
					switch scenario {
					case "subscription denied":
						return armResponse(403, `{"error":{"code":"AuthorizationFailed","message":"body-canary"}}`), nil
					case "wrong scope":
						return armResponse(200, strings.ReplaceAll(subscriptionResponse(), first, second)), nil
					case "registration":
						return armResponse(409, `{"error":{"code":"MissingSubscriptionRegistration","message":"body-canary"}}`), nil
					}
					return armResponse(200, subscriptionResponse()), nil
				}
				if scenario == "both denied" || source == "inventory" && scenario == "inventory denied" || source == "resource_health" && scenario == "health denied" {
					return armResponse(403, `{"error":{"code":"AuthorizationFailed","message":"body-canary"}}`), nil
				}
				records := []string{}
				if source == "inventory" && scenario != "empty" && scenario != "orphan health" {
					records = append(records, inventoryRecord(0))
				}
				if source == "resource_health" && scenario != "empty" && scenario != "missing health" {
					records = append(records, healthRecord(0))
				}
				return armResponse(200, integrationList(source, records, "")), nil
			})
			wantCode, outcome, category := 0, "completed", ""
			switch scenario {
			case "subscription denied":
				wantCode, outcome, category = 1, "failed", "subscription_access_denied"
			case "wrong scope":
				wantCode, outcome, category = 1, "failed", "scope_mismatch"
			case "registration":
				wantCode, outcome, category = 1, "failed", "subscription_validation_failed"
			case "missing session":
				t.Setenv("FAKE_AZ_MODE", "missing")
				wantCode, outcome, category = 2, "failed", "subscription_not_in_session"
			case "expired login":
				t.Setenv("FAKE_AZ_MODE", "expired")
				wantCode, outcome, category = 1, "failed", "login_required"
			case "inventory denied", "health denied":
				wantCode, outcome = 3, "partial"
			case "both denied":
				wantCode, outcome = 1, "failed"
			}
			if code := a.Run(integrationContext(t), []string{"health", "check", "--subscription", first, "--debug"}); code != wantCode {
				t.Fatalf("exit %d want %d: %s", code, wantCode, diagnostics)
			}
			r := assertIntegrationArtifacts(t, a, out, diagnostics)
			if r.Outcome != outcome {
				t.Fatal("incorrect saved outcome")
			}
			if category != "" {
				if len(r.Diagnostics) != 1 || r.Diagnostics[0] != category || counts["inventory"]+counts["resource_health"] != 0 || len(r.Evidence) != 0 {
					t.Fatal("scope failure mislabeled or collection continued")
				}
				wantCalls := 1
				if scenario == "missing session" || scenario == "expired login" {
					wantCalls = 0
				}
				if counts["subscription"] != wantCalls {
					t.Fatal("unexpected retry or registration")
				}
			} else {
				if counts["inventory"] != 1 || counts["resource_health"] != 1 {
					t.Fatal("collectors were not independent")
				}
				for _, source := range []string{"inventory", "resource_health"} {
					denied := scenario == "both denied" || source == "inventory" && scenario == "inventory denied" || source == "resource_health" && scenario == "health denied"
					if denied && (r.Sources[source].Status != "failed" || r.Sources[source].Reason != "access_denied") || !denied && r.Sources[source].Status != "completed" {
						t.Fatal("incorrect per-source coverage")
					}
				}
				wantEvidence, wantFindings, wantGap := 2, 1, ""
				switch scenario {
				case "inventory denied":
					wantEvidence, wantGap = 1, "inventory_unverified"
				case "health denied":
					wantEvidence, wantFindings, wantGap = 1, 0, "health_not_checked"
				case "both denied":
					wantEvidence, wantFindings = 0, 0
				case "empty":
					wantEvidence, wantFindings, wantGap = 0, 0, "no_resources_observed"
				case "missing health":
					wantEvidence, wantFindings, wantGap = 1, 0, "missing_health_evidence"
				case "orphan health":
					wantEvidence, wantGap = 1, "health_resource_absent_from_inventory"
				}
				if len(r.Evidence) != wantEvidence || len(r.Findings) != wantFindings {
					t.Fatal("independent evidence/findings lost or invented")
				}
				if wantGap != "" {
					assertGap(t, r, wantGap)
				}
			}
			assertSyntheticLogin(t, log, scenario != "missing session")
		})
	}
}

func assertGap(t *testing.T, r health.Run, reason string) {
	t.Helper()
	for _, gap := range r.Gaps {
		if gap.Reason == reason {
			return
		}
	}
	t.Fatalf("missing explicit gap: %s", reason)
}

func TestFullFlowLaterPageFailures(t *testing.T) {
	for _, source := range []string{"inventory", "resource_health"} {
		for _, scenario := range []string{"timeout", "throttled", "unsafe host", "wrong subscription", "redirect", "registration", "repeat", "invalid envelope", "mixed records"} {
			t.Run(source+"/"+scenario, func(t *testing.T) {
				counts := map[string]int{}
				var a App
				a, out, diagnostics, _ := integrationApp(t, func(req *http.Request) (*http.Response, error) {
					route := requestSource(req)
					counts[route]++
					if route == "subscription" {
						return armResponse(200, subscriptionResponse()), nil
					}
					record := inventoryRecord(0)
					if route == "resource_health" {
						record = healthRecord(0)
					}
					if route != source {
						return armResponse(200, integrationList(route, []string{record}, "")), nil
					}
					if counts[route] == 1 {
						body := integrationList(route, []string{record}, "next")
						if scenario == "unsafe host" {
							body = strings.Replace(body, "management.azure.com", "evil.example", 1)
						}
						if scenario == "wrong subscription" {
							body = strings.Replace(body, "https://management.azure.com/subscriptions/"+first, "https://management.azure.com/subscriptions/"+second, 1)
						}
						return armResponse(200, body), nil
					}
					if snapshot(t, a.Directory).Sources[source].Pages != 1 {
						t.Fatal("first page not committed before adverse request")
					}
					switch scenario {
					case "timeout":
						return nil, context.DeadlineExceeded
					case "throttled":
						return armResponse(429, `{"error":{"code":"TooManyRequests","message":"body-canary"}}`), nil
					case "redirect":
						response := armResponse(307, "redirect-canary")
						response.Header.Set("Location", "https://evil.example")
						return response, nil
					case "registration":
						return armResponse(409, `{"error":{"code":"MissingSubscriptionRegistration","message":"body-canary"}}`), nil
					case "repeat":
						return armResponse(200, integrationList(route, []string{record}, "next")), nil
					case "invalid envelope":
						return armResponse(200, `{"value":"body-canary"}`), nil
					case "mixed records":
						wrongScope := strings.ReplaceAll(record, first, second)
						return armResponse(200, integrationList(route, []string{record, wrongScope, `{"id":"rejected-canary","properties":42,"tags":42}`}, "")), nil
					}
					t.Fatal("unsafe continuation reached wire")
					return nil, nil
				})
				if code := a.Run(integrationContext(t), []string{"health", "check", "--subscription", first, "--debug"}); code != 3 {
					t.Fatalf("exit %d: %s", code, diagnostics)
				}
				r := assertIntegrationArtifacts(t, a, out, diagnostics)
				reason, calls, pages := "collection_failed", 2, 1
				switch scenario {
				case "timeout":
					reason, calls = "timeout", 5
				case "throttled":
					reason, calls = "throttled", 5
				case "unsafe host", "wrong subscription":
					reason, calls = "unsafe_continuation", 1
				case "redirect":
					reason = "read_boundary_rejected"
				case "repeat":
					reason, pages = "repeated_continuation", 2
				case "invalid envelope":
					reason = "invalid_response"
				case "mixed records":
					reason, pages = "invalid_records", 2
					if r.Sources[source].RejectedRecords != 2 {
						t.Fatal("rejected records not counted")
					}
				}
				other := "inventory"
				if source == other {
					other = "resource_health"
				}
				if counts[source] != calls || counts[other] != 1 || r.Sources[source].Status != "partial" || r.Sources[source].Reason != reason || r.Sources[source].Pages != pages || r.Sources[other].Status != "completed" || len(r.Evidence) != 2 || len(r.Findings) != 1 {
					t.Fatalf("partial flow lost evidence/coverage: requests=%v source=%+v evidence=%d", counts, r.Sources[source], len(r.Evidence))
				}
				assertGap(t, r, reason)
			})
		}
	}
}

func TestFullFlowFinalWriteFailureRetainsNativePages(t *testing.T) {
	counts := map[string]int{}
	a, out, diagnostics, _ := integrationApp(t, func(req *http.Request) (*http.Response, error) {
		source := requestSource(req)
		counts[source]++
		if source == "subscription" {
			return armResponse(200, subscriptionResponse()), nil
		}
		record := inventoryRecord(0)
		if source == "resource_health" {
			record = healthRecord(0)
		}
		return armResponse(200, integrationList(source, []string{record}, "")), nil
	})
	a.NewStore = func(root string, r *health.Run) (Storage, error) {
		s, err := runstore.New(root, r)
		return publicationStore{Storage: s, finalWriteFailure: true}, err
	}
	if code := a.Run(integrationContext(t), []string{"health", "check", "--subscription", first}); code != 1 || out.Len() != 0 || strings.Contains(diagnostics.String(), "Report saved:") || !strings.Contains(diagnostics.String(), "Last saved snapshot:") {
		t.Fatalf("write failure concealed: exit %d %s", code, diagnostics)
	}
	r := snapshot(t, a.Directory)
	if r.EndedAt != nil || len(r.Evidence) != 2 || counts["inventory"] != 1 || counts["resource_health"] != 1 {
		t.Fatal("last native page checkpoint lost")
	}
	if _, err := os.Stat(filepath.Join(a.Directory, "runs", r.ID, "report.md")); !os.IsNotExist(err) {
		t.Fatal("report published after failed final write")
	}
}

type checkpointFailureStore struct{ Storage }

func (s checkpointFailureStore) Save(r *health.Run) error {
	if r.Sources["inventory"].Pages == 2 {
		return errors.New("checkpoint-secret-canary")
	}
	return s.Storage.Save(r)
}

func TestFullFlowCheckpointFailureStopsNativeCollection(t *testing.T) {
	counts := map[string]int{}
	a, out, diagnostics, _ := integrationApp(t, func(req *http.Request) (*http.Response, error) {
		source := requestSource(req)
		counts[source]++
		if source == "subscription" {
			return armResponse(200, subscriptionResponse()), nil
		}
		if source != "inventory" || counts[source] > 2 {
			t.Fatal("collection continued after failed page checkpoint")
		}
		return armResponse(200, integrationList(source, []string{inventoryRecord(counts[source] - 1)}, fmt.Sprint(counts[source]))), nil
	})
	a.NewStore = func(root string, r *health.Run) (Storage, error) {
		s, err := runstore.New(root, r)
		return checkpointFailureStore{Storage: s}, err
	}
	if code := a.Run(integrationContext(t), []string{"health", "check", "--subscription", first}); code != 1 || out.Len() != 0 || strings.Contains(diagnostics.String(), "canary") || !strings.Contains(diagnostics.String(), "Last saved snapshot:") {
		t.Fatalf("checkpoint failure concealed: exit %d %s", code, diagnostics)
	}
	r := snapshot(t, a.Directory)
	if len(r.Evidence) != 1 || r.Sources["inventory"].Pages != 1 || r.EndedAt != nil || counts["inventory"] != 2 || counts["resource_health"] != 0 {
		t.Fatal("prior page checkpoint lost or extra source attempted")
	}
}

func TestGeneratedRunFilesAreIgnored(t *testing.T) {
	root, err := runstore.FindRepository(".")
	if err != nil {
		t.Fatal(err)
	}
	ignore, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	// An isolated Git repository exercises the real ignore rules without changing
	// the workspace's index, commits, configuration, or existing run artifacts.
	scratch := t.TempDir()
	git := func(args ...string) ([]byte, error) {
		cmd := exec.Command("git", args...)
		cmd.Dir = scratch
		cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_DIR="+filepath.Join(scratch, ".git"), "GIT_WORK_TREE="+scratch, "GIT_INDEX_FILE="+filepath.Join(scratch, ".git", "index"))
		return cmd.CombinedOutput()
	}
	if b, err := git("init", "--quiet", "--template="); err != nil {
		t.Fatalf("git init: %v %s", err, b)
	}
	if err := os.WriteFile(filepath.Join(scratch, ".gitignore"), ignore, 0600); err != nil {
		t.Fatal(err)
	}
	r := health.NewRun("synthetic", "test")
	s, err := runstore.New(scratch, &r)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	now := r.UpdatedAt
	r.EndedAt, r.Outcome = &now, "failed"
	r.RebuildCoverage()
	if err := s.Save(&r); err != nil {
		t.Fatal(err)
	}
	if err := s.PublishReport(report.Markdown); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{s.Path(), s.ReportPath()} {
		rel, err := filepath.Rel(scratch, path)
		if err != nil {
			t.Fatal(err)
		}
		if b, err := git("check-ignore", "--no-index", "--", rel); err != nil || strings.TrimSpace(string(b)) != filepath.ToSlash(rel) {
			t.Fatalf("generated artifact is not ignored: %s %v %s", rel, err, b)
		}
	}
	for _, path := range []string{"README.md", "AGENTS.md", "internal/cli/integration_test.go", "internal/cli/testdata/fake-az.sh", "internal/report/testdata/run.json", "internal/report/testdata/mixed-report.md", "context/changes/subscription-resource-health-report/verification.md"} {
		b, err := git("check-ignore", "--no-index", "--", path)
		var exit *exec.ExitError
		if !errors.As(err, &exit) || exit.ExitCode() != 1 || len(b) != 0 {
			t.Fatalf("source/fixture unexpectedly ignored: %s %v %s", path, err, b)
		}
	}
	if b, err := git("ls-files", "--", "runs"); err != nil || len(b) != 0 {
		t.Fatal("generated run artifacts became tracked")
	}
	if b, err := git("status", "--porcelain", "--untracked-files=all"); err != nil || strings.Contains(string(b), "runs/") {
		t.Fatal("generated run artifacts visible in ordinary Git status")
	}
}
