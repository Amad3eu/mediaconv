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

func TestConvertMOVToMP4(t *testing.T) {
	requireFFmpeg(t)
	directory := t.TempDir()
	input := filepath.Join(directory, "camera.mov")
	output := filepath.Join(directory, "camera.mp4")
	generateMOV(t, input)

	result, err := app.New(app.Config{}).Convert(context.Background(), app.ConvertRequest{
		InputPath:  input,
		OutputPath: output,
		Target:     "mp4",
		Preset:     "web",
	}, nil)
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if result.Plan.SourceFormat != "mov" {
		t.Fatalf("SourceFormat = %q, want mov", result.Plan.SourceFormat)
	}
	if result.OutputInfo.VideoStreams()[0].CodecName != "h264" {
		t.Fatalf("video codec = %q, want h264", result.OutputInfo.VideoStreams()[0].CodecName)
	}
	assertNoStagingDirectories(t, directory)
}

// The counterpart of TestConvertVP9OpusWebMToMP4: MP4 in, WebM out. VP9 takes
// different flags from libx264 for the same intent, so this proves the adapter
// translates them into something FFmpeg actually accepts.
func TestConvertMOVToWebM(t *testing.T) {
	requireFFmpeg(t)
	directory := t.TempDir()
	input := filepath.Join(directory, "camera.mov")
	output := filepath.Join(directory, "camera.webm")
	generateMOV(t, input)

	result, err := app.New(app.Config{}).Convert(context.Background(), app.ConvertRequest{
		InputPath:  input,
		OutputPath: output,
		Target:     "webm",
	}, nil)
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	if result.Plan.Profile != "stream" {
		t.Errorf("Profile = %q, want stream", result.Plan.Profile)
	}
	videos := result.OutputInfo.VideoStreams()
	if len(videos) == 0 || videos[0].CodecName != "vp9" {
		t.Fatalf("video codec = %+v, want vp9", videos)
	}
	if !hasFormatName(result.OutputInfo.FormatNames, "webm") {
		t.Errorf("format names = %v, want a webm container", result.OutputInfo.FormatNames)
	}
	assertNoStagingDirectories(t, directory)
}

// A round trip has to survive both directions, because each one re-encodes.
func TestRoundTripWebMToMP4AndBack(t *testing.T) {
	requireFFmpeg(t)
	directory := t.TempDir()
	original := filepath.Join(directory, "original.webm")
	asMP4 := filepath.Join(directory, "step.mp4")
	backToWebM := filepath.Join(directory, "back.webm")
	generateWebM(t, original, true, "0.5")

	service := app.New(app.Config{})
	if _, err := service.Convert(context.Background(), app.ConvertRequest{
		InputPath: original, OutputPath: asMP4, Target: "mp4",
	}, nil); err != nil {
		t.Fatalf("webm to mp4 error = %v", err)
	}

	result, err := service.Convert(context.Background(), app.ConvertRequest{
		InputPath: asMP4, OutputPath: backToWebM, Target: "webm",
	}, nil)
	if err != nil {
		t.Fatalf("mp4 back to webm error = %v", err)
	}

	videos := result.OutputInfo.VideoStreams()
	if len(videos) == 0 || videos[0].CodecName != "vp9" {
		t.Fatalf("video codec = %+v, want vp9", videos)
	}
	audios := result.OutputInfo.AudioStreams()
	if len(audios) == 0 || audios[0].CodecName != "opus" {
		t.Fatalf("audio codec = %+v, want opus", audios)
	}
	assertNoStagingDirectories(t, directory)
}

func hasFormatName(names []string, want string) bool {
	for _, name := range names {
		if name == want {
			return true
		}
	}
	return false
}

func TestConvertWAVToMP3(t *testing.T) {
	requireFFmpeg(t)
	directory := t.TempDir()
	input := filepath.Join(directory, "song.wav")
	output := filepath.Join(directory, "song.mp3")
	generateWAV(t, input)

	result, err := app.New(app.Config{}).Convert(context.Background(), app.ConvertRequest{
		InputPath:  input,
		OutputPath: output,
		Target:     "mp3",
	}, nil)
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if result.Plan.SourceFormat != "wav" {
		t.Fatalf("SourceFormat = %q, want wav", result.Plan.SourceFormat)
	}
	if result.Plan.Profile != "music" {
		t.Fatalf("Profile = %q, want music", result.Plan.Profile)
	}
	if len(result.OutputInfo.VideoStreams()) != 0 {
		t.Fatalf("video stream count = %d, want 0", len(result.OutputInfo.VideoStreams()))
	}
	if result.OutputInfo.AudioStreams()[0].CodecName != "mp3" {
		t.Fatalf("audio codec = %q, want mp3", result.OutputInfo.AudioStreams()[0].CodecName)
	}
	assertNoStagingDirectories(t, directory)
}

