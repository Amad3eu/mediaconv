package app

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCollectBatchCandidates(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "b.mov"))
	writeTestFile(t, filepath.Join(root, "camera.qt"))
	writeTestFile(t, filepath.Join(root, "mobile.m4v"))
	writeTestFile(t, filepath.Join(root, "a.webm"))
	writeTestFile(t, filepath.Join(root, "ignore.mp4"))
	writeTestFile(t, filepath.Join(root, "song.wav"))
	writeTestFile(t, filepath.Join(root, "nested", "clip.mkv"))

	got, err := collectBatchCandidates(root, "mp4", false)
	if err != nil {
		t.Fatalf("collectBatchCandidates() error = %v", err)
	}
	want := []string{
		filepath.Join(root, "a.webm"),
		filepath.Join(root, "b.mov"),
		filepath.Join(root, "camera.qt"),
		filepath.Join(root, "mobile.m4v"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("non-recursive candidates = %#v, want %#v", got, want)
	}

	got, err = collectBatchCandidates(root, "mp4", true)
	if err != nil {
		t.Fatalf("collectBatchCandidates(recursive) error = %v", err)
	}
	want = []string{
		filepath.Join(root, "a.webm"),
		filepath.Join(root, "b.mov"),
		filepath.Join(root, "camera.qt"),
		filepath.Join(root, "mobile.m4v"),
		filepath.Join(root, "nested", "clip.mkv"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("recursive candidates = %#v, want %#v", got, want)
	}
}

func TestCollectBatchCandidatesForMP3(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "book.m4b"))
	writeTestFile(t, filepath.Join(root, "clip.opus"))
	writeTestFile(t, filepath.Join(root, "voice.m4a"))
	writeTestFile(t, filepath.Join(root, "recording.oga"))
	writeTestFile(t, filepath.Join(root, "song.wav"))
	writeTestFile(t, filepath.Join(root, "already.mp3"))
	writeTestFile(t, filepath.Join(root, "video.webm"))

	got, err := collectBatchCandidates(root, "mp3", false)
	if err != nil {
		t.Fatalf("collectBatchCandidates() error = %v", err)
	}
	want := []string{
		filepath.Join(root, "book.m4b"),
		filepath.Join(root, "clip.opus"),
		filepath.Join(root, "recording.oga"),
		filepath.Join(root, "song.wav"),
		filepath.Join(root, "voice.m4a"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("candidates = %#v, want %#v", got, want)
	}
}

func TestBatchOutputPathPreservesRelativePathAndChangesExtension(t *testing.T) {
	t.Parallel()

	inputDir := filepath.Join("media")
	outputDir := filepath.Join("converted")
	inputPath := filepath.Join("media", "nested", "clip.webm")

	got, err := batchOutputPath(inputDir, outputDir, inputPath, "mp4")
	if err != nil {
		t.Fatalf("batchOutputPath() error = %v", err)
	}
	want := filepath.Join("converted", "nested", "clip.mp4")
	if got != want {
		t.Errorf("batchOutputPath() = %q, want %q", got, want)
	}
}

func writeTestFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create parent directory: %v", err)
	}
	if err := os.WriteFile(path, []byte("test"), 0o600); err != nil {
		t.Fatalf("write test file: %v", err)
	}
}
