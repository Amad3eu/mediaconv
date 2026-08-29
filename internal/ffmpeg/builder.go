package ffmpeg

import (
	"strconv"
	"strings"

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
		"-i", plan.InputPath,
		"-map", plan.VideoMap,
	}

	if plan.Audio != nil {
		args = append(args, "-map", plan.AudioMap)
	}

	args = append(args,
		"-sn",
		"-dn",
		"-c:v", plan.Video.Codec,
		"-crf", strconv.Itoa(plan.Video.CRF),
		"-preset", plan.Video.Preset,
		"-pix_fmt", plan.Video.PixelFormat,
	)
	if len(plan.Video.Filters) > 0 {
		args = append(args, "-vf", strings.Join(plan.Video.Filters, ","))
	}
	if plan.Audio != nil {
		args = append(args,
			"-c:a", plan.Audio.Codec,
			"-b:a", plan.Audio.BitRate,
		)
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

	return append(args, "-f", plan.TargetFormat, temporaryOutput)
}
