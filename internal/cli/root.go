package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Amad3eu/mediaconv/internal/app"
	"github.com/Amad3eu/mediaconv/internal/buildinfo"
	"github.com/Amad3eu/mediaconv/internal/failure"
)

type options struct {
	ffmpegPath  string
	ffprobePath string
	json        bool
	verbose     bool
	color       string
	colorMode   colorMode
}

// paletteFor resolves color per stream, because stdout and stderr are not
// always the same kind of destination: `mediaconv convert x.webm > out.txt`
// still has a terminal on stderr.
func (o *options) paletteFor(writer io.Writer) palette {
	return newPalette(writer, o.colorMode, os.LookupEnv)
}

func Execute(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	opts := &options{colorMode: colorAuto}
	root := newRootCommand(opts, stdin, stdout, stderr)
	root.SetArgs(args)
	root.SetContext(ctx)

	err := root.Execute()
	if err == nil {
		return 0
	}
	code := failure.ExitCode(err)
	if !failure.IsReported(err) {
		writeError(stderr, err, code, opts.json, opts.verbose, opts.paletteFor(stderr))
	}
	return code
}

func newRootCommand(opts *options, stdin io.Reader, stdout, stderr io.Writer) *cobra.Command {
	info := buildinfo.Current()
	root := &cobra.Command{
		Use:           "mediaconv",
		Short:         "Convert media files safely with FFmpeg",
		Long:          "MediaConv is a script-friendly media conversion CLI. Its web profile converts common video containers to broadly compatible MP4.",
		SilenceErrors: true,
		SilenceUsage:  true,
		Version:       info.Version,
	}
	root.SetIn(stdin)
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetVersionTemplate("mediaconv {{.Version}}\n")
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return failure.New(failure.Usage, err.Error(), "Run 'mediaconv --help' to see available options.", err)
	})

	flags := root.PersistentFlags()
	flags.StringVar(&opts.ffmpegPath, "ffmpeg-path", "", "Path to the ffmpeg executable or its directory")
	flags.StringVar(&opts.ffprobePath, "ffprobe-path", "", "Path to the ffprobe executable or its directory")
	flags.BoolVar(&opts.json, "json", false, "Write machine-readable JSON")
	flags.BoolVarP(&opts.verbose, "verbose", "v", false, "Include underlying diagnostic details")
	flags.StringVar(&opts.color, "color", string(colorAuto), "Colorize output: auto, always, or never")

	root.PersistentPreRunE = func(_ *cobra.Command, _ []string) error {
		mode, err := parseColorMode(opts.color)
		if err != nil {
			return err
		}
		opts.colorMode = mode
		return nil
	}

	root.AddCommand(
		newConvertCommand(opts, stdout, stderr),
		newBatchCommand(opts, stdout),
		newInspectCommand(opts, stdout),
		newDoctorCommand(opts, stdout),
		newFormatsCommand(opts, stdout),
		newVersionCommand(opts, stdout),
	)
	root.AddCommand(newCompletionCommand(root, stdout))
	return root
}

func newConvertCommand(opts *options, stdout, stderr io.Writer) *cobra.Command {
	var (
		outputPath string
		target     string
		preset     string
		overwrite  bool
		noProgress bool
	)
	command := &cobra.Command{
		Use:   "convert INPUT",
		Short: "Convert a media file",
		Args:  exactArgs(1),
		Example: strings.TrimSpace(`
  mediaconv convert recording.webm
  mediaconv convert clip.mov --output clip.mp4
  mediaconv convert song.wav --to mp3
  mediaconv convert archive.mkv --to mp4 --preset web --overwrite`),
		RunE: func(command *cobra.Command, args []string) error {
			service := app.New(app.Config{FFmpegPath: opts.ffmpegPath, FFprobePath: opts.ffprobePath})
			progress := newProgressWriter(stderr, !noProgress && !opts.json)
			defer progress.Clear()
			result, err := service.Convert(command.Context(), app.ConvertRequest{
				InputPath:  args[0],
				OutputPath: outputPath,
				Target:     target,
				Preset:     preset,
				Overwrite:  overwrite,
			}, progress.Update)
			if err != nil {
				return err
			}
			progress.Clear()
			return writeConvertResult(stdout, result, opts.json, opts.paletteFor(stdout))
		},
	}
	flags := command.Flags()
	flags.StringVarP(&outputPath, "output", "o", "", "Output path (default: INPUT with an .mp4 extension)")
	flags.StringVar(&target, "to", "mp4", "Target format")
	flags.StringVar(&preset, "preset", "", "Conversion profile (default: web for MP4, music for MP3)")
	flags.BoolVar(&overwrite, "overwrite", false, "Replace an existing regular output file")
	flags.BoolVar(&noProgress, "no-progress", false, "Disable interactive progress output")
	return command
}

