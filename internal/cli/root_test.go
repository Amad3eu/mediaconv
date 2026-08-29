package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Amad3eu/mediaconv/internal/buildinfo"
	"github.com/Amad3eu/mediaconv/internal/failure"
	"github.com/Amad3eu/mediaconv/internal/profile"
)

func TestExecuteRootHelp(t *testing.T) {
	code, stdout, stderr := executeForTest(t, "--help")
	if code != 0 {
		t.Errorf("Execute(--help) code = %d, want 0", code)
	}
	if stderr != "" {
		t.Errorf("Execute(--help) stderr = %q, want empty", stderr)
	}
	for _, fragment := range []string{
		"MediaConv is a script-friendly media conversion CLI.",
		"Usage:",
		"mediaconv [command]",
		"convert",
		"doctor",
		"formats",
		"inspect",
		"version",
		"--ffmpeg-path",
		"--json",
	} {
		if !strings.Contains(stdout, fragment) {
			t.Errorf("help output does not contain %q\noutput:\n%s", fragment, stdout)
		}
	}
}

func TestExecuteConvertHelp(t *testing.T) {
	code, stdout, stderr := executeForTest(t, "convert", "--help")
	if code != 0 || stderr != "" {
		t.Fatalf("Execute(convert --help) = code %d, stderr %q", code, stderr)
	}
	for _, fragment := range []string{
		"mediaconv convert INPUT",
		"--output",
		"--overwrite",
		"--preset",
		"--to",
		"--no-progress",
	} {
		if !strings.Contains(stdout, fragment) {
			t.Errorf("convert help does not contain %q\noutput:\n%s", fragment, stdout)
		}
	}
}

func TestExecuteVersion(t *testing.T) {
	originalVersion, originalCommit, originalDate := buildinfo.Version, buildinfo.Commit, buildinfo.Date
	buildinfo.Version = "v1.2.3"
	buildinfo.Commit = "abc1234"
	buildinfo.Date = "2026-08-29T12:34:56Z"
	t.Cleanup(func() {
		buildinfo.Version = originalVersion
		buildinfo.Commit = originalCommit
		buildinfo.Date = originalDate
	})

	t.Run("version command", func(t *testing.T) {
		code, stdout, stderr := executeForTest(t, "version")
		if code != 0 || stderr != "" {
			t.Fatalf("Execute(version) = code %d, stderr %q", code, stderr)
		}
		want := "mediaconv v1.2.3\ncommit: abc1234\nbuilt: 2026-08-29T12:34:56Z\n"
		if stdout != want {
			t.Errorf("version output = %q, want %q", stdout, want)
		}
	})

	t.Run("cobra version flag", func(t *testing.T) {
		code, stdout, stderr := executeForTest(t, "--version")
		if code != 0 || stderr != "" {
			t.Fatalf("Execute(--version) = code %d, stderr %q", code, stderr)
		}
		if want := "mediaconv v1.2.3\n"; stdout != want {
			t.Errorf("--version output = %q, want %q", stdout, want)
		}
	})

	t.Run("JSON", func(t *testing.T) {
		code, stdout, stderr := executeForTest(t, "--json", "version")
		if code != 0 || stderr != "" {
			t.Fatalf("Execute(--json version) = code %d, stderr %q", code, stderr)
		}
		var envelope struct {
			OK    bool `json:"ok"`
			Build struct {
				Version string `json:"version"`
				Commit  string `json:"commit"`
				Date    string `json:"date"`
			} `json:"build"`
		}
		decodeJSON(t, stdout, &envelope)
		if !envelope.OK || envelope.Build.Version != "v1.2.3" || envelope.Build.Commit != "abc1234" || envelope.Build.Date != "2026-08-29T12:34:56Z" {
			t.Errorf("version JSON = %#v", envelope)
		}
	})
}

func TestExecuteFormats(t *testing.T) {
	t.Run("text", func(t *testing.T) {
		code, stdout, stderr := executeForTest(t, "formats")
		if code != 0 || stderr != "" {
			t.Fatalf("Execute(formats) = code %d, stderr %q", code, stderr)
		}
		want := "WEBM -> MP4  profile=web  video=h264 (libx264)  audio=aac\n"
		if stdout != want {
			t.Errorf("formats output = %q, want %q", stdout, want)
		}
	})

	t.Run("JSON", func(t *testing.T) {
		code, stdout, stderr := executeForTest(t, "--json", "formats")
		if code != 0 || stderr != "" {
			t.Fatalf("Execute(--json formats) = code %d, stderr %q", code, stderr)
		}
		var envelope struct {
			OK      bool                      `json:"ok"`
			Formats []profile.SupportedFormat `json:"formats"`
		}
		decodeJSON(t, stdout, &envelope)
		if !envelope.OK || len(envelope.Formats) != 1 {
			t.Fatalf("formats JSON = %#v", envelope)
		}
		format := envelope.Formats[0]
		if format.Source != "webm" || format.Target != "mp4" || format.Profile != "web" || format.VideoCodec != "h264 (libx264)" || format.AudioCodec != "aac" {
			t.Errorf("format JSON = %#v", format)
		}
	})
}

