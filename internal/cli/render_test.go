package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Amad3eu/mediaconv/internal/app"
	"github.com/Amad3eu/mediaconv/internal/media"
)

// decodeObject reads the rendered object back as a map, so a key that was
// renamed or dropped is visible. root_test.go's decodeJSON decodes into a
// typed target instead, which catches added keys but not missing ones.
func decodeObject(t *testing.T, raw []byte) map[string]any {
	t.Helper()

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, raw)
	}
	return decoded
}

// assertKeys pins the exact key set. --json is a public contract: renaming or
// dropping a key breaks every script reading it, and nothing else in the suite
// would notice.
func assertKeys(t *testing.T, label string, object map[string]any, want ...string) {
	t.Helper()

	got := make([]string, 0, len(object))
	for key := range object {
		got = append(got, key)
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("%s keys =\n  %v\nwant\n  %v", label, got, want)
	}
}

func sampleInfo() media.Info {
	return media.Info{
		Path:           "/tmp/clip.mp4",
		FormatNames:    []string{"mov", "mp4"},
		FormatLongName: "QuickTime / MOV",
		Duration:       1500 * time.Millisecond,
		Size:           4096,
		BitRate:        128000,
		ChapterCount:   0,
		Streams: []media.Stream{
			{Index: 0, CodecType: "video", CodecName: "h264", Width: 1920, Height: 1080},
			{Index: 1, CodecType: "audio", CodecName: "aac", Channels: 2},
		},
	}
}

func TestConvertJSONKeepsItsContract(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	result := app.ConvertResult{
		InputPath:  "/tmp/in.webm",
		OutputPath: "/tmp/out.mp4",
		Elapsed:    2500 * time.Millisecond,
		OutputInfo: sampleInfo(),
		Warnings:   []string{"no audio stream"},
	}
	if err := writeConvertResult(&buffer, result, true, palette{}); err != nil {
		t.Fatalf("writeConvertResult() error = %v", err)
	}

	decoded := decodeObject(t, buffer.Bytes())
	assertKeys(t, "convert", decoded,
		"ok", "input_path", "output_path", "elapsed_seconds", "warnings", "plan", "output")

	if decoded["ok"] != true {
		t.Errorf("ok = %v, want true", decoded["ok"])
	}
	if decoded["elapsed_seconds"] != 2.5 {
		t.Errorf("elapsed_seconds = %v, want 2.5", decoded["elapsed_seconds"])
	}
	if decoded["output_path"] != "/tmp/out.mp4" {
		t.Errorf("output_path = %v, want the converted file", decoded["output_path"])
	}
}

func TestBatchJSONKeepsItsContract(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	result := app.BatchResult{
		InputDir:  "/tmp/clips",
		OutputDir: "/tmp/clips/mp3",
		Target:    "mp3",
		Preset:    "music",
		Jobs:      4,
		Total:     2,
		Converted: 2,
		Elapsed:   time.Second,
		Items:     []app.BatchItem{{InputPath: "/tmp/clips/a.wav", OutputPath: "/tmp/clips/mp3/a.mp3", OK: true}},
	}
	if err := writeBatchResult(&buffer, result, true, palette{}); err != nil {
		t.Fatalf("writeBatchResult() error = %v", err)
	}

	decoded := decodeObject(t, buffer.Bytes())
	// "jobs" is deliberately absent: writeBatchResult builds this object by
	// hand and never included it, although app.BatchResult carries the tag.
	// Pinned here so that adding it becomes a visible, documented change.
	assertKeys(t, "batch", decoded,
		"ok", "input_dir", "output_dir", "target", "preset", "recursive",
		"total", "converted", "failed", "elapsed_seconds", "items")
}

// ok mirrors the failure count, not the fact that the command ran. A script
// that trusts it has to be able to tell a clean batch from a partial one.
func TestBatchJSONReportsNotOKWhenAnyFileFailed(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name   string
		failed int
		wantOK bool
	}{
		{name: "every file converted", failed: 0, wantOK: true},
		{name: "one file failed", failed: 1, wantOK: false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var buffer bytes.Buffer
			result := app.BatchResult{Total: 3, Converted: 3 - testCase.failed, Failed: testCase.failed}
			if err := writeBatchResult(&buffer, result, true, palette{}); err != nil {
				t.Fatalf("writeBatchResult() error = %v", err)
			}

			if got := decodeObject(t, buffer.Bytes())["ok"]; got != testCase.wantOK {
				t.Errorf("ok = %v with %d failed, want %v", got, testCase.failed, testCase.wantOK)
			}
		})
	}
}

