//go:build integration

package integration_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Amad3eu/mediaconv/internal/app"
)

func TestConvertVP9OpusWebMToMP4(t *testing.T) {
	requireFFmpeg(t)
	directory := t.TempDir()
	input := filepath.Join(directory, "vídeo de entrada.webm")
	output := filepath.Join(directory, "vídeo convertido.mp4")

	generateWebM(t, input, true, "0.5")
	inputBefore, err := os.ReadFile(input)
	if err != nil {
		t.Fatal(err)
	}

	service := app.New(app.Config{})
	result, err := service.Convert(context.Background(), app.ConvertRequest{
		InputPath:  input,
		OutputPath: output,
		Target:     "mp4",
		Preset:     "web",
	}, nil)
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if result.OutputPath != output {
		t.Fatalf("OutputPath = %q, want %q", result.OutputPath, output)
	}
	if result.OutputInfo.VideoStreams()[0].CodecName != "h264" {
		t.Fatalf("video codec = %q, want h264", result.OutputInfo.VideoStreams()[0].CodecName)
	}
	if result.OutputInfo.AudioStreams()[0].CodecName != "aac" {
		t.Fatalf("audio codec = %q, want aac", result.OutputInfo.AudioStreams()[0].CodecName)
	}
	inputAfter, err := os.ReadFile(input)
	if err != nil {
		t.Fatal(err)
	}
	if string(inputBefore) != string(inputAfter) {
		t.Fatal("input file changed during conversion")
	}
	assertNoStagingDirectories(t, directory)
}

func TestConvertWebMWithoutAudio(t *testing.T) {
	requireFFmpeg(t)
	directory := t.TempDir()
	input := filepath.Join(directory, "silent.webm")
	output := filepath.Join(directory, "silent.mp4")
	generateWebM(t, input, false, "0.5")

	result, err := app.New(app.Config{}).Convert(context.Background(), app.ConvertRequest{
		InputPath:  input,
		OutputPath: output,
		Target:     "mp4",
		Preset:     "web",
	}, nil)
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if len(result.OutputInfo.AudioStreams()) != 0 {
		t.Fatalf("audio stream count = %d, want 0", len(result.OutputInfo.AudioStreams()))
	}
	assertNoStagingDirectories(t, directory)
}

func TestTruncatedWebMIsNotPublished(t *testing.T) {
	requireFFmpeg(t)
	directory := t.TempDir()
	input := filepath.Join(directory, "truncated.webm")
	output := filepath.Join(directory, "must-not-exist.mp4")
	generateWebM(t, input, false, "4")

	info, err := os.Stat(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(input, info.Size()/2); err != nil {
		t.Fatal(err)
	}

	_, err = app.New(app.Config{}).Convert(context.Background(), app.ConvertRequest{
		InputPath:  input,
		OutputPath: output,
		Target:     "mp4",
		Preset:     "web",
	}, nil)
	if err == nil {
		t.Fatal("Convert() error = nil for a truncated input")
	}
	if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
		t.Fatalf("output was published for a truncated input: %v", statErr)
	}
	assertNoStagingDirectories(t, directory)
}

func TestExistingOutputIsNotChangedWithoutOverwrite(t *testing.T) {
	directory := t.TempDir()
	input := filepath.Join(directory, "input.webm")
	output := filepath.Join(directory, "output.mp4")
	if err := os.WriteFile(input, []byte("not empty"), 0o600); err != nil {
		t.Fatal(err)
	}
	const previous = "existing output"
	if err := os.WriteFile(output, []byte(previous), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := app.New(app.Config{}).Convert(context.Background(), app.ConvertRequest{
		InputPath:  input,
		OutputPath: output,
		Target:     "mp4",
		Preset:     "web",
	}, nil)
	if err == nil {
		t.Fatal("Convert() error = nil, want an output conflict")
	}
	contents, readErr := os.ReadFile(output)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(contents) != previous {
		t.Fatalf("existing output changed to %q", contents)
	}
}

func generateWebM(t *testing.T, output string, withAudio bool, duration string) {
	t.Helper()
	args := []string{
		"-hide_banner", "-loglevel", "error", "-nostdin", "-y",
		"-f", "lavfi", "-i", "testsrc2=size=160x90:rate=24",
	}
	if withAudio {
		args = append(args,
			"-f", "lavfi", "-i", "sine=frequency=1000:sample_rate=48000",
			"-map", "0:v:0", "-map", "1:a:0",
		)
	}
	args = append(args, "-t", duration, "-c:v", "libvpx-vp9")
	if withAudio {
		args = append(args, "-c:a", "libopus")
	}
	args = append(args, output)
	command := exec.Command("ffmpeg", args...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("generate WebM: %v\n%s", err, output)
	}
}

func requireFFmpeg(t *testing.T) {
	t.Helper()
	for _, binary := range []string{"ffmpeg", "ffprobe"} {
		if _, err := exec.LookPath(binary); err != nil {
			t.Skipf("%s is not available: %v", binary, err)
		}
	}
}

func assertNoStagingDirectories(t *testing.T, directory string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(directory, ".mediaconv-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("staging directories left behind: %v", matches)
	}
}
