package runstore

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/patryk-plonka/devops-ai-agents/internal/report"
)

func TestDraftPublicationPreservesWorkingCopies(t *testing.T) {
	_, run, _ := newReviewedPair(t, "completed")
	ctx := context.Background()
	store, err := run.OpenDrafts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	first, second := draftTestID("a"), draftTestID("b")
	original := []byte("complete draft\n")
	path := publishTestDraft(t, store, report.Ticket, first, original, true)
	if filepath.Dir(path) != filepath.Join(filepath.Dir(run.ReportPath()), "drafts") ||
		filepath.Base(path) != "ticket-"+first+".md" {
		t.Fatalf("unexpected draft path %s", path)
	}
	for name, want := range map[string]os.FileMode{path: 0600, filepath.Dir(path): 0700} {
		info, err := os.Stat(name)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != want {
			t.Errorf("mode of %s = %o, want %o", name, info.Mode().Perm(), want)
		}
	}
	edited := []byte("operator edits\n")
	writeReviewedFixture(t, path, edited)
	if got := publishTestDraft(t, store, report.Ticket, first, original, false); got != path {
		t.Fatal("retry reported a different path")
	}
	other := publishTestDraft(t, store, report.Ticket, second, original, true)
	recommendation := publishTestDraft(t, store, report.Recommendation, first, original, true)
	for name, want := range map[string][]byte{path: edited, other: original, recommendation: original} {
		if !bytes.Equal(readReviewedFixture(t, name), want) {
			t.Errorf("unexpected bytes in %s", name)
		}
	}
	assertDraftEntries(t, filepath.Dir(path), 3)
}

func TestDraftPublicationConcurrentWinner(t *testing.T) {
	_, run, _ := newReviewedPair(t, "completed")
	ctx := context.Background()
	id := draftTestID("a")
	const writers = 12
	type result struct {
		path    string
		body    []byte
		created bool
		err     error
	}
	start := make(chan struct{})
	results := make(chan result, writers)
	for i := range writers {
		go func() {
			<-start
			store, err := run.OpenDrafts(ctx)
			if err != nil {
				results <- result{err: err}
				return
			}
			defer store.Close()
			body := bytes.Repeat([]byte{byte('a' + i)}, 32*1024)
			path, created, err := store.Publish(ctx, report.Ticket, id, body)
			results <- result{path, body, created, err}
		}()
	}
	close(start)
	var winner result
	count := 0
	for range writers {
		r := <-results
		if r.err != nil {
			t.Error(r.err)
			continue
		}
		if r.path != filepath.Join(filepath.Dir(run.ReportPath()), "drafts", "ticket-"+id+".md") {
			t.Errorf("unexpected path %s", r.path)
		}
		if r.created {
			winner = r
			count++
		}
	}
	if count != 1 || !bytes.Equal(readReviewedFixture(t, winner.path), winner.body) {
		t.Fatalf("expected one complete winner, got %d", count)
	}
	assertDraftEntries(t, filepath.Dir(winner.path), 1)
}

func TestDraftPublicationRejectsObstructionsAndCanRetry(t *testing.T) {
	for _, obstruction := range []string{"drafts symlink", "drafts file", "draft symlink", "draft directory"} {
		t.Run(obstruction, func(t *testing.T) {
			_, run, _ := newReviewedPair(t, "completed")
			ctx := context.Background()
			target := filepath.Join(t.TempDir(), "outside")
			dir := filepath.Join(filepath.Dir(run.ReportPath()), "drafts")
			switch obstruction {
			case "drafts symlink":
				if err := os.Mkdir(target, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, dir); err != nil {
					t.Fatal(err)
				}
				assertDraftEntries(t, target, 0)
			case "drafts file":
				writeReviewedFixture(t, dir, []byte("unrelated data"))
			}
			store, err := run.OpenDrafts(ctx)
			if strings.HasPrefix(obstruction, "drafts ") {
				if err == nil {
					_ = store.Close()
					t.Fatal("accepted an obstructed drafts directory")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			body := []byte("complete draft\n")
			first := publishTestDraft(t, store, report.Ticket, draftTestID("a"), body, true)
			second := draftTestID("b")
			blocked := filepath.Join(dir, "ticket-"+second+".md")
			switch obstruction {
			case "draft symlink":
				writeReviewedFixture(t, target, []byte("outside draft"))
				if err := os.Symlink(target, blocked); err != nil {
					t.Fatal(err)
				}
			case "draft directory":
				if err := os.Mkdir(blocked, 0700); err != nil {
					t.Fatal(err)
				}
			}
			if path, _, err := store.Publish(ctx, report.Ticket, second, body); err == nil || path != "" {
				t.Fatalf("published obstructed destination %s (%v)", path, err)
			}
			assertDraftEntries(t, dir, 2)
			if obstruction == "draft symlink" && string(readReviewedFixture(t, target)) != "outside draft" {
				t.Fatal("symlink target changed")
			}
			if err := os.Remove(blocked); err != nil {
				t.Fatal(err)
			}
			cancelled, cancel := context.WithCancel(ctx)
			cancel()
			if _, _, err := store.Publish(cancelled, report.Ticket, second, body); !errors.Is(err, context.Canceled) {
				t.Fatalf("cancelled publication returned %v", err)
			}
			assertDraftEntries(t, dir, 1)
			publishTestDraft(t, store, report.Ticket, draftTestID("a"), []byte("new contents"), false)
			publishTestDraft(t, store, report.Ticket, second, body, true)
			if !bytes.Equal(readReviewedFixture(t, first), body) {
				t.Fatal("earlier completed draft changed")
			}
			assertDraftEntries(t, dir, 2)
		})
	}
}

func publishTestDraft(t *testing.T, store *DraftStore, kind report.DraftType, id string, body []byte, wantCreated bool) string {
	t.Helper()
	path, created, err := store.Publish(context.Background(), kind, id, body)
	if err != nil || path == "" || created != wantCreated {
		t.Fatalf("publish %s = %q, %t, %v", id, path, created, err)
	}
	return path
}

func assertDraftEntries(t *testing.T, dir string, expected int) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != expected {
		t.Fatalf("entries %v, error %v", entries, err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".draft-") {
			t.Fatal("staging file retained")
		}
	}
}

func draftTestID(value string) string { return "finding-" + strings.Repeat(value, 64) }
