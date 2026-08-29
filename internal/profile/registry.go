package profile

import (
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"time"

	"github.com/Amad3eu/mediaconv/internal/media"
)

var (
	ErrUnsupportedTarget = errors.New("unsupported target format")
	ErrUnsupportedInput  = errors.New("unsupported input")
	ErrMissingCapability = errors.New("required FFmpeg capability is missing")
	ErrUnsupportedPreset = errors.New("unsupported preset")
)

type Registry struct{}

type SupportedFormat struct {
	Source      string `json:"source"`
	Target      string `json:"target"`
	Profile     string `json:"profile"`
	VideoCodec  string `json:"video_codec"`
	AudioCodec  string `json:"audio_codec"`
	Description string `json:"description"`
}

func (Registry) Formats() []SupportedFormat {
	return []SupportedFormat{{
		Source:      "webm",
		Target:      "mp4",
		Profile:     "web",
		VideoCodec:  "h264 (libx264)",
		AudioCodec:  "aac",
		Description: "Broadly compatible MP4 for browsers and media players",
	}}
}

func (Registry) Plan(inputPath, outputPath, target, preset string, info media.Info, capabilities media.Capabilities) (media.Plan, error) {
	target = strings.ToLower(strings.TrimSpace(target))
	preset = strings.ToLower(strings.TrimSpace(preset))
	if target != "mp4" {
		return media.Plan{}, fmt.Errorf("%w: %q", ErrUnsupportedTarget, target)
	}
	if preset != "web" {
		return media.Plan{}, fmt.Errorf("%w: %q", ErrUnsupportedPreset, preset)
	}
	if !hasFormat(info.FormatNames, "webm") {
		return media.Plan{}, fmt.Errorf("%w: ffprobe detected %q instead of WebM", ErrUnsupportedInput, strings.Join(info.FormatNames, ","))
	}

	videos := info.VideoStreams()
	if len(videos) == 0 {
		return media.Plan{}, fmt.Errorf("%w: no video stream was found", ErrUnsupportedInput)
	}
	if !capabilities.HasEncoder("libx264") {
		return media.Plan{}, fmt.Errorf("%w: libx264 encoder", ErrMissingCapability)
	}
	if !capabilities.HasMuxer("mp4") && !capabilities.HasMuxer("mov") {
		return media.Plan{}, fmt.Errorf("%w: MP4 muxer", ErrMissingCapability)
	}

	audios := info.AudioStreams()
	if len(audios) > 0 && !capabilities.HasEncoder("aac") {
		return media.Plan{}, fmt.Errorf("%w: AAC encoder", ErrMissingCapability)
	}

	video := videos[0]
	filters := make([]string, 0, 1)
	if video.Width%2 != 0 || video.Height%2 != 0 {
		filters = append(filters, "pad=ceil(iw/2)*2:ceil(ih/2)*2")
	}

	warnings := make([]string, 0)
	if len(videos) > 1 {
		warnings = append(warnings, "Only the first video stream will be converted.")
	}
	if len(audios) > 1 {
		warnings = append(warnings, "Only the first audio stream will be converted.")
	}
	if len(info.SubtitleStreams()) > 0 {
		warnings = append(warnings, "Subtitle streams are not included in the MP4 output.")
	}
	if info.ChapterCount > 0 {
		warnings = append(warnings, "Chapters are not included in the MP4 output.")
	}
	if strings.Contains(strings.ToLower(video.PixelFormat), "yuva") || strings.Contains(strings.ToLower(video.PixelFormat), "rgba") {
		warnings = append(warnings, "The source has an alpha channel; transparency will be lost.")
	}
	if isHDR(video.ColorTransfer) {
		warnings = append(warnings, "The source appears to use HDR transfer characteristics; the web preset may not preserve HDR correctly.")
	}
	if strings.ToLower(filepath.Ext(inputPath)) != ".webm" {
		warnings = append(warnings, "The input is detected as WebM even though its file extension is not .webm.")
	}

	plan := media.Plan{
		InputPath:    inputPath,
		OutputPath:   outputPath,
		SourceFormat: "webm",
		TargetFormat: "mp4",
		Profile:      "web",
		VideoMap:     "0:v:0",
		Video: media.VideoSettings{
			Codec:       "libx264",
			CRF:         23,
			Preset:      "medium",
			PixelFormat: "yuv420p",
			Filters:     filters,
		},
		MovFlags:      []string{"faststart"},
		CopyMetadata:  true,
		DropChapters:  true,
		Warnings:      warnings,
		InputDuration: info.Duration,
	}
	if len(audios) > 0 {
		plan.AudioMap = "0:a:0"
		plan.Audio = &media.AudioSettings{Codec: "aac", BitRate: "192k"}
	}
	return plan, nil
}

func Verify(plan media.Plan, info media.Info) error {
	if info.Size <= 0 {
		return fmt.Errorf("output file is empty")
	}
	if !hasFormat(info.FormatNames, "mp4") && !hasFormat(info.FormatNames, "mov") {
		return fmt.Errorf("ffprobe did not detect an MP4 container")
	}
	videos := info.VideoStreams()
	if len(videos) == 0 || videos[0].CodecName != "h264" {
		return fmt.Errorf("output does not contain the expected H.264 video stream")
	}
	if videos[0].Width%2 != 0 || videos[0].Height%2 != 0 {
		return fmt.Errorf("output video dimensions are not even")
	}
	if plan.Audio != nil {
		audios := info.AudioStreams()
		if len(audios) == 0 || audios[0].CodecName != "aac" {
			return fmt.Errorf("output does not contain the expected AAC audio stream")
		}
	}
	if plan.InputDuration > 0 {
		if info.Duration <= 0 {
			return fmt.Errorf("output duration could not be verified")
		}
		tolerance := maxDuration(2*time.Second, plan.InputDuration/10)
		if time.Duration(math.Abs(float64(info.Duration-plan.InputDuration))) > tolerance {
			return fmt.Errorf(
				"output duration %s differs unexpectedly from input duration %s",
				info.Duration.Round(time.Millisecond),
				plan.InputDuration.Round(time.Millisecond),
			)
		}
	}
	return nil
}

func hasFormat(formats []string, target string) bool {
	for _, format := range formats {
		if strings.EqualFold(format, target) {
			return true
		}
	}
	return false
}

func isHDR(transfer string) bool {
	transfer = strings.ToLower(transfer)
	return transfer == "smpte2084" || transfer == "arib-std-b67"
}

func maxDuration(left, right time.Duration) time.Duration {
	if left > right {
		return left
	}
	return right
}
