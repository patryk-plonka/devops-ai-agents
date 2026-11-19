package health

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

const (
	advisorSubscription = "/subscriptions/" + testSubscription
	advisorSuffix       = "/providers/Microsoft.Advisor/recommendations/"
	advisorTestType     = "abcdef01-2345-6789-abcd-ef0123456789"
)

var advisorCollected = time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)

func advisorInput(name string) AdvisorInput {
	resource := resourcePrefix + name
	return AdvisorInput{
		ID: resource + advisorSuffix + "opaque-" + name, TargetResourceID: resource,
		TypeID: advisorTestType, Category: "Cost", Impact: "High", Status: "New",
		LastRefreshed: ptr(advisorCollected.Add(-time.Hour)), LastUpdated: ptr(advisorCollected.Add(-48 * time.Hour)),
	}
}

func advisorPage(inputs ...AdvisorInput) Page {
	return Page{Advisor: inputs, CollectedAt: advisorCollected}
}

func TestAdvisorScopedEvidenceAndPrivacy(t *testing.T) {
	valid := advisorInput("vm")
	valid.ID = strings.ToUpper(valid.ID)
	valid.TypeID = strings.ToUpper(valid.TypeID)
	valid.LastRefreshed = ptr(valid.LastRefreshed.In(time.FixedZone("west", -3600)))
	group := advisorInput("group")
	group.ID = advisorSubscription + "/resourceGroups/Group" + advisorSuffix + "opaque"
	subscription := advisorInput("subscription")
	subscription.ID = advisorSubscription + advisorSuffix + "opaque"
	nested := advisorInput("nested")
	nested.TargetResourceID += "/extensions/agent/providers/Microsoft.Insights/diagnosticSettings/setting"
	nested.ID = nested.TargetResourceID + advisorSuffix + "opaque"
	inputs := []AdvisorInput{valid, group, subscription, nested}
	for _, id := range []string{
		strings.Replace(valid.ID, strings.ToUpper(testSubscription), otherSubscription, 1),
		advisorSubscription + "/resourceGroups/.." + advisorSuffix + "invalid-canary",
		advisorSubscription + "/resourceGroups/dangling/providers/Microsoft.Compute/virtualMachines" + advisorSuffix + "invalid-canary",
		advisorSubscription + "/wrong/scope" + advisorSuffix + "invalid-canary",
		advisorSubscription + advisorSuffix, advisorSubscription + advisorSuffix + "..",
		valid.ID + "/invalid-canary", valid.ID + "%2Finvalid-canary",
		valid.ID + "?sig=invalid-canary", valid.ID + "/\x1binvalid-canary",
		advisorSubscription + advisorSuffix + "eyJabc.def.invalid-canary",
		strings.Replace(valid.ID, "MICROSOFT.ADVISOR", "OTHER.PROVIDER", 1),
	} {
		inputs = append(inputs, AdvisorInput{ID: id, TargetResourceID: resourcePrefix + "discarded-canary"})
	}
	unknown := advisorInput("unknown")
	unknown.TargetResourceID = strings.Replace(resourcePrefix, testSubscription, otherSubscription, 1) + "foreign-target-canary"
	unknown.TypeID, unknown.Category, unknown.Impact, unknown.Status = "type-canary", "Password=category-canary", "high", "\x1b[32mNew"
	unknown.LastRefreshed, unknown.LastUpdated = nil, ptr(time.Time{})
	conflict := advisorInput("mismatched")
	conflict.TargetResourceID = resourcePrefix + "conflicting-target-canary"
	missing := advisorInput("missing")
	missing.TargetResourceID = ""
	unsafe := advisorInput("unsafe")
	unsafe.TargetResourceID = resourcePrefix + "vm?sig=unsafe-canary"
	inputs = append(inputs, unknown, conflict, missing, unsafe)

	r := scopedRun()
	r.NormalizePage("advisor", advisorPage(inputs...))
	if len(r.Evidence) != 8 || len(r.AdvisorInstances) != 8 || len(r.Findings) != 7 || r.Sources["advisor"].RejectedRecords != 12 {
		t.Fatalf("unexpected acceptance or classification: %+v", r)
	}
	for i, input := range []AdvisorInput{valid, group, subscription, nested} {
		e := r.Evidence[i]
		if e.EntityID != input.ID || e.SubscriptionID != testSubscription || e.ResourceID != input.TargetResourceID ||
			e.Source != "advisor" || e.APIVersion != "2026-10-01" || e.Operation != AdvisorListOperation ||
			e.Advisor == nil || e.Advisor.TypeID != advisorTestType || e.Advisor.Category != "Cost" || e.Advisor.Impact != "High" || e.Advisor.Status != "New" ||
			*e.Advisor.LastRefreshed != input.LastRefreshed.UTC() || *e.Advisor.LastUpdated != input.LastUpdated.UTC() || e.CollectedAt != advisorCollected {
			t.Fatalf("incorrect saved evidence: %+v", e)
		}
	}
	if facts := r.Evidence[4].Advisor; facts.Status != "unknown" || facts.Category != "unknown" || facts.Impact != "unknown" ||
		facts.TypeID != "" || facts.LastRefreshed != nil || facts.LastUpdated != nil || !hasGap(r, "unknown_advisor_status") {
		t.Fatalf("unexpected facts for unknown fields: %+v", facts)
	}
	for i, want := range []string{"unknown_resource_attribution", "conflicting_resource_attribution", "unknown_resource_attribution", "unknown_resource_attribution"} {
		e := r.Evidence[i+4]
		if e.ResourceID != "" || e.Advisor.AttributionLimitation != want || !hasGap(r, want) {
			t.Fatalf("uncertain resource became a target: %+v", e)
		}
	}
	saved, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	for _, excluded := range []string{"canary", otherSubscription, "\\u001b"} {
		if strings.Contains(string(saved), excluded) {
			t.Fatalf("unsafe or foreign content reached JSON: %s", excluded)
		}
	}
	assertReferences(t, r)
}

