package ffmpeg

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/Amad3eu/mediaconv/internal/media"
)

type Prober struct {
	Binary string
}

func (p Prober) Probe(ctx context.Context, path string) (media.Info, error) {
	args := []string{
		"-v", "error",
		"-protocol_whitelist", "file",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		"-show_chapters",
		path,
	}

	cmd := exec.CommandContext(ctx, p.Binary, args...)
	cmd.WaitDelay = 3 * time.Second
	stdout := newLimitedBuffer(8 * 1024 * 1024)
	stderr := newTailBuffer(64 * 1024)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	if err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = err.Error()
		}
		return media.Info{}, fmt.Errorf("ffprobe failed: %s", detail)
	}
	if stdout.Exceeded() {
		return media.Info{}, fmt.Errorf("ffprobe JSON exceeded the 8 MiB safety limit")
	}

	info, err := parseProbeJSON(stdout.Bytes())
	if err != nil {
		return media.Info{}, fmt.Errorf("decode ffprobe output: %w", err)
	}
	info.Path = path
	return info, nil
}

type limitedBuffer struct {
	data     []byte
	max      int
	exceeded bool
}

func newLimitedBuffer(max int) *limitedBuffer {
	return &limitedBuffer{max: max}
}

func (b *limitedBuffer) Write(data []byte) (int, error) {
	written := len(data)
	remaining := b.max - len(b.data)
	if remaining <= 0 {
		b.exceeded = true
		return written, nil
	}
	if len(data) > remaining {
		b.data = append(b.data, data[:remaining]...)
		b.exceeded = true
		return written, nil
	}
	b.data = append(b.data, data...)
	return written, nil
}

func (b *limitedBuffer) Bytes() []byte  { return b.data }
func (b *limitedBuffer) Exceeded() bool { return b.exceeded }

type probeDocument struct {
	Streams  []probeStream     `json:"streams"`
	Chapters []json.RawMessage `json:"chapters"`
	Format   probeFormat       `json:"format"`
}

type probeStream struct {
	Index         int               `json:"index"`
	CodecName     string            `json:"codec_name"`
	CodecLongName string            `json:"codec_long_name"`
	Profile       string            `json:"profile"`
	CodecType     string            `json:"codec_type"`
	PixelFormat   string            `json:"pix_fmt"`
	Width         int               `json:"width"`
	Height        int               `json:"height"`
	Channels      int               `json:"channels"`
	ChannelLayout string            `json:"channel_layout"`
	SampleRate    string            `json:"sample_rate"`
	ColorTransfer string            `json:"color_transfer"`
	Duration      string            `json:"duration"`
	Tags          map[string]string `json:"tags"`
}

type probeFormat struct {
	FormatName     string            `json:"format_name"`
	FormatLongName string            `json:"format_long_name"`
	Duration       string            `json:"duration"`
	Size           string            `json:"size"`
	BitRate        string            `json:"bit_rate"`
	Tags           map[string]string `json:"tags"`
}

func parseProbeJSON(data []byte) (media.Info, error) {
	var document probeDocument
	if err := json.Unmarshal(data, &document); err != nil {
		return media.Info{}, err
	}

	info := media.Info{
		FormatNames:    splitNames(document.Format.FormatName),
		FormatLongName: document.Format.FormatLongName,
		Duration:       parseSeconds(document.Format.Duration),
		Size:           parseInt64(document.Format.Size),
		BitRate:        parseInt64(document.Format.BitRate),
		ChapterCount:   len(document.Chapters),
		Tags:           document.Format.Tags,
		Streams:        make([]media.Stream, 0, len(document.Streams)),
	}

	for _, stream := range document.Streams {
		info.Streams = append(info.Streams, media.Stream{
			Index:         stream.Index,
			CodecType:     stream.CodecType,
			CodecName:     stream.CodecName,
			CodecLongName: stream.CodecLongName,
			Profile:       stream.Profile,
			PixelFormat:   stream.PixelFormat,
			Width:         stream.Width,
			Height:        stream.Height,
			Channels:      stream.Channels,
			ChannelLayout: stream.ChannelLayout,
			SampleRate:    int(parseInt64(stream.SampleRate)),
			ColorTransfer: stream.ColorTransfer,
			Duration:      parseSeconds(stream.Duration),
			Tags:          stream.Tags,
		})
	}

	return info, nil
}

func splitNames(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if name := strings.TrimSpace(part); name != "" {
			result = append(result, name)
		}
	}
	return result
}

func parseSeconds(value string) time.Duration {
	seconds, err := strconv.ParseFloat(value, 64)
	if err != nil || seconds < 0 {
		return 0
	}
	return time.Duration(seconds * float64(time.Second))
}

func parseInt64(value string) int64 {
	number, _ := strconv.ParseInt(value, 10, 64)
	return number
}
