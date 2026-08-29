package media

import "time"

type Stream struct {
	Index         int               `json:"index"`
	CodecType     string            `json:"codec_type"`
	CodecName     string            `json:"codec_name"`
	CodecLongName string            `json:"codec_long_name,omitempty"`
	Profile       string            `json:"profile,omitempty"`
	PixelFormat   string            `json:"pixel_format,omitempty"`
	Width         int               `json:"width,omitempty"`
	Height        int               `json:"height,omitempty"`
	Channels      int               `json:"channels,omitempty"`
	ChannelLayout string            `json:"channel_layout,omitempty"`
	SampleRate    int               `json:"sample_rate,omitempty"`
	ColorTransfer string            `json:"color_transfer,omitempty"`
	Duration      time.Duration     `json:"-"`
	Tags          map[string]string `json:"tags,omitempty"`
}

type Info struct {
	Path           string            `json:"path"`
	FormatNames    []string          `json:"format_names"`
	FormatLongName string            `json:"format_long_name,omitempty"`
	Duration       time.Duration     `json:"-"`
	Size           int64             `json:"size_bytes"`
	BitRate        int64             `json:"bit_rate,omitempty"`
	Streams        []Stream          `json:"streams"`
	ChapterCount   int               `json:"chapter_count"`
	Tags           map[string]string `json:"tags,omitempty"`
}

func (i Info) VideoStreams() []Stream {
	return streamsOfType(i.Streams, "video")
}

func (i Info) AudioStreams() []Stream {
	return streamsOfType(i.Streams, "audio")
}

func (i Info) SubtitleStreams() []Stream {
	return streamsOfType(i.Streams, "subtitle")
}

func streamsOfType(streams []Stream, kind string) []Stream {
	result := make([]Stream, 0)
	for _, stream := range streams {
		if stream.CodecType == kind {
			result = append(result, stream)
		}
	}
	return result
}

type Capabilities struct {
	FFmpegPath     string          `json:"ffmpeg_path"`
	FFprobePath    string          `json:"ffprobe_path"`
	FFmpegVersion  string          `json:"ffmpeg_version"`
	FFprobeVersion string          `json:"ffprobe_version"`
	Encoders       map[string]bool `json:"-"`
	Muxers         map[string]bool `json:"-"`
}

func (c Capabilities) HasEncoder(name string) bool {
	return c.Encoders[name]
}

func (c Capabilities) HasMuxer(name string) bool {
	return c.Muxers[name]
}

type VideoSettings struct {
	Codec       string   `json:"codec"`
	CRF         int      `json:"crf"`
	Preset      string   `json:"preset"`
	PixelFormat string   `json:"pixel_format"`
	Filters     []string `json:"filters,omitempty"`
}

type AudioSettings struct {
	Codec   string `json:"codec"`
	BitRate string `json:"bit_rate"`
}

type Plan struct {
	InputPath     string         `json:"input_path"`
	OutputPath    string         `json:"output_path"`
	SourceFormat  string         `json:"source_format"`
	TargetFormat  string         `json:"target_format"`
	Profile       string         `json:"profile"`
	VideoMap      string         `json:"video_map"`
	AudioMap      string         `json:"audio_map,omitempty"`
	Video         VideoSettings  `json:"video"`
	Audio         *AudioSettings `json:"audio,omitempty"`
	MovFlags      []string       `json:"movflags,omitempty"`
	CopyMetadata  bool           `json:"copy_metadata"`
	DropChapters  bool           `json:"drop_chapters"`
	Warnings      []string       `json:"warnings,omitempty"`
	InputDuration time.Duration  `json:"-"`
}

type Progress struct {
	Frame     int64
	Processed time.Duration
	Total     time.Duration
	Speed     string
	Done      bool
}