func TestAdvisorARMPathValidation(t *testing.T) {
	group := advisorSubscription + "/resourceGroups/Grüppe_1.(Dev)"
	badGroup := advisorSubscription + "/resourceGroups/not a group"
	incompleteProvider := group + "/providers/Microsoft.Compute/providers/Microsoft.Storage/storageAccounts/demo"
	incompleteExtension := resourcePrefix + "vm/providers/Microsoft.Insights/providers/Microsoft.Storage/storageAccounts/demo"
	blankNamespace := group + "/providers/ /virtualMachines/vm"
	blankType := group + "/providers/Microsoft.Compute/ /vm"
	blankName := resourcePrefix + "\u00a0"
	spacedName := group + "/providers/Microsoft.CustomProviders/resourceProviders/Example Provider"
	for _, tc := range []struct {
		name, parent, target, wantTarget, wantLimitation string
		rejected                                         bool
	}{
		{name: "explicit resource group", parent: group, target: group, wantTarget: group},
		{name: "subscription parent with group target", parent: advisorSubscription, target: group, wantTarget: group},
		{name: "no inferred group target", parent: group, wantLimitation: "unknown_resource_attribution"},
		{name: "invalid group parent", parent: badGroup, rejected: true},
		{name: "group parent with trailing period", parent: group + ".", rejected: true},
		{name: "group parent over length", parent: advisorSubscription + "/resourceGroups/" + strings.Repeat("é", 91), rejected: true},
		{name: "invalid group within resource parent", parent: badGroup + "/providers/Microsoft.Compute/virtualMachines/vm", rejected: true},
		{name: "provider without resource", parent: incompleteProvider, rejected: true},
		{name: "extension without resource", parent: incompleteExtension, rejected: true},
		{name: "blank namespace in parent", parent: blankNamespace, rejected: true},
		{name: "blank type in parent", parent: blankType, rejected: true},
		{name: "blank name in parent", parent: blankName, rejected: true},
		{name: "resource name with an internal space", parent: spacedName, target: spacedName, wantTarget: spacedName},
		{name: "malformed resource target", parent: group, target: incompleteProvider, wantLimitation: "unknown_resource_attribution"},
		{name: "blank namespace in target", parent: group, target: blankNamespace, wantLimitation: "unknown_resource_attribution"},
		{name: "blank type in target", parent: group, target: blankType, wantLimitation: "unknown_resource_attribution"},
		{name: "blank name in target", parent: group, target: blankName, wantLimitation: "unknown_resource_attribution"},
		{name: "malformed group target", parent: group, target: badGroup, wantLimitation: "unknown_resource_attribution"},
		{name: "foreign group target", parent: group, target: strings.Replace(group, testSubscription, otherSubscription, 1), wantLimitation: "unknown_resource_attribution"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := advisorInput("vm")
			input.ID, input.TargetResourceID = tc.parent+advisorSuffix+"opaque:recommendation+0", tc.target
			r := scopedRun()
			r.NormalizePage("advisor", advisorPage(input))
			rejected := r.Sources["advisor"].RejectedRecords
			if tc.rejected {
				if rejected != 1 || len(r.Evidence) != 0 || len(r.Findings) != 0 {
					t.Fatalf("invalid parent reached evidence or findings: rejected=%d evidence=%v findings=%v", rejected, r.Evidence, r.Findings)
				}
				return
			}
			if rejected != 0 || len(r.Evidence) != 1 || len(r.Findings) != 1 {
				t.Fatalf("valid instance lost: rejected=%d evidence=%v findings=%v", rejected, r.Evidence, r.Findings)
			}
			e := r.Evidence[0]
			if e.EntityID != input.ID || e.ResourceID != tc.wantTarget || e.Advisor.AttributionLimitation != tc.wantLimitation ||
				!strings.EqualFold(r.Findings[0].ResourceID, tc.wantTarget) {
				t.Fatalf("wrong identity or attribution: evidence=%+v advisor=%+v", e, e.Advisor)
			}
			assertReferences(t, r)
		})
	}
}

