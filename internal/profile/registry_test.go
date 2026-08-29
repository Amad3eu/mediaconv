package profile

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Amad3eu/mediaconv/internal/media"
)

func TestRegistryFormats(t *testing.T) {
	t.Parallel()

	got := (Registry{}).Formats()
	want := []SupportedFormat{{
		Source:      "webm",
		Target:      "mp4",
		Profile:     "web",
		VideoCodec:  "h264 (libx264)",
		AudioCodec:  "aac",
		Description: "Broadly compatible MP4 for browsers and media players",
	}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Formats() = %#v, want %#v", got, want)
	}
}

func TestRegistryPlanCreatesWebMP4Plan(t *testing.T) {
	t.Parallel()

	info := supportedInputInfo()
	info.Duration = 90*time.Second + 250*time.Millisecond
	info.Streams = append(info.Streams, media.Stream{
		Index:         1,
		CodecType:     "audio",
		CodecName:     "opus",
		Channels:      2,
		ChannelLayout: "stereo",
	})
	caps := supportedCapabilities()
	// The mov muxer is sufficient for MP4 output even if the alias named mp4 is absent.
	delete(caps.Muxers, "mp4")

	got, err := (Registry{}).Plan(
		"/media/Input.WEBM",
		"/media/output.mp4",
		" MP4 ",
		" WEB ",
		info,
		caps,
	)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if got.InputPath != "/media/Input.WEBM" || got.OutputPath != "/media/output.mp4" {
		t.Errorf("plan paths = (%q, %q)", got.InputPath, got.OutputPath)
	}
	if got.SourceFormat != "webm" || got.TargetFormat != "mp4" || got.Profile != "web" {
		t.Errorf("plan formats/profile = %#v", got)
	}
	if got.VideoMap != "0:v:0" || got.AudioMap != "0:a:0" {
		t.Errorf("plan maps = (%q, %q)", got.VideoMap, got.AudioMap)
	}
	wantVideo := media.VideoSettings{
		Codec:       "libx264",
		CRF:         23,
		Preset:      "medium",
		PixelFormat: "yuv420p",
		Filters:     []string{},
	}
	if !reflect.DeepEqual(got.Video, wantVideo) {
		t.Errorf("Video = %#v, want %#v", got.Video, wantVideo)
	}
	wantAudio := &media.AudioSettings{Codec: "aac", BitRate: "192k"}
	if !reflect.DeepEqual(got.Audio, wantAudio) {
		t.Errorf("Audio = %#v, want %#v", got.Audio, wantAudio)
	}
	if !reflect.DeepEqual(got.MovFlags, []string{"faststart"}) {
		t.Errorf("MovFlags = %v", got.MovFlags)
	}
	if !got.CopyMetadata || !got.DropChapters {
		t.Errorf("metadata/chapter settings = copy:%v drop:%v", got.CopyMetadata, got.DropChapters)
	}
	if got.InputDuration != info.Duration {
		t.Errorf("InputDuration = %v, want %v", got.InputDuration, info.Duration)
	}
	if len(got.Warnings) != 0 {
		t.Errorf("Warnings = %v, want none", got.Warnings)
	}
}

func TestRegistryPlanPadsOddDimensionsAndOmitsAudio(t *testing.T) {
	t.Parallel()

	info := supportedInputInfo()
	info.Streams[0].Width = 1279
	info.Streams[0].Height = 719

	got, err := (Registry{}).Plan("clip.webm", "clip.mp4", "mp4", "web", info, supportedCapabilities())
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if got.Audio != nil || got.AudioMap != "" {
		t.Errorf("audio settings = %#v, map = %q; want no audio", got.Audio, got.AudioMap)
	}
	wantFilters := []string{"pad=ceil(iw/2)*2:ceil(ih/2)*2"}
	if !reflect.DeepEqual(got.Video.Filters, wantFilters) {
		t.Errorf("Filters = %v, want %v", got.Video.Filters, wantFilters)
	}
}

