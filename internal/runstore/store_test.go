package runstore

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/patryk-plonka/devops-ai-agents/internal/health"
)

func TestFindRepository(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module "+Module+"\n\ngo 1.27.3\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a", "go.mod"), []byte("module another/project\n"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := FindRepository(nested)
	if err != nil || got != root {
		t.Fatalf("root %s, err %v", got, err)
	}
	if _, err := FindRepository(t.TempDir()); err == nil {
		t.Fatal("accepted missing project")
	}
}

func TestPrivateDistinctRunSnapshots(t *testing.T) {
	root := t.TempDir()
	ids := map[string]bool{}
	for range 5 {
		run := health.NewRun("../../subscription", "test")
		s, err := New(root, &run)
		if err != nil {
			t.Fatal(err)
		}
		if ids[run.ID] {
			t.Fatal("duplicate run")
		}
		ids[run.ID] = true
		if run.Revision != 1 || run.Outcome != "in_progress" || run.StartedAt.IsZero() {
			t.Fatal("invalid initial snapshot")
		}
		if filepath.Dir(s.Path()) != filepath.Join(root, "runs", run.ID) {
			t.Fatal("unsafe run path")
		}
		for _, path := range []string{filepath.Join(root, "runs"), filepath.Dir(s.Path()), s.Path()} {
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			want := os.FileMode(0600)
			if info.IsDir() {
				want = 0700
			}
			if runtime.GOOS != "windows" && info.Mode().Perm() != want {
				t.Fatalf("permissions %s: %v", path, info.Mode())
			}
		}
		run.Outcome = "failed"
		run.Diagnostics = append(run.Diagnostics, "subscription_access_denied")
		if err := s.Save(&run); err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(s.Path())
		if err != nil {
			t.Fatal(err)
		}
		var saved health.Run
		if err := json.Unmarshal(b, &saved); err != nil {
			t.Fatal(err)
		}
		if saved.Revision != 2 || saved.Outcome != "failed" || saved.ID != run.ID {
			t.Fatal("snapshot disagrees")
		}
		if _, err := os.Stat(filepath.Join(filepath.Dir(s.Path()), "report.md")); !os.IsNotExist(err) {
			t.Fatal("premature report")
		}
		entries, err := os.ReadDir(filepath.Dir(s.Path()))
		if err != nil || len(entries) != 1 {
			t.Fatal("temporary write files leaked")
		}
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestInvalidStorageAndSymlinkEscape(t *testing.T) {
	for _, kind := range []string{"file", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			outside := t.TempDir()
			path := filepath.Join(root, "runs")
			var err error
			if kind == "file" {
				err = os.WriteFile(path, []byte("existing"), 0600)
			} else {
				err = os.Symlink(outside, path)
			}
			if err != nil {
				t.Fatal(err)
			}
			run := health.NewRun("client", "test")
			if s, err := New(root, &run); err == nil {
				_ = s.Close()
				t.Fatal("accepted invalid storage")
			}
			entries, err := os.ReadDir(outside)
			if err != nil || len(entries) != 0 {
				t.Fatal("escaped run root")
			}
		})
	}
}

func TestFailedCheckpointPreservesPriorRevision(t *testing.T) {
	run := health.NewRun("client", "test")
	s, err := New(t.TempDir(), &run)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(s.Path())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	run.Outcome = "failed"
	if err := s.Save(&run); err == nil {
		t.Fatal("write to closed store succeeded")
	}
	after, err := os.ReadFile(s.Path())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || run.Revision != 1 {
		t.Fatal("failed write changed committed snapshot/revision")
	}
}

