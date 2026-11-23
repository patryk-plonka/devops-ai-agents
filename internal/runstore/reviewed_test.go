package runstore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/patryk-plonka/devops-ai-agents/internal/health"
	"github.com/patryk-plonka/devops-ai-agents/internal/report"
)

func newReviewedPair(t *testing.T, outcome string) (string, *Store, health.Run) {
	t.Helper()
	repo := t.TempDir()
	writeReviewedFixture(t, filepath.Join(repo, "go.mod"), []byte("module "+Module+"\n"))
	run := health.NewRun("synthetic", "test")
	store, err := New(repo, &run)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	run.Outcome = outcome
	end := run.UpdatedAt
	run.EndedAt = &end
	run.RebuildCoverage()
	if err := store.Save(&run); err != nil {
		t.Fatal(err)
	}
	if err := store.PublishReport(context.Background(), report.Markdown); err != nil {
		t.Fatal(err)
	}
	return repo, store, run
}

func readReviewedFixture(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func writeReviewedFixture(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestOpenReviewedRunKeepsFinalizedPair(t *testing.T) {
	for _, outcome := range []string{"completed", "partial", "cancelled"} {
		t.Run(outcome, func(t *testing.T) {
			repo, store, original := newReviewedPair(t, outcome)
			snapshot := readReviewedFixture(t, store.Path())
			markdown := readReviewedFixture(t, store.ReportPath())
			if !strings.Contains(original.ID, ".") || !bytes.Contains(markdown, []byte("Run ID: "+strings.ReplaceAll(original.ID, ".", "&#46;"))) {
				t.Fatal("expected production ID encoded in the Markdown header")
			}
			invocation := filepath.Join(repo, "nested")
			if err := os.Mkdir(invocation, 0700); err != nil {
				t.Fatal(err)
			}
			relative, err := filepath.Rel(invocation, store.ReportPath())
			if err != nil {
				t.Fatal(err)
			}
			for _, path := range []string{relative, store.ReportPath()} {
				opened, saved, err := OpenReviewedRun(invocation, path)
				if err != nil {
					t.Fatal(err)
				}
				if err := opened.Close(); err != nil {
					t.Fatal(err)
				}
				if opened.ReportPath() != store.ReportPath() || saved.ID != original.ID ||
					saved.Revision != original.Revision || saved.Outcome != outcome || saved.Coverage != "incomplete" {
					t.Fatalf("opened unexpected revision or outcome: %s/%d %s %s", saved.ID, saved.Revision, saved.Outcome, saved.Coverage)
				}
			}
			if !bytes.Equal(snapshot, readReviewedFixture(t, store.Path())) || !bytes.Equal(markdown, readReviewedFixture(t, store.ReportPath())) {
				t.Fatal("review changed an artifact")
			}
			entries, err := os.ReadDir(filepath.Dir(store.Path()))
			if err != nil || len(entries) != 2 {
				t.Fatalf("unexpected run directory contents: %v, %v", entries, err)
			}
		})
	}
}

func TestOpenReviewedRunResolvesSavedSchema4Example(t *testing.T) {
	repo := t.TempDir()
	writeReviewedFixture(t, filepath.Join(repo, "go.mod"), []byte("module "+Module+"\n"))
	example := filepath.Join("..", "report", "testdata", "assessment")
	snapshot := readReviewedFixture(t, filepath.Join(example, "run.json"))
	markdown := readReviewedFixture(t, filepath.Join(example, "report.md"))
	var original health.Run
	if err := json.Unmarshal(snapshot, &original); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join("runs", original.ID)
	if err := os.MkdirAll(filepath.Join(repo, dir), 0700); err != nil {
		t.Fatal(err)
	}
	writeReviewedFixture(t, filepath.Join(repo, dir, "run.json"), snapshot)
	writeReviewedFixture(t, filepath.Join(repo, dir, "report.md"), markdown)
	store, saved, err := OpenReviewedRun(repo, filepath.Join(dir, "report.md"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	var ids []string
	for _, f := range original.Findings {
		if !bytes.Contains(markdown, []byte(f.ID)) {
			t.Fatal("fixture finding is missing from the reviewed Markdown")
		}
		ids = append(ids, f.ID)
	}
	result, err := saved.ResolveDraftSelection(ids)
	if err != nil || len(result.Findings) != len(original.Findings) {
		t.Fatalf("resolving checked-in schema 4 findings: got %d, error %v", len(result.Findings), err)
	}
}

func TestOpenReviewedRunAcceptsRepositoryAlias(t *testing.T) {
	repo, store, run := newReviewedPair(t, "completed")
	alias := filepath.Join(t.TempDir(), "repository-alias")
	if err := os.Symlink(repo, alias); err != nil {
		t.Fatal(err)
	}
	aliasReport := filepath.Join(alias, "runs", run.ID, "report.md")
	for _, paths := range [][2]string{{repo, aliasReport}, {alias, store.ReportPath()}} {
		opened, saved, err := OpenReviewedRun(paths[0], paths[1])
		if err != nil {
			t.Fatal(err)
		}
		opened.Close()
		if saved.ID != run.ID || saved.Revision != run.Revision {
			t.Fatal("alias opened the wrong run or revision")
		}
	}
	moveToSymlink(t, filepath.Dir(store.ReportPath()))
	assertInvalidReviewedRun(t, repo, aliasReport)
}

func TestOpenReviewedRunMatchesIdentityAcrossIntroWording(t *testing.T) {
	repo, store, original := newReviewedPair(t, "completed")
	markdown := readReviewedFixture(t, store.ReportPath())
	revised := bytes.Replace(markdown, []byte("The counts cover observed inventory. "),
		[]byte("Counts describe the observed inventory.\n\n"), 1)
	if bytes.Equal(markdown, revised) {
		t.Fatal("test must vary the explanatory prose")
	}
	writeReviewedFixture(t, store.ReportPath(), revised)

	opened, saved, err := OpenReviewedRun(repo, store.ReportPath())
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	if saved.ID != original.ID || saved.Revision != original.Revision {
		t.Fatal("opened the wrong run or revision")
	}
}

func TestOpenReviewedRunRejectsInvalidPair(t *testing.T) {
	for _, scenario := range []string{
		"missing report", "checkpoint with report", "stale", "older schema", "future schema",
		"zero revision", "directory identity", "Markdown identity", "identity in body",
		"ambiguous identity", "invalid reference", "malformed JSON",
	} {
		t.Run(scenario, func(t *testing.T) {
			repo, store, run := newReviewedPair(t, "completed")
			markdown := string(readReviewedFixture(t, store.ReportPath()))
			switch scenario {
			case "missing report":
				if err := os.Remove(store.ReportPath()); err != nil {
					t.Fatal(err)
				}
			case "checkpoint with report":
				run.EndedAt, run.Outcome = nil, "in_progress"
			case "stale":
				run.Revision++
			case "older schema":
				run.SchemaVersion--
			case "future schema":
				run.SchemaVersion++
			case "zero revision":
				run.Revision = 0
			case "directory identity":
				run.ID = "another-run"
			case "Markdown identity":
				markdown = strings.Replace(markdown, "Run ID: ", "Run ID: another-", 1)
			case "identity in body":
				header, body, _ := strings.Cut(markdown, "\n## Source coverage\n")
				begin := strings.Index(header, "Run ID:")
				end := strings.Index(header, "Execution outcome:")
				markdown = header[:begin] + header[end:] + "\n## Source coverage\n\n" + header[begin:end] + body
			case "ambiguous identity":
				markdown = strings.Replace(markdown, "Execution outcome:", "Revision: 1\n\nExecution outcome:", 1)
			case "invalid reference":
				run.Gaps[0].EvidenceIDs = []string{"ev-missing"}
			}
			snapshot, err := json.Marshal(run)
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "malformed JSON" {
				snapshot = []byte("{")
			}
			writeReviewedFixture(t, store.Path(), snapshot)
			if scenario != "missing report" {
				writeReviewedFixture(t, store.ReportPath(), []byte(markdown))
			}
			assertInvalidReviewedRun(t, repo, store.ReportPath())
		})
	}
}

func TestOpenReviewedRunConfinesReads(t *testing.T) {
	for _, scenario := range []string{"outside path", "wrong basename", "runs symlink", "run symlink", "report symlink", "snapshot symlink", "report directory", "snapshot directory"} {
		t.Run(scenario, func(t *testing.T) {
			repo, store, run := newReviewedPair(t, "completed")
			path := store.ReportPath()
			switch scenario {
			case "outside path":
				outside := filepath.Join(t.TempDir(), "runs", run.ID)
				if err := os.MkdirAll(outside, 0700); err != nil {
					t.Fatal(err)
				}
				path = filepath.Join(outside, "report.md")
				writeReviewedFixture(t, path, readReviewedFixture(t, store.ReportPath()))
				writeReviewedFixture(t, filepath.Join(outside, "run.json"), readReviewedFixture(t, store.Path()))
			case "wrong basename":
				path = filepath.Join(filepath.Dir(path), "copy.md")
				writeReviewedFixture(t, path, readReviewedFixture(t, store.ReportPath()))
			case "runs symlink":
				moveToSymlink(t, filepath.Join(repo, "runs"))
			case "run symlink":
				moveToSymlink(t, filepath.Dir(path))
			case "report symlink":
				moveToSymlink(t, path)
			case "snapshot symlink":
				moveToSymlink(t, store.Path())
			case "report directory", "snapshot directory":
				name := path
				if scenario == "snapshot directory" {
					name = store.Path()
				}
				if err := os.Remove(name); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(name, 0700); err != nil {
					t.Fatal(err)
				}
			}
			assertInvalidReviewedRun(t, repo, path)
		})
	}
}

func moveToSymlink(t *testing.T, path string) {
	t.Helper()
	target := path + "-target"
	if err := os.Rename(path, target); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Base(target), path); err != nil {
		t.Fatal(err)
	}
}

func assertInvalidReviewedRun(t *testing.T, repo, path string) {
	t.Helper()
	store, run, err := OpenReviewedRun(repo, path)
	if store != nil {
		_ = store.Close()
	}
	if !errors.Is(err, ErrInvalidReviewedRun) || store != nil || run.ID != "" {
		t.Fatalf("expected rejected pair, got opened=%t run=%q error=%v", store != nil, run.ID, err)
	}
}
