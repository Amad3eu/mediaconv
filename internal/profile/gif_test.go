package profile

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Amad3eu/mediaconv/internal/media"
)

func gifCapabilities() media.Capabilities {
	return media.Capabilities{
		Encoders: map[string]bool{"gif": true},
		Muxers:   map[string]bool{"gif": true},
	}
}

func videoInput(duration time.Duration, withAudio bool) media.Info {
	info := media.Info{
		FormatNames: []string{"mov", "mp4"},
		Duration:    duration,
		Streams: []media.Stream{
			{Index: 0, CodecType: "video", CodecName: "h264", Width: 1280, Height: 720, PixelFormat: "yuv420p"},
		},
	}
	if withAudio {
		info.Streams = append(info.Streams, media.Stream{Index: 1, CodecType: "audio", CodecName: "aac"})
	}
	return info
}

// A GIF holds 256 colors, so the palette has to come from the clip itself.
// palettegen and paletteuse need named branches, which is why the filter is one
// graph rather than a chain of filters joined with commas.
func TestPlanGIFBuildsThePaletteGraph(t *testing.T) {
	t.Parallel()

	plan, err := (Registry{}).Plan(
		Request{InputPath: filepath.Join("in", "clip.mp4"), OutputPath: filepath.Join("out", "clip.gif"), Target: "gif"},
		videoInput(30*time.Second, true), gifCapabilities(),
	)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if plan.TargetFormat != "gif" || plan.Profile != "preview" {
		t.Errorf("plan targets %q with profile %q, want gif/preview", plan.TargetFormat, plan.Profile)
	}
	if plan.Video == nil || plan.Video.Codec != "gif" {
		t.Fatalf("video settings = %+v, want the gif encoder", plan.Video)
	}
	if len(plan.Video.Filters) != 1 {
		t.Fatalf("filters = %v, want a single graph", plan.Video.Filters)
	}

	graph := plan.Video.Filters[0]
	for _, part := range []string{"fps=10", "scale=480:-1", "lanczos", "split", "palettegen", "paletteuse"} {
		if !strings.Contains(graph, part) {
			t.Errorf("filter graph is missing %q\ngot: %s", part, graph)
		}
	}

	// The palette lives in the file; forcing a pixel format fights paletteuse.
	if plan.Video.PixelFormat != "" {
		t.Errorf("PixelFormat = %q, want empty for GIF", plan.Video.PixelFormat)
	}
	if plan.Audio != nil {
		t.Errorf("audio settings = %+v, want none: GIF has no audio", plan.Audio)
	}
}

func TestPlanGIFTrimsToThePreviewLength(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input time.Duration
		want  time.Duration
	}{
		{name: "long input is cut to five seconds", input: 10 * time.Minute, want: 5 * time.Second},
		{name: "short input keeps its own length", input: 2 * time.Second, want: 2 * time.Second},
		{name: "unknown duration falls back to five seconds", input: 0, want: 5 * time.Second},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			plan, err := (Registry{}).Plan(
				Request{InputPath: filepath.Join("in", "clip.mp4"), OutputPath: filepath.Join("out", "clip.gif"), Target: "gif"},
				videoInput(test.input, false), gifCapabilities(),
			)
			if err != nil {
				t.Fatalf("Plan() error = %v", err)
			}
			if plan.TrimDuration != test.want {
				t.Errorf("TrimDuration = %s, want %s", plan.TrimDuration, test.want)
			}
		})
	}
}

// The preview is deliberately shorter than its source, so verification has to
// compare it against the trim. Comparing against the input would reject every
// preview of anything longer than five seconds.
func TestVerifyGIFComparesAgainstTheTrimNotTheSource(t *testing.T) {
	t.Parallel()

	plan, err := (Registry{}).Plan(
		Request{InputPath: filepath.Join("in", "clip.mp4"), OutputPath: filepath.Join("out", "clip.gif"), Target: "gif"},
		videoInput(10*time.Minute, false), gifCapabilities(),
	)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	preview := media.Info{
		Size:        4096,
		FormatNames: []string{"gif"},
		Duration:    5 * time.Second,
		Streams:     []media.Stream{{Index: 0, CodecType: "video", CodecName: "gif", Width: 480, Height: 270}},
	}
	if err := Verify(plan, preview); err != nil {
		t.Errorf("Verify() rejected a correct five second preview of a ten minute file: %v", err)
	}

	// A preview that came out the length of the whole source means the trim
	// never took effect.
	untrimmed := preview
	untrimmed.Duration = 10 * time.Minute
	if err := Verify(plan, untrimmed); err == nil {
		t.Error("Verify() accepted a preview as long as the source")
	}
}

