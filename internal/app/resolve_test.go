package app

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Amad3eu/mediaconv/internal/failure"
	"github.com/Amad3eu/mediaconv/internal/output"
)

func TestResolveInputRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	directory := filepath.Join(root, "folder")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatalf("create directory: %v", err)
	}
	empty := filepath.Join(root, "empty.webm")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatalf("write empty file: %v", err)
	}

	tests := []struct {
		name  string
		input string
		want  failure.Kind
	}{
		{name: "empty", input: "", want: failure.Usage},
		{name: "whitespace", input: "   ", want: failure.Usage},
		{name: "missing", input: filepath.Join(root, "absent.webm"), want: failure.Input},
		{name: "directory", input: directory, want: failure.Input},
		{name: "empty file", input: empty, want: failure.Input},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, _, err := resolveInput(test.input)
			assertFailureKind(t, err, test.want)
		})
	}
}

func TestResolveInputReturnsAbsolutePathAndInfo(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	input := filepath.Join(root, "recording.webm")
	writeTestFile(t, input)

	path, info, err := resolveInput(input)
	if err != nil {
		t.Fatalf("resolveInput() error = %v", err)
	}
	if !filepath.IsAbs(path) {
		t.Errorf("resolveInput() path = %q, want an absolute path", path)
	}
	if path != filepath.Clean(input) {
		t.Errorf("resolveInput() path = %q, want %q", path, filepath.Clean(input))
	}
	if info == nil {
		t.Fatal("resolveInput() info = nil, want file info")
	}
	if !info.Mode().IsRegular() {
		t.Errorf("resolveInput() info mode = %v, want a regular file", info.Mode())
	}
}