func TestAdvisorExactEnumsAndDistinctInstances(t *testing.T) {
	r := scopedRun()
	inputs := []AdvisorInput{advisorInput("a"), advisorInput("b"), advisorInput("untyped-a"), advisorInput("untyped-b"),
		advisorInput("postponed"), advisorInput("dismissed"), advisorInput("completed"), advisorInput("missing-status")}
	inputs[1].Category, inputs[1].Impact = "HighAvailability", "Medium"
	inputs[2].TypeID, inputs[2].Category, inputs[2].Impact = "", "Performance", "Low"
	inputs[3].TypeID, inputs[3].Category, inputs[3].Impact = "invalid", "Security", ""
	inputs[4].Status, inputs[4].Category = "Postponed", "OperationalExcellence"
	inputs[5].Status, inputs[6].Status, inputs[7].Status = "Dismissed", "Completed", ""
	inputs[7].Category = ""
	r.NormalizePage("advisor", advisorPage(inputs...))
	if len(r.AdvisorInstances) != len(inputs) || len(r.Findings) != 4 || r.Sources["advisor"].RejectedRecords != 0 {
		t.Fatalf("instances merged or valid unclassified rows rejected: %+v", r)
	}
	for i, input := range inputs {
		entityID, resourceID := strings.ToLower(input.ID), strings.ToLower(input.TargetResourceID)
		idx := slices.IndexFunc(r.AdvisorInstances, func(row AdvisorInstance) bool { return row.EntityID == entityID })
		if idx < 0 {
			t.Fatal("instance missing from review")
		}
		row := r.AdvisorInstances[idx]
		if row.ResourceID != resourceID || !slices.Equal(row.EvidenceIDs, []string{r.Evidence[i].ID}) {
			t.Fatalf("row lost instance-specific data: %+v", row)
		}
		want := AdvisorFacts{TypeID: advisorTestType, Category: input.Category, Impact: input.Impact, Status: input.Status,
			LastRefreshed: input.LastRefreshed, LastUpdated: input.LastUpdated}
		switch i {
		case 2:
			want.TypeID = ""
		case 3:
			want.TypeID, want.Impact = "", "unknown"
		case 7:
			want.Category, want.Status = "unknown", "unknown"
		}
		if !reflect.DeepEqual(row.AdvisorFacts, want) || (row.FindingID != "") != (input.Status == "New") {
			t.Fatalf("got %+v, want %+v", row, want)
		}
	}
	assertReferences(t, r)
}