func TestRegistryPlanReportsLossAndSelectionWarnings(t *testing.T) {
	t.Parallel()

	info := supportedInputInfo()
	info.Streams[0].PixelFormat = "yuva420p"
	info.Streams[0].ColorTransfer = "smpte2084"
	info.Streams = append(info.Streams,
		media.Stream{Index: 1, CodecType: "video", CodecName: "vp8", Width: 640, Height: 360},
		media.Stream{Index: 2, CodecType: "audio", CodecName: "opus"},
		media.Stream{Index: 3, CodecType: "audio", CodecName: "vorbis"},
		media.Stream{Index: 4, CodecType: "subtitle", CodecName: "webvtt"},
	)
	info.ChapterCount = 2

	got, err := (Registry{}).Plan("clip.bin", "clip.mp4", "mp4", "web", info, supportedCapabilities())
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	wantWarnings := []string{
		"Only the first video stream will be converted.",
		"Only the first audio stream will be converted.",
		"Subtitle streams are not included in the MP4 output.",
		"Chapters are not included in the MP4 output.",
		"The source has an alpha channel; transparency will be lost.",
		"The source appears to use HDR transfer characteristics; the web preset may not preserve HDR correctly.",
		"The input is detected as WebM even though its file extension is not .webm.",
	}
	if !reflect.DeepEqual(got.Warnings, wantWarnings) {
		t.Errorf("Warnings mismatch\n got: %q\nwant: %q", got.Warnings, wantWarnings)
	}
}

func TestRegistryPlanRejectsUnsupportedInputsAndMissingCapabilities(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		target  string
		preset  string
		info    media.Info
		caps    media.Capabilities
		wantErr error
		want    string
	}{
		{
			name:   "target",
			target: "avi", preset: "web",
			info: supportedInputInfo(), caps: supportedCapabilities(),
			wantErr: ErrUnsupportedTarget, want: `"avi"`,
		},
		{
			name:   "preset",
			target: "mp4", preset: "archive",
			info: supportedInputInfo(), caps: supportedCapabilities(),
			wantErr: ErrUnsupportedPreset, want: `"archive"`,
		},
		{
			name:   "container",
			target: "mp4", preset: "web",
			info: media.Info{FormatNames: []string{"matroska"}, Streams: supportedInputInfo().Streams},
			caps: supportedCapabilities(), wantErr: ErrUnsupportedInput, want: "instead of WebM",
		},
		{
			name:   "video stream",
			target: "mp4", preset: "web",
			info: media.Info{FormatNames: []string{"webm"}, Streams: []media.Stream{{CodecType: "audio", CodecName: "opus"}}},
			caps: supportedCapabilities(), wantErr: ErrUnsupportedInput, want: "no video stream",
		},
		{
			name:   "h264 encoder",
			target: "mp4", preset: "web",
			info: supportedInputInfo(), caps: media.Capabilities{Encoders: map[string]bool{"aac": true}, Muxers: map[string]bool{"mp4": true}},
			wantErr: ErrMissingCapability, want: "libx264",
		},
		{
			name:   "mp4 muxer",
			target: "mp4", preset: "web",
			info: supportedInputInfo(), caps: media.Capabilities{Encoders: map[string]bool{"libx264": true, "aac": true}, Muxers: map[string]bool{"webm": true}},
			wantErr: ErrMissingCapability, want: "MP4 muxer",
		},
		{
			name:   "aac encoder when input has audio",
			target: "mp4", preset: "web",
			info: inputInfoWithAudio(), caps: media.Capabilities{Encoders: map[string]bool{"libx264": true}, Muxers: map[string]bool{"mp4": true}},
			wantErr: ErrMissingCapability, want: "AAC encoder",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := (Registry{}).Plan("clip.webm", "clip.mp4", test.target, test.preset, test.info, test.caps)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Plan() error = %v, want errors.Is(..., %v)", err, test.wantErr)
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Errorf("Plan() error = %q, want substring %q", err, test.want)
			}
		})
	}
}