func TestResolveOutputDerivesPathFromInput(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	video := filepath.Join(root, "recording.webm")
	writeTestFile(t, video)
	audio := filepath.Join(root, "song.wav")
	writeTestFile(t, audio)

	tests := []struct {
		name   string
		input  string
		target string
		want   string
	}{
		{name: "defaults to mp4", input: video, target: "", want: filepath.Join(root, "recording.mp4")},
		{name: "explicit mp4", input: video, target: "mp4", want: filepath.Join(root, "recording.mp4")},
		{name: "explicit mp3", input: audio, target: "mp3", want: filepath.Join(root, "song.mp3")},
		{name: "target is case insensitive", input: video, target: "MP4", want: filepath.Join(root, "recording.mp4")},
		{name: "target is trimmed", input: audio, target: "  mp3  ", want: filepath.Join(root, "song.mp3")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			info, err := os.Stat(test.input)
			if err != nil {
				t.Fatalf("stat input: %v", err)
			}
			got, err := resolveOutput(test.input, info, "", test.target, false)
			if err != nil {
				t.Fatalf("resolveOutput() error = %v", err)
			}
			if got != test.want {
				t.Errorf("resolveOutput() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestResolveOutputRejectsConflicts(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	input := filepath.Join(root, "recording.webm")
	writeTestFile(t, input)
	inputInfo, err := os.Stat(input)
	if err != nil {
		t.Fatalf("stat input: %v", err)
	}
	existing := filepath.Join(root, "taken.mp4")
	writeTestFile(t, existing)
	directoryOutput := filepath.Join(root, "folder.mp4")
	if err := os.MkdirAll(directoryOutput, 0o755); err != nil {
		t.Fatalf("create directory output: %v", err)
	}

	tests := []struct {
		name      string
		requested string
		target    string
		overwrite bool
		want      failure.Kind
	}{
		{name: "unsupported target", requested: "", target: "avi", want: failure.Usage},
		{
			name:      "extension does not match target",
			requested: filepath.Join(root, "out.mkv"),
			target:    "mp4",
			want:      failure.Usage,
		},
		{
			name:      "missing output directory",
			requested: filepath.Join(root, "absent", "out.mp4"),
			target:    "mp4",
			want:      failure.OutputConflict,
		},
		{
			name:      "existing output without overwrite",
			requested: existing,
			target:    "mp4",
			want:      failure.OutputConflict,
		},
		{
			name:      "output is a directory",
			requested: directoryOutput,
			target:    "mp4",
			overwrite: true,
			want:      failure.OutputConflict,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := resolveOutput(input, inputInfo, test.requested, test.target, test.overwrite)
			assertFailureKind(t, err, test.want)
		})
	}
}

func TestResolveOutputExistingFileRequiresOverwrite(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	input := filepath.Join(root, "recording.webm")
	writeTestFile(t, input)
	inputInfo, err := os.Stat(input)
	if err != nil {
		t.Fatalf("stat input: %v", err)
	}
	existing := filepath.Join(root, "recording.mp4")
	writeTestFile(t, existing)

	_, err = resolveOutput(input, inputInfo, existing, "mp4", false)
	assertFailureKind(t, err, failure.OutputConflict)
	if !errors.Is(err, output.ErrExists) {
		t.Errorf("resolveOutput() error = %v, want it to wrap output.ErrExists", err)
	}

	got, err := resolveOutput(input, inputInfo, existing, "mp4", true)
	if err != nil {
		t.Fatalf("resolveOutput(overwrite) error = %v", err)
	}
	if got != existing {
		t.Errorf("resolveOutput(overwrite) = %q, want %q", got, existing)
	}
}

func TestResolveOutputRejectsSamePathAsInput(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	input := filepath.Join(root, "clip.mp4")
	writeTestFile(t, input)
	inputInfo, err := os.Stat(input)
	if err != nil {
		t.Fatalf("stat input: %v", err)
	}

	_, err = resolveOutput(input, inputInfo, input, "mp4", true)
	assertFailureKind(t, err, failure.OutputConflict)
}

func TestResolveOutputRejectsHardLinkToInput(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	input := filepath.Join(root, "clip.mp4")
	writeTestFile(t, input)
	linked := filepath.Join(root, "same.mp4")
	if err := os.Link(input, linked); err != nil {
		t.Skipf("hard links are unavailable on this filesystem: %v", err)
	}
	inputInfo, err := os.Stat(input)
	if err != nil {
		t.Fatalf("stat input: %v", err)
	}

	_, err = resolveOutput(input, inputInfo, linked, "mp4", true)
	assertFailureKind(t, err, failure.OutputConflict)
}

func TestResolveOutputRejectsSymlink(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks on Windows requires elevated privileges")
	}

	root := t.TempDir()
	input := filepath.Join(root, "recording.webm")
	writeTestFile(t, input)
	inputInfo, err := os.Stat(input)
	if err != nil {
		t.Fatalf("stat input: %v", err)
	}
	target := filepath.Join(root, "real.mp4")
	writeTestFile(t, target)
	link := filepath.Join(root, "link.mp4")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks are unavailable on this filesystem: %v", err)
	}

	_, err = resolveOutput(input, inputInfo, link, "mp4", true)
	assertFailureKind(t, err, failure.OutputConflict)
	if !errors.Is(err, output.ErrSymlink) {
		t.Errorf("resolveOutput() error = %v, want it to wrap output.ErrSymlink", err)
	}
}

func TestResolveInputDir(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	file := filepath.Join(root, "clip.webm")
	writeTestFile(t, file)

	t.Run("accepts a directory", func(t *testing.T) {
		t.Parallel()

		got, err := resolveInputDir(root)
		if err != nil {
			t.Fatalf("resolveInputDir() error = %v", err)
		}
		if got != filepath.Clean(root) {
			t.Errorf("resolveInputDir() = %q, want %q", got, filepath.Clean(root))
		}
	})

	tests := []struct {
		name  string
		input string
		want  failure.Kind
	}{
		{name: "empty", input: "", want: failure.Usage},
		{name: "whitespace", input: "  ", want: failure.Usage},
		{name: "missing", input: filepath.Join(root, "absent"), want: failure.Input},
		{name: "regular file", input: file, want: failure.Input},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := resolveInputDir(test.input)
			assertFailureKind(t, err, test.want)
		})
	}
}

