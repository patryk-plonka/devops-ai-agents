package runstore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/patryk-plonka/devops-ai-agents/internal/health"
	"github.com/patryk-plonka/devops-ai-agents/internal/report"
)

func TestOperationalAndLegacySnapshotsPublishCommittedFacts(t *testing.T) {
	for _, fixture := range []string{"run.json", "operational/run.json"} {
		t.Run(fixture, func(t *testing.T) {
			b, err := os.ReadFile("../report/testdata/" + fixture)
			if err != nil {
				t.Fatal(err)
			}
			var run health.Run
			if json.Unmarshal(b, &run) != nil {
				t.Fatal("invalid fixture")
			}
			s, err := New(t.TempDir(), &run)
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			if err := s.Save(&run); err != nil {
				t.Fatal(err)
			}
			committed, err := os.ReadFile(s.Path())
			if err != nil {
				t.Fatal(err)
			}
			var saved health.Run
			if json.Unmarshal(committed, &saved) != nil {
				t.Fatal("bad saved JSON")
			}
			if !reflect.DeepEqual(saved, run) {
				t.Fatal("normalized facts lost in Save/JSON round trip")
			}
			// Publication must use stored facts even after the mutable caller is changed.
			run.Evidence = nil
			run.Findings = nil
			run.Sources = nil
			if err := s.PublishReport(report.Markdown); err != nil {
				t.Fatal(err)
			}
			published, err := os.ReadFile(s.ReportPath())
			if err != nil {
				t.Fatal(err)
			}
			expected, err := report.Markdown(saved)
			if err != nil || !bytes.Equal(published, expected) {
				t.Fatal("report did not use complete committed snapshot")
			}
			text := html.UnescapeString(string(published))
			if !strings.Contains(text, "Run ID: "+saved.ID) || !strings.Contains(text, fmt.Sprintf("Revision: %d", saved.Revision)) {
				t.Fatal("report run/revision mismatch")
			}
			if fixture == "operational/run.json" {
				if saved.SchemaVersion != 2 || saved.Sources["service_health"].Query == nil || saved.Sources["azure_monitor"].Query == nil {
					t.Fatal("new query metadata lost")
				}
				service, monitor := false, false
				for _, e := range saved.Evidence {
					service = service || e.ServiceHealth != nil
					monitor = monitor || e.Monitor != nil
					if !strings.Contains(text, e.ID) {
						t.Fatal("stored evidence missing from index")
					}
				}
				if !service || !monitor || !strings.Contains(text, "workflow response: Closed") || !strings.Contains(text, "queryStartTime: 10/16/2026") {
					t.Fatal("operational facts/provenance lost")
				}
			}
			saved.Findings[0].EvidenceIDs = []string{"nonexistent-evidence"}
			if err := s.Save(&saved); err != nil {
				t.Fatal(err)
			}
			if err := s.PublishReport(report.Markdown); err == nil {
				t.Fatal("unsupported reference published")
			}
			after, err := os.ReadFile(s.ReportPath())
			if err != nil || !bytes.Equal(after, published) {
				t.Fatal("failed publication damaged prior report")
			}
		})
	}
}
