package output

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var (
	ErrExists  = errors.New("output already exists")
	ErrSymlink = errors.New("output path is a symbolic link")
)

type Workspace struct {
	directory string
	stagePath string
	closed    bool
}

func NewWorkspace(finalPath string) (*Workspace, error) {
	parent := filepath.Dir(finalPath)
	directory, err := os.MkdirTemp(parent, ".mediaconv-*")
	if err != nil {
		return nil, fmt.Errorf("create staging directory: %w", err)
	}

	return &Workspace{
		directory: directory,
		stagePath: filepath.Join(directory, "output.mp4"),
	}, nil
}

func (w *Workspace) StagePath() string { return w.stagePath }

func (w *Workspace) Cleanup() error {
	if w == nil || w.closed {
		return nil
	}
	if err := os.RemoveAll(w.directory); err != nil {
		return fmt.Errorf("remove staging directory: %w", err)
	}
	w.closed = true
	return nil
}

type Publisher struct{}

func (Publisher) Publish(stagePath, finalPath string, overwrite bool) error {
	info, err := os.Lstat(finalPath)
	switch {
	case err == nil:
		if info.Mode()&os.ModeSymlink != 0 {
			return ErrSymlink
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("output exists and is not a regular file")
		}
		if !overwrite {
			return ErrExists
		}
	case !errors.Is(err, os.ErrNotExist):
		return fmt.Errorf("inspect output: %w", err)
	}

	if overwrite {
		if err := replaceFile(stagePath, finalPath); err != nil {
			return fmt.Errorf("replace output: %w", err)
		}
		return nil
	}

	// A hard link publishes the staged file atomically while guaranteeing that
	// an output created by another process is never overwritten.
	if err := os.Link(stagePath, finalPath); err != nil {
		if errors.Is(err, os.ErrExist) {
			return ErrExists
		}
		return fmt.Errorf("publish output without overwriting: %w", err)
	}
	// Workspace cleanup removes the staging link. The published hard link stays
	// valid and points to the same verified file data.
	return nil
}
