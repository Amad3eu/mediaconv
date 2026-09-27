package profile

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/Amad3eu/mediaconv/internal/media"
)

func audioCapabilities() media.Capabilities {
	return media.Capabilities{
		Encoders: map[string]bool{"aac": true, "pcm_s16le": true, "libmp3lame": true},
		Muxers:   map[string]bool{"ipod": true, "mp4": true, "wav": true, "mp3": true},
	}
}

func wavInput(streams int) media.Info {
	info := media.Info{
		FormatNames: []string{"wav"},
		Duration:    4 * time.Second,
		Streams: []media.Stream{
			{Index: 0, CodecType: "audio", CodecName: "pcm_s16le", Channels: 2},
		},
	}
	for i := 1; i < streams; i++ {
		info.Streams = append(info.Streams, media.Stream{Index: i, CodecType: "audio", CodecName: "pcm_s16le"})
	}
	return info
}

// FFmpeg has no muxer called m4a; that container is written with ipod. The
// plan has to say so, or the conversion fails with "output format is not
// known" at the very last step.
func TestPlanM4ANamesTheIpodMuxer(t *testing.T) {
	t.Parallel()

	plan, err := (Registry{}).Plan(
		filepath.Join("in", "song.wav"), filepath.Join("out", "song.m4a"),
		"m4a", "", wavInput(1), audioCapabilities(),
	)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if plan.TargetFormat != "m4a" {
		t.Errorf("TargetFormat = %q, want m4a", plan.TargetFormat)
	}
	if plan.Muxer != "ipod" {
		t.Errorf("Muxer = %q, want ipod, which is what FFmpeg calls this container", plan.Muxer)
	}
	if plan.Profile != "aac" {
		t.Errorf("Profile = %q, want aac", plan.Profile)
	}
	if plan.Audio == nil || plan.Audio.Codec != "aac" || plan.Audio.BitRate != "192k" {
		t.Fatalf("audio settings = %+v, want aac at 192k", plan.Audio)
	}
	if plan.Video != nil {
		t.Errorf("audio target carries video settings: %+v", plan.Video)
	}
}

// PCM has no bitrate to choose: the sample format fixes it. Leaving it empty
// is what makes the adapter omit -b:a instead of passing a value PCM ignores.
func TestPlanWAVLeavesTheBitRateEmpty(t *testing.T) {
	t.Parallel()

	plan, err := (Registry{}).Plan(
		filepath.Join("in", "song.mp3"), filepath.Join("out", "song.wav"),
		"wav", "", media.Info{
			FormatNames: []string{"mp3"},
			Duration:    4 * time.Second,
			Streams:     []media.Stream{{Index: 0, CodecType: "audio", CodecName: "mp3"}},
		}, audioCapabilities(),
	)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if plan.Audio == nil || plan.Audio.Codec != "pcm_s16le" {
		t.Fatalf("audio settings = %+v, want pcm_s16le", plan.Audio)
	}
	if plan.Audio.BitRate != "" {
		t.Errorf("BitRate = %q, want empty: PCM has no bitrate to set", plan.Audio.BitRate)
	}
	// The WAV muxer is spelled the same as the target, so nothing to override.
	if plan.Muxer != "" {
		t.Errorf("Muxer = %q, want empty so the target name is used", plan.Muxer)
	}
	if plan.Profile != "master" {
		t.Errorf("Profile = %q, want master", plan.Profile)
	}
}

func TestAudioTargetsRequireTheirCapabilities(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		target       string
		capabilities media.Capabilities
	}{
		{
			name:         "m4a without the AAC encoder",
			target:       "m4a",
			capabilities: media.Capabilities{Muxers: map[string]bool{"ipod": true}},
		},
		{
			name:         "m4a without a container muxer",
			target:       "m4a",
			capabilities: media.Capabilities{Encoders: map[string]bool{"aac": true}},
		},
		{
			name:         "wav without the PCM encoder",
			target:       "wav",
			capabilities: media.Capabilities{Muxers: map[string]bool{"wav": true}},
		},
		{
			name:         "wav without the WAV muxer",
			target:       "wav",
			capabilities: media.Capabilities{Encoders: map[string]bool{"pcm_s16le": true}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := (Registry{}).Plan(
				filepath.Join("in", "song.wav"), filepath.Join("out", "song."+test.target),
				test.target, "", wavInput(1), test.capabilities,
			)
			if !errors.Is(err, ErrMissingCapability) {
				t.Errorf("Plan() error = %v, want ErrMissingCapability", err)
			}
		})
	}
}