func TestVerifyGIFRejectsWrongOutput(t *testing.T) {
	t.Parallel()

	plan, err := (Registry{}).Plan(
		Request{InputPath: filepath.Join("in", "clip.mp4"), OutputPath: filepath.Join("out", "clip.gif"), Target: "gif"},
		videoInput(30*time.Second, false), gifCapabilities(),
	)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	good := []media.Stream{{Index: 0, CodecType: "video", CodecName: "gif", Width: 480, Height: 270}}

	tests := []struct {
		name   string
		output media.Info
	}{
		{name: "empty file", output: media.Info{Size: 0, FormatNames: []string{"gif"}, Streams: good}},
		{
			name:   "not a gif",
			output: media.Info{Size: 4096, FormatNames: []string{"mov", "mp4"}, Duration: 5 * time.Second, Streams: good},
		},
		{
			name: "wrong codec",
			output: media.Info{Size: 4096, FormatNames: []string{"gif"}, Duration: 5 * time.Second,
				Streams: []media.Stream{{Index: 0, CodecType: "video", CodecName: "png", Width: 480, Height: 270}}},
		},
		{
			name: "no dimensions",
			output: media.Info{Size: 4096, FormatNames: []string{"gif"}, Duration: 5 * time.Second,
				Streams: []media.Stream{{Index: 0, CodecType: "video", CodecName: "gif"}}},
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

func TestPlanGIFRequiresItsCapabilities(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		capabilities media.Capabilities
	}{
		{name: "no encoder", capabilities: media.Capabilities{Muxers: map[string]bool{"gif": true}}},
		{name: "no muxer", capabilities: media.Capabilities{Encoders: map[string]bool{"gif": true}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := (Registry{}).Plan(
				Request{InputPath: filepath.Join("in", "clip.mp4"), OutputPath: filepath.Join("out", "clip.gif"), Target: "gif"},
				videoInput(10*time.Second, false), test.capabilities,
			)
			if !errors.Is(err, ErrMissingCapability) {
				t.Errorf("Plan() error = %v, want ErrMissingCapability", err)
			}
		})
	}
}

func TestPlanGIFRejectsAnotherPresetAndAudioOnlyInput(t *testing.T) {
	t.Parallel()

	_, err := (Registry{}).Plan(
		Request{InputPath: filepath.Join("in", "clip.mp4"), OutputPath: filepath.Join("out", "clip.gif"), Target: "gif", Preset: "web"},
		videoInput(10*time.Second, false), gifCapabilities(),
	)
	if !errors.Is(err, ErrUnsupportedPreset) {
		t.Errorf("Plan(preset=web) error = %v, want ErrUnsupportedPreset", err)
	}

	audioOnly := media.Info{
		FormatNames: []string{"mov", "mp4"},
		Duration:    10 * time.Second,
		Streams:     []media.Stream{{Index: 0, CodecType: "audio", CodecName: "aac"}},
	}
	_, err = (Registry{}).Plan(
		Request{InputPath: filepath.Join("in", "song.mp4"), OutputPath: filepath.Join("out", "song.gif"), Target: "gif"},
		audioOnly, gifCapabilities(),
	)
	if !errors.Is(err, ErrUnsupportedInput) {
		t.Errorf("Plan(audio only) error = %v, want ErrUnsupportedInput", err)
	}
}

func TestPlanGIFWarnsAboutWhatThePreviewLeavesOut(t *testing.T) {
	t.Parallel()

	plan, err := (Registry{}).Plan(
		Request{InputPath: filepath.Join("in", "clip.mp4"), OutputPath: filepath.Join("out", "clip.gif"), Target: "gif"},
		videoInput(time.Minute, true), gifCapabilities(),
	)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	joined := strings.Join(plan.Warnings, " | ")
	for _, want := range []string{"480 pixels wide", "no audio", "part of the input"} {
		if !strings.Contains(joined, want) {
			t.Errorf("warnings do not mention %q\ngot: %s", want, joined)
		}
	}
}

// --start and --duration replace the preview's own defaults. The offset also
// caps the length: a preview cannot run past the end of its source.
func TestPlanGIFHonoursAnExplicitRange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        time.Duration
		start        time.Duration
		duration     time.Duration
		wantStart    time.Duration
		wantDuration time.Duration
	}{
		{
			name:  "duration replaces the five second default",
			input: time.Minute, duration: 12 * time.Second,
			wantDuration: 12 * time.Second,
		},
		{
			name:  "start offsets without changing the default length",
			input: time.Minute, start: 30 * time.Second,
			wantStart: 30 * time.Second, wantDuration: 5 * time.Second,
		},
		{
			name:  "both together",
			input: time.Minute, start: 10 * time.Second, duration: 3 * time.Second,
			wantStart: 10 * time.Second, wantDuration: 3 * time.Second,
		},
		{
			name:  "length is capped by what is left after the offset",
			input: 12 * time.Second, start: 10 * time.Second, duration: time.Minute,
			wantStart: 10 * time.Second, wantDuration: 2 * time.Second,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			plan, err := (Registry{}).Plan(
				Request{
					InputPath:    filepath.Join("in", "clip.mp4"),
					OutputPath:   filepath.Join("out", "clip.gif"),
					Target:       "gif",
					TrimStart:    test.start,
					TrimDuration: test.duration,
				},
				videoInput(test.input, false), gifCapabilities(),
			)
			if err != nil {
				t.Fatalf("Plan() error = %v", err)
			}
			if plan.TrimStart != test.wantStart {
				t.Errorf("TrimStart = %s, want %s", plan.TrimStart, test.wantStart)
			}
			if plan.TrimDuration != test.wantDuration {
				t.Errorf("TrimDuration = %s, want %s", plan.TrimDuration, test.wantDuration)
			}
		})
	}
}

