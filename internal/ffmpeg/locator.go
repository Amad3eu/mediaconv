package ffmpeg

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

type Paths struct {
	FFmpeg  string
	FFprobe string
}

type Locator struct{}

func (Locator) Locate(ffmpegOverride, ffprobeOverride string) (Paths, error) {
	ffmpegPath, err := (Locator{}).LocateFFmpeg(ffmpegOverride)
	if err != nil {
		return Paths{}, err
	}

	ffprobePath := ""
	if ffprobeOverride == "" {
		candidate := filepath.Join(filepath.Dir(ffmpegPath), executableName("ffprobe"))
		if executableFile(candidate) == nil {
			ffprobePath = candidate
		}
	}
	if ffprobePath == "" {
		ffprobePath, err = (Locator{}).LocateFFprobe(ffprobeOverride)
		if err != nil {
			return Paths{}, err
		}
	}

	return Paths{FFmpeg: ffmpegPath, FFprobe: ffprobePath}, nil
}

func (Locator) LocateFFmpeg(override string) (string, error) {
	return locateExecutable(override, "ffmpeg")
}

func (Locator) LocateFFprobe(override string) (string, error) {
	return locateExecutable(override, "ffprobe")
}

func locateExecutable(override, name string) (string, error) {
	if override == "" {
		path, err := exec.LookPath(executableName(name))
		if err != nil {
			return "", fmt.Errorf("%s was not found in PATH", name)
		}
		return filepath.Abs(path)
	}

	path, err := filepath.Abs(override)
	if err != nil {
		return "", fmt.Errorf("resolve %s path: %w", name, err)
	}

	info, err := os.Stat(path)
	if err == nil && info.IsDir() {
		path = filepath.Join(path, executableName(name))
	}

	if err := executableFile(path); err != nil {
		return "", fmt.Errorf("invalid %s executable %q: %w", name, path, err)
	}
	return path, nil
}

func executableFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("not a regular file")
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o111 == 0 {
		return fmt.Errorf("file is not executable")
	}
	return nil
}

func executableName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}
