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
	videoSources := []string{"webm", "mov", "qt", "mkv", "avi", "mp4", "m4v"}
	audioSources := []string{"wav", "flac", "m4a", "m4b", "aac", "ogg", "oga", "opus", "mp3"}
	formats := make([]SupportedFormat, 0, len(videoSources)+len(audioSources))
	for _, source := range videoSources {
		formats = append(formats, SupportedFormat{
			Source:      source,
			Target:      "mp4",
			Profile:     "web",
			VideoCodec:  "h264 (libx264)",
			AudioCodec:  "aac",
			Description: "Broadly compatible MP4 for browsers and media players",
		})
	}
	for _, source := range audioSources {
		formats = append(formats, SupportedFormat{
			Source:      source,
			Target:      "mp3",
			Profile:     "music",
			VideoCodec:  "none",
			AudioCodec:  "mp3 (libmp3lame)",
			Description: "Portable MP3 audio for music players and sharing",
		})
	}
	return formats
}

func (Registry) Plan(inputPath, outputPath, target, preset string, info media.Info, capabilities media.Capabilities) (media.Plan, error) {
	target = strings.ToLower(strings.TrimSpace(target))
	preset = strings.ToLower(strings.TrimSpace(preset))
	if target == "" {
		target = "mp4"
	}
	if preset == "" {
		preset = defaultPreset(target)
	}
	switch target {
	case "mp4":
		return planWebMP4(inputPath, outputPath, preset, info, capabilities)
	case "mp3":
		return planMusicMP3(inputPath, outputPath, preset, info, capabilities)
	default:
		return media.Plan{}, fmt.Errorf("%w: %q", ErrUnsupportedTarget, target)
	}
}

