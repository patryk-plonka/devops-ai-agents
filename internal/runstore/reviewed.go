package runstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/patryk-plonka/devops-ai-agents/internal/health"
	"github.com/patryk-plonka/devops-ai-agents/internal/report"
)

var ErrInvalidReviewedRun = errors.New("invalid reviewed run")

// OpenReviewedRun opens the existing directory without creating or advancing
// a run. The caller closes the returned store.
func OpenReviewedRun(invocationDir, reportPath string) (*Store, health.Run, error) {
	var empty health.Run
	if reportPath == "" {
		return nil, empty, fmt.Errorf("%w: a report path is required", ErrInvalidReviewedRun)
	}
	repository, err := FindRepository(invocationDir)
	if err != nil {
		return nil, empty, fmt.Errorf("%w: %v", ErrInvalidReviewedRun, err)
	}
	if !filepath.IsAbs(reportPath) {
		reportPath = filepath.Join(invocationDir, reportPath)
	}
	absolute, err := filepath.Abs(reportPath)
	if err != nil {
		return nil, empty, err
	}
	runDir := filepath.Dir(absolute)
	runsDir := filepath.Dir(runDir)
	if filepath.Base(absolute) != "report.md" || filepath.Base(runsDir) != "runs" {
		return nil, empty, fmt.Errorf("%w: report must be under runs/<run-id>/report.md", ErrInvalidReviewedRun)
	}
	repo, err := os.OpenRoot(repository)
	if err != nil {
		return nil, empty, err
	}
	defer repo.Close()
	// Match aliases of the repository itself while opening the run components
	// through the same rooted checks for every spelling.
	if prefix := filepath.Dir(runsDir); prefix != repository {
		actual, err := repo.Stat(".")
		if err != nil {
			return nil, empty, err
		}
		provided, err := os.Stat(prefix)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, empty, err
		}
		if err != nil || !os.SameFile(actual, provided) {
			return nil, empty, fmt.Errorf("%w: report must be under runs/<run-id>/report.md", ErrInvalidReviewedRun)
		}
	}
	runs, err := openReviewedDir(repo, "runs")
	if err != nil {
		return nil, empty, err
	}
	defer runs.Close()
	runID := filepath.Base(runDir)
	root, err := openReviewedDir(runs, runID)
	if err != nil {
		return nil, empty, err
	}
	saved, err := readReviewedPair(root, runID)
	if err != nil {
		_ = root.Close()
		return nil, empty, err
	}
	return &Store{root: root, path: filepath.Join(repository, "runs", runID), revision: saved.Revision}, saved, nil
}

func openReviewedDir(parent *os.Root, name string) (*os.Root, error) {
	info, err := parent.Lstat(name)
	if err != nil {
		return nil, reviewedPathError(err, name)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%w: expected a real run directory", ErrInvalidReviewedRun)
	}
	dir, err := parent.OpenRoot(name)
	if err != nil {
		return nil, reviewedPathError(err, name)
	}
	opened, err := dir.Stat(".")
	if err != nil || !os.SameFile(info, opened) {
		_ = dir.Close()
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("%w: run directory changed during opening", ErrInvalidReviewedRun)
	}
	return dir, nil
}

func readReviewedPair(root *os.Root, runID string) (health.Run, error) {
	var saved health.Run
	snapshot, err := readReviewedFile(root, "run.json")
	if err != nil {
		return saved, err
	}
	if err := json.Unmarshal(snapshot, &saved); err != nil {
		return saved, fmt.Errorf("%w: malformed run.json", ErrInvalidReviewedRun)
	}
	if saved.ID != runID {
		return saved, fmt.Errorf("%w: snapshot run ID does not match its directory", ErrInvalidReviewedRun)
	}
	markdown, err := readReviewedFile(root, "report.md")
	if err != nil {
		return saved, err
	}
	if err := report.ValidateReviewedMarkdown(saved, markdown); err != nil {
		return saved, fmt.Errorf("%w: %v", ErrInvalidReviewedRun, err)
	}
	return saved, nil
}

func readReviewedFile(root *os.Root, name string) ([]byte, error) {
	info, err := root.Lstat(name)
	if err != nil {
		return nil, reviewedPathError(err, name)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: %s must be a regular file", ErrInvalidReviewedRun, name)
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, reviewedPathError(err, name)
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
		return nil, fmt.Errorf("%w: %s changed during opening", ErrInvalidReviewedRun, name)
	}
	return io.ReadAll(file)
}

func reviewedPathError(err error, name string) error {
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%w: %s is missing", ErrInvalidReviewedRun, name)
	}
	return err
}
