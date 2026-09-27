package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// stubEnvVar turns this test binary into a stand-in for ffmpeg and ffprobe, so
// the full Doctor path can be exercised on machines without FFmpeg installed.
// The value is "<encoders>;<muxers>", each a comma-separated list.
const stubEnvVar = "MEDIACONV_TEST_FFMPEG_STUB"

// stubLogEnvVar, when set, makes the stub append one line per invocation so a
// test can count how many subprocesses the code under test actually spawned.
const stubLogEnvVar = "MEDIACONV_TEST_FFMPEG_STUB_LOG"

func TestMain(m *testing.M) {
	if script, ok := os.LookupEnv(stubEnvVar); ok {
		os.Exit(runFFmpegStub(script, os.Args[1:]))
	}
	os.Exit(m.Run())
}

func runFFmpegStub(script string, args []string) int {
	if path := os.Getenv(stubLogEnvVar); path != "" {
		if file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600); err == nil {
			fmt.Fprintln(file, strings.Join(args, " "))
			_ = file.Close()
		}
	}

	encoders, muxers, _ := strings.Cut(script, ";")

	switch {
	case slices.Contains(args, "-version"):
		fmt.Println("ffmpeg version 7.1-stub Copyright (c) the FFmpeg developers")
		fmt.Println("built with stub")
	case slices.Contains(args, "-encoders"):
		printStubCapabilities("Encoders:", "V.....", encoders)
	case slices.Contains(args, "-muxers"):
		printStubCapabilities("Muxers:", "E", muxers)
	default:
		fmt.Fprintf(os.Stderr, "stub: unexpected arguments %v\n", args)
		return 1
	}
	return 0
}

func printStubCapabilities(header, flags, list string) {
	fmt.Println(" " + header)
	fmt.Println(" ------")
	for _, name := range strings.Split(list, ",") {
		if name = strings.TrimSpace(name); name != "" {
			fmt.Printf(" %s %s stub description\n", flags, name)
		}
	}
}

// stubbedService points the service at this test binary acting as FFmpeg.
func stubbedService(t *testing.T, encoders, muxers string) *Service {
	t.Helper()

	executable, err := os.Executable()
	if err != nil {
		t.Skipf("cannot locate the test binary: %v", err)
	}
	t.Setenv(stubEnvVar, encoders+";"+muxers)
	return New(Config{FFmpegPath: executable, FFprobePath: executable})
}

func TestDoctorReportsACompleteFFmpegBuild(t *testing.T) {
	service := stubbedService(t, "libx264,aac,libmp3lame,libvpx-vp9,libopus,pcm_s16le", "mp4,mp3,webm,ipod,wav")
	report := service.Doctor(context.Background())

	if !report.OK {
		t.Errorf("Doctor() reported a problem with a complete build: %+v", report.Checks)
	}
	want := []string{
		"ffmpeg",
		"ffprobe",
		"libx264 encoder",
		"AAC encoder",
		"libmp3lame encoder",
		"libvpx-vp9 encoder",
		"libopus encoder",
		"PCM encoder",
		"MP4 muxer",
		"MP3 muxer",
		"WebM muxer",
		"M4A muxer",
		"WAV muxer",
	}
	if len(report.Checks) != len(want) {
		t.Fatalf("Checks = %d, want %d", len(report.Checks), len(want))
	}
	for i, name := range want {
		if report.Checks[i].Name != name {
			t.Errorf("check %d = %q, want %q", i, report.Checks[i].Name, name)
		}
		if !report.Checks[i].OK {
			t.Errorf("check %q reported a failure", name)
		}
	}
	if !strings.Contains(report.Checks[0].Detail, "7.1-stub") {
		t.Errorf("ffmpeg detail = %q, want it to carry the reported version", report.Checks[0].Detail)
	}
	if !strings.Contains(report.Checks[1].Detail, "7.1-stub") {
		t.Errorf("ffprobe detail = %q, want it to carry the reported version", report.Checks[1].Detail)
	}
}

func TestDoctorFlagsMissingCapabilities(t *testing.T) {
	service := stubbedService(t, "libx264,aac,libvpx-vp9,libopus,pcm_s16le", "mp4,webm,ipod,wav")
	report := service.Doctor(context.Background())

	if report.OK {
		t.Error("Doctor() reported OK while libmp3lame and the MP3 muxer are missing")
	}

	failed := make(map[string]string)
	for _, check := range report.Checks {
		if !check.OK {
			failed[check.Name] = check.Detail
		}
	}
	for _, name := range []string{"libmp3lame encoder", "MP3 muxer"} {
		detail, ok := failed[name]
		if !ok {
			t.Errorf("check %q passed, want it to be reported as missing", name)
			continue
		}
		if detail != "missing" {
			t.Errorf("check %q detail = %q, want \"missing\"", name, detail)
		}
	}
	if len(failed) != 2 {
		t.Errorf("failed checks = %v, want exactly libmp3lame and the MP3 muxer", failed)
	}
}

func TestDoctorAcceptsTheMovMuxerForMP4(t *testing.T) {
	service := stubbedService(t, "libx264,aac,libmp3lame,libvpx-vp9,libopus,pcm_s16le", "mov,mp3,webm,ipod,wav")
	report := service.Doctor(context.Background())

	for _, check := range report.Checks {
		if check.Name == "MP4 muxer" && !check.OK {
			t.Error("MP4 muxer check failed although the mov muxer is available")
		}
	}
}

// Detection costs four subprocesses. Repeating it per file is what made batch
// runs over folders of small inputs spend most of their time not converting.
func TestResolveFFmpegDetectsCapabilitiesOnlyOnce(t *testing.T) {
	spawnLog := filepath.Join(t.TempDir(), "spawns.log")
	t.Setenv(stubLogEnvVar, spawnLog)
	service := stubbedService(t, "libx264,aac,libmp3lame,libvpx-vp9,libopus,pcm_s16le", "mp4,mp3,webm,ipod,wav")

	for attempt := 1; attempt <= 3; attempt++ {
		_, capabilities, err := service.resolveFFmpeg(context.Background())
		if err != nil {
			t.Fatalf("resolveFFmpeg() attempt %d error = %v", attempt, err)
		}
		if !capabilities.HasEncoder("libx264") {
			t.Fatalf("attempt %d lost the detected capabilities", attempt)
		}
	}

	recorded, err := os.ReadFile(spawnLog)
	if err != nil {
		t.Fatalf("read spawn log: %v", err)
	}
	spawns := strings.Count(string(recorded), "\n")
	if spawns != 4 {
		t.Errorf("three calls spawned %d subprocesses, want the 4 of a single detection:\n%s", spawns, recorded)
	}
}