func TestAdvisorCaseEquivalentDuplicatesProduceStableViews(t *testing.T) {
	ascii := advisorInput("vm")
	asciiVariant := ascii
	asciiVariant.ID, asciiVariant.TargetResourceID = strings.ToUpper(ascii.ID), strings.ToUpper(ascii.TargetResourceID)
	greek := ascii
	greek.ID = strings.Replace(ascii.ID, "/Group/", "/ΟΣ/", 1)
	greek.TargetResourceID = strings.Replace(ascii.TargetResourceID, "/Group/", "/ος/", 1)
	greekVariant := ascii
	greekVariant.ID = strings.Replace(ascii.ID, "/Group/", "/ος/", 1)
	greekVariant.TargetResourceID = strings.Replace(ascii.TargetResourceID, "/Group/", "/ΟΣ/", 1)
	for _, tc := range []struct {
		name           string
		first, variant AdvisorInput
	}{
		{"ASCII", ascii, asciiVariant},
		{"Greek sigma", greek, greekVariant},
	} {
		t.Run(tc.name, func(t *testing.T) {
			first, second := advisorPage(tc.first), advisorPage(tc.variant)
			second.CollectedAt = first.CollectedAt.Add(time.Hour)
			forward, reversed := scopedRun(), scopedRun()
			for i, r := range []*Run{&forward, &reversed} {
				pages := []Page{first, second}
				if i == 1 {
					slices.Reverse(pages)
				}
				for _, page := range pages {
					r.NormalizePage("advisor", page)
				}
				if len(r.Evidence) != 1 || len(r.AdvisorInstances) != 1 || len(r.Findings) != 1 {
					t.Fatalf("case variants should be one observation and finding: evidence=%d rows=%d findings=%d",
						len(r.Evidence), len(r.AdvisorInstances), len(r.Findings))
				}
				e, original := r.Evidence[0], pages[0].Advisor[0]
				if e.EntityID != original.ID || e.ResourceID != original.TargetResourceID || e.CollectedAt != pages[0].CollectedAt ||
					e.Advisor.AttributionLimitation != "" {
					t.Fatalf("first observation or case-insensitive attribution lost: %+v", e)
				}
				assertReferences(t, *r)
			}
			if forward.Evidence[0].ID != reversed.Evidence[0].ID || !reflect.DeepEqual(forward.AdvisorInstances, reversed.AdvisorInstances) ||
				!reflect.DeepEqual(forward.Findings, reversed.Findings) || !reflect.DeepEqual(forward.Gaps, reversed.Gaps) {
				t.Fatal("reversing equivalent observations changed evidence identity or derived views")
			}

			findingID := forward.Findings[0].ID
			refreshed := tc.variant
			refreshed.LastRefreshed = ptr(refreshed.LastRefreshed.Add(time.Minute))
			forward.NormalizePage("advisor", advisorPage(refreshed))
			if len(forward.Evidence) != 2 || len(forward.AdvisorInstances) != 1 || len(forward.Findings) != 1 ||
				forward.Findings[0].ID != findingID || len(forward.Findings[0].EvidenceIDs) != 2 ||
				!reflect.DeepEqual(forward.AdvisorInstances[0].LastRefreshed, refreshed.LastRefreshed) {
				t.Fatal("a refresh of the same case-insensitive instance changed its identity or lost evidence")
			}
		})
	}
}

