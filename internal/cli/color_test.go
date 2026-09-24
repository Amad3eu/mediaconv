package cli

import (
	"bytes"
	"errors"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/Amad3eu/mediaconv/internal/app"
	"github.com/Amad3eu/mediaconv/internal/failure"
)

// noEnv is an environment with nothing set in it.
func noEnv(string) (string, bool) { return "", false }

func envWith(pairs map[string]string) func(string) (string, bool) {
	return func(name string) (string, bool) {
		value, ok := pairs[name]
		return value, ok
	}
}

func TestNewPaletteHonorsExplicitMode(t *testing.T) {
	t.Parallel()

	var writer bytes.Buffer // isTerminal only accepts *os.File, so a buffer is never a terminal

	if !newPalette(&writer, colorAlways, noEnv).enabled {
		t.Error("--color always did not enable color on a non-terminal")
	}
	if newPalette(&writer, colorNever, noEnv).enabled {
		t.Error("--color never enabled color")
	}
}

func TestNewPaletteExplicitAlwaysOverridesEnvironment(t *testing.T) {
	t.Parallel()

	var writer bytes.Buffer // isTerminal only accepts *os.File, so a buffer is never a terminal
	env := envWith(map[string]string{"NO_COLOR": "1", "TERM": "dumb"})

	if !newPalette(&writer, colorAlways, env).enabled {
		t.Error("--color always was overridden by NO_COLOR; an explicit flag is the user speaking and must win")
	}
}

func TestNewPaletteAutoRespectsEnvironment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		env  map[string]string
	}{
		{name: "NO_COLOR set to 1", env: map[string]string{"NO_COLOR": "1"}},
		{name: "NO_COLOR set to anything", env: map[string]string{"NO_COLOR": "0"}},
		{name: "dumb terminal", env: map[string]string{"TERM": "dumb"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var writer bytes.Buffer // isTerminal only accepts *os.File, so a buffer is never a terminal
			if newPalette(&writer, colorAuto, envWith(test.env)).enabled {
				t.Errorf("color stayed enabled with %v", test.env)
			}
		})
	}
}

func TestNewPaletteAutoIgnoresEmptyNoColor(t *testing.T) {
	t.Parallel()

	// https://no-color.org: the variable disables color when present and not
	// an empty string. An empty value must not disable it.
	var writer bytes.Buffer // isTerminal only accepts *os.File, so a buffer is never a terminal
	pal := newPalette(&writer, colorAuto, envWith(map[string]string{"NO_COLOR": ""}))

	// The writer is still not a terminal, so color is off either way; what
	// this asserts is that the empty value was not the reason.
	if pal.enabled {
		t.Error("unexpected: a non-terminal writer should not be colored")
	}
	if newPalette(&writer, colorAlways, envWith(map[string]string{"NO_COLOR": ""})).enabled == false {
		t.Error("an empty NO_COLOR was treated as a request to disable color")
	}
}

func TestNewPaletteAutoDisablesColorOnANonTerminal(t *testing.T) {
	t.Parallel()

	var writer bytes.Buffer // isTerminal only accepts *os.File, so a buffer is never a terminal
	if newPalette(&writer, colorAuto, noEnv).enabled {
		t.Error("color was enabled on a stream that is not a terminal")
	}
}

func TestParseColorMode(t *testing.T) {
	t.Parallel()

	valid := []struct {
		input string
		want  colorMode
	}{
		{input: "auto", want: colorAuto},
		{input: "always", want: colorAlways},
		{input: "never", want: colorNever},
		{input: "ALWAYS", want: colorAlways},
		{input: "  auto ", want: colorAuto},
	}
	for _, test := range valid {
		got, err := parseColorMode(test.input)
		if err != nil {
			t.Errorf("parseColorMode(%q) error = %v", test.input, err)
			continue
		}
		if got != test.want {
			t.Errorf("parseColorMode(%q) = %q, want %q", test.input, got, test.want)
		}
	}

	for _, input := range []string{"", "yes", "true", "16m"} {
		_, err := parseColorMode(input)
		if err == nil {
			t.Errorf("parseColorMode(%q) error = nil, want a usage failure", input)
			continue
		}
		var target *failure.Error
		if !errors.As(err, &target) || target.Kind != failure.Usage {
			t.Errorf("parseColorMode(%q) error = %v, want a usage failure", input, err)
		}
	}
}

