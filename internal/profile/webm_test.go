package profile

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/Amad3eu/mediaconv/internal/media"
)

func webmCapabilities() media.Capabilities {
	return media.Capabilities{
		Encoders: map[string]bool{"libvpx-vp9": true, "libopus": true},
		Muxers:   map[string]bool{"webm": true},
	}
}

func mp4Input(width, height int, withAudio bool) media.Info {
	info := media.Info{
		FormatNames: []string{"mov", "mp4", "m4a"},
		Duration:    5 * time.Second,
		Streams: []media.Stream{
			{Index: 0, CodecType: "video", CodecName: "h264", Width: width, Height: height, PixelFormat: "yuv420p"},
		},
	}
	if withAudio {
		info.Streams = append(info.Streams, media.Stream{Index: 1, CodecType: "audio", CodecName: "aac", Channels: 2})
	}
	return info
}

func TestPlanWebMUsesVP9AndOpus(t *testing.T) {
	t.Parallel()

	plan, err := (Registry{}).Plan(
		filepath.Join("in", "clip.mp4"), filepath.Join("out", "clip.webm"),
		"webm", "", mp4Input(1920, 1080, true), webmCapabilities(),
	)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if plan.TargetFormat != "webm" || plan.Profile != "stream" {
		t.Errorf("plan targets %q with profile %q, want webm/stream", plan.TargetFormat, plan.Profile)
	}
	if plan.Video == nil || plan.Video.Codec != "libvpx-vp9" {
		t.Fatalf("video settings = %+v, want libvpx-vp9", plan.Video)
	}
	if plan.Video.PixelFormat != "yuv420p" {
		t.Errorf("pixel format = %q, want yuv420p", plan.Video.PixelFormat)
	}
	if plan.Audio == nil || plan.Audio.Codec != "libopus" {
		t.Fatalf("audio settings = %+v, want libopus", plan.Audio)
	}
	if plan.VideoMap != "0:v:0" || plan.AudioMap != "0:a:0" {
		t.Errorf("maps = %q / %q, want the first video and audio streams", plan.VideoMap, plan.AudioMap)
	}
	// MP4 fast start is an MP4 container trick and means nothing in WebM.
	if len(plan.MovFlags) != 0 {
		t.Errorf("plan carries movflags %v into a WebM output", plan.MovFlags)
	}
	if len(plan.Video.Filters) != 0 {
		t.Errorf("even dimensions should need no filter, got %v", plan.Video.Filters)
	}
}

func TestPlanWebMOmitsAudioWhenTheSourceHasNone(t *testing.T) {
	t.Parallel()

	plan, err := (Registry{}).Plan(
		filepath.Join("in", "silent.mp4"), filepath.Join("out", "silent.webm"),
		"webm", "stream", mp4Input(640, 480, false), webmCapabilities(),
	)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if plan.Audio != nil {
		t.Errorf("audio settings = %+v, want none for a silent source", plan.Audio)
	}
	if plan.AudioMap != "" {
		t.Errorf("audio map = %q, want empty", plan.AudioMap)
	}
}

func TestPlanWebMPadsOddDimensions(t *testing.T) {
	t.Parallel()

	plan, err := (Registry{}).Plan(
		filepath.Join("in", "odd.mp4"), filepath.Join("out", "odd.webm"),
		"webm", "", mp4Input(641, 481, false), webmCapabilities(),
	)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if len(plan.Video.Filters) != 1 {
		t.Fatalf("filters = %v, want one pad filter", plan.Video.Filters)
	}
}

func TestPlanWebMRequiresItsCodecs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		capabilities media.Capabilities
		withAudio    bool
	}{
		{
			name:         "no vp9 encoder",
			capabilities: media.Capabilities{Encoders: map[string]bool{"libopus": true}, Muxers: map[string]bool{"webm": true}},
		},
		{
			name:         "no webm muxer",
			capabilities: media.Capabilities{Encoders: map[string]bool{"libvpx-vp9": true, "libopus": true}},
		},
		{
			name:         "no opus encoder for an input that has audio",
			capabilities: media.Capabilities{Encoders: map[string]bool{"libvpx-vp9": true}, Muxers: map[string]bool{"webm": true}},
			withAudio:    true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := (Registry{}).Plan(
				filepath.Join("in", "clip.mp4"), filepath.Join("out", "clip.webm"),
				"webm", "", mp4Input(320, 240, test.withAudio), test.capabilities,
			)
			if !errors.Is(err, ErrMissingCapability) {
				t.Errorf("Plan() error = %v, want ErrMissingCapability", err)
			}
		})
	}
}

