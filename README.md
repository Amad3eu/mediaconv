# MediaConv

[English](README.md) · [Português (Brasil)](README.pt-BR.md)

[![CI](https://github.com/Amad3eu/mediaconv/actions/workflows/ci.yml/badge.svg)](https://github.com/Amad3eu/mediaconv/actions/workflows/ci.yml)
[![CodeQL](https://github.com/Amad3eu/mediaconv/actions/workflows/codeql.yml/badge.svg)](https://github.com/Amad3eu/mediaconv/actions/workflows/codeql.yml)
[![Go](https://img.shields.io/badge/Go-1.26%2B-00ADD8?logo=go)](go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A safe, script-friendly command-line media converter powered by FFmpeg.

The first supported profile converts WebM video to broadly compatible MP4 using
H.264 video and AAC audio. MediaConv validates the input, converts into a private
staging directory, verifies the result, and only then publishes the output.

> [!NOTE]
> MediaConv is in early development. Until v1.0, commands and flags may change
> between minor releases.

## Features

- Local WebM to MP4 conversion with a compatibility-focused profile.
- Interactive progress when stderr is a terminal; clean output in scripts.
- No overwrite unless `--overwrite` is explicitly provided.
- Temporary output cleanup after failure or interruption.
- Paths containing spaces and Unicode are passed directly to FFmpeg without a shell.
- Human-readable and JSON output.
- Dependency and codec diagnostics through `mediaconv doctor`.
- Native release binaries for Linux, macOS, and Windows on AMD64 and ARM64.

## Quick start

```bash
# Verify FFmpeg and the required codecs.
mediaconv doctor

# Inspect an input file.
mediaconv inspect "recording.webm"

# Create recording.mp4 beside the input.
mediaconv convert "recording.webm"

# Select an output and explicitly allow replacement.
mediaconv convert "recording.webm" \
  --output "exports/recording.mp4" \
  --overwrite
```

## Requirements

MediaConv does not bundle or download FFmpeg. Install `ffmpeg` and `ffprobe` before
using it. The initial `web` profile also requires the `libx264` video encoder, the
AAC audio encoder, and the MP4 muxer.

Common installation commands:

```bash
# Debian / Ubuntu
sudo apt update && sudo apt install ffmpeg

# macOS with Homebrew
brew install ffmpeg

# Arch Linux
sudo pacman -S ffmpeg
```

On Windows, one option referenced by the
[official FFmpeg download page](https://ffmpeg.org/download.html) is:

```powershell
winget install --id Gyan.FFmpeg --exact --source winget
```

FFmpeg builds differ. Run `mediaconv doctor` rather than assuming a particular
package includes every codec.

## Install MediaConv

### Release archive

Download the archive for your operating system from
[GitHub Releases](https://github.com/Amad3eu/mediaconv/releases/latest), verify it
against the published checksum, extract it, and place `mediaconv` in a directory
included in `PATH`.

### Go toolchain

```bash
go install github.com/Amad3eu/mediaconv/cmd/mediaconv@latest
```

Installing MediaConv with Go does not install FFmpeg.

### Build from source

```bash
git clone https://github.com/Amad3eu/mediaconv.git
cd mediaconv
go build -trimpath -o ./bin/mediaconv ./cmd/mediaconv
```

Development requires Go 1.26 or newer.

## Commands

```text
mediaconv convert INPUT [--to mp4] [-o OUTPUT] [--preset web] [--overwrite]
mediaconv inspect INPUT
mediaconv doctor
mediaconv formats
mediaconv version
mediaconv completion bash|zsh|fish|powershell
```

Use `mediaconv COMMAND --help` for the complete flags and examples. Global flags
include `--json`, `--verbose`, `--ffmpeg-path`, and `--ffprobe-path`.

### JSON and exit codes

Use `--json` for automation. Successful results are written to stdout; progress
and diagnostics use stderr. Interactive progress is automatically disabled when
stderr is not a terminal.

| Code | Meaning |
| ---: | --- |
| 0 | Success |
| 1 | Unexpected internal error |
| 2 | Invalid command, flag, or option |
| 3 | Missing FFmpeg dependency or capability |
| 4 | Invalid, corrupt, or unsupported input |
| 5 | Output conflict or publication failure |
| 6 | Conversion or output verification failure |
| 130 | Interrupted by the user |

## Supported conversions

| Input | Output | Profile | Video | Audio | Status |
| --- | --- | --- | --- | --- | --- |
| WebM | MP4 | `web` | H.264 (`libx264`, CRF 23) | AAC 192 kbit/s | Initial |

The `web` profile converts the first video stream and the first optional audio
stream. It produces `yuv420p`, preserves compatible metadata, drops chapters and
subtitles, pads odd dimensions to even values, and enables MP4 fast start. The CLI
warns when extra streams, transparency, chapters, subtitles, or HDR may be lost.

## Safety and privacy

- Only regular local files are accepted. URLs, devices, and pipes are not supported.
- FFmpeg is started with an argument array, never through `sh`, `cmd.exe`, or another shell.
- Conversion happens in a private staging directory on the output filesystem.
- Existing outputs and symlink outputs are rejected unless a regular file is explicitly replaced.
- The verified output is published atomically on supported filesystems.
- Media files are processed locally and are never uploaded by MediaConv.
- There is no telemetry.

Without `--overwrite`, publication uses a hard link so another process cannot race
MediaConv into replacing an existing destination. The output filesystem must
support hard links. This is standard on common local NTFS, APFS, ext4, and similar
filesystems, but may not be available on some removable or network filesystems.

## Roadmap

- Additional profiles such as MP4 to WebM and MOV/MKV to MP4.
- Audio extraction to MP3, AAC, and WAV.
- Batch conversion with conservative concurrency controls.
- Package-manager distribution after the release interface stabilizes.
- Optional hardware acceleration after capability-specific tests are available.

Dynamic plugins and bundled FFmpeg binaries are intentionally outside the initial
scope. See [the architecture](docs/ARCHITECTURE.md) for the design boundaries.

## Contributing and security

See [CONTRIBUTING.md](CONTRIBUTING.md) before opening a pull request. Report
security issues privately according to [SECURITY.md](SECURITY.md). Repository
maintainers should also apply the settings in
[docs/REPOSITORY_SETUP.md](docs/REPOSITORY_SETUP.md).

## License and FFmpeg

MediaConv is available under the [MIT License](LICENSE). FFmpeg is a separate
project with licensing determined by its build configuration. MediaConv invokes
the user's FFmpeg executables and does not redistribute them. See
[THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) for details.