// Trimming is not a preview feature: cutting a range out of a long recording
// is as useful for the other targets.
func TestEveryTargetAcceptsARange(t *testing.T) {
	t.Parallel()

	video := videoInput(time.Minute, true)
	audio := media.Info{
		FormatNames: []string{"wav"},
		Duration:    time.Minute,
		Streams:     []media.Stream{{Index: 0, CodecType: "audio", CodecName: "pcm_s16le"}},
	}
	all := media.Capabilities{
		Encoders: map[string]bool{"libx264": true, "aac": true, "libmp3lame": true, "libvpx-vp9": true, "libopus": true, "pcm_s16le": true, "gif": true},
		Muxers:   map[string]bool{"mp4": true, "mp3": true, "webm": true, "ipod": true, "wav": true, "gif": true},
	}

	for _, target := range []string{"mp4", "webm", "mp3", "m4a", "wav"} {
		info, source := video, "clip.mp4"
		if target == "mp3" || target == "m4a" || target == "wav" {
			info, source = audio, "song.wav"
		}

		plan, err := (Registry{}).Plan(
			Request{
				InputPath:    filepath.Join("in", source),
				OutputPath:   filepath.Join("out", "out."+target),
				Target:       target,
				TrimStart:    20 * time.Second,
				TrimDuration: 5 * time.Second,
			},
			info, all,
		)
		if err != nil {
			t.Errorf("Plan(%s) error = %v", target, err)
			continue
		}
		if plan.TrimStart != 20*time.Second || plan.TrimDuration != 5*time.Second {
			t.Errorf("%s dropped the range: start=%s duration=%s", target, plan.TrimStart, plan.TrimDuration)
		}
	}
}

func TestPlanRejectsAnImpossibleRange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		start    time.Duration
		duration time.Duration
	}{
		{name: "negative start", start: -time.Second},
		{name: "negative duration", duration: -time.Second},
		{name: "start at the end", start: 30 * time.Second},
		{name: "start past the end", start: time.Hour},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := (Registry{}).Plan(
				Request{
					InputPath:    filepath.Join("in", "clip.mp4"),
					OutputPath:   filepath.Join("out", "clip.gif"),
					Target:       "gif",
					TrimStart:    test.start,
					TrimDuration: test.duration,
				},
				videoInput(30*time.Second, false), gifCapabilities(),
			)
			if !errors.Is(err, ErrInvalidRange) {
				t.Errorf("Plan() error = %v, want ErrInvalidRange", err)
			}
		})
	}
}
