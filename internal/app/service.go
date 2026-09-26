package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
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

	ffmpegOnce   sync.Once
	ffmpegPaths  ffmpeg.Paths
	capabilities media.Capabilities
	ffmpegErr    error
}

func New(config Config) *Service {
	return &Service{config: config}
}

// resolveFFmpeg locates FFmpeg and reads its capabilities once per service.
//
// Detection costs four subprocesses, and a batch converts every file with the
// same FFmpeg, so repeating it per file dominated the runtime for folders of
// small inputs. The result is cached for the life of the service, which is one
// command invocation: the installed FFmpeg does not change underneath a
// running conversion.
func (s *Service) resolveFFmpeg(ctx context.Context) (ffmpeg.Paths, media.Capabilities, error) {
	s.ffmpegOnce.Do(func() {
		paths, err := (ffmpeg.Locator{}).Locate(s.config.FFmpegPath, s.config.FFprobePath)
		if err != nil {
			s.ffmpegErr = failure.New(
				failure.Dependency,
				"FFmpeg and ffprobe are required but could not be located.",
				"Install FFmpeg, run 'mediaconv doctor', or provide --ffmpeg-path and --ffprobe-path.",
				err,
			)
			return
		}
		capabilities, err := (ffmpeg.CapabilityDetector{}).Detect(ctx, paths)
		if err != nil {
			s.ffmpegErr = dependencyOrInterrupted(ctx, "Could not inspect the installed FFmpeg.", err)
			return
		}
		s.ffmpegPaths = paths
		s.capabilities = capabilities
	})
	return s.ffmpegPaths, s.capabilities, s.ffmpegErr
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

type BatchRequest struct {
	InputDir  string
	OutputDir string
	Target    string
	Preset    string
	Overwrite bool
	Recursive bool
	// Jobs is how many files to convert at once. Zero means sequential.
	Jobs int
}

type BatchItem struct {
	InputPath  string   `json:"input_path"`
	OutputPath string   `json:"output_path,omitempty"`
	OK         bool     `json:"ok"`
	Error      string   `json:"error,omitempty"`
	Warnings   []string `json:"warnings,omitempty"`
}

type BatchResult struct {
	InputDir  string        `json:"input_dir"`
	OutputDir string        `json:"output_dir"`
	Target    string        `json:"target"`
	Preset    string        `json:"preset,omitempty"`
	Recursive bool          `json:"recursive"`
	Jobs      int           `json:"jobs"`
	Total     int           `json:"total"`
	Converted int           `json:"converted"`
	Failed    int           `json:"failed"`
	Items     []BatchItem   `json:"items"`
	Elapsed   time.Duration `json:"-"`
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

	paths, capabilities, err := s.resolveFFmpeg(ctx)
	if err != nil {
		return ConvertResult{}, err
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
			"Check that the path points to a valid local video file.",
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

func (s *Service) BatchConvert(ctx context.Context, request BatchRequest) (BatchResult, error) {
	started := time.Now()
	inputDir, err := resolveInputDir(request.InputDir)
	if err != nil {
		return BatchResult{}, err
	}
	target, err := normalizeTarget(request.Target)
	if err != nil {
		return BatchResult{}, err
	}
	outputDir, err := resolveBatchOutputDir(inputDir, request.OutputDir)
	if err != nil {
		return BatchResult{}, err
	}

	candidates, err := collectBatchCandidates(inputDir, target, request.Recursive)
	if err != nil {
		return BatchResult{}, err
	}
	if len(candidates) == 0 {
		return BatchResult{}, failure.New(
			failure.Input,
			"No supported input files were found.",
			"Run 'mediaconv formats' to list supported batch inputs.",
			nil,
		)
	}

	jobs, err := normalizeJobs(request.Jobs, len(candidates))
	if err != nil {
		return BatchResult{}, err
	}

	result := BatchResult{
		InputDir:  inputDir,
		OutputDir: outputDir,
		Target:    target,
		Preset:    request.Preset,
		Recursive: request.Recursive,
		Jobs:      jobs,
		Total:     len(candidates),
		Items:     make([]BatchItem, 0, len(candidates)),
	}

	// Each worker writes to its own index, so the slices need no lock and the
	// collection below can read them once every worker has finished.
	items := make([]BatchItem, len(candidates))
	converted := make([]bool, len(candidates))

	queue := make(chan int)
	var group sync.WaitGroup
	for range jobs {
		group.Add(1)
		go func() {
			defer group.Done()
			for index := range queue {
				if ctx.Err() != nil {
					return
				}
				items[index] = s.convertBatchItem(ctx, inputDir, outputDir, candidates[index], target, request)
				converted[index] = true
			}
		}()
	}

feed:
	for index := range candidates {
		select {
		case queue <- index:
		case <-ctx.Done():
			break feed
		}
	}
	close(queue)
	group.Wait()

	// Collected in candidate order rather than completion order, so a
	// concurrent run reports exactly what a sequential one would.
	for index := range candidates {
		if !converted[index] {
			continue
		}
		result.Items = append(result.Items, items[index])
		if items[index].OK {
			result.Converted++
		} else {
			result.Failed++
		}
	}

	result.Elapsed = time.Since(started)
	if err := ctx.Err(); err != nil {
		return result, interrupted(err)
	}
	return result, nil
}

// convertBatchItem converts one file and never returns an error: a batch
// reports per-file outcomes and keeps going.
func (s *Service) convertBatchItem(ctx context.Context, inputDir, outputDir, inputPath, target string, request BatchRequest) BatchItem {
	outputPath, err := batchOutputPath(inputDir, outputDir, inputPath, target)
	if err != nil {
		return BatchItem{InputPath: inputPath, OK: false, Error: err.Error()}
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return BatchItem{InputPath: inputPath, OutputPath: outputPath, OK: false, Error: err.Error()}
	}

	result, err := s.Convert(ctx, ConvertRequest{
		InputPath:  inputPath,
		OutputPath: outputPath,
		Target:     target,
		Preset:     request.Preset,
		Overwrite:  request.Overwrite,
	}, nil)
	if err != nil {
		return BatchItem{
			InputPath:  inputPath,
			OutputPath: outputPath,
			OK:         false,
			Error:      failure.Format(err, false),
		}
	}
	return BatchItem{
		InputPath:  result.InputPath,
		OutputPath: result.OutputPath,
		OK:         true,
		Warnings:   result.Warnings,
	}
}

// normalizeJobs resolves the requested concurrency. Zero is the unset zero
// value and means sequential; more workers than files would only park idle
// goroutines, so the count is capped at the amount of work available.
func normalizeJobs(requested, candidates int) (int, error) {
	if requested < 0 {
		return 0, failure.New(
			failure.Usage,
			fmt.Sprintf("--jobs cannot be negative, received %d.", requested),
			"Use --jobs 1 to convert one file at a time.",
			nil,
		)
	}
	if requested == 0 {
		requested = 1
	}
	if requested > candidates {
		return candidates, nil
	}
	return requested, nil
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
	report := DoctorReport{OK: true, Checks: make([]DoctorCheck, 0, 9)}
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
	report.add("libmp3lame encoder", capabilities.HasEncoder("libmp3lame"), capabilityDetail(capabilities.HasEncoder("libmp3lame")))
	report.add("libvpx-vp9 encoder", capabilities.HasEncoder("libvpx-vp9"), capabilityDetail(capabilities.HasEncoder("libvpx-vp9")))
	report.add("libopus encoder", capabilities.HasEncoder("libopus"), capabilityDetail(capabilities.HasEncoder("libopus")))
	hasMP4 := capabilities.HasMuxer("mp4") || capabilities.HasMuxer("mov")
	report.add("MP4 muxer", hasMP4, capabilityDetail(hasMP4))
	report.add("MP3 muxer", capabilities.HasMuxer("mp3"), capabilityDetail(capabilities.HasMuxer("mp3")))
	report.add("WebM muxer", capabilities.HasMuxer("webm"), capabilityDetail(capabilities.HasMuxer("webm")))
	return report
}

func (s *Service) Formats() []profile.SupportedFormat {
	return (profile.Registry{}).Formats()
}

func resolveInputDir(input string) (string, error) {
	if strings.TrimSpace(input) == "" {
		return "", failure.New(failure.Usage, "An input directory is required.", "Run 'mediaconv batch --help' for examples.", nil)
	}
	path, err := filepath.Abs(input)
	if err != nil {
		return "", failure.Wrap(failure.Input, "The input directory path is invalid.", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", failure.New(failure.Input, "The input directory does not exist or cannot be accessed.", "Check the path and directory permissions.", err)
	}
	if !info.IsDir() {
		return "", failure.New(failure.Input, "The batch input must be a directory.", "Use 'mediaconv convert' for a single file.", nil)
	}
	return filepath.Clean(path), nil
}

func normalizeTarget(target string) (string, error) {
	target = strings.ToLower(strings.TrimSpace(target))
	if target == "" {
		target = defaultTarget
	}
	if _, ok := (profile.Registry{}).Target(target); !ok {
		return "", unsupportedTarget(target)
	}
	return target, nil
}

// defaultTarget is what --to falls back to when it is not given.
const defaultTarget = "mp4"

func unsupportedTarget(target string) error {
	return failure.New(
		failure.Usage,
		fmt.Sprintf("Unsupported target format %q.", target),
		fmt.Sprintf("Supported formats are %s. Run 'mediaconv formats' to list the conversions.", strings.Join((profile.Registry{}).TargetNames(), ", ")),
		nil,
	)
}

func resolveBatchOutputDir(inputDir, requested string) (string, error) {
	if strings.TrimSpace(requested) == "" {
		return inputDir, nil
	}
	outputDir, err := filepath.Abs(requested)
	if err != nil {
		return "", failure.Wrap(failure.OutputConflict, "The output directory path is invalid.", err)
	}
	outputDir = filepath.Clean(outputDir)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", failure.New(failure.OutputConflict, "The output directory could not be created.", "Check the path and directory permissions.", err)
	}
	info, err := os.Stat(outputDir)
	if err != nil || !info.IsDir() {
		return "", failure.New(failure.OutputConflict, "The output path is not a directory.", "Choose a directory for --output-dir.", err)
	}
	return outputDir, nil
}

func collectBatchCandidates(root, target string, recursive bool) ([]string, error) {
	extensions := batchInputExtensions(target)
	candidates := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != root && !recursive {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		if extensions[strings.ToLower(filepath.Ext(path))] {
			candidates = append(candidates, path)
		}
		return nil
	})
	if err != nil {
		return nil, failure.Wrap(failure.Input, "The input directory could not be scanned.", err)
	}
	sort.Strings(candidates)
	return candidates, nil
}

func batchInputExtensions(target string) map[string]bool {
	found, ok := (profile.Registry{}).Target(target)
	if !ok {
		return nil
	}
	extensions := make(map[string]bool, len(found.BatchExtensions))
	for _, extension := range found.BatchExtensions {
		extensions[extension] = true
	}
	return extensions
}

func batchOutputPath(inputDir, outputDir, inputPath, target string) (string, error) {
	relative, err := filepath.Rel(inputDir, inputPath)
	if err != nil {
		return "", fmt.Errorf("resolve relative path: %w", err)
	}
	extension := filepath.Ext(relative)
	outputRelative := strings.TrimSuffix(relative, extension) + "." + target
	return filepath.Join(outputDir, outputRelative), nil
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
		return "", nil, failure.New(failure.Input, "The input file is empty.", "Choose a non-empty video file.", nil)
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
		target = defaultTarget
	}
	if _, ok := (profile.Registry{}).Target(target); !ok {
		return "", unsupportedTarget(target)
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
		return "", failure.New(failure.Usage, fmt.Sprintf("The output extension does not match --to %s.", target), fmt.Sprintf("Use an output path ending in .%s.", target), nil)
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
		return failure.New(failure.Dependency, "The installed FFmpeg does not provide a required codec or muxer.", "Run 'mediaconv doctor' and install a complete FFmpeg build.", err)
	case errors.Is(err, profile.ErrUnsupportedTarget), errors.Is(err, profile.ErrUnsupportedPreset):
		return failure.New(failure.Usage, err.Error(), "Run 'mediaconv formats' to list supported conversions and profiles.", err)
	default:
		return failure.New(failure.Input, "The input is not supported by the selected conversion profile.", "Run 'mediaconv formats' to list supported conversions and profiles.", err)
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