func TestMediaInfoJSONKeepsItsContract(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	if err := writeMediaInfo(&buffer, sampleInfo(), true); err != nil {
		t.Fatalf("writeMediaInfo() error = %v", err)
	}

	decoded := decodeObject(t, buffer.Bytes())
	assertKeys(t, "info", decoded, "ok", "media")

	object, ok := decoded["media"].(map[string]any)
	if !ok {
		t.Fatalf("media = %T, want an object", decoded["media"])
	}
	assertKeys(t, "media", object,
		"path", "format_names", "format_long_name", "duration_seconds",
		"size_bytes", "bit_rate", "chapter_count", "streams", "tags")

	if object["duration_seconds"] != 1.5 {
		t.Errorf("duration_seconds = %v, want 1.5", object["duration_seconds"])
	}

	streams, ok := object["streams"].([]any)
	if !ok || len(streams) != 2 {
		t.Fatalf("streams = %v, want two entries", object["streams"])
	}
	stream, ok := streams[0].(map[string]any)
	if !ok {
		t.Fatalf("stream = %T, want an object", streams[0])
	}
	// "color_transfer" is absent for the same reason "jobs" is: mediaInfoView
	// builds the object by hand and media.Stream's tag never reaches it.
	assertKeys(t, "stream", stream,
		"index", "codec_type", "codec_name", "codec_long_name", "profile",
		"pixel_format", "width", "height", "channels", "channel_layout",
		"sample_rate", "duration_seconds", "tags")
}

func TestConvertTextWritesEveryWarning(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	result := app.ConvertResult{
		OutputPath: "/tmp/out.mp4",
		Warnings:   []string{"first warning", "second warning"},
	}
	if err := writeConvertResult(&buffer, result, false, palette{}); err != nil {
		t.Fatalf("writeConvertResult() error = %v", err)
	}

	output := buffer.String()
	for _, want := range []string{"Converted: /tmp/out.mp4", "Warning: first warning", "Warning: second warning"} {
		if !strings.Contains(output, want) {
			t.Errorf("output is missing %q:\n%s", want, output)
		}
	}
}

// The arrow and the reason are each conditional, so a file that failed before
// producing anything must not print a dangling "-> ".
func TestBatchTextOmitsTheArrowWhenThereIsNoOutput(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	result := app.BatchResult{
		Total:     2,
		Converted: 1,
		Failed:    1,
		Items: []app.BatchItem{
			{InputPath: "/tmp/a.wav", OutputPath: "/tmp/a.mp3", OK: true, Warnings: []string{"clipping"}},
			{InputPath: "/tmp/b.wav", OK: false, Error: "unsupported codec"},
		},
	}
	if err := writeBatchResult(&buffer, result, false, palette{}); err != nil {
		t.Fatalf("writeBatchResult() error = %v", err)
	}

	output := buffer.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 4 {
		t.Fatalf("got %d lines, want a summary, two items and one warning:\n%s", len(lines), output)
	}
	if !strings.Contains(lines[0], "2 file(s), 1 converted, 1 failed") {
		t.Errorf("summary = %q", lines[0])
	}
	if !strings.Contains(lines[1], "/tmp/a.wav -> /tmp/a.mp3") {
		t.Errorf("converted line = %q, want the arrow and the output path", lines[1])
	}
	if strings.Contains(lines[3], "->") {
		t.Errorf("failed line = %q, want no arrow when nothing was produced", lines[3])
	}
	if !strings.Contains(lines[3], "(unsupported codec)") {
		t.Errorf("failed line = %q, want the reason", lines[3])
	}
}