func TestExecuteWritesUsageErrorsAsJSON(t *testing.T) {
	code, stdout, stderr := executeForTest(t, "--json", "convert")
	if code != failure.ExitUsage {
		t.Errorf("Execute(--json convert) code = %d, want %d", code, failure.ExitUsage)
	}
	if stdout != "" {
		t.Errorf("usage error stdout = %q, want empty", stdout)
	}
	envelope := decodeErrorEnvelope(t, stderr)
	if envelope.OK {
		t.Error("error JSON ok = true")
	}
	if envelope.Error.ExitCode != failure.ExitUsage {
		t.Errorf("error JSON exit_code = %d", envelope.Error.ExitCode)
	}
	if !strings.Contains(envelope.Error.Message, "expects 1 argument(s), received 0") {
		t.Errorf("error JSON message = %q", envelope.Error.Message)
	}
	if !strings.Contains(envelope.Error.Hint, "mediaconv convert --help") {
		t.Errorf("error JSON hint = %q", envelope.Error.Hint)
	}
}

func TestExecuteWritesMissingFFmpegDependenciesAsJSON(t *testing.T) {
	directory := t.TempDir()
	inputPath := filepath.Join(directory, "input.webm")
	if err := os.WriteFile(inputPath, []byte("not parsed because dependency lookup fails first"), 0o600); err != nil {
		t.Fatalf("write input: %v", err)
	}
	missingFFmpeg := filepath.Join(directory, "missing-ffmpeg")
	missingFFprobe := filepath.Join(directory, "missing-ffprobe")

	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "convert",
			args: []string{"--json", "--ffmpeg-path", missingFFmpeg, "--ffprobe-path", missingFFprobe, "convert", inputPath},
			want: "FFmpeg and ffprobe are required",
		},
		{
			name: "inspect",
			args: []string{"--json", "--ffprobe-path", missingFFprobe, "inspect", inputPath},
			want: "ffprobe is required",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			code, stdout, stderr := executeForTest(t, test.args...)
			if code != failure.ExitDependency {
				t.Errorf("Execute() code = %d, want %d", code, failure.ExitDependency)
			}
			if stdout != "" {
				t.Errorf("dependency error stdout = %q, want empty", stdout)
			}
			envelope := decodeErrorEnvelope(t, stderr)
			if envelope.OK || envelope.Error.ExitCode != failure.ExitDependency {
				t.Errorf("dependency JSON = %#v", envelope)
			}
			if !strings.Contains(envelope.Error.Message, test.want) {
				t.Errorf("dependency message = %q, want substring %q", envelope.Error.Message, test.want)
			}
			if !strings.Contains(envelope.Error.Hint, "Install FFmpeg") {
				t.Errorf("dependency hint = %q", envelope.Error.Hint)
			}
		})
	}
}

func TestExecuteDoctorJSONReportsMissingFFmpegWithoutDuplicateError(t *testing.T) {
	directory := t.TempDir()
	missingFFmpeg := filepath.Join(directory, "missing-ffmpeg")
	missingFFprobe := filepath.Join(directory, "missing-ffprobe")

	code, stdout, stderr := executeForTest(
		t,
		"--json",
		"--ffmpeg-path", missingFFmpeg,
		"--ffprobe-path", missingFFprobe,
		"doctor",
	)
	if code != failure.ExitDependency {
		t.Errorf("Execute(doctor) code = %d, want %d", code, failure.ExitDependency)
	}
	if stderr != "" {
		t.Errorf("doctor stderr = %q, want empty because report already contains failures", stderr)
	}
	var report struct {
		OK     bool `json:"ok"`
		Checks []struct {
			Name   string `json:"name"`
			OK     bool   `json:"ok"`
			Detail string `json:"detail"`
		} `json:"checks"`
	}
	decodeJSON(t, stdout, &report)
	if report.OK {
		t.Error("doctor report ok = true")
	}
	if len(report.Checks) != 2 {
		t.Fatalf("doctor checks = %#v, want ffmpeg and ffprobe", report.Checks)
	}
	if report.Checks[0].Name != "ffmpeg" || report.Checks[0].OK || !strings.Contains(report.Checks[0].Detail, "invalid ffmpeg executable") {
		t.Errorf("ffmpeg check = %#v", report.Checks[0])
	}
	if report.Checks[1].Name != "ffprobe" || report.Checks[1].OK || !strings.Contains(report.Checks[1].Detail, "invalid ffprobe executable") {
		t.Errorf("ffprobe check = %#v", report.Checks[1])
	}
}

type errorEnvelope struct {
	OK    bool `json:"ok"`
	Error struct {
		Message  string `json:"message"`
		Hint     string `json:"hint"`
		ExitCode int    `json:"exit_code"`
	} `json:"error"`
}

func executeForTest(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var stdoutBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer
	code = Execute(context.Background(), args, strings.NewReader(""), &stdoutBuffer, &stderrBuffer)
	return code, stdoutBuffer.String(), stderrBuffer.String()
}

func decodeErrorEnvelope(t *testing.T, value string) errorEnvelope {
	t.Helper()
	var envelope errorEnvelope
	decodeJSON(t, value, &envelope)
	return envelope
}

func decodeJSON(t *testing.T, value string, target any) {
	t.Helper()
	decoder := json.NewDecoder(strings.NewReader(value))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		t.Fatalf("decode JSON %q: %v", value, err)
	}
	if decoder.More() {
		t.Fatalf("JSON contains additional values: %q", value)
	}
}
