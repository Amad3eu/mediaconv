package ffmpeg

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// The fixtures below are the verbatim output of a real FFmpeg build, so the
// parser is exercised against the genuine format instead of one invented here.
// See testdata/README.md for the commands that produced them.

//go:embed testdata/ffmpeg-version.txt
var stubVersionOutput string

//go:embed testdata/ffmpeg-encoders.txt
var stubEncodersOutput string

//go:embed testdata/ffmpeg-muxers.txt
var stubMuxersOutput string

// capabilityStubEnvVar turns this test binary into a stand-in for ffmpeg, so
// detection can be tested on machines and CI runners without FFmpeg installed.
// The value selects which invocation, if any, should fail.
const capabilityStubEnvVar = "MEDIACONV_TEST_CAPABILITY_STUB"

const (
	stubHealthy       = "healthy"
	stubFailVersion   = "fail-version"
	stubFailEncoders  = "fail-encoders"
	stubFailMuxers    = "fail-muxers"
	stubFailureDetail = "Unrecognized option, see -help"
)

func TestMain(m *testing.M) {
	if mode, ok := os.LookupEnv(capabilityStubEnvVar); ok {
		os.Exit(runCapabilityStub(mode, os.Args[1:]))
	}
	os.Exit(m.Run())
}

func runCapabilityStub(mode string, args []string) int {
	switch {
	case slices.Contains(args, "-version"):
		// Exits without writing anything, to cover the branch where there is
		// no output to attach to the error.
		if mode == stubFailVersion {
			return 1
		}
		fmt.Print(stubVersionOutput)
	case slices.Contains(args, "-encoders"):
		if mode == stubFailEncoders {
			fmt.Fprintln(os.Stderr, stubFailureDetail)
			return 1
		}
		fmt.Print(stubEncodersOutput)
	case slices.Contains(args, "-muxers"):
		if mode == stubFailMuxers {
			fmt.Fprintln(os.Stderr, stubFailureDetail)
			return 1
		}
		fmt.Print(stubMuxersOutput)
	default:
		fmt.Fprintf(os.Stderr, "stub: unexpected arguments %v\n", args)
		return 1
	}
	return 0
}

// stubPaths points both binaries at this test binary acting as FFmpeg.
func stubPaths(t *testing.T, mode string) Paths {
	t.Helper()

	executable, err := os.Executable()
	if err != nil {
		t.Skipf("cannot locate the test binary: %v", err)
	}
	t.Setenv(capabilityStubEnvVar, mode)
	return Paths{FFmpeg: executable, FFprobe: executable}
}

// Detection costs four subprocesses, so the assertions about a healthy build
// share one run instead of repeating it.
func TestDetectReadsRealFFmpegOutput(t *testing.T) {
	paths := stubPaths(t, stubHealthy)

	capabilities, err := (CapabilityDetector{}).Detect(context.Background(), paths)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	t.Run("the capabilities the six profiles need", func(t *testing.T) {
		// A parser change that silently stops recognizing one of these fails
		// here instead of at conversion time.
		for _, encoder := range []string{"libx264", "aac", "libmp3lame", "libvpx-vp9", "libopus", "pcm_s16le", "gif"} {
			if !capabilities.HasEncoder(encoder) {
				t.Errorf("HasEncoder(%q) = false, want it found in the real output", encoder)
			}
		}
		for _, muxer := range []string{"mp4", "mov", "mp3", "webm", "ipod", "wav", "gif"} {
			if !capabilities.HasMuxer(muxer) {
				t.Errorf("HasMuxer(%q) = false, want it found in the real output", muxer)
			}
		}
	})

	// Only the name column counts. Recording whole lines would make Doctor
	// report every codec as present, which is the failure that hurts most: the
	// check passes and the conversion fails afterwards.
	t.Run("only the name column", func(t *testing.T) {
		// Words from the description column, on the same lines as encoders the
		// profiles use.
		for _, word := range []string{"H.264", "MP3", "Opus", "encoder"} {
			if capabilities.HasEncoder(word) {
				t.Errorf("HasEncoder(%q) = true, want description text ignored", word)
			}
		}
		// "Muxing supported" is the legend printed above the muxer list.
		for _, word := range []string{"Muxing", "supported", "QuickTime"} {
			if capabilities.HasMuxer(word) {
				t.Errorf("HasMuxer(%q) = true, want legend and description text ignored", word)
			}
		}
		if capabilities.HasEncoder("libx265_not_really") {
			t.Error("HasEncoder() answered true for a name that is not in the output")
		}
	})

	// FFmpeg lists some formats under several names on one line, such as
	// "stream_segment,ssegment". Both names have to resolve.
	t.Run("names listed together on one line", func(t *testing.T) {
		for _, muxer := range []string{"stream_segment", "ssegment"} {
			if !capabilities.HasMuxer(muxer) {
				t.Errorf("HasMuxer(%q) = false, want both comma-separated names to resolve", muxer)
			}
		}
	})

	t.Run("the first version line only", func(t *testing.T) {
		for name, version := range map[string]string{
			"ffmpeg":  capabilities.FFmpegVersion,
			"ffprobe": capabilities.FFprobeVersion,
		} {
			if !strings.HasPrefix(version, "ffmpeg version ") {
				t.Errorf("%s version = %q, want the reported version line", name, version)
			}
			if strings.Contains(version, "\n") || strings.Contains(version, "built with") {
				t.Errorf("%s version = %q, want only the first line", name, version)
			}
		}
	})

	t.Run("the paths it was given", func(t *testing.T) {
		if capabilities.FFmpegPath != paths.FFmpeg || capabilities.FFprobePath != paths.FFprobe {
			t.Errorf("Detect() lost the paths: %q and %q", capabilities.FFmpegPath, capabilities.FFprobePath)
		}
	})
}