func TestAdvisorUnrelatedCaseFoldClassesStayDistinct(t *testing.T) {
	first := advisorInput("vm")
	first.ID = strings.Replace(first.ID, "/Group/", "/i/", 1)
	first.TargetResourceID = strings.Replace(first.TargetResourceID, "/Group/", "/i/", 1)
	second := first
	second.ID = strings.Replace(first.ID, "/i/", "/İ/", 1)
	second.TargetResourceID = strings.Replace(first.TargetResourceID, "/i/", "/İ/", 1)
	r := scopedRun()
	r.NormalizePage("advisor", advisorPage(first, second))
	if len(r.Evidence) != 2 || len(r.AdvisorInstances) != 2 || len(r.Findings) != 2 {
		t.Fatalf("distinct ARM IDs incorrectly merged: evidence=%d rows=%d findings=%d", len(r.Evidence), len(r.AdvisorInstances), len(r.Findings))
	}
	assertReferences(t, r)
}

func TestAdvisorRebuildOrdersVersionsByStateChange(t *testing.T) {
	first := advisorInput("vm")
	refreshed := first
	refreshed.ID, refreshed.TargetResourceID = strings.ToUpper(first.ID), strings.ToUpper(first.TargetResourceID)
	refreshed.LastRefreshed = ptr(first.LastRefreshed.Add(30 * time.Minute))
	olderDismissed := refreshed
	olderDismissed.Status, olderDismissed.LastUpdated = "Dismissed", ptr(first.LastUpdated.Add(-time.Hour))
	newerDismissed := first
	newerDismissed.Status, newerDismissed.LastUpdated = "Dismissed", ptr(first.LastUpdated.Add(time.Hour))
	tiedDismissed := refreshed
	tiedDismissed.Status = "Dismissed"
	untimedDismissed := tiedDismissed
	untimedDismissed.LastUpdated = nil
	conflictingTarget := refreshed
	conflictingTarget.TargetResourceID = ""
	initial := scopedRun()
	initial.NormalizePage("advisor", advisorPage(first))
	stableFindingID := initial.Findings[0].ID

	for _, tc := range []struct {
		name, status string
		input        AdvisorInput
		evaluated    *time.Time
		conflict     bool
		findingRefs  int
	}{
		{"later evaluation", "New", refreshed, refreshed.LastRefreshed, false, 2},
		{"later evaluation on older status", "New", olderDismissed, first.LastRefreshed, false, 1},
		{"later state change", "Dismissed", newerDismissed, newerDismissed.LastRefreshed, false, 0},
		{"tied status", "unknown", tiedDismissed, nil, true, 0},
		{"untimed contradiction", "unknown", untimedDismissed, nil, true, 0},
		{"contradictory attribution", "unknown", conflictingTarget, nil, true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := advisorPage(first), advisorPage(tc.input)
			b.CollectedAt = a.CollectedAt.Add(time.Hour)
			r, reversed := scopedRun(), scopedRun()
			for i, run := range []*Run{&r, &reversed} {
				pages := []Page{a, b}
				if i == 1 {
					slices.Reverse(pages)
				}
				for _, page := range pages {
					run.NormalizePage("advisor", page)
				}
				if len(run.AdvisorInstances) != 1 || len(run.Evidence) != 2 {
					t.Fatalf("native identity or source times lost: %+v", run)
				}
				row := run.AdvisorInstances[0]
				wantIDs := []string{run.Evidence[0].ID, run.Evidence[1].ID}
				slices.Sort(wantIDs)
				if row.Status != tc.status || !reflect.DeepEqual(row.LastRefreshed, tc.evaluated) || !slices.Equal(row.EvidenceIDs, wantIDs) {
					t.Fatalf("wrong version chosen: %+v", row)
				}
				if tc.conflict {
					idx := slices.IndexFunc(run.Gaps, func(g Gap) bool { return g.Reason == "conflicting_advisor_evidence" })
					if idx < 0 || !slices.Equal(run.Gaps[idx].EvidenceIDs, wantIDs) || row.ResourceID != "" || row.TypeID != "" ||
						row.Category != "unknown" || row.Impact != "unknown" {
						t.Fatalf("conflict hidden or given a determinate summary: %+v", run)
					}
				}
				if tc.findingRefs == 0 {
					if len(run.Findings) != 0 || row.FindingID != "" {
						t.Fatal("unconfirmed New instance entered findings")
					}
				} else if len(run.Findings) != 1 || row.FindingID != stableFindingID || run.Findings[0].ID != stableFindingID ||
					len(run.Findings[0].EvidenceIDs) != tc.findingRefs {
					t.Fatalf("finding identity or references changed: %+v", run.Findings)
				}
				assertReferences(t, *run)
			}
			if !reflect.DeepEqual(r.AdvisorInstances, reversed.AdvisorInstances) || !reflect.DeepEqual(r.Findings, reversed.Findings) || !reflect.DeepEqual(r.Gaps, reversed.Gaps) {
				t.Fatal("reversing page order changed derived Advisor views")
			}
		})
	}
}

