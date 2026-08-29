package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Amad3eu/mediaconv/internal/failure"
	"github.com/Amad3eu/mediaconv/internal/ffmpeg"
	"github.com/Amad3eu/mediaconv/internal/media"
	"github.com/Amad3eu/mediaconv/internal/output"
	"github.com/Amad3eu/mediaconv/internal/profile"
)

type Config struct {
	FFmpegPath  string
	FFprobePath string
}

type Service struct {
	config Config
}

func New(config Config) *Service {
	return &Service{config: config}
}

type ConvertRequest struct {
	InputPath  string
	OutputPath string
	Target     string
	Preset     string
	Overwrite  bool
}

type ConvertResult struct {
	InputPath  string        `json:"input_path"`
	OutputPath string        `json:"output_path"`
	Elapsed    time.Duration `json:"-"`
	Plan       media.Plan    `json:"plan"`
	OutputInfo media.Info    `json:"output"`
	Warnings   []string      `json:"warnings,omitempty"`
}

func (s *Service) Convert(ctx context.Context, request ConvertRequest, sink func(media.Progress)) (_ ConvertResult, returnErr error) {
	started := time.Now()
	inputPath, inputFileInfo, err := resolveInput(request.InputPath)
	if err != nil {
		return ConvertResult{}, err
	}
	outputPath, err := resolveOutput(inputPath, inputFileInfo, request.OutputPath, request.Target, request.Overwrite)
	if err != nil {
		return ConvertResult{}, err
	}

	paths, err := (ffmpeg.Locator{}).Locate(s.config.FFmpegPath, s.config.FFprobePath)
	if err != nil {
		return ConvertResult{}, failure.New(
			failure.Dependency,
			"FFmpeg and ffprobe are required but could not be located.",
			"Install FFmpeg, run 'mediaconv doctor', or provide --ffmpeg-path and --ffprobe-path.",
			err,
		)
	}
	capabilities, err := (ffmpeg.CapabilityDetector{}).Detect(ctx, paths)
	if err != nil {
		return ConvertResult{}, dependencyOrInterrupted(ctx, "Could not inspect the installed FFmpeg.", err)
	}

	prober := ffmpeg.Prober{Binary: paths.FFprobe}
	inputInfo, err := prober.Probe(ctx, inputPath)
	if err != nil {
		if ctx.Err() != nil {
			return ConvertResult{}, interrupted(ctx.Err())
		}
		return ConvertResult{}, failure.New(
			failure.Input,
			"The input could not be read as a supported media file.",
			"Check that the path points to a valid local WebM file.",
			err,
		)
	}

	plan, err := (profile.Registry{}).Plan(inputPath, outputPath, request.Target, request.Preset, inputInfo, capabilities)
	if err != nil {
		return ConvertResult{}, planFailure(err)
	}

	workspace, err := output.NewWorkspace(outputPath)
	if err != nil {
		return ConvertResult{}, failure.New(
			failure.OutputConflict,
			"Could not create a temporary file beside the output.",
			"Check that the output directory exists and is writable.",
			err,
		)
	}
	defer func() {
		if cleanupErr := workspace.Cleanup(); cleanupErr != nil && returnErr == nil {
			returnErr = failure.Wrap(failure.Conversion, "The conversion succeeded, but temporary files could not be cleaned up.", cleanupErr)
		}
	}()

	runner := ffmpeg.Runner{Binary: paths.FFmpeg}
	if err := runner.Run(ctx, plan, workspace.StagePath(), sink); err != nil {
		if ctx.Err() != nil {
			return ConvertResult{}, interrupted(ctx.Err())
		}
		return ConvertResult{}, failure.New(
			failure.Conversion,
			"FFmpeg could not complete the conversion.",
			"Run again with --verbose to see the FFmpeg diagnostic.",
			err,
		)
	}

	outputInfo, err := prober.Probe(ctx, workspace.StagePath())
	if err != nil {
		if ctx.Err() != nil {
			return ConvertResult{}, interrupted(ctx.Err())
		}
		return ConvertResult{}, failure.Wrap(failure.Conversion, "The converted file could not be verified.", err)
	}
	if err := profile.Verify(plan, outputInfo); err != nil {
		return ConvertResult{}, failure.New(
			failure.Conversion,
			"The converted file failed validation and was not published.",
			"The original input and any existing output were left unchanged.",
			err,
		)
	}

	if err := (output.Publisher{}).Publish(workspace.StagePath(), outputPath, request.Overwrite); err != nil {
		return ConvertResult{}, publishFailure(err)
	}
	outputInfo.Path = outputPath

	return ConvertResult{
		InputPath:  inputPath,
		OutputPath: outputPath,
		Elapsed:    time.Since(started),
		Plan:       plan,
		OutputInfo: outputInfo,
		Warnings:   plan.Warnings,
	}, nil
}