func newBatchCommand(opts *options, stdout io.Writer) *cobra.Command {
	var (
		outputDir string
		target    string
		preset    string
		overwrite bool
		recursive bool
	)
	command := &cobra.Command{
		Use:   "batch DIRECTORY",
		Short: "Convert supported files in a directory",
		Args:  exactArgs(1),
		Example: strings.Join([]string{
			"mediaconv batch ./recordings --to mp4",
			"mediaconv batch ./audio --to mp3 --output-dir ./converted",
			"mediaconv batch ./media --to mp4 --recursive --overwrite",
		}, "\n"),
		RunE: func(command *cobra.Command, args []string) error {
			service := app.New(app.Config{FFmpegPath: opts.ffmpegPath, FFprobePath: opts.ffprobePath})
			result, err := service.BatchConvert(command.Context(), app.BatchRequest{
				InputDir:  args[0],
				OutputDir: outputDir,
				Target:    target,
				Preset:    preset,
				Overwrite: overwrite,
				Recursive: recursive,
			})
			if err != nil && result.Total == 0 {
				return err
			}
			if writeErr := writeBatchResult(stdout, result, opts.json, opts.paletteFor(stdout)); writeErr != nil {
				return writeErr
			}
			if err != nil {
				return err
			}
			if result.Failed > 0 {
				return failure.Reported(failure.New(
					failure.Conversion,
					"One or more files failed during batch conversion.",
					"Review the batch summary and rerun failed files with --verbose if needed.",
					nil,
				))
			}
			return nil
		},
	}
	flags := command.Flags()
	flags.StringVarP(&outputDir, "output-dir", "o", "", "Output directory (default: input directory)")
	flags.StringVar(&target, "to", "mp4", "Target format")
	flags.StringVar(&preset, "preset", "", "Conversion profile (default: web for MP4, music for MP3)")
	flags.BoolVar(&overwrite, "overwrite", false, "Replace existing regular output files")
	flags.BoolVarP(&recursive, "recursive", "r", false, "Scan subdirectories recursively")
	return command
}

func newInspectCommand(opts *options, stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "inspect INPUT",
		Short: "Inspect a local media file with ffprobe",
		Args:  exactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			service := app.New(app.Config{FFmpegPath: opts.ffmpegPath, FFprobePath: opts.ffprobePath})
			info, err := service.Inspect(command.Context(), args[0])
			if err != nil {
				return err
			}
			return writeMediaInfo(stdout, info, opts.json)
		},
	}
}

func newDoctorCommand(opts *options, stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check FFmpeg and the codecs required by MediaConv",
		Args:  exactArgs(0),
		RunE: func(command *cobra.Command, _ []string) error {
			service := app.New(app.Config{FFmpegPath: opts.ffmpegPath, FFprobePath: opts.ffprobePath})
			report := service.Doctor(command.Context())
			if err := writeDoctorReport(stdout, report, opts.json, opts.paletteFor(stdout)); err != nil {
				return failure.Wrap(failure.Unexpected, "Could not write the doctor report.", err)
			}
			if !report.OK {
				return failure.Reported(failure.New(
					failure.Dependency,
					"One or more required FFmpeg capabilities are missing.",
					"Install a build with ffmpeg, ffprobe, libx264, AAC, and MP4 support.",
					nil,
				))
			}
			return nil
		},
	}
}

func newFormatsCommand(opts *options, stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "formats",
		Short: "List conversion profiles supported by MediaConv",
		Args:  exactArgs(0),
		RunE: func(_ *cobra.Command, _ []string) error {
			service := app.New(app.Config{})
			return writeFormats(stdout, service.Formats(), opts.json)
		},
	}
}

func newVersionCommand(opts *options, stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version and build information",
		Args:  exactArgs(0),
		RunE: func(_ *cobra.Command, _ []string) error {
			info := buildinfo.Current()
			if opts.json {
				return writeJSON(stdout, map[string]any{"ok": true, "build": info})
			}
			_, err := fmt.Fprintf(stdout, "mediaconv %s\ncommit: %s\nbuilt: %s\n", info.Version, info.Commit, info.Date)
			return err
		},
	}
}

func newCompletionCommand(root *cobra.Command, stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:       "completion [bash|zsh|fish|powershell]",
		Short:     "Generate a shell completion script",
		Args:      exactArgs(1),
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		RunE: func(_ *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return root.GenBashCompletion(stdout)
			case "zsh":
				return root.GenZshCompletion(stdout)
			case "fish":
				return root.GenFishCompletion(stdout, true)
			case "powershell":
				return root.GenPowerShellCompletionWithDesc(stdout)
			default:
				return failure.New(failure.Usage, fmt.Sprintf("Unsupported shell %q.", args[0]), "Choose bash, zsh, fish, or powershell.", nil)
			}
		},
	}
}

func exactArgs(expected int) cobra.PositionalArgs {
	return func(command *cobra.Command, args []string) error {
		if len(args) != expected {
			return failure.New(
				failure.Usage,
				fmt.Sprintf("%s expects %d argument(s), received %d.", command.CommandPath(), expected, len(args)),
				fmt.Sprintf("Run '%s --help' for usage.", command.CommandPath()),
				nil,
			)
		}
		return nil
	}
}

func writeError(writer io.Writer, err error, code int, asJSON, verbose bool, pal palette) {
	message, hint := failure.Details(err)
	if asJSON {
		_ = writeJSON(writer, map[string]any{
			"ok": false,
			"error": map[string]any{
				"message":   message,
				"hint":      hint,
				"exit_code": code,
			},
		})
		return
	}
	_, _ = fmt.Fprintf(writer, "%s %s\n", pal.failed("Error:"), message)
	if hint != "" {
		_, _ = fmt.Fprintf(writer, "%s %s\n", pal.muted("Hint:"), hint)
	}
	if verbose {
		if detail := failure.Detail(err); detail != "" {
			_, _ = fmt.Fprintf(writer, "%s %s\n", pal.muted("Details:"), detail)
		}
	}
}

func writeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