// A silent source must not demand an Opus encoder it will never use.
func TestPlanWebMWithoutAudioDoesNotNeedOpus(t *testing.T) {
	t.Parallel()

	_, err := (Registry{}).Plan(
		filepath.Join("in", "silent.mp4"), filepath.Join("out", "silent.webm"),
		"webm", "", mp4Input(320, 240, false),
		media.Capabilities{Encoders: map[string]bool{"libvpx-vp9": true}, Muxers: map[string]bool{"webm": true}},
	)
	if err != nil {
		t.Errorf("Plan() error = %v, want a plan without Opus", err)
	}
}

func TestPlanWebMRejectsAnotherPreset(t *testing.T) {
	t.Parallel()

	_, err := (Registry{}).Plan(
		filepath.Join("in", "clip.mp4"), filepath.Join("out", "clip.webm"),
		"webm", "web", mp4Input(320, 240, true), webmCapabilities(),
	)
	if !errors.Is(err, ErrUnsupportedPreset) {
		t.Errorf("Plan() error = %v, want ErrUnsupportedPreset", err)
	}
}

func TestVerifyWebMAcceptsExpectedOutput(t *testing.T) {
	t.Parallel()

	plan, err := (Registry{}).Plan(
		filepath.Join("in", "clip.mp4"), filepath.Join("out", "clip.webm"),
		"webm", "", mp4Input(1280, 720, true), webmCapabilities(),
	)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	output := media.Info{
		Size:        4096,
		FormatNames: []string{"matroska", "webm"},
		Duration:    5 * time.Second,
		Streams: []media.Stream{
			{Index: 0, CodecType: "video", CodecName: "vp9", Width: 1280, Height: 720},
			{Index: 1, CodecType: "audio", CodecName: "opus", Channels: 2},
		},
	}
	if err := Verify(plan, output); err != nil {
		t.Errorf("Verify() error = %v", err)
	}
}

func TestVerifyWebMRejectsWrongOutput(t *testing.T) {
	t.Parallel()

	plan, err := (Registry{}).Plan(
		filepath.Join("in", "clip.mp4"), filepath.Join("out", "clip.webm"),
		"webm", "", mp4Input(1280, 720, true), webmCapabilities(),
	)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	good := []media.Stream{
		{Index: 0, CodecType: "video", CodecName: "vp9", Width: 1280, Height: 720},
		{Index: 1, CodecType: "audio", CodecName: "opus", Channels: 2},
	}

	tests := []struct {
		name   string
		output media.Info
	}{
		{name: "empty file", output: media.Info{Size: 0, FormatNames: []string{"webm"}, Streams: good}},
		{
			name:   "not a webm container",
			output: media.Info{Size: 4096, FormatNames: []string{"mov", "mp4"}, Duration: 5 * time.Second, Streams: good},
		},
		{
			name: "video is not vp9",
			output: media.Info{Size: 4096, FormatNames: []string{"webm"}, Duration: 5 * time.Second, Streams: []media.Stream{
				{Index: 0, CodecType: "video", CodecName: "vp8", Width: 1280, Height: 720},
				{Index: 1, CodecType: "audio", CodecName: "opus"},
			}},
		},
		{
			name: "audio is not opus although the plan asked for it",
			output: media.Info{Size: 4096, FormatNames: []string{"webm"}, Duration: 5 * time.Second, Streams: []media.Stream{
				{Index: 0, CodecType: "video", CodecName: "vp9", Width: 1280, Height: 720},
				{Index: 1, CodecType: "audio", CodecName: "vorbis"},
			}},
		},
		{
			name:   "duration drifted",
			output: media.Info{Size: 4096, FormatNames: []string{"webm"}, Duration: 30 * time.Second, Streams: good},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if err := Verify(plan, test.output); err == nil {
				t.Error("Verify() error = nil, want the output to be rejected")
			}
		})
	}
}
