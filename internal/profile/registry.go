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
	ErrInvalidRange      = errors.New("invalid range")
)

type Registry struct{}

// Request is what the caller is asking for. It is a struct rather than a list
// of parameters because the trim fields are optional and a positional empty
// string tells the reader nothing about which knob it is.
type Request struct {
	InputPath  string
	OutputPath string
	Target     string
	Preset     string
	// TrimStart and TrimDuration limit the conversion to part of the input.
	// Zero means from the beginning, and to the end. A profile may supply its
	// own default duration, which these override when set.
	TrimStart    time.Duration
	TrimDuration time.Duration
}

// Target is an output format the registry can plan for. It is the single
// source of truth: internal/app asks here instead of keeping its own copy of
// the list, so adding a format stays inside this package, as ADR 0003 intends.
type Target struct {
	Name          string
	DefaultPreset string
	// BatchExtensions are the input extensions `mediaconv batch` picks up for
	// this target. The target's own extension is deliberately absent, so a
	// batch never converts a file onto itself. A single `convert` is less
	// restrictive: re-encoding an MP4 to MP4 is a legitimate request.
	BatchExtensions []string
}

var targets = []Target{
	{
		Name:            "mp4",
		DefaultPreset:   "web",
		BatchExtensions: []string{".webm", ".mov", ".qt", ".mkv", ".avi", ".m4v"},
	},
	{
		Name:          "webm",
		DefaultPreset: "stream",
		// MP4 is here and .mp4 is absent from the mp4 target above: the two
		// directions are what make a round trip possible.
		BatchExtensions: []string{".mp4", ".m4v", ".mov", ".qt", ".mkv", ".avi"},
	},
	{
		Name:            "mp3",
		DefaultPreset:   "music",
		BatchExtensions: []string{".wav", ".flac", ".m4a", ".m4b", ".aac", ".ogg", ".oga", ".opus"},
	},
	{
		Name:          "gif",
		DefaultPreset: "preview",
		// GIF is made from video, so the sources are the video containers.
		BatchExtensions: []string{".mp4", ".m4v", ".mov", ".qt", ".mkv", ".avi", ".webm"},
	},
	{
		Name:            "m4a",
		DefaultPreset:   "aac",
		BatchExtensions: []string{".wav", ".flac", ".m4b", ".aac", ".ogg", ".oga", ".opus", ".mp3"},
	},
	{
		Name:            "wav",
		DefaultPreset:   "master",
		BatchExtensions: []string{".flac", ".m4a", ".m4b", ".aac", ".ogg", ".oga", ".opus", ".mp3"},
	},
}

// Targets lists every supported output format.
func (Registry) Targets() []Target {
	out := make([]Target, len(targets))
	copy(out, targets)
	return out
}

// Target looks up one output format by name.
func (Registry) Target(name string) (Target, bool) {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, target := range targets {
		if target.Name == name {
			return target, true
		}
	}
	return Target{}, false
}

