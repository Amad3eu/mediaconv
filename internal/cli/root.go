package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
}

func Execute(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	opts := &options{}
	root := newRootCommand(opts, stdin, stdout, stderr)
	root.SetArgs(args)
	root.SetContext(ctx)

	err := root.Execute()
	if err == nil {
		return 0
	}
	code := failure.ExitCode(err)
	if !failure.IsReported(err) {
		writeError(stderr, err, code, opts.json, opts.verbose)
	}
	return code
}

func newRootCommand(opts *options, stdin io.Reader, stdout, stderr io.Writer) *cobra.Command {
	info := buildinfo.Current()
	root := &cobra.Command{
		Use:           "mediaconv",
		Short:         "Convert media files safely with FFmpeg",
		Long:          "MediaConv is a script-friendly media conversion CLI. Its first profile converts WebM video to broadly compatible MP4.",
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

	root.AddCommand(
		newConvertCommand(opts, stdout, stderr),
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
  mediaconv convert recording.webm --output recording.mp4
  mediaconv convert recording.webm --to mp4 --preset web --overwrite`),
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
			return writeConvertResult(stdout, result, opts.json)
		},
	}
	flags := command.Flags()
	flags.StringVarP(&outputPath, "output", "o", "", "Output path (default: INPUT with an .mp4 extension)")
	flags.StringVar(&target, "to", "mp4", "Target format")
	flags.StringVar(&preset, "preset", "web", "Conversion profile")
	flags.BoolVar(&overwrite, "overwrite", false, "Replace an existing regular output file")
	flags.BoolVar(&noProgress, "no-progress", false, "Disable interactive progress output")
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
			if err := writeDoctorReport(stdout, report, opts.json); err != nil {
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

func writeError(writer io.Writer, err error, code int, asJSON, verbose bool) {
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
	_, _ = fmt.Fprintf(writer, "Error: %s\n", failure.Format(err, verbose))
}

func writeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
