package ffmpeg

import (
	"strconv"
	"strings"
	"time"

	"github.com/Amad3eu/mediaconv/internal/media"
)

func BuildArgs(plan media.Plan, temporaryOutput string) []string {
	args := []string{
		"-hide_banner",
		"-nostdin",
		"-loglevel", "error",
		"-nostats",
		"-stats_period", "0.5",
		"-progress", "pipe:1",
		"-n",
		"-protocol_whitelist", "file",
	}

	// -ss before -i seeks by jumping rather than decoding, which matters when
	// a preview starts deep into a long file.
	if plan.TrimStart > 0 {
		args = append(args, "-ss", seconds(plan.TrimStart))
	}
	args = append(args, "-i", plan.InputPath)
	if plan.TrimDuration > 0 {
		args = append(args, "-t", seconds(plan.TrimDuration))
	}

	if plan.Video != nil {
		args = append(args, "-map", plan.VideoMap)
	}
	if plan.Audio != nil {
		args = append(args, "-map", plan.AudioMap)
	}

	args = append(args, "-sn", "-dn")
	if plan.Video != nil {
		args = append(args, "-c:v", plan.Video.Codec)
		args = append(args, videoQualityArgs(plan.Video)...)
		// GIF carries its palette in the file, so forcing a pixel format on it
		// would fight paletteuse.
		if plan.Video.PixelFormat != "" {
			args = append(args, "-pix_fmt", plan.Video.PixelFormat)
		}
		if len(plan.Video.Filters) > 0 {
			args = append(args, "-vf", strings.Join(plan.Video.Filters, ","))
		}
	} else {
		args = append(args, "-vn")
	}
	if plan.Audio != nil {
		args = append(args, "-c:a", plan.Audio.Codec)
		// PCM has no bitrate to set: the sample format already fixes it, and
		// passing -b:a to it is meaningless.
		if plan.Audio.BitRate != "" {
			args = append(args, "-b:a", plan.Audio.BitRate)
		}
	}
	if plan.CopyMetadata {
		args = append(args, "-map_metadata", "0")
	}
	if plan.DropChapters {
		args = append(args, "-map_chapters", "-1")
	}
	if len(plan.MovFlags) > 0 {
		args = append(args, "-movflags", "+"+strings.Join(plan.MovFlags, "+"))
	}

	return append(args, "-f", outputFormat(plan), temporaryOutput)
}

// outputFormat is the name FFmpeg knows the container by, which is not always
// the extension the user asked for.
func outputFormat(plan media.Plan) string {
	if plan.Muxer != "" {
		return plan.Muxer
	}
	return plan.TargetFormat
}

// seconds formats a duration the way FFmpeg reads -ss and -t.
func seconds(d time.Duration) string {
	return strconv.FormatFloat(d.Seconds(), 'f', 3, 64)
}

// videoQualityArgs maps the plan's quality intent onto the flags the chosen
// encoder actually understands. The profile says "constant quality at this CRF,
// at this speed"; translating that is the adapter's job, per ADR 0003.
//
// libx264 reads -crf on its own and calls its speed control -preset. libvpx-vp9
// treats -crf as a ceiling unless -b:v 0 puts it in constant quality mode, calls
// the same control -deadline, and stays single threaded without -row-mt.
func videoQualityArgs(video *media.VideoSettings) []string {
	switch video.Codec {
	case "gif":
		// The GIF encoder takes neither a CRF nor a speed preset: quality is
		// decided by the palette and the frame rate, both set as filters.
		return nil
	case "libvpx-vp9":
		return []string{
			"-b:v", "0",
			"-crf", strconv.Itoa(video.CRF),
			"-deadline", video.Preset,
			"-row-mt", "1",
		}
	default:
		return []string{
			"-crf", strconv.Itoa(video.CRF),
			"-preset", video.Preset,
		}
	}
}