// TargetNames lists the supported output formats, for error messages.
func (Registry) TargetNames() []string {
	names := make([]string, 0, len(targets))
	for _, target := range targets {
		names = append(names, target.Name)
	}
	return names
}

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
	webmSources := []string{"mp4", "m4v", "mov", "qt", "mkv", "avi", "webm"}
	audioSources := []string{"wav", "flac", "m4a", "m4b", "aac", "ogg", "oga", "opus", "mp3"}
	gifSources := []string{"mp4", "m4v", "mov", "qt", "mkv", "avi", "webm"}
	m4aSources := []string{"wav", "flac", "m4b", "aac", "ogg", "oga", "opus", "mp3", "m4a"}
	wavSources := []string{"flac", "m4a", "m4b", "aac", "ogg", "oga", "opus", "mp3", "wav"}
	formats := make([]SupportedFormat, 0, len(videoSources)+len(webmSources)+len(gifSources)+len(m4aSources)+len(wavSources)+len(audioSources))
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
	for _, source := range webmSources {
		formats = append(formats, SupportedFormat{
			Source:      source,
			Target:      "webm",
			Profile:     "stream",
			VideoCodec:  "vp9 (libvpx-vp9)",
			AudioCodec:  "opus (libopus)",
			Description: "Royalty-free WebM for the web, without H.264 patent licensing",
		})
	}
	for _, source := range gifSources {
		formats = append(formats, SupportedFormat{
			Source:      source,
			Target:      "gif",
			Profile:     "preview",
			VideoCodec:  "gif",
			AudioCodec:  "none",
			Description: "Short looping preview, five seconds at 480 pixels wide",
		})
	}
	for _, source := range m4aSources {
		formats = append(formats, SupportedFormat{
			Source:      source,
			Target:      "m4a",
			Profile:     "aac",
			VideoCodec:  "none",
			AudioCodec:  "aac",
			Description: "AAC audio in an M4A container, for phones and modern players",
		})
	}
	for _, source := range wavSources {
		formats = append(formats, SupportedFormat{
			Source:      source,
			Target:      "wav",
			Profile:     "master",
			VideoCodec:  "none",
			AudioCodec:  "pcm_s16le",
			Description: "Uncompressed PCM, for editing and as an intermediate",
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

func (Registry) Plan(request Request, info media.Info, capabilities media.Capabilities) (media.Plan, error) {
	if request.TrimStart < 0 || request.TrimDuration < 0 {
		return media.Plan{}, fmt.Errorf("%w: a negative offset or length is not a range", ErrInvalidRange)
	}
	if info.Duration > 0 && request.TrimStart >= info.Duration {
		return media.Plan{}, fmt.Errorf(
			"%w: the start offset %s is at or past the end of a %s input",
			ErrInvalidRange,
			request.TrimStart.Round(time.Millisecond),
			info.Duration.Round(time.Millisecond),
		)
	}

	target := strings.ToLower(strings.TrimSpace(request.Target))
	preset := strings.ToLower(strings.TrimSpace(request.Preset))
	if target == "" {
		target = "mp4"
	}
	if preset == "" {
		preset = defaultPreset(target)
	}
	switch target {
	case "mp4":
		return planWebMP4(request, preset, info, capabilities)
	case "webm":
		return planStreamWebM(request, preset, info, capabilities)
	case "mp3":
		return planMusicMP3(request, preset, info, capabilities)
	case "m4a":
		return planAacM4A(request, preset, info, capabilities)
	case "wav":
		return planMasterWAV(request, preset, info, capabilities)
	case "gif":
		return planPreviewGIF(request, preset, info, capabilities)
	default:
		return media.Plan{}, fmt.Errorf("%w: %q", ErrUnsupportedTarget, target)
	}
}

func planWebMP4(request Request, preset string, info media.Info, capabilities media.Capabilities) (media.Plan, error) {
	inputPath, outputPath := request.InputPath, request.OutputPath
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
	applyTrim(&plan, request)
	return plan, nil
}

func planMusicMP3(request Request, preset string, info media.Info, capabilities media.Capabilities) (media.Plan, error) {
	inputPath, outputPath := request.InputPath, request.OutputPath
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

	plan := media.Plan{
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
	}
	applyTrim(&plan, request)
	return plan, nil
}

// planStreamWebM converts video to VP9 with Opus audio in a WebM container.
//
// The quality knobs differ from the web preset on purpose. CRF 32 is roughly
// the perceptual match of libx264 at CRF 23, because the two encoders do not
// share a scale. The deadline is "good" rather than "best": "best" costs
// several times the encode time for a difference most viewers cannot see, and
// this tool converts whole folders.
func planStreamWebM(request Request, preset string, info media.Info, capabilities media.Capabilities) (media.Plan, error) {
	inputPath, outputPath := request.InputPath, request.OutputPath
	if preset != "stream" {
		return media.Plan{}, fmt.Errorf("%w: %q", ErrUnsupportedPreset, preset)
	}
	sourceFormat, ok := supportedVideoSource(inputPath, info.FormatNames)
	if !ok {
		return media.Plan{}, fmt.Errorf(
			"%w: ffprobe detected %q instead of mp4, mov, mkv, avi, or webm",
			ErrUnsupportedInput,
			strings.Join(info.FormatNames, ","),
		)
	}

	videos := info.VideoStreams()
	if len(videos) == 0 {
		return media.Plan{}, fmt.Errorf("%w: no video stream was found", ErrUnsupportedInput)
	}
	if !capabilities.HasEncoder("libvpx-vp9") {
		return media.Plan{}, fmt.Errorf("%w: libvpx-vp9 encoder", ErrMissingCapability)
	}
	if !capabilities.HasMuxer("webm") {
		return media.Plan{}, fmt.Errorf("%w: WebM muxer", ErrMissingCapability)
	}

	audios := info.AudioStreams()
	if len(audios) > 0 && !capabilities.HasEncoder("libopus") {
		return media.Plan{}, fmt.Errorf("%w: libopus encoder", ErrMissingCapability)
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
		warnings = append(warnings, "Subtitle streams are not included in the WebM output.")
	}
	if info.ChapterCount > 0 {
		warnings = append(warnings, "Chapters are not included in the WebM output.")
	}
	if isHDR(video.ColorTransfer) {
		warnings = append(warnings, "The source appears to use HDR transfer characteristics; the stream preset may not preserve HDR correctly.")
	}
	if inputExtensionDoesNotMatchSource(inputPath, sourceFormat) {
		warnings = append(warnings, fmt.Sprintf("The input is detected as %s even though its file extension is %s.", strings.ToUpper(sourceFormat), strings.ToLower(filepath.Ext(inputPath))))
	}

	plan := media.Plan{
		InputPath:    inputPath,
		OutputPath:   outputPath,
		SourceFormat: sourceFormat,
		TargetFormat: "webm",
		Profile:      "stream",
		VideoMap:     "0:v:0",
		Video: &media.VideoSettings{
			Codec:       "libvpx-vp9",
			CRF:         32,
			Preset:      "good",
			PixelFormat: "yuv420p",
			Filters:     filters,
		},
		CopyMetadata:  true,
		DropChapters:  true,
		Warnings:      warnings,
		InputDuration: info.Duration,
	}
	if len(audios) > 0 {
		plan.AudioMap = "0:a:0"
		plan.Audio = &media.AudioSettings{Codec: "libopus", BitRate: "128k"}
	}
	applyTrim(&plan, request)
	return plan, nil
}

func verifyWebM(plan media.Plan, info media.Info) error {
	if !hasFormat(info.FormatNames, "webm") && !hasFormat(info.FormatNames, "matroska") {
		return fmt.Errorf("ffprobe did not detect a WebM container")
	}
	videos := info.VideoStreams()
	if len(videos) == 0 || videos[0].CodecName != "vp9" {
		return fmt.Errorf("output does not contain the expected VP9 video stream")
	}
	if videos[0].Width%2 != 0 || videos[0].Height%2 != 0 {
		return fmt.Errorf("output video dimensions are not even")
	}
	if plan.Audio != nil {
		audios := info.AudioStreams()
		if len(audios) == 0 || audios[0].CodecName != "opus" {
			return fmt.Errorf("output does not contain the expected Opus audio stream")
		}
	}
	if plan.InputDuration > 0 && info.Duration > 0 {
		if drift := info.Duration - plan.InputDuration; drift < -time.Second || drift > time.Second {
			return fmt.Errorf("output duration %s differs from the input duration %s", info.Duration, plan.InputDuration)
		}
	}
	return nil
}

// applyTrim copies the requested range onto a plan. Trimming is not specific
// to any one target: cutting thirty seconds out of a long recording is as
// useful for MP4 as it is for a preview.
func applyTrim(plan *media.Plan, request Request) {
	plan.TrimStart = request.TrimStart
	if request.TrimDuration > 0 {
		plan.TrimDuration = request.TrimDuration
	}
}

// remainingAfter is how much of the input is left once the start offset is
// taken out, or zero when the length is unknown.
func remainingAfter(info media.Info, start time.Duration) time.Duration {
	if info.Duration <= 0 {
		return 0
	}
	return info.Duration - start
}

// planAudioOnly is the shape the audio profiles share: one stream in, one
// stream out, everything else dropped, with the codec and container decided by
// the caller.
func planAudioOnly(
	request Request, target, profileName string,
	audio media.AudioSettings, muxer string,
	info media.Info,
) (media.Plan, error) {
	inputPath, outputPath := request.InputPath, request.OutputPath
	sourceFormat, ok := supportedAudioSource(inputPath, info.FormatNames)
	if !ok {
		return media.Plan{}, fmt.Errorf(
			"%w: ffprobe detected %q instead of wav, flac, m4a, aac, ogg, or mp3",
			ErrUnsupportedInput,
			strings.Join(info.FormatNames, ","),
		)
	}
	if len(info.AudioStreams()) == 0 {
		return media.Plan{}, fmt.Errorf("%w: no audio stream was found", ErrUnsupportedInput)
	}

	upper := strings.ToUpper(target)
	warnings := make([]string, 0)
	if len(info.VideoStreams()) > 0 {
		warnings = append(warnings, fmt.Sprintf("Video streams are not included in the %s output.", upper))
	}
	if len(info.AudioStreams()) > 1 {
		warnings = append(warnings, "Only the first audio stream will be converted.")
	}
	if len(info.SubtitleStreams()) > 0 {
		warnings = append(warnings, fmt.Sprintf("Subtitle streams are not included in the %s output.", upper))
	}
	if info.ChapterCount > 0 {
		warnings = append(warnings, fmt.Sprintf("Chapters are not included in the %s output.", upper))
	}
	if inputExtensionDoesNotMatchSource(inputPath, sourceFormat) {
		warnings = append(warnings, fmt.Sprintf("The input is detected as %s even though its file extension is %s.", strings.ToUpper(sourceFormat), strings.ToLower(filepath.Ext(inputPath))))
	}

	plan := media.Plan{
		InputPath:     inputPath,
		OutputPath:    outputPath,
		SourceFormat:  sourceFormat,
		TargetFormat:  target,
		Muxer:         muxer,
		Profile:       profileName,
		AudioMap:      "0:a:0",
		Audio:         &audio,
		CopyMetadata:  true,
		DropChapters:  true,
		Warnings:      warnings,
		InputDuration: info.Duration,
	}
	applyTrim(&plan, request)
	return plan, nil
}

// planAacM4A writes AAC into an M4A container. FFmpeg has no muxer named m4a;
// the container is written with ipod, which is why the plan carries the muxer
// name separately from the extension the user asked for.
func planAacM4A(request Request, preset string, info media.Info, capabilities media.Capabilities) (media.Plan, error) {
	if preset != "aac" {
		return media.Plan{}, fmt.Errorf("%w: %q", ErrUnsupportedPreset, preset)
	}
	if !capabilities.HasEncoder("aac") {
		return media.Plan{}, fmt.Errorf("%w: AAC encoder", ErrMissingCapability)
	}
	if !capabilities.HasMuxer("ipod") && !capabilities.HasMuxer("mp4") {
		return media.Plan{}, fmt.Errorf("%w: M4A muxer", ErrMissingCapability)
	}
	return planAudioOnly(
		request, "m4a", "aac",
		media.AudioSettings{Codec: "aac", BitRate: "192k"},
		"ipod", info,
	)
}

// planMasterWAV writes uncompressed PCM. There is no bitrate to choose: the
// sample format fixes it, so the plan leaves it empty and the adapter omits
// -b:a rather than passing a value PCM would ignore.
func planMasterWAV(request Request, preset string, info media.Info, capabilities media.Capabilities) (media.Plan, error) {
	if preset != "master" {
		return media.Plan{}, fmt.Errorf("%w: %q", ErrUnsupportedPreset, preset)
	}
	if !capabilities.HasEncoder("pcm_s16le") {
		return media.Plan{}, fmt.Errorf("%w: pcm_s16le encoder", ErrMissingCapability)
	}
	if !capabilities.HasMuxer("wav") {
		return media.Plan{}, fmt.Errorf("%w: WAV muxer", ErrMissingCapability)
	}
	return planAudioOnly(
		request, "wav", "master",
		media.AudioSettings{Codec: "pcm_s16le"},
		"", info,
	)
}

// verifyAudio is the check the audio targets share: right container, right
// codec, and a duration that did not drift.
func verifyAudio(plan media.Plan, info media.Info, label string, containers []string, codec string) error {
	matched := false
	for _, container := range containers {
		if hasFormat(info.FormatNames, container) {
			matched = true
			break
		}
	}
	if !matched {
		return fmt.Errorf("ffprobe did not detect a %s container", strings.ToUpper(label))
	}
	audios := info.AudioStreams()
	if len(audios) == 0 || audios[0].CodecName != codec {
		return fmt.Errorf("output does not contain the expected %s audio stream", codec)
	}
	return verifyDuration(plan, info)
}

// Preview defaults. A preview is meant to be glanceable and small enough to
// drop into a chat or a readme, so it is short, narrow and low frame rate.
const (
	previewWidth    = 480
	previewFPS      = 10
	previewDuration = 5 * time.Second
)

// planPreviewGIF builds a short looping GIF.
//
// A GIF holds at most 256 colors, so a good one needs a palette computed from
// the clip itself rather than a generic one. FFmpeg can do that in a single
// pass: split the stream, let palettegen read one branch and paletteuse apply
// it to the other. The two-pass form with an intermediate palette file gives a
// byte-identical result here, so the simpler graph is the one worth carrying.
func planPreviewGIF(request Request, preset string, info media.Info, capabilities media.Capabilities) (media.Plan, error) {
	inputPath, outputPath := request.InputPath, request.OutputPath
	if preset != "preview" {
		return media.Plan{}, fmt.Errorf("%w: %q", ErrUnsupportedPreset, preset)
	}
	sourceFormat, ok := supportedVideoSource(inputPath, info.FormatNames)
	if !ok {
		return media.Plan{}, fmt.Errorf(
			"%w: ffprobe detected %q instead of mp4, mov, mkv, avi, or webm",
			ErrUnsupportedInput,
			strings.Join(info.FormatNames, ","),
		)
	}

	videos := info.VideoStreams()
	if len(videos) == 0 {
		return media.Plan{}, fmt.Errorf("%w: no video stream was found", ErrUnsupportedInput)
	}
	if !capabilities.HasEncoder("gif") {
		return media.Plan{}, fmt.Errorf("%w: GIF encoder", ErrMissingCapability)
	}
	if !capabilities.HasMuxer("gif") {
		return media.Plan{}, fmt.Errorf("%w: GIF muxer", ErrMissingCapability)
	}

	// The preview has a length of its own, which an explicit --duration
	// replaces. Either way it cannot outrun what is left after the offset.
	trim := previewDuration
	if request.TrimDuration > 0 {
		trim = request.TrimDuration
	}
	if remaining := remainingAfter(info, request.TrimStart); remaining > 0 && remaining < trim {
		trim = remaining
	}

	position := "the first"
	if request.TrimStart > 0 {
		position = fmt.Sprintf("%s in,", request.TrimStart.Round(time.Millisecond))
	}
	warnings := []string{
		fmt.Sprintf("The preview is %s %s of the input, at %d pixels wide and %d frames per second.",
			position, trim.Round(time.Second), previewWidth, previewFPS),
	}
	if len(info.AudioStreams()) > 0 {
		warnings = append(warnings, "GIF has no audio; the sound is not included.")
	}
	if info.Duration > trim+request.TrimStart {
		warnings = append(warnings, "Only part of the input becomes the preview.")
	}
	if inputExtensionDoesNotMatchSource(inputPath, sourceFormat) {
		warnings = append(warnings, fmt.Sprintf("The input is detected as %s even though its file extension is %s.", strings.ToUpper(sourceFormat), strings.ToLower(filepath.Ext(inputPath))))
	}

	return media.Plan{
		InputPath:    inputPath,
		OutputPath:   outputPath,
		SourceFormat: sourceFormat,
		TargetFormat: "gif",
		Profile:      "preview",
		VideoMap:     "0:v:0",
		Video: &media.VideoSettings{
			Codec: "gif",
			// One graph, not a chain: palettegen and paletteuse need named
			// branches, which a comma-separated list cannot express.
			Filters: []string{fmt.Sprintf(
				"fps=%d,scale=%d:-1:flags=lanczos,split[s0][s1];[s0]palettegen[p];[s1][p]paletteuse",
				previewFPS, previewWidth,
			)},
		},
		TrimStart:     request.TrimStart,
		TrimDuration:  trim,
		Warnings:      warnings,
		InputDuration: info.Duration,
	}, nil
}

func verifyGIF(plan media.Plan, info media.Info) error {
	if !hasFormat(info.FormatNames, "gif") {
		return fmt.Errorf("ffprobe did not detect a GIF")
	}
	videos := info.VideoStreams()
	if len(videos) == 0 || videos[0].CodecName != "gif" {
		return fmt.Errorf("output does not contain the expected GIF stream")
	}
	if videos[0].Width <= 0 || videos[0].Height <= 0 {
		return fmt.Errorf("output has no dimensions")
	}
	return verifyDuration(plan, info)
}

func Verify(plan media.Plan, info media.Info) error {
	if info.Size <= 0 {
		return fmt.Errorf("output file is empty")
	}
	switch plan.TargetFormat {
	case "webm":
		return verifyWebM(plan, info)
	case "mp3":
		return verifyMP3(plan, info)
	case "m4a":
		return verifyAudio(plan, info, "m4a", []string{"mov", "mp4"}, "aac")
	case "wav":
		return verifyAudio(plan, info, "wav", []string{"wav"}, "pcm_s16le")
	case "gif":
		return verifyGIF(plan, info)
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
	return verifyDuration(plan, info)
}

// verifyDuration catches a conversion that stopped early: the process can exit
// zero and still leave a file that is half the length of the input.
func verifyDuration(plan media.Plan, info media.Info) error {
	expected := plan.InputDuration
	// A trimmed output is meant to be shorter than its source, so the source
	// is the wrong thing to compare it against.
	if plan.TrimDuration > 0 && plan.TrimDuration < expected {
		expected = plan.TrimDuration
	}
	if expected <= 0 {
		return nil
	}
	if info.Duration <= 0 {
		return fmt.Errorf("output duration could not be verified")
	}
	tolerance := maxDuration(2*time.Second, expected/10)
	if time.Duration(math.Abs(float64(info.Duration-expected))) > tolerance {
		return fmt.Errorf(
			"output duration %s differs unexpectedly from the expected %s",
			info.Duration.Round(time.Millisecond),
			expected.Round(time.Millisecond),
		)
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
	if found, ok := (Registry{}).Target(target); ok {
		return found.DefaultPreset
	}
	return "web"
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