func (s *Service) Inspect(ctx context.Context, input string) (media.Info, error) {
	path, _, err := resolveInput(input)
	if err != nil {
		return media.Info{}, err
	}
	ffprobePath, err := (ffmpeg.Locator{}).LocateFFprobe(s.config.FFprobePath)
	if err != nil {
		return media.Info{}, failure.New(
			failure.Dependency,
			"ffprobe is required but could not be located.",
			"Install FFmpeg, run 'mediaconv doctor', or provide --ffprobe-path.",
			err,
		)
	}
	info, err := (ffmpeg.Prober{Binary: ffprobePath}).Probe(ctx, path)
	if err != nil {
		if ctx.Err() != nil {
			return media.Info{}, interrupted(ctx.Err())
		}
		return media.Info{}, failure.Wrap(failure.Input, "The media file could not be inspected.", err)
	}
	return info, nil
}

type DoctorCheck struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

type DoctorReport struct {
	OK     bool          `json:"ok"`
	Checks []DoctorCheck `json:"checks"`
}

func (s *Service) Doctor(ctx context.Context) DoctorReport {
	report := DoctorReport{OK: true, Checks: make([]DoctorCheck, 0, 5)}
	locator := ffmpeg.Locator{}
	ffmpegPath, ffmpegErr := locator.LocateFFmpeg(s.config.FFmpegPath)
	report.add("ffmpeg", ffmpegErr == nil, chooseDetail(ffmpegPath, ffmpegErr))
	var ffprobePath string
	var ffprobeErr error
	if ffmpegErr == nil {
		paths, pairErr := locator.Locate(s.config.FFmpegPath, s.config.FFprobePath)
		if pairErr == nil {
			ffprobePath = paths.FFprobe
		} else {
			ffprobeErr = pairErr
		}
	} else {
		ffprobePath, ffprobeErr = locator.LocateFFprobe(s.config.FFprobePath)
	}
	report.add("ffprobe", ffprobeErr == nil, chooseDetail(ffprobePath, ffprobeErr))
	if ffmpegErr != nil || ffprobeErr != nil {
		return report
	}

	capabilities, err := (ffmpeg.CapabilityDetector{}).Detect(ctx, ffmpeg.Paths{FFmpeg: ffmpegPath, FFprobe: ffprobePath})
	if err != nil {
		report.add("capabilities", false, err.Error())
		return report
	}
	report.Checks[0].Detail = capabilities.FFmpegVersion + " (" + capabilities.FFmpegPath + ")"
	report.Checks[1].Detail = capabilities.FFprobeVersion + " (" + capabilities.FFprobePath + ")"
	report.add("libx264 encoder", capabilities.HasEncoder("libx264"), capabilityDetail(capabilities.HasEncoder("libx264")))
	report.add("AAC encoder", capabilities.HasEncoder("aac"), capabilityDetail(capabilities.HasEncoder("aac")))
	hasMP4 := capabilities.HasMuxer("mp4") || capabilities.HasMuxer("mov")
	report.add("MP4 muxer", hasMP4, capabilityDetail(hasMP4))
	return report
}

func (s *Service) Formats() []profile.SupportedFormat {
	return (profile.Registry{}).Formats()
}

func (r *DoctorReport) add(name string, ok bool, detail string) {
	r.Checks = append(r.Checks, DoctorCheck{Name: name, OK: ok, Detail: detail})
	if !ok {
		r.OK = false
	}
}

func chooseDetail(path string, err error) string {
	if err != nil {
		return err.Error()
	}
	return path
}

func capabilityDetail(ok bool) string {
	if ok {
		return "available"
	}
	return "missing"
}

