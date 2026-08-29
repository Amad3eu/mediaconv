package ffmpeg

import (
	"strconv"
	"strings"
	"time"

	"github.com/Amad3eu/mediaconv/internal/media"
)

type progressParser struct {
	values map[string]string
	total  time.Duration
}

func newProgressParser(total time.Duration) *progressParser {
	return &progressParser{values: make(map[string]string), total: total}
}

func (p *progressParser) Feed(line string) (media.Progress, bool) {
	key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
	if !ok {
		return media.Progress{}, false
	}
	p.values[key] = value
	if key != "progress" {
		return media.Progress{}, false
	}

	progress := media.Progress{
		Frame:     parseProgressInt(p.values["frame"]),
		Processed: progressDuration(p.values),
		Total:     p.total,
		Speed:     p.values["speed"],
		Done:      value == "end",
	}
	p.values = make(map[string]string)
	return progress, true
}

func progressDuration(values map[string]string) time.Duration {
	if clock := values["out_time"]; clock != "" && clock != "N/A" {
		if duration, ok := parseClock(clock); ok {
			return duration
		}
	}
	for _, key := range []string{"out_time_us", "out_time_ms"} {
		if value := parseProgressInt(values[key]); value > 0 {
			return time.Duration(value) * time.Microsecond
		}
	}
	return 0
}

func parseClock(value string) (time.Duration, bool) {
	parts := strings.Split(value, ":")
	if len(parts) != 3 {
		return 0, false
	}
	hours, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, false
	}
	minutes, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, false
	}
	seconds, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return 0, false
	}
	duration := time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute
	duration += time.Duration(seconds * float64(time.Second))
	return duration, true
}

func parseProgressInt(value string) int64 {
	number, _ := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	return number
}
