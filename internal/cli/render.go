package cli

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Amad3eu/mediaconv/internal/app"
	"github.com/Amad3eu/mediaconv/internal/media"
	"github.com/Amad3eu/mediaconv/internal/profile"
)

func writeConvertResult(writer io.Writer, result app.ConvertResult, asJSON bool) error {
	if asJSON {
		return writeJSON(writer, map[string]any{
			"ok":              true,
			"input_path":      result.InputPath,
			"output_path":     result.OutputPath,
			"elapsed_seconds": seconds(result.Elapsed),
			"warnings":        result.Warnings,
			"plan":            result.Plan,
			"output":          mediaInfoView(result.OutputInfo),
		})
	}

	if _, err := fmt.Fprintf(writer, "Converted: %s\n", result.OutputPath); err != nil {
		return err
	}
	for _, warning := range result.Warnings {
		if _, err := fmt.Fprintf(writer, "Warning: %s\n", warning); err != nil {
			return err
		}
	}
	return nil
}

func writeMediaInfo(writer io.Writer, info media.Info, asJSON bool) error {
	if asJSON {
		return writeJSON(writer, map[string]any{"ok": true, "media": mediaInfoView(info)})
	}

	if _, err := fmt.Fprintf(writer, "File: %s\n", info.Path); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "Format: %s\n", strings.Join(info.FormatNames, ", ")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "Duration: %s\n", humanDuration(info.Duration)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "Size: %d bytes\n", info.Size); err != nil {
		return err
	}
	for _, stream := range info.Streams {
		if _, err := fmt.Fprintf(writer, "Stream #%d: %s %s", stream.Index, stream.CodecType, stream.CodecName); err != nil {
			return err
		}
		if stream.CodecType == "video" {
			if _, err := fmt.Fprintf(writer, " %dx%d", stream.Width, stream.Height); err != nil {
				return err
			}
		}
		if stream.CodecType == "audio" && stream.Channels > 0 {
			if _, err := fmt.Fprintf(writer, " %d channel(s)", stream.Channels); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(writer); err != nil {
			return err
		}
	}
	return nil
}

func writeDoctorReport(writer io.Writer, report app.DoctorReport, asJSON bool) error {
	if asJSON {
		return writeJSON(writer, report)
	}
	for _, check := range report.Checks {
		status := "OK"
		if !check.OK {
			status = "MISSING"
		}
		if _, err := fmt.Fprintf(writer, "%-7s %-18s %s\n", status, check.Name, check.Detail); err != nil {
			return err
		}
	}
	return nil
}

func writeFormats(writer io.Writer, formats []profile.SupportedFormat, asJSON bool) error {
	if asJSON {
		return writeJSON(writer, map[string]any{"ok": true, "formats": formats})
	}
	for _, format := range formats {
		if _, err := fmt.Fprintf(
			writer,
			"%s -> %s  profile=%s  video=%s  audio=%s\n",
			strings.ToUpper(format.Source),
			strings.ToUpper(format.Target),
			format.Profile,
			format.VideoCodec,
			format.AudioCodec,
		); err != nil {
			return err
		}
	}
	return nil
}

func mediaInfoView(info media.Info) map[string]any {
	streams := make([]map[string]any, 0, len(info.Streams))
	for _, stream := range info.Streams {
		streams = append(streams, map[string]any{
			"index":            stream.Index,
			"codec_type":       stream.CodecType,
			"codec_name":       stream.CodecName,
			"codec_long_name":  stream.CodecLongName,
			"profile":          stream.Profile,
			"pixel_format":     stream.PixelFormat,
			"width":            stream.Width,
			"height":           stream.Height,
			"channels":         stream.Channels,
			"channel_layout":   stream.ChannelLayout,
			"sample_rate":      stream.SampleRate,
			"duration_seconds": seconds(stream.Duration),
			"tags":             stream.Tags,
		})
	}
	return map[string]any{
		"path":             info.Path,
		"format_names":     info.FormatNames,
		"format_long_name": info.FormatLongName,
		"duration_seconds": seconds(info.Duration),
		"size_bytes":       info.Size,
		"bit_rate":         info.BitRate,
		"chapter_count":    info.ChapterCount,
		"streams":          streams,
		"tags":             info.Tags,
	}
}

func seconds(duration time.Duration) float64 {
	return float64(duration) / float64(time.Second)
}

func humanDuration(duration time.Duration) string {
	if duration <= 0 {
		return "unknown"
	}
	return duration.Round(time.Millisecond).String()
}