func TestPaletteDisabledLeavesTextUntouched(t *testing.T) {
	t.Parallel()

	var pal palette // the zero value is disabled
	for _, text := range []string{"OK", "Error:", "anything"} {
		if got := pal.ok(text); got != text {
			t.Errorf("ok(%q) = %q, want the text unchanged", text, got)
		}
		if got := pal.failed(text); got != text {
			t.Errorf("failed(%q) = %q, want the text unchanged", text, got)
		}
	}
}

func TestPaletteEnabledWrapsAndAlwaysResets(t *testing.T) {
	t.Parallel()

	pal := palette{enabled: true}

	if got := pal.ok("OK"); got != ansiGreen+"OK"+ansiReset {
		t.Errorf("ok() = %q", got)
	}
	if got := pal.failed("FAIL"); got != ansiRed+"FAIL"+ansiReset {
		t.Errorf("failed() = %q", got)
	}
	if got := pal.warning("Warning:"); got != ansiYellow+"Warning:"+ansiReset {
		t.Errorf("warning() = %q", got)
	}
	if got := pal.muted("Hint:"); got != ansiDim+"Hint:"+ansiReset {
		t.Errorf("muted() = %q", got)
	}
	// Styling empty text would emit a dangling escape pair for nothing.
	if got := pal.ok(""); got != "" {
		t.Errorf("ok(\"\") = %q, want an empty string", got)
	}
}

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(text string) string { return ansiPattern.ReplaceAllString(text, "") }

// The padding has to be applied before the ANSI codes, otherwise the width
// verbs count the escape bytes and the columns go ragged the moment color is
// switched on. Stripping the codes must give back exactly the plain layout.
func TestColoredOutputKeepsColumnsAligned(t *testing.T) {
	t.Parallel()

	report := app.DoctorReport{
		OK: false,
		Checks: []app.DoctorCheck{
			{Name: "ffmpeg", OK: true, Detail: "/usr/bin/ffmpeg"},
			{Name: "libmp3lame encoder", OK: false, Detail: "missing"},
			{Name: "MP4 muxer", OK: true, Detail: "available"},
		},
	}

	var plain, colored bytes.Buffer
	if err := writeDoctorReport(&plain, report, false, palette{}); err != nil {
		t.Fatalf("writeDoctorReport(plain) error = %v", err)
	}
	if err := writeDoctorReport(&colored, report, false, palette{enabled: true}); err != nil {
		t.Fatalf("writeDoctorReport(colored) error = %v", err)
	}

	if !strings.Contains(colored.String(), ansiGreen) || !strings.Contains(colored.String(), ansiRed) {
		t.Fatal("colored doctor report carries no color")
	}
	if got := stripANSI(colored.String()); got != plain.String() {
		t.Errorf("stripping color changed the layout:\n got: %q\nwant: %q", got, plain.String())
	}
}

func TestColoredBatchOutputKeepsColumnsAligned(t *testing.T) {
	t.Parallel()

	result := app.BatchResult{
		InputDir: "/in", OutputDir: "/out", Target: "mp4",
		Total: 2, Converted: 1, Failed: 1,
		Items: []app.BatchItem{
			{InputPath: "/in/a.webm", OutputPath: "/out/a.mp4", OK: true},
			{InputPath: "/in/b.mkv", OK: false, Error: "boom"},
		},
	}

	var plain, colored bytes.Buffer
	if err := writeBatchResult(&plain, result, false, palette{}); err != nil {
		t.Fatalf("writeBatchResult(plain) error = %v", err)
	}
	if err := writeBatchResult(&colored, result, false, palette{enabled: true}); err != nil {
		t.Fatalf("writeBatchResult(colored) error = %v", err)
	}

	if got := stripANSI(colored.String()); got != plain.String() {
		t.Errorf("stripping color changed the layout:\n got: %q\nwant: %q", got, plain.String())
	}
}

func TestJSONOutputIsNeverColored(t *testing.T) {
	t.Parallel()

	report := app.DoctorReport{OK: true, Checks: []app.DoctorCheck{{Name: "ffmpeg", OK: true, Detail: "ok"}}}

	var buffer bytes.Buffer
	if err := writeDoctorReport(&buffer, report, true, palette{enabled: true}); err != nil {
		t.Fatalf("writeDoctorReport() error = %v", err)
	}
	if strings.Contains(buffer.String(), "\x1b[") {
		t.Errorf("JSON output carries ANSI codes: %q", buffer.String())
	}
}