func TestAdvisorSourceCheckpoints(t *testing.T) {
	unknown := AdvisorInput{ID: advisorSubscription + advisorSuffix + "unknown"}
	unrecognized := AdvisorInput{ID: advisorSubscription + advisorSuffix + "future", Status: "FutureStatus"}
	for _, tc := range []struct {
		name, wantStatus       string
		pages                  []Page
		code, wantUnclassified int
	}{
		{"failure before first page", "failed", nil, 1, 0},
		{"empty enumeration", "completed", []Page{advisorPage()}, 0, 0},
		{"duplicate and empty pages", "completed", []Page{advisorPage(advisorInput("vm")), advisorPage(advisorInput("vm")), advisorPage()}, 0, 0},
		{"mixed valid and rejected", "partial", []Page{advisorPage(advisorInput("vm"), AdvisorInput{})}, 3, 0},
		{"unknown and duplicate", "completed", []Page{advisorPage(unknown, unrecognized), advisorPage(unknown)}, 0, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := scopedRun()
			q := AdvisorQuery(advisorCollected.Add(-24 * time.Hour))
			r.Sources["advisor"] = Source{Query: &q}
			saved := []Run{}
			collect := func(_ context.Context, _ string, emit func(Page) error) error {
				if len(tc.pages) == 0 {
					return errors.New("unavailable")
				}
				for i, page := range tc.pages {
					page.CollectedAt = advisorCollected.Add(time.Duration(i) * time.Hour).In(time.FixedZone("east", 3600))
					if err := emit(page); err != nil {
						return err
					}
				}
				return nil
			}
			code, err := Check(context.Background(), &r, []NamedCollector{{Name: "advisor", Collect: collect}}, func(run *Run) error {
				b, err := json.Marshal(run)
				if err != nil {
					return err
				}
				if run.Sources["advisor"].Pages == 0 && (strings.Contains(string(b), "first_page_at") || strings.Contains(string(b), "last_page_at")) {
					t.Fatal("timestamps persisted before a page was accepted")
				}
				var snapshot Run
				if err := json.Unmarshal(b, &snapshot); err != nil {
					return err
				}
				saved = append(saved, snapshot)
				return nil
			})
			if err != nil || code != tc.code || len(saved) != 2+len(tc.pages) {
				t.Fatalf("outcome %d, checkpoints %d, error %v", code, len(saved), err)
			}
			for i, snapshot := range saved {
				s := snapshot.Sources["advisor"]
				if s.Query.Operation != AdvisorListOperation || s.Query.APIVersion != AdvisorAPI || s.Query.RequestedAt == nil ||
					!s.Query.ReferenceAt.Equal(q.ReferenceAt) || !s.Query.RequestedAt.Equal(*saved[0].Sources["advisor"].Query.RequestedAt) {
					t.Fatalf("checkpoint lost list provenance: %+v", s.Query)
				}
				pages := min(i, len(tc.pages))
				if pages > 0 && (s.Pages != pages || s.FirstPageAt == nil || *s.FirstPageAt != advisorCollected ||
					s.LastPageAt == nil || *s.LastPageAt != advisorCollected.Add(time.Duration(pages-1)*time.Hour)) {
					t.Fatalf("wrong page timing at checkpoint %d: %+v", i, s)
				}
			}
			final := saved[len(saved)-1]
			if final.Sources["advisor"].Status != tc.wantStatus || final.Coverage != "incomplete" {
				t.Fatalf("wrong coverage: %+v", final)
			}
			if got := final.Sources["advisor"].UnclassifiedRecords; got != tc.wantUnclassified {
				t.Fatalf("unclassified records %d, want %d", got, tc.wantUnclassified)
			}
			if tc.wantUnclassified > 0 && (len(final.Evidence) != 2 || len(final.Findings) != 0) {
				t.Fatal("unknown duplicates affected evidence or introduced findings")
			}
			if len(tc.pages) == 3 && (len(final.Evidence) != 1 || final.Evidence[0].CollectedAt != advisorCollected) {
				t.Fatal("duplicate changed the original evidence collection time")
			}
		})
	}
}