// The mp4 muxer can write this container too, so an FFmpeg without the ipod
// muxer but with mp4 must not be refused.
func TestPlanM4AAcceptsTheMP4MuxerAsAFallback(t *testing.T) {
	t.Parallel()

	_, err := (Registry{}).Plan(
		filepath.Join("in", "song.wav"), filepath.Join("out", "song.m4a"),
		"m4a", "",
		wavInput(1),
		media.Capabilities{
			Encoders: map[string]bool{"aac": true},
			Muxers:   map[string]bool{"mp4": true},
		},
	)
	if err != nil {
		t.Errorf("Plan() error = %v, want the mp4 muxer to be accepted", err)
	}
}

func TestAudioTargetsRejectAnotherPreset(t *testing.T) {
	t.Parallel()

	for target, wrong := range map[string]string{"m4a": "master", "wav": "aac"} {
		_, err := (Registry{}).Plan(
			filepath.Join("in", "song.wav"), filepath.Join("out", "song."+target),
			target, wrong, wavInput(1), audioCapabilities(),
		)
		if !errors.Is(err, ErrUnsupportedPreset) {
			t.Errorf("Plan(%s, preset=%s) error = %v, want ErrUnsupportedPreset", target, wrong, err)
		}
	}
}

func TestAudioTargetsWarnAboutWhatIsDropped(t *testing.T) {
	t.Parallel()

	info := wavInput(2)
	info.ChapterCount = 3
	info.Streams = append(info.Streams,
		media.Stream{Index: 8, CodecType: "video", CodecName: "mjpeg"},
		media.Stream{Index: 9, CodecType: "subtitle", CodecName: "subrip"},
	)

	for _, target := range []string{"m4a", "wav"} {
		plan, err := (Registry{}).Plan(
			filepath.Join("in", "song.wav"), filepath.Join("out", "song."+target),
			target, "", info, audioCapabilities(),
		)
		if err != nil {
			t.Fatalf("Plan(%s) error = %v", target, err)
		}
		if len(plan.Warnings) < 4 {
			t.Errorf("Plan(%s) warnings = %v, want video, extra audio, subtitles and chapters all mentioned", target, plan.Warnings)
		}
	}
}

func TestVerifyAudioTargets(t *testing.T) {
	t.Parallel()

	m4a, err := (Registry{}).Plan(
		filepath.Join("in", "song.wav"), filepath.Join("out", "song.m4a"),
		"m4a", "", wavInput(1), audioCapabilities(),
	)
	if err != nil {
		t.Fatalf("Plan(m4a) error = %v", err)
	}

	good := media.Info{
		Size:        8192,
		FormatNames: []string{"mov", "mp4", "m4a"},
		Duration:    4 * time.Second,
		Streams:     []media.Stream{{Index: 0, CodecType: "audio", CodecName: "aac"}},
	}
	if err := Verify(m4a, good); err != nil {
		t.Errorf("Verify(m4a) error = %v", err)
	}

	tests := []struct {
		name   string
		output media.Info
	}{
		{name: "empty file", output: media.Info{Size: 0, FormatNames: []string{"mp4"}, Streams: good.Streams}},
		{
			name:   "wrong container",
			output: media.Info{Size: 8192, FormatNames: []string{"wav"}, Duration: 4 * time.Second, Streams: good.Streams},
		},
		{
			name: "wrong codec",
			output: media.Info{Size: 8192, FormatNames: []string{"mp4"}, Duration: 4 * time.Second,
				Streams: []media.Stream{{Index: 0, CodecType: "audio", CodecName: "alac"}}},
		},
		{
			name:   "conversion stopped early",
			output: media.Info{Size: 8192, FormatNames: []string{"mp4"}, Duration: 1 * time.Second, Streams: good.Streams},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if err := Verify(m4a, test.output); err == nil {
				t.Error("Verify() error = nil, want the output to be rejected")
			}
		})
	}
}
