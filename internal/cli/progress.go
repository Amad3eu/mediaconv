package cli

import (
	"fmt"
	"io"
	"math"
	"os"
	"time"

	"github.com/Amad3eu/mediaconv/internal/media"
)

type progressWriter struct {
	writer     io.Writer
	enabled    bool
	visible    bool
	percentage float64
}

func newProgressWriter(writer io.Writer, requested bool) *progressWriter {
	return &progressWriter{
		writer:  writer,
		enabled: requested && isTerminal(writer),
	}
}

func (p *progressWriter) Update(progress media.Progress) {
	if !p.enabled {
		return
	}
	p.visible = true
	if progress.Done {
		_, _ = fmt.Fprint(p.writer, "\rFinalizing output...")
		return
	}

	processed := progress.Processed.Round(time.Second)
	if progress.Total > 0 {
		percentage := float64(progress.Processed) / float64(progress.Total) * 100
		percentage = math.Max(0, math.Min(100, percentage))
		if percentage < p.percentage {
			percentage = p.percentage
		}
		p.percentage = percentage
		_, _ = fmt.Fprintf(p.writer, "\rConverting... %5.1f%%  %s", percentage, progress.Speed)
		return
	}
	_, _ = fmt.Fprintf(p.writer, "\rConverting... %s  %s", processed, progress.Speed)
}

func (p *progressWriter) Clear() {
	if !p.enabled || !p.visible {
		return
	}
	_, _ = fmt.Fprint(p.writer, "\r\x1b[2K")
	p.visible = false
}

func isTerminal(writer io.Writer) bool {
	file, ok := writer.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
