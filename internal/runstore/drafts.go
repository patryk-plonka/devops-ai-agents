package runstore

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/patryk-plonka/devops-ai-agents/internal/health"
	"github.com/patryk-plonka/devops-ai-agents/internal/report"
)

// DraftStore confines publication to a real directory inside the reviewed run.
type DraftStore struct {
	root *os.Root
	path string
}

func (s *Store) OpenDrafts(ctx context.Context) (*DraftStore, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := s.root.Mkdir("drafts", 0700); err != nil && !errors.Is(err, os.ErrExist) {
		return nil, err
	}
	root, err := openReviewedDir(s.root, "drafts")
	if err != nil {
		return nil, err
	}
	if err := root.Chmod(".", 0700); err != nil {
		_ = root.Close()
		return nil, err
	}
	return &DraftStore{root: root, path: filepath.Join(s.path, "drafts")}, nil
}

func (s *DraftStore) Close() error { return s.root.Close() }

// Publish returns a path only for a complete or previously existing draft.
// Linking a complete sibling file gives each destination one atomic winner.
func (s *DraftStore) Publish(ctx context.Context, kind report.DraftType, id string, markdown []byte) (path string, created bool, err error) {
	if !kind.Valid() || !health.ValidFindingID(id) {
		return "", false, errors.New("invalid draft filename")
	}
	if err := ctx.Err(); err != nil {
		return "", false, err
	}
	name := string(kind) + "-" + id + ".md"
	path = filepath.Join(s.path, name)
	if exists, err := s.existing(name); err != nil {
		return "", false, err
	} else if exists {
		return path, false, nil
	}

	staging := ".draft-" + randomID() + ".tmp"
	f, err := s.root.OpenFile(staging, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", false, err
	}
	defer func() { _ = s.root.Remove(staging) }()
	_, writeErr := f.Write(markdown)
	closeErr := f.Close()
	if err := errors.Join(writeErr, closeErr, ctx.Err()); err != nil {
		return "", false, err
	}
	if err := s.root.Link(staging, name); err != nil {
		if errors.Is(err, os.ErrExist) {
			if exists, inspectErr := s.existing(name); inspectErr != nil {
				return "", false, inspectErr
			} else if exists {
				return path, false, nil
			}
		}
		return "", false, err
	}
	return path, true, nil
}

func (s *DraftStore) existing(name string) (bool, error) {
	info, err := s.root.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !info.Mode().IsRegular() {
		return false, errors.New("draft destination must be a regular file")
	}
	return true, nil
}
