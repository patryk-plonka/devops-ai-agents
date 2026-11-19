package runstore

import (
	"encoding/json"
	"html"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/patryk-plonka/devops-ai-agents/internal/health"
	"github.com/patryk-plonka/devops-ai-agents/internal/report"
)

func TestAdvisorPublicationUsesCommittedPageTimes(t *testing.T) {
	const sub = "11111111-1111-1111-1111-111111111111"
	instance := "/subscriptions/" + sub + "/providers/Microsoft.Advisor/recommendations/synthetic-instance"
	reference := time.Date(2026, 11, 18, 8, 0, 0, 0, time.UTC)
	prepared, firstPage, secondPage := reference.Add(time.Minute), reference.Add(2*time.Minute), reference.Add(3*time.Minute)
	input := health.AdvisorInput{ID: instance, Status: "New"}
	for _, tc := range []struct {
		name, status, reason, wording string
		pages                         []health.Page
		wantRow                       bool
	}{
		{"empty", "completed", "", "The completed cached list contained no recommendations.", []health.Page{{CollectedAt: firstPage}}, false},
		{"duplicate final page", "completed", "", "### New recommendations", []health.Page{
			{CollectedAt: firstPage, Advisor: []health.AdvisorInput{input}}, {CollectedAt: secondPage, Advisor: []health.AdvisorInput{input}},
		}, true},
		{"failure before a page", "failed", "access_denied", "The cached list could not be read (access_denied).", nil, false},
		{"partial", "partial", "unsafe_continuation", "The cached list could not be fully checked (unsafe_continuation).", []health.Page{
			{CollectedAt: firstPage, Advisor: []health.AdvisorInput{input}},
		}, true},
		{"unattempted", "not_checked", "not_attempted", "Advisor collection was not attempted.", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := health.NewRun("synthetic", "test")
			s, err := New(t.TempDir(), &r)
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			r.Scope = &health.Scope{SubscriptionID: sub}
			q := health.AdvisorQuery(reference)
			if tc.status != "not_checked" {
				q.RequestedAt = &prepared
			}
			r.Sources["advisor"] = health.Source{Status: tc.status, Reason: tc.reason, Query: &q}
			for _, p := range tc.pages {
				r.NormalizePage("advisor", p)
			}
			r.Outcome = tc.status
			if tc.status == "not_checked" {
				r.Outcome = "cancelled"
			}
			ended := reference.Add(time.Hour)
			r.EndedAt = &ended
			r.RebuildCoverage()
			if err := s.Save(&r); err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(s.Path())
			if err != nil {
				t.Fatal(err)
			}
			var saved health.Run
			if err := json.Unmarshal(raw, &saved); err != nil {
				t.Fatal(err)
			}
			r.Sources["advisor"] = health.Source{}
			if err := s.PublishReport(report.Markdown); err != nil {
				t.Fatal(err)
			}
			b, err := os.ReadFile(s.ReportPath())
			if err != nil {
				t.Fatal(err)
			}
			_, section, _ := strings.Cut(html.UnescapeString(string(b)), "## Azure Advisor recommendations")
			section, _, _ = strings.Cut(section, "\n## ")
			wantPrepared := "not recorded"
			if q.RequestedAt != nil {
				wantPrepared = prepared.Format(time.RFC3339)
			}
			for _, want := range []string{tc.wording, "Collection status: " + tc.status, "Cached list scope: selected subscription " + sub, "Operation: Recommendations.List", "API version: 2026-10-01", "Query prepared (UTC): " + wantPrepared} {
				if !strings.Contains(section, want) {
					t.Fatalf("Advisor section missing %q", want)
				}
			}
			if (strings.Contains(section, "contained no recommendations") != (tc.name == "empty")) ||
				(strings.Contains(section, health.CanonicalID(instance)) != tc.wantRow) {
				t.Fatal("wrong empty wording or retained rows")
			}
			source := saved.Sources["advisor"]
			if source.Pages != len(tc.pages) {
				t.Fatal("wrong saved page count")
			}
			if len(tc.pages) == 0 {
				if strings.Contains(string(raw), "first_page_at") || strings.Contains(string(raw), "last_page_at") || !strings.Contains(section, "No Advisor pages were accepted.") {
					t.Fatal("zero-page provenance is missing or misleading")
				}
				return
			}
			for _, timeLabel := range []string{"First accepted page (UTC): " + firstPage.Format(time.RFC3339), "Last accepted page (UTC): " + tc.pages[len(tc.pages)-1].CollectedAt.Format(time.RFC3339)} {
				if !strings.Contains(section, timeLabel) {
					t.Fatalf("Advisor section missing %s", timeLabel)
				}
			}
			if tc.wantRow {
				if len(saved.Evidence) != 1 || !saved.Evidence[0].CollectedAt.Equal(firstPage) {
					t.Fatal("retained evidence collection time was replaced")
				}
				if !strings.Contains(string(b), "Collected (UTC): "+firstPage.Format(time.RFC3339)) {
					t.Fatal("evidence index missing original collection time")
				}
			}
		})
	}
}