func TestSnapshotSymlinkCannotModifyOutsideFile(t *testing.T) {
	run := health.NewRun("client", "test")
	s, err := New(t.TempDir(), &run)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	outside := filepath.Join(t.TempDir(), "private")
	if err := os.WriteFile(outside, []byte("untouched"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(s.Path()); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, s.Path()); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(&run); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(outside)
	if err != nil || string(b) != "untouched" {
		t.Fatal("followed snapshot symlink")
	}
}

func TestReportPublicationUsesCommittedRevisionAndPreservesPriorReport(t *testing.T) {
	run := health.NewRun("synthetic", "test")
	s, err := New(t.TempDir(), &run)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	now := run.UpdatedAt
	run.EndedAt = &now
	run.Outcome = "completed"
	if err := s.Save(&run); err != nil {
		t.Fatal(err)
	}
	render := func(saved health.Run) ([]byte, error) {
		if saved.ID != run.ID || saved.Revision != run.Revision {
			t.Fatal("rendered uncommitted revision")
		}
		return []byte(fmt.Sprintf("run=%s revision=%d", saved.ID, saved.Revision)), nil
	}
	if err := s.PublishReport(render); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(s.ReportPath())
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(s.ReportPath())
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
		t.Fatal("report permissions")
	}
	if yes, err := s.ReportExists(); err != nil || !yes {
		t.Fatal("report not found")
	}
	run.Outcome = "partial"
	if err := s.Save(&run); err != nil {
		t.Fatal(err)
	}
	// A render failure keeps both the newer JSON checkpoint and older report.
	if err := s.PublishReport(func(saved health.Run) ([]byte, error) {
		if saved.Outcome != "partial" {
			t.Fatal("stale snapshot rendered")
		}
		return nil, errors.New("render failed")
	}); err == nil {
		t.Fatal("render failure ignored")
	}
	after, err := os.ReadFile(s.ReportPath())
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("render failure damaged report")
	}
	snapshot, err := os.ReadFile(s.Path())
	if err != nil {
		t.Fatal(err)
	}
	var saved health.Run
	if json.Unmarshal(snapshot, &saved) != nil || saved.Revision != run.Revision {
		t.Fatal("checkpoint lost")
	}
	// Publishing reads persisted facts even if the caller changes its in-memory copy.
	run.Outcome = "failed"
	if err := s.PublishReport(func(saved health.Run) ([]byte, error) {
		if saved.Outcome != "partial" {
			t.Fatal("uncommitted state rendered")
		}
		return []byte("new report"), nil
	}); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Dir(s.Path()))
	if err != nil || len(entries) != 2 {
		t.Fatal("temporary files leaked")
	}
}
func TestReportWriteFailureAndSymlinkStayInsideRun(t *testing.T) {
	run := health.NewRun("synthetic", "test")
	s, err := New(t.TempDir(), &run)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	outside := filepath.Join(t.TempDir(), "private")
	if err := os.WriteFile(outside, []byte("untouched"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, s.ReportPath()); err != nil {
		t.Fatal(err)
	}
	render := func(health.Run) ([]byte, error) { return []byte("report"), nil }
	if err := s.PublishReport(render); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(outside)
	if err != nil || string(b) != "untouched" {
		t.Fatal("followed report symlink")
	}
	if err := os.Remove(s.ReportPath()); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(s.ReportPath(), 0700); err != nil {
		t.Fatal(err)
	}
	checkpoint, err := os.ReadFile(s.Path())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.PublishReport(render); err == nil {
		t.Fatal("report directory accepted")
	}
	after, err := os.ReadFile(s.Path())
	if err != nil || !bytes.Equal(checkpoint, after) {
		t.Fatal("report failure damaged JSON")
	}
	entries, err := os.ReadDir(filepath.Dir(s.Path()))
	if err != nil || len(entries) != 2 {
		t.Fatal("failed publication leaked temporary file")
	}
}

func TestReportExistsClassifiesEntriesWithoutFollowingSymlinks(t *testing.T) {
	for _, kind := range []string{"missing", "regular", "directory", "symlink", "dangling symlink"} {
		t.Run(kind, func(t *testing.T) {
			run := health.NewRun("synthetic", "test")
			s, err := New(t.TempDir(), &run)
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			switch kind {
			case "regular":
				err = os.WriteFile(s.ReportPath(), []byte("older report"), 0600)
			case "directory":
				err = os.Mkdir(s.ReportPath(), 0700)
			case "symlink", "dangling symlink":
				target := filepath.Join(t.TempDir(), "outside.md")
				if kind == "symlink" {
					if err := os.WriteFile(target, []byte("outside report"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				err = os.Symlink(target, s.ReportPath())
			}
			if err != nil {
				t.Fatal(err)
			}
			exists, err := s.ReportExists()
			wantError := kind != "missing" && kind != "regular"
			if exists != (kind == "regular") || (err != nil) != wantError {
				t.Fatalf("exists %v error %v", exists, err)
			}
		})
	}
}
