// Package runstore writes owner-only normalized snapshots beneath the repository.
package runstore

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/patryk-plonka/devops-ai-agents/internal/health"
)

const Module = "github.com/patryk-plonka/devops-ai-agents"

func FindRepository(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		b, err := os.ReadFile(filepath.Join(dir, "go.mod"))
		if err == nil {
			for _, line := range strings.Split(string(b), "\n") {
				fields := strings.Fields(line)
				if len(fields) >= 2 && fields[0] == "module" && strings.Trim(fields[1], `"`) == Module {
					return dir, nil
				}
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("run from this project's repository tree (containing its go.mod)")
		}
		dir = parent
	}
}

type Store struct {
	root     *os.Root
	path     string
	revision int
}

func (s *Store) Path() string       { return filepath.Join(s.path, "run.json") }
func (s *Store) ReportPath() string { return filepath.Join(s.path, "report.md") }
func (s *Store) Close() error       { return s.root.Close() }

// ReportExists reports regular files only, without following symlinks.
// Other entry types are obstructions whose report availability is unknown.
func (s *Store) ReportExists() (bool, error) {
	info, err := s.root.Lstat("report.md")
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !info.Mode().IsRegular() {
		return false, errors.New("report path is not a regular file")
	}
	return true, nil
}

// PublishReport renders only the committed snapshot. Snapshot and report renames
// are separate operations; a failed publication leaves any older report intact.
// The caller serializes Save and PublishReport for this single-run store.
func (s *Store) PublishReport(render func(health.Run) ([]byte, error)) error {
	b, err := s.root.ReadFile("run.json")
	if err != nil {
		return err
	}
	var saved health.Run
	if err := json.Unmarshal(b, &saved); err != nil {
		return err
	}
	if saved.ID != filepath.Base(s.path) || saved.Revision != s.revision {
		return errors.New("snapshot identity or revision mismatch")
	}
	b, err = render(saved)
	if err != nil {
		return err
	}
	return s.replace("report.md", b)
}

func New(repository string, run *health.Run) (*Store, error) {
	repo, err := os.OpenRoot(repository)
	if err != nil {
		return nil, err
	}
	defer repo.Close()
	if err := repo.Mkdir("runs", 0700); err != nil && !errors.Is(err, os.ErrExist) {
		return nil, err
	}
	info, err := repo.Lstat("runs")
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("runs must be a real directory")
	}
	runs, err := repo.OpenRoot("runs")
	if err != nil {
		return nil, err
	}
	defer runs.Close()
	if err := runs.Chmod(".", 0700); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	id := now.Format("20060102T150405.000000000Z") + "-" + randomID()
	if err := runs.Mkdir(id, 0700); err != nil {
		return nil, err
	}
	root, err := runs.OpenRoot(id)
	if err != nil {
		return nil, err
	}
	s := &Store{root: root, path: filepath.Join(repository, "runs", id)}
	run.ID = id
	run.StartedAt = now
	if err := s.Save(run); err != nil {
		_ = s.Close()
		return nil, err
	}
	return s, nil
}

func randomID() string { b := make([]byte, 16); _, _ = rand.Read(b); return hex.EncodeToString(b) }

// Save advances the in-memory revision only after atomic publication succeeds.
func (s *Store) Save(run *health.Run) error {
	next := *run
	next.Revision = s.revision + 1
	next.UpdatedAt = time.Now().UTC()
	b, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if err := s.replace("run.json", b); err != nil {
		return err
	}
	*run = next
	s.revision = next.Revision
	return nil
}

func (s *Store) replace(target string, b []byte) error {
	name := ".write-" + randomID() + ".tmp"
	f, err := s.root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer func() { _ = s.root.Remove(name) }()
	_, writeErr := f.Write(b)
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	return s.root.Rename(name, target)
}