func TestNormalizeTarget(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		target  string
		want    string
		wantErr bool
	}{
		{name: "defaults to mp4", target: "", want: "mp4"},
		{name: "lowercases", target: "MP4", want: "mp4"},
		{name: "trims", target: "  mp3  ", want: "mp3"},
		{name: "rejects unsupported", target: "avi", wantErr: true},
		{name: "rejects arbitrary text", target: "not a format", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := normalizeTarget(test.target)
			if test.wantErr {
				assertFailureKind(t, err, failure.Usage)
				return
			}
			if err != nil {
				t.Fatalf("normalizeTarget() error = %v", err)
			}
			if got != test.want {
				t.Errorf("normalizeTarget() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestResolveBatchOutputDir(t *testing.T) {
	t.Parallel()

	t.Run("defaults to the input directory", func(t *testing.T) {
		t.Parallel()

		inputDir := t.TempDir()
		got, err := resolveBatchOutputDir(inputDir, "")
		if err != nil {
			t.Fatalf("resolveBatchOutputDir() error = %v", err)
		}
		if got != inputDir {
			t.Errorf("resolveBatchOutputDir() = %q, want %q", got, inputDir)
		}
	})

	t.Run("creates a missing directory", func(t *testing.T) {
		t.Parallel()

		inputDir := t.TempDir()
		requested := filepath.Join(inputDir, "converted", "nested")

		got, err := resolveBatchOutputDir(inputDir, requested)
		if err != nil {
			t.Fatalf("resolveBatchOutputDir() error = %v", err)
		}
		if got != filepath.Clean(requested) {
			t.Errorf("resolveBatchOutputDir() = %q, want %q", got, filepath.Clean(requested))
		}
		info, err := os.Stat(got)
		if err != nil {
			t.Fatalf("stat output directory: %v", err)
		}
		if !info.IsDir() {
			t.Errorf("resolveBatchOutputDir() created %v, want a directory", info.Mode())
		}
	})

	t.Run("rejects a regular file", func(t *testing.T) {
		t.Parallel()

		inputDir := t.TempDir()
		file := filepath.Join(inputDir, "not-a-directory")
		writeTestFile(t, file)

		_, err := resolveBatchOutputDir(inputDir, file)
		assertFailureKind(t, err, failure.OutputConflict)
	})
}

func TestBatchInputExtensionsExcludeTheTargetFormat(t *testing.T) {
	t.Parallel()

	if batchInputExtensions("mp4")[".mp4"] {
		t.Error("mp4 batch inputs include .mp4, which would convert files onto themselves")
	}
	if batchInputExtensions("mp3")[".mp3"] {
		t.Error("mp3 batch inputs include .mp3, which would convert files onto themselves")
	}
	if !batchInputExtensions("mp4")[".webm"] {
		t.Error("mp4 batch inputs are missing .webm")
	}
	if !batchInputExtensions("mp3")[".wav"] {
		t.Error("mp3 batch inputs are missing .wav")
	}
}

func TestSamePath(t *testing.T) {
	t.Parallel()

	if !samePath(filepath.Join("a", "b"), filepath.Join("a", ".", "b")) {
		t.Error("samePath() did not treat equivalent unclean paths as equal")
	}
	if samePath(filepath.Join("a", "b"), filepath.Join("a", "c")) {
		t.Error("samePath() treated distinct paths as equal")
	}

	mixedCase := samePath(filepath.Join("a", "B"), filepath.Join("a", "b"))
	if runtime.GOOS == "windows" && !mixedCase {
		t.Error("samePath() is case sensitive on Windows, where paths are not")
	}
	if runtime.GOOS != "windows" && mixedCase {
		t.Error("samePath() is case insensitive on a case-sensitive platform")
	}
}

func assertFailureKind(t *testing.T, err error, want failure.Kind) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected a %v failure, got nil", want)
	}
	var target *failure.Error
	if !errors.As(err, &target) {
		t.Fatalf("error %v is not a *failure.Error", err)
	}
	if target.Kind != want {
		t.Errorf("failure kind = %v, want %v (error: %v)", target.Kind, want, err)
	}
}
