package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/patryk-plonka/devops-ai-agents/internal/azure"
	"github.com/patryk-plonka/devops-ai-agents/internal/health"
	"github.com/patryk-plonka/devops-ai-agents/internal/report"
	"github.com/patryk-plonka/devops-ai-agents/internal/runstore"
)

func assertPublished(t *testing.T, root, path string) health.Run {
	t.Helper()
	r := snapshot(t, root)
	if filepath.Base(path) != "report.md" || filepath.Base(filepath.Dir(path)) != r.ID {
		t.Fatal("stdout does not identify this report")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := html.UnescapeString(string(b))
	for _, want := range []string{"Run ID: " + r.ID, fmt.Sprintf("Revision: %d", r.Revision), "Execution outcome: **" + r.Outcome + "**", "INCOMPLETE coverage"} {
		if !strings.Contains(text, want) {
			t.Fatalf("report/snapshot disagree: %q", want)
		}
	}
	for _, f := range r.Findings {
		for _, id := range f.EvidenceIDs {
			if !strings.Contains(text, "["+id+"](run.json)") {
				t.Fatal("unresolved finding")
			}
		}
	}
	return r
}
func TestPublishedReportMatchesCommittedSnapshot(t *testing.T) {
	a, out, _, _ := testApp(t)
	a.Inventory = func(_ context.Context, _ string, emit func(health.Page) error) error {
		return emit(health.Page{Resources: []health.ResourceInput{{ID: cliVM, Tags: map[string]*string{"team": stringPointer("consulting"), "password": stringPointer("secret-canary")}}}})
	}
	a.ResourceHealth = func(_ context.Context, _ string, emit func(health.Page) error) error {
		return emit(health.Page{Health: []health.HealthInput{{StatusID: cliVM + "/providers/Microsoft.ResourceHealth/availabilityStatuses/current", PropertiesPresent: true, State: stringPointer("Unavailable")}}})
	}
	if code := a.Run(context.Background(), []string{"health", "check", "--subscription", first}); code != 0 {
		t.Fatalf("code %d", code)
	}
	r := assertPublished(t, a.Directory, strings.TrimSpace(out.String()))
	if len(r.Findings) != 1 {
		t.Fatal("finding lost")
	}
	b, err := os.ReadFile(strings.TrimSpace(out.String()))
	if err != nil {
		t.Fatal(err)
	}
	text := html.UnescapeString(string(b))
	if !strings.Contains(text, "team=consulting") || !strings.Contains(text, "password=[REDACTED]") || strings.Contains(text, "secret-canary") {
		t.Fatal("tags not filtered/preserved")
	}
}

type publicationStore struct {
	Storage
	finalWriteFailure bool
}

func (s publicationStore) Save(r *health.Run) error {
	if s.finalWriteFailure && r.EndedAt != nil {
		return errors.New("secret-canary")
	}
	return s.Storage.Save(r)
}
func (s publicationStore) PublishReport(render func(health.Run) ([]byte, error)) error {
	return errors.New("secret-canary")
}
func TestFinalizationFailuresExposeRetainedArtifacts(t *testing.T) {
	for _, kind := range []string{"final JSON write", "render", "publication", "publication with older report", "final JSON with older report"} {
		t.Run(kind, func(t *testing.T) {
			a, out, diag, _ := testApp(t)
			older := strings.Contains(kind, "older")
			a.NewStore = func(root string, r *health.Run) (Storage, error) {
				s, err := runstore.New(root, r)
				if err != nil {
					return nil, err
				}
				if older {
					now := r.UpdatedAt
					r.EndedAt = &now
					r.Outcome = "failed"
					if err := s.Save(r); err != nil {
						t.Fatal(err)
					}
					if err := s.PublishReport(report.Markdown); err != nil {
						t.Fatal(err)
					}
					r.EndedAt = nil
					r.Outcome = "in_progress"
				}
				if strings.HasPrefix(kind, "final JSON") {
					return publicationStore{Storage: s, finalWriteFailure: true}, nil
				}
				if strings.HasPrefix(kind, "publication") {
					return publicationStore{Storage: s}, nil
				}
				return s, nil
			}
			if kind == "render" {
				a.Render = func(health.Run) ([]byte, error) { return nil, errors.New("secret-canary") }
			}
			code := a.Run(context.Background(), []string{"health", "check", "--subscription", first})
			if code != 1 || out.Len() != 0 || strings.Contains(diag.String(), "secret-canary") || strings.Contains(diag.String(), "Report saved:") || !strings.Contains(diag.String(), "Last saved snapshot:") {
				t.Fatalf("failure feedback: code %d %s", code, diag)
			}
			if older && !strings.Contains(diag.String(), "older report exists") {
				t.Fatal("stale report not disclosed")
			}
			if !older && !strings.Contains(diag.String(), "No older report exists") {
				t.Fatal("report absence not disclosed")
			}
			r := snapshot(t, a.Directory)
			if strings.HasPrefix(kind, "final JSON") && r.EndedAt != nil {
				t.Fatal("failed final write replaced checkpoint")
			}
			if !strings.HasPrefix(kind, "final JSON") && r.Outcome != "failed" {
				t.Fatal("publication failure missing from snapshot")
			}
		})
	}
}
func TestCancelledAtEveryStageFinalizesWithoutMoreAzure(t *testing.T) {
	for _, stage := range []string{"before discovery", "discovery", "validation", "inventory next page", "health next page", "selection"} {
		t.Run(stage, func(t *testing.T) {
			a, out, diag, _ := testApp(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			collect := func(_ context.Context, _ string, emit func(health.Page) error) error {
				return emit(health.Page{Resources: []health.ResourceInput{{ID: cliVM}}})
			}
			switch stage {
			case "before discovery":
				cancel()
				a.Discover = func(context.Context) ([]azure.Account, error) { t.Fatal("discovery after cancel"); return nil, nil }
			case "discovery":
				a.Discover = func(ctx context.Context) ([]azure.Account, error) { cancel(); return nil, ctx.Err() }
			case "validation":
				a.Validate = func(ctx context.Context, _ azure.Account) error { cancel(); return ctx.Err() }
			case "inventory next page":
				a.Inventory = func(ctx context.Context, id string, emit func(health.Page) error) error {
					if err := collect(ctx, id, emit); err != nil {
						return err
					}
					cancel()
					return ctx.Err()
				}
			case "health next page":
				a.Inventory = collect
				a.ResourceHealth = func(ctx context.Context, _ string, emit func(health.Page) error) error {
					if err := emit(health.Page{Health: []health.HealthInput{{StatusID: cliVM + "/providers/Microsoft.ResourceHealth/availabilityStatuses/current", PropertiesPresent: true, State: stringPointer("Unavailable")}}}); err != nil {
						return err
					}
					cancel()
					return ctx.Err()
				}
			case "selection":
				reader, writer := io.Pipe()
				defer reader.Close()
				defer writer.Close()
				a.Terminal = true
				a.In = reader
				a.Discover = func(context.Context) ([]azure.Account, error) {
					return []azure.Account{cached(first, "Client"), cached(second, "Client")}, nil
				}
				// The input blocks after signalling entry; cancellation must not require EOF.
				a.In = cancelReader{Reader: reader, cancel: cancel}
			}
			if stage != "inventory next page" && stage != "health next page" {
				a.Inventory = func(context.Context, string, func(health.Page) error) error {
					t.Fatal("collection after interruption")
					return nil
				}
			}
			if stage != "health next page" {
				a.ResourceHealth = func(context.Context, string, func(health.Page) error) error {
					t.Fatal("second source after interruption")
					return nil
				}
			}
			if code := a.Run(ctx, []string{"health", "check", "--subscription", "Client"}); code != 130 {
				t.Fatalf("code %d: %s", code, diag)
			}
			r := assertPublished(t, a.Directory, strings.TrimSpace(out.String()))
			if r.Outcome != "cancelled" {
				t.Fatal("interruption not saved")
			}
			if strings.Contains(stage, "next page") && len(r.Evidence) == 0 {
				t.Fatal("checkpoint evidence lost")
			}
			if stage == "inventory next page" && r.Sources["resource_health"].Reason != "not_attempted" {
				t.Fatal("unattempted stage lost")
			}
		})
	}
}

type cancelReader struct {
	io.Reader
	cancel context.CancelFunc
}

func (r cancelReader) Read(b []byte) (int, error) { r.cancel(); return r.Reader.Read(b) }

func TestInterruptedFinalWriteFailureRetainsCheckpoint(t *testing.T) {
	a, out, diag, _ := testApp(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a.NewStore = func(root string, r *health.Run) (Storage, error) {
		s, err := runstore.New(root, r)
		return publicationStore{Storage: s, finalWriteFailure: true}, err
	}
	a.Inventory = func(ctx context.Context, _ string, emit func(health.Page) error) error {
		if err := emit(health.Page{Resources: []health.ResourceInput{{ID: cliVM}}}); err != nil {
			return err
		}
		cancel()
		return ctx.Err()
	}
	if code := a.Run(ctx, []string{"health", "check", "--subscription", first}); code != 130 || out.Len() != 0 || !strings.Contains(diag.String(), "Final run write failed") || !strings.Contains(diag.String(), "Last saved snapshot:") {
		t.Fatalf("code %d %s", code, diag)
	}
	r := snapshot(t, a.Directory)
	if len(r.Evidence) != 1 || r.EndedAt != nil {
		t.Fatal("checkpoint lost or finalization claimed")
	}
	b, _ := json.Marshal(r)
	if strings.Contains(string(b)+diag.String(), "secret-canary") {
		t.Fatal("storage error leaked")
	}
}

func TestInterruptDuringRenderingFinalizesCancelledRevision(t *testing.T) {
	a, out, diag, _ := testApp(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	renders := 0
	a.Render = func(r health.Run) ([]byte, error) {
		renders++
		if renders == 1 {
			cancel()
		}
		return report.Markdown(r)
	}
	if code := a.Run(ctx, []string{"health", "check", "--subscription", first}); code != 130 {
		t.Fatalf("code %d: %s", code, diag)
	}
	r := assertPublished(t, a.Directory, strings.TrimSpace(out.String()))
	if r.Outcome != "cancelled" || renders != 2 {
		t.Fatal("late interruption produced completed output")
	}
}
func TestInterruptedInitializationReportsNoArtifact(t *testing.T) {
	a, out, diag, calls := testApp(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a.NewStore = func(string, *health.Run) (Storage, error) { cancel(); return nil, errors.New("secret-canary") }
	if code := a.Run(ctx, []string{"health", "check", "--subscription", first}); code != 130 || out.Len() != 0 || *calls != 0 || !strings.Contains(diag.String(), "no artifact could be saved") {
		t.Fatalf("code %d: %s", code, diag)
	}
}

// Interrupt inside Save so the context is still live when finalization starts.
// This differs from interruption during collection, before finish is entered.
type interruptFinalizationStore struct {
	Storage
	cancel          context.CancelFunc
	diagnosticWrite bool
	failWrite       bool
	before          []byte
	triggered       bool
}

func (s *interruptFinalizationStore) Save(r *health.Run) error {
	diagnostic := false
	for _, category := range r.Diagnostics {
		if category == "report_publication_failed" {
			diagnostic = true
		}
	}
	if r.EndedAt != nil && diagnostic == s.diagnosticWrite {
		var err error
		s.before, err = os.ReadFile(s.Path())
		if err != nil {
			return err
		}
		s.triggered = true
		s.cancel()
		if s.failWrite {
			return errors.New("storage-secret-canary")
		}
	}
	return s.Storage.Save(r)
}

func (s *interruptFinalizationStore) PublishReport(render func(health.Run) ([]byte, error)) error {
	if s.diagnosticWrite {
		return errors.New("publication-secret-canary")
	}
	return s.Storage.PublishReport(render)
}

func TestInterruptionDuringFinalizationWritesPreservesExit130(t *testing.T) {
	for _, tc := range []struct {
		name             string
		diagnostic, fail bool
	}{
		{"failed final snapshot", false, true},
		{"failed publication diagnostic", true, true},
		{"committed publication diagnostic", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, out, diag, _ := testApp(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var storage *interruptFinalizationStore
			a.NewStore = func(root string, r *health.Run) (Storage, error) {
				s, err := runstore.New(root, r)
				if err != nil {
					return nil, err
				}
				storage = &interruptFinalizationStore{Storage: s, cancel: cancel, diagnosticWrite: tc.diagnostic, failWrite: tc.fail}
				return storage, nil
			}
			a.Inventory = func(_ context.Context, _ string, emit func(health.Page) error) error {
				return emit(health.Page{Resources: []health.ResourceInput{{ID: cliVM}}})
			}
			code := a.Run(ctx, []string{"health", "check", "--subscription", first})
			if !storage.triggered || ctx.Err() == nil {
				t.Fatal("test did not interrupt inside finalization write")
			}
			if code != 130 || out.Len() != 0 || !strings.Contains(diag.String(), "Run cancelled; finalization did not complete.") || !strings.Contains(diag.String(), "Last saved snapshot:") {
				t.Fatalf("code %d diagnostics %s", code, diag)
			}
			if strings.Contains(diag.String(), "secret-canary") || strings.Contains(diag.String(), "Report saved:") {
				t.Fatal("unsafe or misleading failure feedback")
			}
			r := snapshot(t, a.Directory)
			if len(r.Evidence) != 1 {
				t.Fatal("committed evidence lost")
			}
			if tc.fail {
				after, err := os.ReadFile(storage.Path())
				if err != nil || string(after) != string(storage.before) {
					t.Fatal("failed write replaced prior checkpoint")
				}
			}
			if _, err := os.Stat(storage.ReportPath()); !os.IsNotExist(err) {
				t.Fatal("report published after failed finalization")
			}
		})
	}
}

type reportPathWriter func([]byte) (int, error)

func (w reportPathWriter) Write(b []byte) (int, error) { return w(b) }

func TestReportPathDeliveryPreservesPublishedArtifacts(t *testing.T) {
	for _, tc := range []struct {
		name, outcome                                   string
		partial, cancelBefore, cancelDuring, shortWrite bool
		success                                         bool
		wantCode                                        int
	}{
		{name: "completed", outcome: "completed", wantCode: 1},
		{name: "partial", outcome: "partial", partial: true, wantCode: 1},
		{name: "already interrupted", outcome: "cancelled", cancelBefore: true, wantCode: 130},
		{name: "interrupted during delivery", outcome: "completed", cancelDuring: true, wantCode: 130},
		{name: "partial output", outcome: "completed", shortWrite: true, wantCode: 1},
		{name: "interrupted during successful delivery", outcome: "completed", cancelDuring: true, success: true, wantCode: 130},
		{name: "partial run interrupted during successful delivery", outcome: "partial", partial: true, cancelDuring: true, success: true, wantCode: 130},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, _, diag, _ := testApp(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			a.Inventory = func(_ context.Context, _ string, emit func(health.Page) error) error {
				return emit(health.Page{Resources: []health.ResourceInput{{ID: cliVM}}})
			}
			a.ResourceHealth = func(ctx context.Context, _ string, emit func(health.Page) error) error {
				if tc.cancelBefore {
					cancel()
					return ctx.Err()
				}
				if tc.partial {
					return errors.New("collector-secret-canary")
				}
				return emit(health.Page{Health: []health.HealthInput{{StatusID: cliVM + "/providers/Microsoft.ResourceHealth/availabilityStatuses/current", PropertiesPresent: true, State: stringPointer("Unavailable")}}})
			}
			var reportPath, snapshotPath string
			var beforeReport, beforeSnapshot []byte
			writes := 0
			a.Out = reportPathWriter(func(b []byte) (int, error) {
				writes++
				reportPath = strings.TrimSpace(string(b))
				snapshotPath = filepath.Join(filepath.Dir(reportPath), "run.json")
				r := assertPublished(t, a.Directory, reportPath)
				if r.Outcome != tc.outcome || len(r.Evidence) == 0 {
					t.Fatal("wrong snapshot before delivery")
				}
				var err error
				beforeReport, err = os.ReadFile(reportPath)
				if err != nil {
					t.Fatal(err)
				}
				beforeSnapshot, err = os.ReadFile(snapshotPath)
				if err != nil {
					t.Fatal(err)
				}
				if tc.cancelDuring {
					cancel()
				}
				if tc.success {
					return len(b), nil
				}
				n := 0
				if tc.shortWrite {
					n = len(b) / 2
				}
				return n, errors.New("stdout-secret-canary")
			})
			code := a.Run(ctx, []string{"health", "check", "--subscription", first})
			if code != tc.wantCode || writes != 1 {
				t.Fatalf("code %d writes %d diagnostics %s", code, writes, diag)
			}
			if strings.Contains(diag.String(), "Could not write the saved report path to stdout.") == tc.success || !strings.Contains(diag.String(), reportPath) {
				t.Fatalf("incorrect delivery feedback or saved path missing: %s", diag)
			}
			if tc.wantCode == 130 && !strings.Contains(diag.String(), "Run cancelled") {
				t.Fatal("interruption feedback missing")
			}
			if tc.cancelDuring && !strings.Contains(diag.String(), "Run cancelled; the saved report remains available.") {
				t.Fatal("late interruption did not identify the retained report")
			}
			if strings.Contains(diag.String(), "secret-canary") || strings.Contains(diag.String(), "not current") || strings.Contains(diag.String(), "no report was saved") {
				t.Fatal("unsafe error or misleading artifact diagnostic")
			}
			afterReport, err := os.ReadFile(reportPath)
			if err != nil || string(afterReport) != string(beforeReport) {
				t.Fatal("output failure changed published report")
			}
			afterSnapshot, err := os.ReadFile(snapshotPath)
			if err != nil || string(afterSnapshot) != string(beforeSnapshot) {
				t.Fatal("output failure changed committed snapshot")
			}
			assertPublished(t, a.Directory, reportPath)
		})
	}
}

func TestReportDirectoryIsNotDescribedAsAnOlderReport(t *testing.T) {
	a, out, diag, _ := testApp(t)
	a.NewStore = func(root string, r *health.Run) (Storage, error) {
		s, err := runstore.New(root, r)
		if err != nil {
			return nil, err
		}
		if err := os.Mkdir(s.ReportPath(), 0700); err != nil {
			t.Fatal(err)
		}
		return s, nil
	}
	if code := a.Run(context.Background(), []string{"health", "check", "--subscription", first}); code != 1 || out.Len() != 0 {
		t.Fatalf("code %d stdout %s diagnostics %s", code, out, diag)
	}
	if strings.Contains(diag.String(), "An older report exists") || !strings.Contains(diag.String(), "Could not determine whether an older report exists; no current report was published.") {
		t.Fatalf("directory misclassified as a report: %s", diag)
	}
	r := snapshot(t, a.Directory)
	if r.Outcome != "failed" || !strings.Contains(strings.Join(r.Diagnostics, " "), "report_publication_failed") {
		t.Fatal("publication failure not saved in JSON")
	}
}