func planWebMP4(inputPath, outputPath, preset string, info media.Info, capabilities media.Capabilities) (media.Plan, error) {
	if preset != "web" {
		return media.Plan{}, fmt.Errorf("%w: %q", ErrUnsupportedPreset, preset)
	}
	sourceFormat, ok := supportedVideoSource(inputPath, info.FormatNames)
	if !ok {
		return media.Plan{}, fmt.Errorf(
			"%w: ffprobe detected %q instead of webm, mov, mkv, avi, or mp4",
			ErrUnsupportedInput,
			strings.Join(info.FormatNames, ","),
		)
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
	if inputExtensionDoesNotMatchSource(inputPath, sourceFormat) {
		warnings = append(warnings, fmt.Sprintf("The input is detected as %s even though its file extension is %s.", strings.ToUpper(sourceFormat), strings.ToLower(filepath.Ext(inputPath))))
	}

	plan := media.Plan{
		InputPath:    inputPath,
		OutputPath:   outputPath,
		SourceFormat: sourceFormat,
		TargetFormat: "mp4",
		Profile:      "web",
		VideoMap:     "0:v:0",
		Video: &media.VideoSettings{
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

func planMusicMP3(inputPath, outputPath, preset string, info media.Info, capabilities media.Capabilities) (media.Plan, error) {
	if preset != "music" {
		return media.Plan{}, fmt.Errorf("%w: %q", ErrUnsupportedPreset, preset)
	}
	sourceFormat, ok := supportedAudioSource(inputPath, info.FormatNames)
	if !ok {
		return media.Plan{}, fmt.Errorf(
			"%w: ffprobe detected %q instead of wav, flac, m4a, aac, ogg, or mp3",
			ErrUnsupportedInput,
			strings.Join(info.FormatNames, ","),
		)
	}

	audios := info.AudioStreams()
	if len(audios) == 0 {
		return media.Plan{}, fmt.Errorf("%w: no audio stream was found", ErrUnsupportedInput)
	}
	if !capabilities.HasEncoder("libmp3lame") {
		return media.Plan{}, fmt.Errorf("%w: libmp3lame encoder", ErrMissingCapability)
	}
	if !capabilities.HasMuxer("mp3") {
		return media.Plan{}, fmt.Errorf("%w: MP3 muxer", ErrMissingCapability)
	}

	warnings := make([]string, 0)
	if len(info.VideoStreams()) > 0 {
		warnings = append(warnings, "Video streams are not included in the MP3 output.")
	}
	if len(audios) > 1 {
		warnings = append(warnings, "Only the first audio stream will be converted.")
	}
	if len(info.SubtitleStreams()) > 0 {
		warnings = append(warnings, "Subtitle streams are not included in the MP3 output.")
	}
	if info.ChapterCount > 0 {
		warnings = append(warnings, "Chapters are not included in the MP3 output.")
	}
	if inputExtensionDoesNotMatchSource(inputPath, sourceFormat) {
		warnings = append(warnings, fmt.Sprintf("The input is detected as %s even though its file extension is %s.", strings.ToUpper(sourceFormat), strings.ToLower(filepath.Ext(inputPath))))
	}

	return media.Plan{
		InputPath:     inputPath,
		OutputPath:    outputPath,
		SourceFormat:  sourceFormat,
		TargetFormat:  "mp3",
		Profile:       "music",
		AudioMap:      "0:a:0",
		Audio:         &media.AudioSettings{Codec: "libmp3lame", BitRate: "192k"},
		CopyMetadata:  true,
		DropChapters:  true,
		Warnings:      warnings,
		InputDuration: info.Duration,
	}, nil
}

func Verify(plan media.Plan, info media.Info) error {
	if info.Size <= 0 {
		return fmt.Errorf("output file is empty")
	}
	switch plan.TargetFormat {
	case "mp3":
		return verifyMP3(plan, info)
	default:
		return verifyMP4(plan, info)
	}
}

func verifyMP4(plan media.Plan, info media.Info) error {
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

func verifyMP3(plan media.Plan, info media.Info) error {
	if !hasFormat(info.FormatNames, "mp3") {
		return fmt.Errorf("ffprobe did not detect an MP3 container")
	}
	audios := info.AudioStreams()
	if len(audios) == 0 || audios[0].CodecName != "mp3" {
		return fmt.Errorf("output does not contain the expected MP3 audio stream")
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

func defaultPreset(target string) string {
	switch target {
	case "mp3":
		return "music"
	default:
		return "web"
	}
}

func supportedVideoSource(inputPath string, formats []string) (string, bool) {
	switch strings.TrimPrefix(strings.ToLower(filepath.Ext(inputPath)), ".") {
	case "webm":
		if hasFormat(formats, "webm") {
			return "webm", true
		}
	case "mov", "qt":
		if hasFormat(formats, "mov") || hasFormat(formats, "mp4") {
			return "mov", true
		}
	case "mkv", "mka", "mks":
		if hasFormat(formats, "matroska") {
			return "mkv", true
		}
	case "avi":
		if hasFormat(formats, "avi") {
			return "avi", true
		}
	case "mp4", "m4v":
		if hasFormat(formats, "mp4") || hasFormat(formats, "mov") {
			return "mp4", true
		}
	}

	switch {
	case hasFormat(formats, "webm"):
		return "webm", true
	case hasFormat(formats, "avi"):
		return "avi", true
	case hasFormat(formats, "matroska"):
		return "mkv", true
	case hasFormat(formats, "mp4"):
		return "mp4", true
	case hasFormat(formats, "mov"):
		return "mov", true
	default:
		return "", false
	}
}

func supportedAudioSource(inputPath string, formats []string) (string, bool) {
	switch strings.TrimPrefix(strings.ToLower(filepath.Ext(inputPath)), ".") {
	case "wav":
		if hasFormat(formats, "wav") {
			return "wav", true
		}
	case "flac":
		if hasFormat(formats, "flac") {
			return "flac", true
		}
	case "m4a", "m4b":
		if hasFormat(formats, "mov") || hasFormat(formats, "mp4") {
			return "m4a", true
		}
	case "aac":
		if hasFormat(formats, "aac") {
			return "aac", true
		}
	case "ogg", "oga", "opus":
		if hasFormat(formats, "ogg") {
			return "ogg", true
		}
	case "mp3":
		if hasFormat(formats, "mp3") {
			return "mp3", true
		}
	}

	switch {
	case hasFormat(formats, "wav"):
		return "wav", true
	case hasFormat(formats, "flac"):
		return "flac", true
	case hasFormat(formats, "aac"):
		return "aac", true
	case hasFormat(formats, "ogg"):
		return "ogg", true
	case hasFormat(formats, "mp3"):
		return "mp3", true
	case hasFormat(formats, "mov"), hasFormat(formats, "mp4"):
		return "m4a", true
	default:
		return "", false
	}
}

func inputExtensionDoesNotMatchSource(inputPath, source string) bool {
	extension := strings.TrimPrefix(strings.ToLower(filepath.Ext(inputPath)), ".")
	if extension == "" {
		return true
	}
	switch source {
	case "mkv":
		return extension != "mkv" && extension != "mka" && extension != "mks"
	case "mov":
		return extension != "mov" && extension != "qt"
	case "mp4":
		return extension != "mp4" && extension != "m4v"
	case "m4a":
		return extension != "m4a" && extension != "m4b"
	case "ogg":
		return extension != "ogg" && extension != "oga" && extension != "opus"
	default:
		return extension != source
	}
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
