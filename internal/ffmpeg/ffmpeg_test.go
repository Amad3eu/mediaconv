package ffmpeg

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/Amad3eu/mediaconv/internal/media"
)

func TestParseProbeJSON(t *testing.T) {
	t.Parallel()

	data := []byte(`{
  "streams": [
    {
      "index": 0,
      "codec_name": "vp9",
      "codec_long_name": "Google VP9",
      "profile": "Profile 0",
      "codec_type": "video",
      "pix_fmt": "yuv420p",
      "width": 1920,
      "height": 1080,
      "color_transfer": "bt709",
      "duration": "12.345000",
      "tags": {"language": "eng"}
    },
    {
      "index": 1,
      "codec_name": "opus",
      "codec_long_name": "Opus",
      "codec_type": "audio",
      "channels": 2,
      "channel_layout": "stereo",
      "sample_rate": "48000",
      "duration": "12.300000",
      "tags": {"title": "Main audio"}
    }
  ],
  "chapters": [{"id": 0}, {"id": 1}],
  "format": {
    "format_name": "matroska, webm",
    "format_long_name": "Matroska / WebM",
    "duration": "12.345000",
    "size": "1234567",
    "bit_rate": "800000",
    "tags": {"encoder": "test-suite"}
  }
}`)

	got, err := parseProbeJSON(data)
	if err != nil {
		t.Fatalf("parseProbeJSON() error = %v", err)
	}

	if want := []string{"matroska", "webm"}; !reflect.DeepEqual(got.FormatNames, want) {
		t.Errorf("FormatNames = %v, want %v", got.FormatNames, want)
	}
	if got.FormatLongName != "Matroska / WebM" {
		t.Errorf("FormatLongName = %q", got.FormatLongName)
	}
	if got.Duration != 12*time.Second+345*time.Millisecond {
		t.Errorf("Duration = %v, want 12.345s", got.Duration)
	}
	if got.Size != 1_234_567 {
		t.Errorf("Size = %d, want 1234567", got.Size)
	}
	if got.BitRate != 800_000 {
		t.Errorf("BitRate = %d, want 800000", got.BitRate)
	}
	if got.ChapterCount != 2 {
		t.Errorf("ChapterCount = %d, want 2", got.ChapterCount)
	}
	if got.Tags["encoder"] != "test-suite" {
		t.Errorf("Tags = %v", got.Tags)
	}
	if len(got.Streams) != 2 {
		t.Fatalf("len(Streams) = %d, want 2", len(got.Streams))
	}

	video := got.Streams[0]
	if video.Index != 0 || video.CodecType != "video" || video.CodecName != "vp9" {
		t.Errorf("video identity = %#v", video)
	}
	if video.CodecLongName != "Google VP9" || video.Profile != "Profile 0" {
		t.Errorf("video codec metadata = %#v", video)
	}
	if video.PixelFormat != "yuv420p" || video.Width != 1920 || video.Height != 1080 {
		t.Errorf("video geometry = %#v", video)
	}
	if video.ColorTransfer != "bt709" || video.Duration != 12*time.Second+345*time.Millisecond {
		t.Errorf("video timing/color = %#v", video)
	}
	if video.Tags["language"] != "eng" {
		t.Errorf("video tags = %v", video.Tags)
	}

	audio := got.Streams[1]
	if audio.Index != 1 || audio.CodecType != "audio" || audio.CodecName != "opus" {
		t.Errorf("audio identity = %#v", audio)
	}
	if audio.Channels != 2 || audio.ChannelLayout != "stereo" || audio.SampleRate != 48_000 {
		t.Errorf("audio layout = %#v", audio)
	}
	if audio.Duration != 12*time.Second+300*time.Millisecond {
		t.Errorf("audio duration = %v", audio.Duration)
	}
	if audio.Tags["title"] != "Main audio" {
		t.Errorf("audio tags = %v", audio.Tags)
	}
}