func TestExecuteRejectsAnUnknownColorMode(t *testing.T) {
	code, _, stderr := executeForTest(t, "--color", "sometimes", "formats")

	if code != failure.ExitUsage {
		t.Errorf("Execute(--color sometimes) code = %d, want %d", code, failure.ExitUsage)
	}
	if !strings.Contains(stderr, "Unsupported --color value") {
		t.Errorf("stderr = %q, want it to name the bad value", stderr)
	}
}

func TestExecuteColorAlwaysReachesTheOutput(t *testing.T) {
	directory := t.TempDir()

	code, stdout, _ := executeForTest(
		t,
		"--color", "always",
		"--ffmpeg-path", filepath.Join(directory, "missing-ffmpeg"),
		"--ffprobe-path", filepath.Join(directory, "missing-ffprobe"),
		"doctor",
	)

	if code != failure.ExitDependency {
		t.Fatalf("Execute(doctor) code = %d, want %d", code, failure.ExitDependency)
	}
	if !strings.Contains(stdout, ansiRed) {
		t.Errorf("doctor output has no red MISSING label with --color always:\n%q", stdout)
	}
}

func TestExecuteColorNeverKeepsOutputPlain(t *testing.T) {
	directory := t.TempDir()

	code, stdout, stderr := executeForTest(
		t,
		"--color", "never",
		"--ffmpeg-path", filepath.Join(directory, "missing-ffmpeg"),
		"--ffprobe-path", filepath.Join(directory, "missing-ffprobe"),
		"doctor",
	)

	if code != failure.ExitDependency {
		t.Fatalf("Execute(doctor) code = %d, want %d", code, failure.ExitDependency)
	}
	for name, stream := range map[string]string{"stdout": stdout, "stderr": stderr} {
		if strings.Contains(stream, "\x1b[") {
			t.Errorf("%s carries ANSI codes with --color never: %q", name, stream)
		}
	}
}

// Piped output is the common case in scripts, and it must stay clean without
// anyone passing a flag.
func TestExecuteLeavesPipedOutputPlainByDefault(t *testing.T) {
	directory := t.TempDir()

	code, stdout, stderr := executeForTest(
		t,
		"--ffmpeg-path", filepath.Join(directory, "missing-ffmpeg"),
		"--ffprobe-path", filepath.Join(directory, "missing-ffprobe"),
		"doctor",
	)

	if code != failure.ExitDependency {
		t.Fatalf("Execute(doctor) code = %d, want %d", code, failure.ExitDependency)
	}
	for name, stream := range map[string]string{"stdout": stdout, "stderr": stderr} {
		if strings.Contains(stream, "\x1b[") {
			t.Errorf("%s carries ANSI codes by default: %q", name, stream)
		}
	}
}

func TestErrorOutputKeepsItsPlainTextShape(t *testing.T) {
	t.Parallel()

	err := failure.New(failure.Input, "The input file is empty.", "Choose a non-empty video file.", errors.New("stat said zero bytes"))

	var buffer bytes.Buffer
	writeError(&buffer, err, failure.ExitInput, false, true, palette{})

	want := "Error: The input file is empty.\nHint: Choose a non-empty video file.\nDetails: stat said zero bytes\n"
	if buffer.String() != want {
		t.Errorf("writeError() =\n%q\nwant\n%q", buffer.String(), want)
	}
}

func TestErrorOutputColorsTheLabelsOnly(t *testing.T) {
	t.Parallel()

	err := failure.New(failure.Input, "The input file is empty.", "Choose a non-empty video file.", nil)

	var buffer bytes.Buffer
	writeError(&buffer, err, failure.ExitInput, false, false, palette{enabled: true})

	if !strings.HasPrefix(buffer.String(), ansiRed+"Error:"+ansiReset+" ") {
		t.Errorf("writeError() = %q, want a red Error label", buffer.String())
	}
	if !strings.Contains(buffer.String(), ansiDim+"Hint:"+ansiReset+" ") {
		t.Errorf("writeError() = %q, want a dim Hint label", buffer.String())
	}
	if got := stripANSI(buffer.String()); got != "Error: The input file is empty.\nHint: Choose a non-empty video file.\n" {
		t.Errorf("stripping color changed the text: %q", got)
	}
}
