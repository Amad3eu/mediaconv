package ffmpeg

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/Amad3eu/mediaconv/internal/media"
)

type CapabilityDetector struct{}

func (CapabilityDetector) Detect(ctx context.Context, paths Paths) (media.Capabilities, error) {
	ffmpegVersion, err := firstVersionLine(ctx, paths.FFmpeg)
	if err != nil {
		return media.Capabilities{}, fmt.Errorf("read ffmpeg version: %w", err)
	}
	ffprobeVersion, err := firstVersionLine(ctx, paths.FFprobe)
	if err != nil {
		return media.Capabilities{}, fmt.Errorf("read ffprobe version: %w", err)
	}
	encoders, err := listCapabilities(ctx, paths.FFmpeg, "-encoders")
	if err != nil {
		return media.Capabilities{}, fmt.Errorf("list ffmpeg encoders: %w", err)
	}
	muxers, err := listCapabilities(ctx, paths.FFmpeg, "-muxers")
	if err != nil {
		return media.Capabilities{}, fmt.Errorf("list ffmpeg muxers: %w", err)
	}

	return media.Capabilities{
		FFmpegPath:     paths.FFmpeg,
		FFprobePath:    paths.FFprobe,
		FFmpegVersion:  ffmpegVersion,
		FFprobeVersion: ffprobeVersion,
		Encoders:       encoders,
		Muxers:         muxers,
	}, nil
}

func firstVersionLine(ctx context.Context, binary string) (string, error) {
	cmd := exec.CommandContext(ctx, binary, "-version")
	cmd.WaitDelay = 3 * time.Second
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", commandOutputError(err, output)
	}
	line, _, _ := strings.Cut(strings.TrimSpace(string(output)), "\n")
	return strings.TrimSpace(line), nil
}

func listCapabilities(ctx context.Context, binary, flag string) (map[string]bool, error) {
	cmd := exec.CommandContext(ctx, binary, "-hide_banner", flag)
	cmd.WaitDelay = 3 * time.Second
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, commandOutputError(err, output)
	}

	capabilities := make(map[string]bool)
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || strings.HasPrefix(fields[0], "-") {
			continue
		}
		for _, name := range strings.Split(fields[1], ",") {
			if name = strings.TrimSpace(name); name != "" {
				capabilities[name] = true
			}
		}
	}
	return capabilities, nil
}

func commandOutputError(err error, output []byte) error {
	detail := strings.TrimSpace(string(output))
	if detail == "" {
		return err
	}
	return fmt.Errorf("%w: %s", err, detail)
}
