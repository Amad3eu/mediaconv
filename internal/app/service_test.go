package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Amad3eu/mediaconv/internal/failure"
	"github.com/Amad3eu/mediaconv/internal/output"
	"github.com/Amad3eu/mediaconv/internal/profile"
)

// missingBinaries points the service at paths that cannot exist, so tests stay
// deterministic whether or not FFmpeg is installed on the machine running them.
func missingBinaries(t *testing.T) Config {
	t.Helper()

	root := t.TempDir()
	return Config{
		FFmpegPath:  filepath.Join(root, "absent-ffmpeg"),
		FFprobePath: filepath.Join(root, "absent-ffprobe"),
	}
}

func TestConvertRejectsInputBeforeTouchingFFmpeg(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	input := filepath.Join(root, "recording.webm")
	writeTestFile(t, input)

	tests := []struct {
		name    string
		request ConvertRequest
		want    failure.Kind
	}{
		{name: "empty input", request: ConvertRequest{}, want: failure.Usage},
		{
			name:    "missing input",
			request: ConvertRequest{InputPath: filepath.Join(root, "absent.webm")},
			want:    failure.Input,
		},
		{
			name:    "unsupported target",
			request: ConvertRequest{InputPath: input, Target: "avi"},
			want:    failure.Usage,
		},
		{
			name: "output extension does not match target",
			request: ConvertRequest{
				InputPath:  input,
				OutputPath: filepath.Join(root, "out.mkv"),
				Target:     "mp4",
			},
			want: failure.Usage,
		},
	}

	service := New(missingBinaries(t))
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := service.Convert(context.Background(), test.request, nil)
			assertFailureKind(t, err, test.want)
		})
	}
}

func TestConvertReportsMissingFFmpegAsDependencyFailure(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	input := filepath.Join(root, "recording.webm")
	writeTestFile(t, input)

	service := New(missingBinaries(t))
	_, err := service.Convert(context.Background(), ConvertRequest{InputPath: input}, nil)

	assertFailureKind(t, err, failure.Dependency)
	if code := failure.ExitCode(err); code != failure.ExitDependency {
		t.Errorf("exit code = %d, want %d", code, failure.ExitDependency)
	}
}

func TestConvertDoesNotCreateOutputWhenItFails(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	input := filepath.Join(root, "recording.webm")
	writeTestFile(t, input)
	outputPath := filepath.Join(root, "recording.mp4")

	service := New(missingBinaries(t))
	if _, err := service.Convert(context.Background(), ConvertRequest{InputPath: input}, nil); err == nil {
		t.Fatal("Convert() error = nil, want a dependency failure")
	}

	if _, err := os.Stat(outputPath); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("stat output = %v, want the output to be absent after a failure", err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read directory: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("directory holds %d entries, want only the input file", len(entries))
	}
}

func TestBatchConvertRejectsInvalidRequests(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	file := filepath.Join(root, "clip.webm")
	writeTestFile(t, file)
	emptyDir := t.TempDir()

	tests := []struct {
		name    string
		request BatchRequest
		want    failure.Kind
	}{
		{name: "empty directory argument", request: BatchRequest{}, want: failure.Usage},
		{
			name:    "missing directory",
			request: BatchRequest{InputDir: filepath.Join(root, "absent")},
			want:    failure.Input,
		},
		{
			name:    "file instead of directory",
			request: BatchRequest{InputDir: file},
			want:    failure.Input,
		},
		{
			name:    "unsupported target",
			request: BatchRequest{InputDir: root, Target: "avi"},
			want:    failure.Usage,
		},
		{
			name:    "no supported inputs",
			request: BatchRequest{InputDir: emptyDir, Target: "mp4"},
			want:    failure.Input,
		},
	}

	service := New(missingBinaries(t))
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := service.BatchConvert(context.Background(), test.request)
			assertFailureKind(t, err, test.want)
		})
	}
}

func TestBatchConvertRecordsPerFileFailuresWithoutAbortingTheRun(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "a.webm"))
	writeTestFile(t, filepath.Join(root, "b.mov"))
	outputDir := filepath.Join(root, "converted")

	service := New(missingBinaries(t))
	result, err := service.BatchConvert(context.Background(), BatchRequest{
		InputDir:  root,
		OutputDir: outputDir,
		Target:    "mp4",
	})
	if err != nil {
		t.Fatalf("BatchConvert() error = %v, want per-item failures instead", err)
	}

	if result.Total != 2 {
		t.Errorf("Total = %d, want 2", result.Total)
	}
	if result.Failed != 2 {
		t.Errorf("Failed = %d, want 2", result.Failed)
	}
	if result.Converted != 0 {
		t.Errorf("Converted = %d, want 0", result.Converted)
	}
	if len(result.Items) != 2 {
		t.Fatalf("Items = %d, want 2", len(result.Items))
	}
	for _, item := range result.Items {
		if item.OK {
			t.Errorf("item %q reported OK, want a failure", item.InputPath)
		}
		if item.Error == "" {
			t.Errorf("item %q has an empty error message", item.InputPath)
		}
	}
	if result.OutputDir != filepath.Clean(outputDir) {
		t.Errorf("OutputDir = %q, want %q", result.OutputDir, filepath.Clean(outputDir))
	}
}