func TestAdvisorOutOfRangeSourceTimesDoNotBlockCheckpoint(t *testing.T) {
	parse := func(value string) *time.Time {
		t.Helper()
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			t.Fatal(err)
		}
		return &parsed
	}
	valid := advisorInput("valid")
	valid.LastRefreshed = parse("9999-12-31T22:59:59-01:00")
	valid.LastUpdated = parse("0000-01-01T01:00:00+01:00")
	inputs := []AdvisorInput{valid}
	for _, tc := range []struct{ name, value string }{
		{"above", "9999-12-31T23:59:59-01:00"},
		{"below", "0000-01-01T00:00:00+01:00"},
	} {
		refreshed, updated := advisorInput(tc.name+"-refresh"), advisorInput(tc.name+"-update")
		refreshed.LastRefreshed, updated.LastUpdated = parse(tc.value), parse(tc.value)
		inputs = append(inputs, refreshed, updated)
	}

	r := scopedRun()
	var checkpoint Run
	collect := func(_ context.Context, _ string, emit func(Page) error) error {
		return emit(advisorPage(inputs...))
	}
	code, err := Check(context.Background(), &r, []NamedCollector{{Name: "advisor", Collect: collect}}, func(run *Run) error {
		data, err := json.Marshal(run)
		if err == nil && run.Sources["advisor"].Status == "in_progress" && run.Sources["advisor"].Pages == 1 {
			return json.Unmarshal(data, &checkpoint)
		}
		return err
	})
	if s := r.Sources["advisor"]; err != nil || code != 3 || s.Status != "partial" || s.Reason != "invalid_records" {
		t.Fatalf("want a partial enumeration, got code=%d source=%+v err=%v", code, s, err)
	}
	if checkpoint.Sources["advisor"].RejectedRecords != 4 || len(checkpoint.Evidence) != 1 || len(checkpoint.Findings) != 1 {
		t.Fatalf("valid sibling or rejection count missing at the page checkpoint: %+v", checkpoint)
	}
	e := checkpoint.Evidence[0]
	if e.EntityID != valid.ID || e.Advisor.LastRefreshed == nil || *e.Advisor.LastRefreshed != valid.LastRefreshed.UTC() ||
		e.Advisor.LastUpdated == nil || *e.Advisor.LastUpdated != valid.LastUpdated.UTC() {
		t.Fatalf("valid boundary timestamps lost: %+v", e.Advisor)
	}
	assertReferences(t, checkpoint)
}