func TestVerifyAcceptsExpectedOutput(t *testing.T) {
	t.Parallel()

	plan := media.Plan{Audio: &media.AudioSettings{Codec: "aac", BitRate: "192k"}}
	info := validOutputInfo()
	if err := Verify(plan, info); err != nil {
		t.Errorf("Verify() error = %v", err)
	}

	// An audio stream is optional when the conversion plan did not request one.
	info.Streams = info.Streams[:1]
	if err := Verify(media.Plan{}, info); err != nil {
		t.Errorf("Verify(video-only) error = %v", err)
	}
}

func TestVerifyRejectsInvalidOutput(t *testing.T) {
	t.Parallel()

	audioPlan := media.Plan{Audio: &media.AudioSettings{Codec: "aac"}}
	tests := []struct {
		name   string
		plan   media.Plan
		mutate func(*media.Info)
		want   string
	}{
		{name: "empty", plan: audioPlan, mutate: func(info *media.Info) { info.Size = 0 }, want: "empty"},
		{name: "container", plan: audioPlan, mutate: func(info *media.Info) { info.FormatNames = []string{"matroska"} }, want: "MP4 container"},
		{name: "no video", plan: audioPlan, mutate: func(info *media.Info) { info.Streams = info.Streams[1:] }, want: "H.264 video"},
		{name: "video codec", plan: audioPlan, mutate: func(info *media.Info) { info.Streams[0].CodecName = "hevc" }, want: "H.264 video"},
		{name: "odd width", plan: audioPlan, mutate: func(info *media.Info) { info.Streams[0].Width = 1279 }, want: "dimensions are not even"},
		{name: "odd height", plan: audioPlan, mutate: func(info *media.Info) { info.Streams[0].Height = 719 }, want: "dimensions are not even"},
		{name: "no audio", plan: audioPlan, mutate: func(info *media.Info) { info.Streams = info.Streams[:1] }, want: "AAC audio"},
		{name: "audio codec", plan: audioPlan, mutate: func(info *media.Info) { info.Streams[1].CodecName = "mp3" }, want: "AAC audio"},
		{
			name: "missing output duration",
			plan: media.Plan{InputDuration: 10 * time.Second},
			mutate: func(info *media.Info) {
				info.Duration = 0
			},
			want: "duration could not be verified",
		},
		{
			name: "implausibly short output",
			plan: media.Plan{InputDuration: 10 * time.Second},
			mutate: func(info *media.Info) {
				info.Duration = 5 * time.Second
			},
			want: "differs unexpectedly",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			info := validOutputInfo()
			test.mutate(&info)
			err := Verify(test.plan, info)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Errorf("Verify() error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func supportedInputInfo() media.Info {
	return media.Info{
		FormatNames: []string{"matroska", "WebM"},
		Streams: []media.Stream{{
			Index:       0,
			CodecType:   "video",
			CodecName:   "vp9",
			PixelFormat: "yuv420p",
			Width:       1280,
			Height:      720,
		}},
	}
}

func inputInfoWithAudio() media.Info {
	info := supportedInputInfo()
	info.Streams = append(info.Streams, media.Stream{Index: 1, CodecType: "audio", CodecName: "opus"})
	return info
}

func supportedCapabilities() media.Capabilities {
	return media.Capabilities{
		Encoders: map[string]bool{"libx264": true, "aac": true},
		Muxers:   map[string]bool{"mp4": true, "mov": true},
	}
}

func validOutputInfo() media.Info {
	return media.Info{
		FormatNames: []string{"mov", "mp4"},
		Size:        1024,
		Streams: []media.Stream{
			{Index: 0, CodecType: "video", CodecName: "h264", Width: 1280, Height: 720},
			{Index: 1, CodecType: "audio", CodecName: "aac", Channels: 2},
		},
	}
}