// The audio targets both have a trap the unit tests can only describe: PCM
// rejects a bitrate, and FFmpeg has no muxer named m4a. Only a real run proves
// the adapter got them right.
func TestConvertWAVToM4AAndBack(t *testing.T) {
	requireFFmpeg(t)
	directory := t.TempDir()
	source := filepath.Join(directory, "song.wav")
	asM4A := filepath.Join(directory, "song.m4a")
	backToWAV := filepath.Join(directory, "round.wav")
	generateWAV(t, source)

	service := app.New(app.Config{})
	result, err := service.Convert(context.Background(), app.ConvertRequest{
		InputPath: source, OutputPath: asM4A, Target: "m4a",
	}, nil)
	if err != nil {
		t.Fatalf("wav to m4a error = %v", err)
	}
	audios := result.OutputInfo.AudioStreams()
	if len(audios) == 0 || audios[0].CodecName != "aac" {
		t.Fatalf("audio codec = %+v, want aac", audios)
	}
	if len(result.OutputInfo.VideoStreams()) != 0 {
		t.Errorf("audio target produced a video stream: %+v", result.OutputInfo.VideoStreams())
	}

	result, err = service.Convert(context.Background(), app.ConvertRequest{
		InputPath: asM4A, OutputPath: backToWAV, Target: "wav",
	}, nil)
	if err != nil {
		t.Fatalf("m4a to wav error = %v", err)
	}
	audios = result.OutputInfo.AudioStreams()
	if len(audios) == 0 || audios[0].CodecName != "pcm_s16le" {
		t.Fatalf("audio codec = %+v, want pcm_s16le", audios)
	}
	assertNoStagingDirectories(t, directory)
}

func TestBatchConvertWAVToMP3(t *testing.T) {
	requireFFmpeg(t)
	directory := t.TempDir()
	inputDir := filepath.Join(directory, "input")
	outputDir := filepath.Join(directory, "output")
	if err := os.MkdirAll(inputDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatal(err)
	}
	generateWAV(t, filepath.Join(inputDir, "first.wav"))
	generateWAV(t, filepath.Join(inputDir, "second.wav"))

	result, err := app.New(app.Config{}).BatchConvert(context.Background(), app.BatchRequest{
		InputDir:  inputDir,
		OutputDir: outputDir,
		Target:    "mp3",
	})
	if err != nil {
		t.Fatalf("BatchConvert() error = %v", err)
	}
	if result.Total != 2 || result.Converted != 2 || result.Failed != 0 {
		t.Fatalf("BatchConvert() totals = total %d, converted %d, failed %d; want 2, 2, 0", result.Total, result.Converted, result.Failed)
	}
	for _, name := range []string{"first.mp3", "second.mp3"} {
		if _, err := os.Stat(filepath.Join(outputDir, name)); err != nil {
			t.Fatalf("expected output %s: %v", name, err)
		}
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

func generateMOV(t *testing.T, output string) {
	t.Helper()
	args := []string{
		"-hide_banner", "-loglevel", "error", "-nostdin", "-y",
		"-f", "lavfi", "-i", "testsrc2=size=160x90:rate=24",
		"-t", "0.5",
		"-c:v", "mpeg4",
		"-q:v", "5",
		output,
	}
	command := exec.Command("ffmpeg", args...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("generate MOV: %v\n%s", err, output)
	}
}

func generateWAV(t *testing.T, output string) {
	t.Helper()
	args := []string{
		"-hide_banner", "-loglevel", "error", "-nostdin", "-y",
		"-f", "lavfi", "-i", "sine=frequency=1000:sample_rate=44100",
		"-t", "0.5",
		"-c:a", "pcm_s16le",
		output,
	}
	command := exec.Command("ffmpeg", args...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("generate WAV: %v\n%s", err, output)
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