func TestParseProbeJSONHandlesMissingAndInvalidNumericFields(t *testing.T) {
	t.Parallel()

	got, err := parseProbeJSON([]byte(`{
  "format": {"format_name": " webm, ,matroska ", "duration": "N/A", "size": "invalid", "bit_rate": ""},
  "streams": [{"index": 0, "codec_type": "audio", "sample_rate": "unknown", "duration": "-1"}]
}`))
	if err != nil {
		t.Fatalf("parseProbeJSON() error = %v", err)
	}

	if want := []string{"webm", "matroska"}; !reflect.DeepEqual(got.FormatNames, want) {
		t.Errorf("FormatNames = %v, want %v", got.FormatNames, want)
	}
	if got.Duration != 0 || got.Size != 0 || got.BitRate != 0 {
		t.Errorf("invalid format numbers were not normalized to zero: %#v", got)
	}
	if len(got.Streams) != 1 || got.Streams[0].SampleRate != 0 || got.Streams[0].Duration != 0 {
		t.Errorf("invalid stream numbers were not normalized to zero: %#v", got.Streams)
	}
}

func TestParseProbeJSONRejectsMalformedJSON(t *testing.T) {
	t.Parallel()

	if _, err := parseProbeJSON([]byte(`{"format":`)); err == nil {
		t.Fatal("parseProbeJSON() error = nil, want malformed JSON error")
	}
}

func TestLocatorFindsFFprobeBesideCustomFFmpeg(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	ffmpegPath := filepath.Join(directory, executableName("ffmpeg"))
	ffprobePath := filepath.Join(directory, executableName("ffprobe"))
	for _, path := range []string{ffmpegPath, ffprobePath} {
		if err := os.WriteFile(path, []byte("test executable"), 0o700); err != nil {
			t.Fatal(err)
		}
	}

	paths, err := (Locator{}).Locate(ffmpegPath, "")
	if err != nil {
		t.Fatalf("Locate() error = %v", err)
	}
	if paths.FFmpeg != ffmpegPath || paths.FFprobe != ffprobePath {
		t.Fatalf("Locate() = %#v, want ffmpeg=%q ffprobe=%q", paths, ffmpegPath, ffprobePath)
	}
}

func TestProgressParserFeed(t *testing.T) {
	t.Parallel()

	parser := newProgressParser(10 * time.Second)
	for _, line := range []string{
		"frame=125",
		"out_time=00:00:02.500000",
		"out_time_us=9999999",
		"speed=1.25x",
	} {
		if _, ok := parser.Feed(line); ok {
			t.Fatalf("Feed(%q) emitted progress before progress marker", line)
		}
	}

	got, ok := parser.Feed("progress=continue")
	if !ok {
		t.Fatal("Feed(progress=continue) did not emit progress")
	}
	want := media.Progress{
		Frame:     125,
		Processed: 2500 * time.Millisecond,
		Total:     10 * time.Second,
		Speed:     "1.25x",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("progress = %#v, want %#v", got, want)
	}

	// A progress marker starts a fresh record; stale values must not leak.
	if _, ok := parser.Feed("not-a-key-value-line"); ok {
		t.Fatal("malformed line emitted progress")
	}
	if _, ok := parser.Feed("out_time=N/A"); ok {
		t.Fatal("out_time emitted progress")
	}
	if _, ok := parser.Feed("out_time_us=3750000"); ok {
		t.Fatal("out_time_us emitted progress")
	}
	got, ok = parser.Feed("progress=end")
	if !ok {
		t.Fatal("Feed(progress=end) did not emit progress")
	}
	if got.Frame != 0 || got.Processed != 3750*time.Millisecond || got.Speed != "" || !got.Done {
		t.Errorf("final progress = %#v", got)
	}
}

func TestProgressDurationFallsBackToOutTimeMS(t *testing.T) {
	t.Parallel()

	values := map[string]string{
		"out_time":    "invalid",
		"out_time_us": "0",
		"out_time_ms": "1500000",
	}
	if got := progressDuration(values); got != 1500*time.Millisecond {
		t.Errorf("progressDuration() = %v, want 1.5s", got)
	}
}

func TestParseClock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		want  time.Duration
		ok    bool
	}{
		{name: "fractional", value: "01:02:03.250000", want: time.Hour + 2*time.Minute + 3250*time.Millisecond, ok: true},
		{name: "zero", value: "00:00:00.000000", want: 0, ok: true},
		{name: "too few fields", value: "02:03", ok: false},
		{name: "invalid hour", value: "xx:02:03", ok: false},
		{name: "invalid minute", value: "01:xx:03", ok: false},
		{name: "invalid second", value: "01:02:xx", ok: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, ok := parseClock(test.value)
			if ok != test.ok || got != test.want {
				t.Errorf("parseClock(%q) = (%v, %v), want (%v, %v)", test.value, got, ok, test.want, test.ok)
			}
		})
	}
}