func resolveInput(input string) (string, os.FileInfo, error) {
	if strings.TrimSpace(input) == "" {
		return "", nil, failure.New(failure.Usage, "An input path is required.", "Run 'mediaconv convert --help' for examples.", nil)
	}
	path, err := filepath.Abs(input)
	if err != nil {
		return "", nil, failure.Wrap(failure.Input, "The input path is invalid.", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", nil, failure.New(failure.Input, "The input file does not exist or cannot be accessed.", "Check the path and file permissions.", err)
	}
	if !info.Mode().IsRegular() {
		return "", nil, failure.New(failure.Input, "The input must be a regular local file.", "Directories, devices, pipes, and URLs are not supported.", nil)
	}
	if info.Size() == 0 {
		return "", nil, failure.New(failure.Input, "The input file is empty.", "Choose a non-empty WebM file.", nil)
	}
	file, err := os.Open(path)
	if err != nil {
		return "", nil, failure.New(failure.Input, "The input file is not readable.", "Check the file permissions.", err)
	}
	if err := file.Close(); err != nil {
		return "", nil, failure.Wrap(failure.Input, "The input file could not be closed after validation.", err)
	}
	return filepath.Clean(path), info, nil
}

func resolveOutput(inputPath string, inputInfo os.FileInfo, requested, target string, overwrite bool) (string, error) {
	target = strings.ToLower(strings.TrimSpace(target))
	if target == "" {
		target = "mp4"
	}
	if target != "mp4" {
		return "", failure.New(failure.Usage, fmt.Sprintf("Unsupported target format %q.", target), "Run 'mediaconv formats' to list supported conversions.", nil)
	}

	outputPath := requested
	if strings.TrimSpace(outputPath) == "" {
		extension := filepath.Ext(inputPath)
		outputPath = strings.TrimSuffix(inputPath, extension) + "." + target
	}
	absolute, err := filepath.Abs(outputPath)
	if err != nil {
		return "", failure.Wrap(failure.OutputConflict, "The output path is invalid.", err)
	}
	absolute = filepath.Clean(absolute)
	if !strings.EqualFold(filepath.Ext(absolute), "."+target) {
		return "", failure.New(failure.Usage, "The output extension does not match --to mp4.", "Use an output path ending in .mp4.", nil)
	}
	if samePath(inputPath, absolute) {
		return "", failure.New(failure.OutputConflict, "The input and output paths must be different.", "Choose a different --output path.", nil)
	}

	parentInfo, err := os.Stat(filepath.Dir(absolute))
	if err != nil || !parentInfo.IsDir() {
		return "", failure.New(failure.OutputConflict, "The output directory does not exist or cannot be accessed.", "Create the directory before converting.", err)
	}
	if existing, err := os.Lstat(absolute); err == nil {
		if existing.Mode()&os.ModeSymlink != 0 {
			return "", failure.New(failure.OutputConflict, "The output path is a symbolic link.", "Choose a regular output path; symlink outputs are rejected for safety.", output.ErrSymlink)
		}
		if !existing.Mode().IsRegular() {
			return "", failure.New(failure.OutputConflict, "The output path exists and is not a regular file.", "Choose another output path.", nil)
		}
		if os.SameFile(inputInfo, existing) {
			return "", failure.New(failure.OutputConflict, "The input and output refer to the same file.", "Choose a different --output path.", nil)
		}
		if !overwrite {
			return "", failure.New(failure.OutputConflict, "The output file already exists.", "Pass --overwrite to replace it explicitly.", output.ErrExists)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", failure.Wrap(failure.OutputConflict, "The output path cannot be inspected.", err)
	}
	return absolute, nil
}

func samePath(left, right string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(filepath.Clean(left), filepath.Clean(right))
	}
	return filepath.Clean(left) == filepath.Clean(right)
}

func dependencyOrInterrupted(ctx context.Context, message string, err error) error {
	if ctx.Err() != nil {
		return interrupted(ctx.Err())
	}
	return failure.New(failure.Dependency, message, "Run 'mediaconv doctor' to see which capabilities are missing.", err)
}

func interrupted(err error) error {
	return failure.New(failure.Interrupted, "The operation was interrupted.", "No partial output was published.", err)
}

func planFailure(err error) error {
	switch {
	case errors.Is(err, profile.ErrMissingCapability):
		return failure.New(failure.Dependency, "The installed FFmpeg does not provide a required codec or muxer.", "Run 'mediaconv doctor' and install an FFmpeg build with libx264, AAC, and MP4 support.", err)
	case errors.Is(err, profile.ErrUnsupportedTarget), errors.Is(err, profile.ErrUnsupportedPreset):
		return failure.New(failure.Usage, err.Error(), "Run 'mediaconv formats' to list supported conversions and profiles.", err)
	default:
		return failure.New(failure.Input, "The input is not supported by the selected conversion profile.", "The initial profile accepts local WebM files containing at least one video stream.", err)
	}
}

func publishFailure(err error) error {
	switch {
	case errors.Is(err, output.ErrExists):
		return failure.New(failure.OutputConflict, "The output file was created by another process before publication.", "Choose another path or pass --overwrite.", err)
	case errors.Is(err, output.ErrSymlink):
		return failure.New(failure.OutputConflict, "The output path became a symbolic link and was not replaced.", "Choose a regular output path.", err)
	default:
		return failure.New(failure.OutputConflict, "The verified conversion could not be published.", "Check output directory permissions and filesystem support for hard links.", err)
	}
}