func TestBatchConvertStopsOnCancelledContext(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "a.webm"))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	service := New(missingBinaries(t))
	result, err := service.BatchConvert(ctx, BatchRequest{InputDir: root, Target: "mp4"})

	assertFailureKind(t, err, failure.Interrupted)
	if code := failure.ExitCode(err); code != failure.ExitInterrupted {
		t.Errorf("exit code = %d, want %d", code, failure.ExitInterrupted)
	}
	if len(result.Items) != 0 {
		t.Errorf("Items = %d, want no item processed after cancellation", len(result.Items))
	}
}

func TestDoctorReportsMissingBinaries(t *testing.T) {
	t.Parallel()

	service := New(missingBinaries(t))
	report := service.Doctor(context.Background())

	if report.OK {
		t.Error("Doctor() reported OK with missing binaries")
	}
	if len(report.Checks) != 2 {
		t.Fatalf("Checks = %d, want the report to stop after ffmpeg and ffprobe", len(report.Checks))
	}
	for _, check := range report.Checks {
		if check.OK {
			t.Errorf("check %q reported OK, want a failure", check.Name)
		}
		if check.Detail == "" {
			t.Errorf("check %q has an empty detail", check.Name)
		}
	}
}

func TestDoctorReportAddTracksOverallStatus(t *testing.T) {
	t.Parallel()

	report := DoctorReport{OK: true}
	report.add("first", true, "available")
	if !report.OK {
		t.Error("report.OK = false after only successful checks")
	}

	report.add("second", false, "missing")
	if report.OK {
		t.Error("report.OK = true after a failed check")
	}
	if len(report.Checks) != 2 {
		t.Fatalf("Checks = %d, want 2", len(report.Checks))
	}
	if report.Checks[1].Name != "second" || report.Checks[1].Detail != "missing" {
		t.Errorf("second check = %+v, want name \"second\" and detail \"missing\"", report.Checks[1])
	}
}

func TestChooseDetailPrefersTheError(t *testing.T) {
	t.Parallel()

	if got := chooseDetail("/usr/bin/ffmpeg", nil); got != "/usr/bin/ffmpeg" {
		t.Errorf("chooseDetail() = %q, want the path", got)
	}
	if got := chooseDetail("/usr/bin/ffmpeg", errors.New("not found")); got != "not found" {
		t.Errorf("chooseDetail() = %q, want the error message", got)
	}
}

func TestCapabilityDetail(t *testing.T) {
	t.Parallel()

	if got := capabilityDetail(true); got != "available" {
		t.Errorf("capabilityDetail(true) = %q, want \"available\"", got)
	}
	if got := capabilityDetail(false); got != "missing" {
		t.Errorf("capabilityDetail(false) = %q, want \"missing\"", got)
	}
}

func TestFormatsAreNotEmpty(t *testing.T) {
	t.Parallel()

	formats := New(Config{}).Formats()
	if len(formats) == 0 {
		t.Fatal("Formats() returned no supported conversions")
	}
}

func TestPlanFailureMapsProfileErrorsToExitCodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want failure.Kind
	}{
		{name: "missing capability", err: profile.ErrMissingCapability, want: failure.Dependency},
		{name: "unsupported target", err: profile.ErrUnsupportedTarget, want: failure.Usage},
		{name: "unsupported preset", err: profile.ErrUnsupportedPreset, want: failure.Usage},
		{
			name: "wrapped missing capability",
			err:  fmt.Errorf("plan web profile: %w", profile.ErrMissingCapability),
			want: failure.Dependency,
		},
		{name: "unknown error", err: errors.New("something else"), want: failure.Input},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := planFailure(test.err)
			assertFailureKind(t, err, test.want)
			if !errors.Is(err, test.err) {
				t.Errorf("planFailure() dropped the cause %v", test.err)
			}
		})
	}
}

func TestPublishFailureAlwaysReportsAnOutputConflict(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
	}{
		{name: "already exists", err: output.ErrExists},
		{name: "symlink", err: output.ErrSymlink},
		{name: "wrapped symlink", err: fmt.Errorf("publish: %w", output.ErrSymlink)},
		{name: "unknown error", err: errors.New("permission denied")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := publishFailure(test.err)
			assertFailureKind(t, err, failure.OutputConflict)
			if !errors.Is(err, test.err) {
				t.Errorf("publishFailure() dropped the cause %v", test.err)
			}
		})
	}
}

func TestDependencyOrInterruptedPrefersInterruption(t *testing.T) {
	t.Parallel()

	cause := errors.New("detect capabilities")

	err := dependencyOrInterrupted(context.Background(), "message", cause)
	assertFailureKind(t, err, failure.Dependency)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = dependencyOrInterrupted(ctx, "message", cause)
	assertFailureKind(t, err, failure.Interrupted)
}

func TestInterruptedUsesTheConventionalExitCode(t *testing.T) {
	t.Parallel()

	err := interrupted(context.Canceled)
	assertFailureKind(t, err, failure.Interrupted)
	if code := failure.ExitCode(err); code != failure.ExitInterrupted {
		t.Errorf("exit code = %d, want %d", code, failure.ExitInterrupted)
	}
	if !errors.Is(err, context.Canceled) {
		t.Error("interrupted() dropped the context cause")
	}
}

func TestInspectRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	service := New(missingBinaries(t))

	_, err := service.Inspect(context.Background(), "")
	assertFailureKind(t, err, failure.Usage)

	root := t.TempDir()
	input := filepath.Join(root, "recording.webm")
	writeTestFile(t, input)
	_, err = service.Inspect(context.Background(), input)
	assertFailureKind(t, err, failure.Dependency)
}