func TestBuildArgs(t *testing.T) {
	t.Parallel()

	plan := media.Plan{
		InputPath:    "/media/source clip.webm",
		TargetFormat: "mp4",
		VideoMap:     "0:v:0",
		AudioMap:     "0:a:0",
		Video: media.VideoSettings{
			Codec:       "libx264",
			CRF:         23,
			Preset:      "medium",
			PixelFormat: "yuv420p",
			Filters:     []string{"pad=ceil(iw/2)*2:ceil(ih/2)*2", "setsar=1"},
		},
		Audio:        &media.AudioSettings{Codec: "aac", BitRate: "192k"},
		CopyMetadata: true,
		DropChapters: true,
		MovFlags:     []string{"faststart", "use_metadata_tags"},
	}

	want := []string{
		"-hide_banner",
		"-nostdin",
		"-loglevel", "error",
		"-nostats",
		"-stats_period", "0.5",
		"-progress", "pipe:1",
		"-n",
		"-protocol_whitelist", "file",
		"-i", "/media/source clip.webm",
		"-map", "0:v:0",
		"-map", "0:a:0",
		"-sn",
		"-dn",
		"-c:v", "libx264",
		"-crf", "23",
		"-preset", "medium",
		"-pix_fmt", "yuv420p",
		"-vf", "pad=ceil(iw/2)*2:ceil(ih/2)*2,setsar=1",
		"-c:a", "aac",
		"-b:a", "192k",
		"-map_metadata", "0",
		"-map_chapters", "-1",
		"-movflags", "+faststart+use_metadata_tags",
		"-f", "mp4",
		"/tmp/staged output.mp4",
	}

	if got := BuildArgs(plan, "/tmp/staged output.mp4"); !reflect.DeepEqual(got, want) {
		t.Errorf("BuildArgs() mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestBuildArgsWithoutOptionalAudioOrContainerOptions(t *testing.T) {
	t.Parallel()

	plan := media.Plan{
		InputPath:    "input.webm",
		TargetFormat: "mp4",
		VideoMap:     "0:v:0",
		Video: media.VideoSettings{
			Codec:       "libx264",
			CRF:         20,
			Preset:      "slow",
			PixelFormat: "yuv420p",
		},
	}
	args := BuildArgs(plan, "stage.mp4")

	for _, forbidden := range []string{"-c:a", "-b:a", "-vf", "-map_metadata", "-map_chapters", "-movflags"} {
		if containsArgument(args, forbidden) {
			t.Errorf("BuildArgs() contains optional argument %q: %v", forbidden, args)
		}
	}
	if countArgument(args, "-map") != 1 {
		t.Errorf("BuildArgs() map count = %d, want 1: %v", countArgument(args, "-map"), args)
	}
}

func TestTailBufferRetainsOnlyTail(t *testing.T) {
	t.Parallel()

	buffer := newTailBuffer(5)
	writes := []struct {
		input string
		want  string
	}{
		{input: "abc", want: "abc"},
		{input: "defg", want: "cdefg"},
		{input: "1234567", want: "34567"},
		{input: "XY", want: "567XY"},
	}
	for _, write := range writes {
		written, err := buffer.Write([]byte(write.input))
		if err != nil {
			t.Fatalf("Write(%q) error = %v", write.input, err)
		}
		if written != len(write.input) {
			t.Errorf("Write(%q) wrote %d bytes, want %d", write.input, written, len(write.input))
		}
		if got := buffer.String(); got != write.want {
			t.Errorf("after Write(%q), String() = %q, want %q", write.input, got, write.want)
		}
	}
}

func TestLimitedBufferCapsMemoryAndReportsOverflow(t *testing.T) {
	t.Parallel()

	buffer := newLimitedBuffer(5)
	for _, value := range []string{"abc", "def", "ignored"} {
		written, err := buffer.Write([]byte(value))
		if err != nil {
			t.Fatalf("Write(%q) error = %v", value, err)
		}
		if written != len(value) {
			t.Fatalf("Write(%q) = %d, want %d", value, written, len(value))
		}
	}
	if got := string(buffer.Bytes()); got != "abcde" {
		t.Fatalf("Bytes() = %q, want %q", got, "abcde")
	}
	if !buffer.Exceeded() {
		t.Fatal("Exceeded() = false, want true")
	}
}

func containsArgument(args []string, target string) bool {
	return countArgument(args, target) > 0
}

func countArgument(args []string, target string) int {
	count := 0
	for _, arg := range args {
		if arg == target {
			count++
		}
	}
	return count
}
