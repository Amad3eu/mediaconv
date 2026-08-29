package output

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewWorkspaceStagesBesideFinalAndCleansUp(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	finalPath := filepath.Join(parent, "movie.mp4")
	workspace, err := NewWorkspace(finalPath)
	if err != nil {
		t.Fatalf("NewWorkspace() error = %v", err)
	}

	if filepath.Dir(workspace.directory) != parent {
		t.Errorf("workspace parent = %q, want %q", filepath.Dir(workspace.directory), parent)
	}
	if !strings.HasPrefix(filepath.Base(workspace.directory), ".mediaconv-") {
		t.Errorf("workspace directory = %q, want .mediaconv-* prefix", workspace.directory)
	}
	if got, want := workspace.StagePath(), filepath.Join(workspace.directory, "output.mp4"); got != want {
		t.Errorf("StagePath() = %q, want %q", got, want)
	}
	if info, err := os.Stat(workspace.directory); err != nil || !info.IsDir() {
		t.Fatalf("workspace directory was not created: info=%v err=%v", info, err)
	}
	if _, err := os.Stat(workspace.StagePath()); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("staged output exists before conversion: %v", err)
	}
	if err := os.WriteFile(workspace.StagePath(), []byte("staged media"), 0o600); err != nil {
		t.Fatalf("create staged output: %v", err)
	}

	if err := workspace.Cleanup(); err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}
	if _, err := os.Stat(workspace.directory); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("workspace directory remains after Cleanup(): %v", err)
	}
	if err := workspace.Cleanup(); err != nil {
		t.Errorf("second Cleanup() error = %v", err)
	}

	var nilWorkspace *Workspace
	if err := nilWorkspace.Cleanup(); err != nil {
		t.Errorf("nil Cleanup() error = %v", err)
	}
}

func TestNewWorkspaceFailsWhenOutputParentDoesNotExist(t *testing.T) {
	t.Parallel()

	finalPath := filepath.Join(t.TempDir(), "missing", "movie.mp4")
	workspace, err := NewWorkspace(finalPath)
	if err == nil {
		if workspace != nil {
			_ = workspace.Cleanup()
		}
		t.Fatal("NewWorkspace() error = nil, want missing parent error")
	}
	if !strings.Contains(err.Error(), "create staging directory") {
		t.Errorf("NewWorkspace() error = %q", err)
	}
}

func TestPublisherPublishesWithoutOverwrite(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	stagePath := filepath.Join(directory, "stage.mp4")
	finalPath := filepath.Join(directory, "final.mp4")
	writeTestFile(t, stagePath, "new media")

	if err := (Publisher{}).Publish(stagePath, finalPath, false); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	assertFileContents(t, finalPath, "new media")
	// No-overwrite publication uses a hard link. The owning Workspace removes
	// the staged name during Cleanup while the final link remains valid.
	assertFileContents(t, stagePath, "new media")
	stageInfo, err := os.Stat(stagePath)
	if err != nil {
		t.Fatalf("stat staged file: %v", err)
	}
	finalInfo, err := os.Stat(finalPath)
	if err != nil {
		t.Fatalf("stat final file: %v", err)
	}
	if !os.SameFile(stageInfo, finalInfo) {
		t.Error("stage and final paths do not refer to the same hard-linked file")
	}
}

func TestPublisherDoesNotOverwriteExistingFileByDefault(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	stagePath := filepath.Join(directory, "stage.mp4")
	finalPath := filepath.Join(directory, "final.mp4")
	writeTestFile(t, stagePath, "new media")
	writeTestFile(t, finalPath, "original media")

	err := (Publisher{}).Publish(stagePath, finalPath, false)
	if !errors.Is(err, ErrExists) {
		t.Fatalf("Publish() error = %v, want ErrExists", err)
	}
	assertFileContents(t, finalPath, "original media")
	assertFileContents(t, stagePath, "new media")
}

func TestPublisherOverwritesExistingRegularFile(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	stagePath := filepath.Join(directory, "stage.mp4")
	finalPath := filepath.Join(directory, "final.mp4")
	writeTestFile(t, stagePath, "replacement media")
	writeTestFile(t, finalPath, "original media")

	if err := (Publisher{}).Publish(stagePath, finalPath, true); err != nil {
		t.Fatalf("Publish(overwrite=true) error = %v", err)
	}
	assertFileContents(t, finalPath, "replacement media")
	if _, err := os.Stat(stagePath); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("stage still exists after replacement: %v", err)
	}
}

func TestPublisherRejectsSymlinkOutput(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	stagePath := filepath.Join(directory, "stage.mp4")
	targetPath := filepath.Join(directory, "target.mp4")
	finalPath := filepath.Join(directory, "final.mp4")
	writeTestFile(t, stagePath, "new media")
	writeTestFile(t, targetPath, "symlink target")
	if err := os.Symlink(targetPath, finalPath); err != nil {
		t.Skipf("symbolic links are unavailable: %v", err)
	}

	for _, overwrite := range []bool{false, true} {
		t.Run(overwriteName(overwrite), func(t *testing.T) {
			err := (Publisher{}).Publish(stagePath, finalPath, overwrite)
			if !errors.Is(err, ErrSymlink) {
				t.Errorf("Publish(overwrite=%v) error = %v, want ErrSymlink", overwrite, err)
			}
			assertFileContents(t, targetPath, "symlink target")
			assertFileContents(t, stagePath, "new media")
		})
	}
}

func TestPublisherRejectsNonRegularOutput(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	stagePath := filepath.Join(directory, "stage.mp4")
	finalPath := filepath.Join(directory, "existing-directory.mp4")
	writeTestFile(t, stagePath, "new media")
	if err := os.Mkdir(finalPath, 0o700); err != nil {
		t.Fatalf("create output directory: %v", err)
	}

	err := (Publisher{}).Publish(stagePath, finalPath, true)
	if err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("Publish() error = %v, want non-regular-file error", err)
	}
	assertFileContents(t, stagePath, "new media")
}

func writeTestFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func assertFileContents(t *testing.T, path, want string) {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if got := string(contents); got != want {
		t.Errorf("contents of %s = %q, want %q", path, got, want)
	}
}

func overwriteName(overwrite bool) string {
	if overwrite {
		return "overwrite"
	}
	return "no-overwrite"
}