// Detection asks two different binaries. When one of them is broken, the error
// has to say which, otherwise the user goes looking at the wrong one.
func TestDetectSaysWhichBinaryFailed(t *testing.T) {
	paths := stubPaths(t, stubHealthy)
	paths.FFprobe = filepath.Join(t.TempDir(), executableName("ffprobe"))

	_, err := (CapabilityDetector{}).Detect(context.Background(), paths)
	if err == nil {
		t.Fatal("Detect() succeeded with an ffprobe path that does not exist")
	}
	if !strings.Contains(err.Error(), "read ffprobe version") {
		t.Errorf("Detect() error = %v, want it to name ffprobe", err)
	}
	if strings.Contains(err.Error(), "ffmpeg version") {
		t.Errorf("Detect() error = %v, want it not to blame ffmpeg", err)
	}
}

func TestDetectAttachesTheCommandOutputToTheError(t *testing.T) {
	for _, testCase := range []struct {
		mode string
		want string
	}{
		{mode: stubFailEncoders, want: "list ffmpeg encoders"},
		{mode: stubFailMuxers, want: "list ffmpeg muxers"},
	} {
		t.Run(testCase.mode, func(t *testing.T) {
			paths := stubPaths(t, testCase.mode)

			_, err := (CapabilityDetector{}).Detect(context.Background(), paths)
			if err == nil {
				t.Fatalf("Detect() succeeded while %s fails", testCase.mode)
			}
			if !strings.Contains(err.Error(), testCase.want) {
				t.Errorf("Detect() error = %v, want it to mention %q", err, testCase.want)
			}
			// Without the command's own output the message is just "exit
			// status 1", which tells the user nothing.
			if !strings.Contains(err.Error(), stubFailureDetail) {
				t.Errorf("Detect() error = %v, want it to carry the command output %q", err, stubFailureDetail)
			}
		})
	}
}

// A command can fail silently. The underlying error still has to survive, so
// callers can inspect it.
func TestDetectKeepsTheUnderlyingErrorWhenThereIsNoOutput(t *testing.T) {
	paths := stubPaths(t, stubFailVersion)

	_, err := (CapabilityDetector{}).Detect(context.Background(), paths)
	if err == nil {
		t.Fatal("Detect() succeeded while the version command fails")
	}
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) {
		t.Fatalf("Detect() error = %v, want it to wrap an *exec.ExitError", err)
	}
	if !strings.Contains(err.Error(), "read ffmpeg version") {
		t.Errorf("Detect() error = %v, want it to name the step that failed", err)
	}
}

func TestDetectStopsWhenTheContextIsCancelled(t *testing.T) {
	paths := stubPaths(t, stubHealthy)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := (CapabilityDetector{}).Detect(ctx, paths); err == nil {
		t.Error("Detect() succeeded with a canceled context")
	}
}
