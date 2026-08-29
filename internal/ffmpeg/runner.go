package ffmpeg

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/Amad3eu/mediaconv/internal/media"
)

type Runner struct {
	Binary string
}

type RunError struct {
	Err    error
	Stderr string
}

func (e *RunError) Error() string {
	if e.Stderr != "" {
		return e.Stderr
	}
	return e.Err.Error()
}

func (e *RunError) Unwrap() error { return e.Err }

func (r Runner) Run(ctx context.Context, plan media.Plan, temporaryOutput string, sink func(media.Progress)) error {
	cmd := exec.CommandContext(ctx, r.Binary, BuildArgs(plan, temporaryOutput)...)
	cmd.WaitDelay = 3 * time.Second

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("open ffmpeg progress pipe: %w", err)
	}
	stderr := newTailBuffer(64 * 1024)
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start ffmpeg: %w", err)
	}

	progressDone := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		parser := newProgressParser(plan.InputDuration)
		for scanner.Scan() {
			if progress, ok := parser.Feed(scanner.Text()); ok && sink != nil {
				sink(progress)
			}
		}
		progressDone <- scanner.Err()
	}()

	waitErr := cmd.Wait()
	scanErr := <-progressDone
	stderrText := strings.TrimSpace(stderr.String())
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if waitErr != nil {
		return &RunError{Err: waitErr, Stderr: stderrText}
	}
	if scanErr != nil && !errors.Is(scanErr, context.Canceled) {
		return fmt.Errorf("read ffmpeg progress: %w", scanErr)
	}
	// The command uses -loglevel error, so any stderr content represents a
	// recoverable or fatal media error even when FFmpeg exits with status zero.
	// Publishing such an output could silently accept a truncated input.
	if stderrText != "" {
		return &RunError{Err: errors.New("ffmpeg reported an error"), Stderr: stderrText}
	}
	return nil
}

type tailBuffer struct {
	data []byte
	max  int
}

func newTailBuffer(max int) *tailBuffer {
	return &tailBuffer{max: max}
}

func (b *tailBuffer) Write(data []byte) (int, error) {
	written := len(data)
	if len(data) >= b.max {
		b.data = append(b.data[:0], data[len(data)-b.max:]...)
		return written, nil
	}
	if overflow := len(b.data) + len(data) - b.max; overflow > 0 {
		copy(b.data, b.data[overflow:])
		b.data = b.data[:len(b.data)-overflow]
	}
	b.data = append(b.data, data...)
	return written, nil
}

func (b *tailBuffer) String() string { return string(b.data) }
