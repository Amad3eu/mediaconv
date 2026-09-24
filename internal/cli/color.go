package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/Amad3eu/mediaconv/internal/failure"
)

// colorMode is the resolved value of the --color flag.
type colorMode string

const (
	colorAuto   colorMode = "auto"
	colorAlways colorMode = "always"
	colorNever  colorMode = "never"
)

func parseColorMode(value string) (colorMode, error) {
	switch mode := colorMode(strings.ToLower(strings.TrimSpace(value))); mode {
	case colorAuto, colorAlways, colorNever:
		return mode, nil
	default:
		return "", failure.New(
			failure.Usage,
			fmt.Sprintf("Unsupported --color value %q.", value),
			"Use auto, always, or never.",
			nil,
		)
	}
}

const (
	ansiReset  = "\x1b[0m"
	ansiRed    = "\x1b[31m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
	ansiDim    = "\x1b[2m"
)

// palette styles short status labels. The zero value is disabled, which is
// what pipes, files, and tests get, so output stays byte for byte what it was
// before color existed.
type palette struct {
	enabled bool
}

// newPalette decides whether a stream should carry color.
//
// An explicit --color wins over everything, because it is the user saying what
// they want. Otherwise NO_COLOR (https://no-color.org) disables color when set
// to any non-empty value, a dumb terminal disables it, and what is left is
// whether the stream is a terminal at all.
//
// lookupEnv is a parameter rather than a direct os.LookupEnv call so the
// decision can be tested without mutating the process environment, which would
// force these tests to stop running in parallel.
func newPalette(writer io.Writer, mode colorMode, lookupEnv func(string) (string, bool)) palette {
	switch mode {
	case colorNever:
		return palette{}
	case colorAlways:
		return palette{enabled: true}
	}
	if value, ok := lookupEnv("NO_COLOR"); ok && value != "" {
		return palette{}
	}
	if term, _ := lookupEnv("TERM"); term == "dumb" {
		return palette{}
	}
	return palette{enabled: isTerminal(writer)}
}

func (p palette) ok(text string) string      { return p.wrap(text, ansiGreen) }
func (p palette) failed(text string) string  { return p.wrap(text, ansiRed) }
func (p palette) warning(text string) string { return p.wrap(text, ansiYellow) }
func (p palette) muted(text string) string   { return p.wrap(text, ansiDim) }

func (p palette) wrap(text, code string) string {
	if !p.enabled || text == "" {
		return text
	}
	return code + text + ansiReset
}