// Dimensions belong to video streams and channel counts to audio streams.
// Printing either on the wrong one reports 0x0 to the user.
func TestMediaInfoTextSeparatesVideoFromAudio(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	if err := writeMediaInfo(&buffer, sampleInfo(), false); err != nil {
		t.Fatalf("writeMediaInfo() error = %v", err)
	}

	output := buffer.String()
	for _, want := range []string{
		"File: /tmp/clip.mp4",
		"Format: mov, mp4",
		"Duration: 1.5s",
		"Size: 4096 bytes",
		"Stream #0: video h264 1920x1080",
		"Stream #1: audio aac 2 channel(s)",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("output is missing %q:\n%s", want, output)
		}
	}
	// Checked per line, and for the label rather than a count: a broken
	// guard prints "0 channel(s)" on the video stream, which a search for the
	// audio stream's "2 channel(s)" would miss.
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "Stream #0:") && strings.Contains(line, "channel(s)") {
			t.Errorf("video stream line carries a channel count: %q", line)
		}
		if strings.HasPrefix(line, "Stream #1:") && strings.Contains(line, "x0") {
			t.Errorf("audio stream line carries dimensions: %q", line)
		}
	}
}

func TestMediaInfoTextOmitsChannelsWhenThereAreNone(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	info := media.Info{Streams: []media.Stream{{Index: 0, CodecType: "audio", CodecName: "aac"}}}
	if err := writeMediaInfo(&buffer, info, false); err != nil {
		t.Fatalf("writeMediaInfo() error = %v", err)
	}

	if strings.Contains(buffer.String(), "channel(s)") {
		t.Errorf("output reports a channel count it does not have:\n%s", buffer.String())
	}
}

func TestHumanDuration(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name     string
		duration time.Duration
		want     string
	}{
		// ffprobe leaves Duration at zero for streams that do not report one,
		// and "0s" would read as a real measurement.
		{name: "zero", duration: 0, want: "unknown"},
		{name: "negative", duration: -time.Second, want: "unknown"},
		{name: "rounded to the millisecond", duration: 1500400 * time.Microsecond, want: "1.5s"},
		{name: "minutes", duration: 90 * time.Second, want: "1m30s"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := humanDuration(testCase.duration); got != testCase.want {
				t.Errorf("humanDuration(%v) = %q, want %q", testCase.duration, got, testCase.want)
			}
		})
	}
}

func TestSeconds(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		duration time.Duration
		want     float64
	}{
		{duration: 0, want: 0},
		{duration: 500 * time.Millisecond, want: 0.5},
		{duration: 2 * time.Second, want: 2},
		{duration: 90 * time.Second, want: 90},
	} {
		if got := seconds(testCase.duration); got != testCase.want {
			t.Errorf("seconds(%v) = %v, want %v", testCase.duration, got, testCase.want)
		}
	}
}

// failingWriter refuses every write, the way a closed pipe does when the user
// pipes mediaconv into head.
type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

// A renderer that swallowed the error would report success to the caller while
// the user saw nothing, and the exit code would be wrong.
func TestRenderersReportWriteFailures(t *testing.T) {
	t.Parallel()

	broken := failingWriter{err: errors.New("broken pipe")}
	info := sampleInfo()
	convert := app.ConvertResult{OutputPath: "/tmp/out.mp4", Warnings: []string{"a warning"}}
	batch := app.BatchResult{
		Total: 1,
		Items: []app.BatchItem{{InputPath: "/tmp/a.wav", OutputPath: "/tmp/a.mp3", OK: true, Warnings: []string{"clipping"}}},
	}

	for _, testCase := range []struct {
		name   string
		render func(w io.Writer, asJSON bool) error
	}{
		{"convert", func(w io.Writer, asJSON bool) error {
			return writeConvertResult(w, convert, asJSON, palette{})
		}},
		{"batch", func(w io.Writer, asJSON bool) error {
			return writeBatchResult(w, batch, asJSON, palette{})
		}},
		{"info", func(w io.Writer, asJSON bool) error {
			return writeMediaInfo(w, info, asJSON)
		}},
	} {
		for _, asJSON := range []bool{false, true} {
			name := testCase.name
			if asJSON {
				name += " as JSON"
			}
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				if err := testCase.render(broken, asJSON); err == nil {
					t.Error("renderer reported success although every write failed")
				}
			})
		}
	}
}
