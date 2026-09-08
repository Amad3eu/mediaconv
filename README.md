# MediaConv

[English](README.md) · [Português (Brasil)](README.pt-BR.md)

[![CI](https://github.com/Amad3eu/mediaconv/actions/workflows/ci.yml/badge.svg)](https://github.com/Amad3eu/mediaconv/actions/workflows/ci.yml)
[![CodeQL](https://github.com/Amad3eu/mediaconv/actions/workflows/codeql.yml/badge.svg)](https://github.com/Amad3eu/mediaconv/actions/workflows/codeql.yml)
[![Go](https://img.shields.io/badge/Go-1.26%2B-00ADD8?logo=go)](go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Website](https://img.shields.io/badge/Website-GitHub%20Pages-222222?logo=githubpages)](https://amad3eu.github.io/mediaconv/)

MediaConv is a safe, script-friendly command-line media converter powered by
FFmpeg.

It starts with one polished profile: converting common video containers to
broadly compatible MP4 using H.264 video and AAC audio. MediaConv validates the
input, converts into a private staging directory, verifies the result, and only
then publishes the output.

It also includes a `music` profile for converting common audio files to portable
MP3 using libmp3lame.

> [!NOTE]
> MediaConv is in early development. Until v1.0, commands and flags may change
> between minor releases.

## Why MediaConv?

FFmpeg is powerful, but the command line for a safe, compatible MP4 conversion is
easy to get wrong. MediaConv packages that workflow into a small CLI with
predictable defaults, clear diagnostics, structured output for automation, and a
project layout ready for more converters over time.

Use MediaConv when you want:

- a short command instead of remembering FFmpeg flags;
- output that is verified before it replaces or creates the final file;
- readable errors for missing codecs, corrupt input, or output conflicts;
- a CLI that works in scripts through JSON and typed exit codes;
- a foundation that can grow into more media conversion profiles.

## Features

- Local video to MP4 conversion with a compatibility-focused profile.
- Local audio to MP3 conversion with a music-focused profile.
- Interactive progress when stderr is a terminal; clean output in scripts.
- No overwrite unless `--overwrite` is explicitly provided.
- Temporary output cleanup after failure or interruption.
- Paths containing spaces and Unicode are passed directly to FFmpeg without a shell.
- Human-readable and JSON output.
- Dependency and codec diagnostics through `mediaconv doctor`.
- Native release binaries for Linux, macOS, and Windows on AMD64 and ARM64.

## Quick start

Install FFmpeg first, then install MediaConv from the latest release or with Go.

```bash
# Verify FFmpeg and the required codecs.
mediaconv doctor

# Inspect an input file.
mediaconv inspect "recording.webm"

# Create recording.mp4 beside the input.
mediaconv convert "recording.webm"

# Convert a camera export to MP4.
mediaconv convert "camera.mov" --output "camera.mp4"

# Convert audio to MP3.
mediaconv convert "song.wav" --to mp3

# Convert a whole directory.
mediaconv batch "./recordings" --to mp4 --output-dir "./converted"

# Select an output and explicitly allow replacement.
mediaconv convert "recording.webm" \
  --output "exports/recording.mp4" \
  --overwrite
```

Project site: <https://amad3eu.github.io/mediaconv/>

Latest release: <https://github.com/Amad3eu/mediaconv/releases/latest>

## Requirements

MediaConv does not bundle or download FFmpeg. Install `ffmpeg` and `ffprobe`
before using it. The `web` profile requires `libx264`, AAC encoding, and MP4
muxing. The `music` profile requires `libmp3lame` and MP3 muxing.

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

### Install script

Linux and macOS users can install the latest release without Go:

```bash
curl -fsSL https://amad3eu.github.io/mediaconv/install.sh | sh
```

To install into a custom directory:

```bash
curl -fsSL https://amad3eu.github.io/mediaconv/install.sh \
  | MEDIACONV_INSTALL_DIR="$HOME/.local/bin" sh
```

Windows users can install the latest release with PowerShell:

```powershell
irm https://amad3eu.github.io/mediaconv/install.ps1 | iex
```

To install into a custom directory:

```powershell
$env:MEDIACONV_INSTALL_DIR="$env:USERPROFILE\bin"
irm https://amad3eu.github.io/mediaconv/install.ps1 | iex
```

### Release archive

Download the archive for your operating system from
[GitHub Releases](https://github.com/Amad3eu/mediaconv/releases/latest), verify it
against the published checksum, extract it, and place `mediaconv` in a directory
included in `PATH`.

### Homebrew

```bash
brew install --cask Amad3eu/tap/mediaconv
```

### Debian, Ubuntu, Fedora, and Alpine

Download the Linux package for your platform from
[GitHub Releases](https://github.com/Amad3eu/mediaconv/releases/latest), then
install it with your system package manager:

```bash
# Debian / Ubuntu
sudo apt install ./mediaconv_0.3.0_linux_amd64.deb

# Fedora / RHEL
sudo dnf install ./mediaconv_0.3.0_linux_amd64.rpm

# Alpine
sudo apk add --allow-untrusted ./mediaconv_0.3.0_linux_amd64.apk
```

The package names above use `0.3.0` as an example. Use the latest available
version from the release page.

### Windows with Scoop

```powershell
scoop bucket add amad3eu https://github.com/Amad3eu/scoop-bucket
scoop install amad3eu/mediaconv
```

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
mediaconv convert INPUT [--to mp4|mp3] [-o OUTPUT] [--preset web|music] [--overwrite]
mediaconv batch DIRECTORY [--to mp4|mp3] [-o OUTPUT_DIR] [--recursive] [--overwrite]
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
| WebM | MP4 | `web` | H.264 (`libx264`, CRF 23) | AAC 192 kbit/s | Stable |
| MOV / QT | MP4 | `web` | H.264 (`libx264`, CRF 23) | AAC 192 kbit/s | Stable |
| MKV | MP4 | `web` | H.264 (`libx264`, CRF 23) | AAC 192 kbit/s | Stable |
| AVI | MP4 | `web` | H.264 (`libx264`, CRF 23) | AAC 192 kbit/s | Stable |
| MP4 / M4V | MP4 | `web` | H.264 (`libx264`, CRF 23) | AAC 192 kbit/s | Stable |
| WAV | MP3 | `music` | none | MP3 (`libmp3lame`) 192 kbit/s | Stable |
| FLAC | MP3 | `music` | none | MP3 (`libmp3lame`) 192 kbit/s | Stable |
| M4A / M4B | MP3 | `music` | none | MP3 (`libmp3lame`) 192 kbit/s | Stable |
| AAC | MP3 | `music` | none | MP3 (`libmp3lame`) 192 kbit/s | Stable |
| OGG / OGA / OPUS | MP3 | `music` | none | MP3 (`libmp3lame`) 192 kbit/s | Stable |
| MP3 | MP3 | `music` | none | MP3 (`libmp3lame`) 192 kbit/s | Stable |

The `web` profile converts the first video stream and the first optional audio
stream. It produces `yuv420p`, preserves compatible metadata, drops chapters and
subtitles, pads odd dimensions to even values, and enables MP4 fast start. The CLI
warns when extra streams, transparency, chapters, subtitles, or HDR may be lost.

The `music` profile converts the first audio stream, writes MP3 with
`libmp3lame` at 192 kbit/s, drops video/subtitle streams, and verifies the MP3
output before publishing it.

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

- Additional profiles such as MP4 to WebM and GIF previews.
- Audio output profiles such as AAC and WAV.
- Batch concurrency controls for larger folders.
- Native package repositories for `apt`, `dnf`, and `apk`.
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
