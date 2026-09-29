package ffmpeg

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// writeFakeExecutable creates a file the locator will accept as a binary. The
// contents never run: the locator only stats the file.
func writeFakeExecutable(t *testing.T, directory, name string) string {
	t.Helper()

	path := filepath.Join(directory, executableName(name))
	if err := os.WriteFile(path, []byte("test executable"), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

// Pointing --ffprobe at the folder instead of the binary is an easy mistake to
// make, and the locator is meant to absorb it.
func TestLocateAcceptsADirectoryInsteadOfTheBinary(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	ffmpegPath := writeFakeExecutable(t, directory, "ffmpeg")
	ffprobePath := writeFakeExecutable(t, directory, "ffprobe")

	paths, err := (Locator{}).Locate(directory, directory)
	if err != nil {
		t.Fatalf("Locate() error = %v", err)
	}
	if paths.FFmpeg != ffmpegPath {
		t.Errorf("FFmpeg = %q, want %q", paths.FFmpeg, ffmpegPath)
	}
	if paths.FFprobe != ffprobePath {
		t.Errorf("FFprobe = %q, want %q", paths.FFprobe, ffprobePath)
	}
}

func TestLocateFFprobeResolvesAnOverrideOnItsOwn(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	ffprobePath := writeFakeExecutable(t, directory, "ffprobe")

	located, err := (Locator{}).LocateFFprobe(ffprobePath)
	if err != nil {
		t.Fatalf("LocateFFprobe() error = %v", err)
	}
	if located != ffprobePath {
		t.Errorf("LocateFFprobe() = %q, want %q", located, ffprobePath)
	}
}

// An explicit --ffprobe has to win over the copy sitting next to ffmpeg,
// otherwise the flag would be silently ignored whenever the sibling exists.
func TestLocatePrefersAnExplicitFFprobeOverTheSibling(t *testing.T) {
	t.Parallel()

	ffmpegDirectory := t.TempDir()
	ffmpegPath := writeFakeExecutable(t, ffmpegDirectory, "ffmpeg")
	siblingPath := writeFakeExecutable(t, ffmpegDirectory, "ffprobe")

	chosenPath := writeFakeExecutable(t, t.TempDir(), "ffprobe")

	paths, err := (Locator{}).Locate(ffmpegPath, chosenPath)
	if err != nil {
		t.Fatalf("Locate() error = %v", err)
	}
	if paths.FFprobe == siblingPath {
		t.Fatal("Locate() used the sibling ffprobe and ignored the explicit override")
	}
	if paths.FFprobe != chosenPath {
		t.Errorf("FFprobe = %q, want %q", paths.FFprobe, chosenPath)
	}
}

// A custom ffmpeg with no ffprobe beside it still has to work, by falling back
// to PATH.
func TestLocateFallsBackToPATHWhenFFmpegHasNoSibling(t *testing.T) {
	ffmpegPath := writeFakeExecutable(t, t.TempDir(), "ffmpeg")

	pathDirectory := t.TempDir()
	ffprobePath := writeFakeExecutable(t, pathDirectory, "ffprobe")
	t.Setenv("PATH", pathDirectory)

	paths, err := (Locator{}).Locate(ffmpegPath, "")
	if err != nil {
		t.Fatalf("Locate() error = %v", err)
	}
	if paths.FFprobe != ffprobePath {
		t.Errorf("FFprobe = %q, want the one found in PATH, %q", paths.FFprobe, ffprobePath)
	}
}

func TestLocateReportsAMissingBinary(t *testing.T) {
	t.Parallel()

	missing := filepath.Join(t.TempDir(), executableName("ffmpeg"))

	_, err := (Locator{}).Locate(missing, "")
	if err == nil {
		t.Fatal("Locate() succeeded with a path that does not exist")
	}
	if !strings.Contains(err.Error(), "invalid ffmpeg executable") {
		t.Errorf("Locate() error = %v, want it to name the invalid ffmpeg path", err)
	}
}

func TestLocateRejectsAPathThatIsNotAFile(t *testing.T) {
	t.Parallel()

	// A directory called "ffmpeg" inside the directory that was passed: the
	// locator appends the name, then finds something that is not a file.
	directory := t.TempDir()
	if err := os.Mkdir(filepath.Join(directory, executableName("ffmpeg")), 0o700); err != nil {
		t.Fatal(err)
	}

	_, err := (Locator{}).Locate(directory, "")
	if err == nil {
		t.Fatal("Locate() succeeded with a directory where the binary should be")
	}
	if !strings.Contains(err.Error(), "not a regular file") {
		t.Errorf("Locate() error = %v, want it to report a non-regular file", err)
	}
}

func TestLocateRejectsAFileWithoutThePermissionBit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not carry a permission bit for executables")
	}
	t.Parallel()

	path := filepath.Join(t.TempDir(), "ffmpeg")
	if err := os.WriteFile(path, []byte("not executable"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := (Locator{}).Locate(path, "")
	if err == nil {
		t.Fatal("Locate() succeeded with a file that has no executable bit")
	}
	if !strings.Contains(err.Error(), "not executable") {
		t.Errorf("Locate() error = %v, want it to report the missing executable bit", err)
	}
}

// Falling back to PATH when nothing is there has to fail with a message that
// says what was looked for.
func TestLocateFFmpegReportsAnEmptyPATH(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	_, err := (Locator{}).LocateFFmpeg("")
	if err == nil {
		t.Fatal("LocateFFmpeg() succeeded with nothing in PATH")
	}
	if !strings.Contains(err.Error(), "ffmpeg was not found in PATH") {
		t.Errorf("LocateFFmpeg() error = %v, want it to say ffmpeg was not found", err)
	}
}
