# Architecture

MediaConv is a local CLI that orchestrates user-installed FFmpeg executables. The
Go process owns validation, planning, progress, error classification, output
verification, and publication. FFmpeg owns media decoding and encoding.

## Conversion flow

```text
Cobra command
    -> application service
    -> local path validation
    -> ffprobe input
    -> profile registry and structured plan
    -> FFmpeg runner and progress parser
    -> ffprobe staged output
    -> profile verification
    -> output publisher
```

The final destination is never passed to FFmpeg. A conversion is written into a
private directory created by `os.MkdirTemp` beside the destination, which keeps the
staged file on the same filesystem. After FFmpeg succeeds, ffprobe verifies the
expected MP4 container and streams. Only a verified file is published.

Without overwrite, the publisher uses a hard link. Creating the destination link
is atomic and fails if another file already exists. With explicit overwrite, Unix
uses `rename(2)` semantics and Windows uses `MoveFileEx` with replacement and
write-through flags.

## Package boundaries

| Package | Responsibility |
| --- | --- |
| `cmd/mediaconv` | Signal-aware process entry point |
| `internal/cli` | Cobra commands and human/JSON rendering |
| `internal/app` | Use-case orchestration and path preflight |
| `internal/media` | Media, stream, plan, capability, and progress types |
| `internal/profile` | Supported targets, presets, planning, and output verification |
| `internal/ffmpeg` | Binary discovery, capability detection, probing, argv building, execution, and progress parsing |
| `internal/output` | Staging cleanup and race-resistant publication |
| `internal/failure` | Typed public error categories and exit codes |
| `internal/buildinfo` | Version information injected by GoReleaser |

Packages remain under `internal/` until there is a real external Go API consumer.
New conversions should normally be new profiles, not dynamic Go plugins.

## Security boundaries

- The MVP accepts regular local files only.
- Input and output paths become absolute before FFmpeg is started.
- Child processes are created by `exec.CommandContext` with distinct arguments;
  media paths are never parsed by a shell.
- FFmpeg stdin is disabled and the input protocol whitelist contains only `file`.
- FFmpeg stderr is limited to its last 64 KiB.
- Cancellation kills the child process and cleanup removes only the exact staging
  directory returned by `os.MkdirTemp`.
- Existing symlink outputs are rejected.
- FFmpeg is not downloaded, bundled, or updated by MediaConv.

If MediaConv ever accepts URLs, becomes a service, bundles FFmpeg, or loads external
profiles, those features require a separate threat model and architecture decision.

## Release model

The source uses SemVer tags. GoReleaser creates pure-Go binaries for Linux, macOS,
and Windows on AMD64 and ARM64. Archives include documentation and license files;
FFmpeg is never included. Releases also contain SHA-256 checksums, SBOMs, a keyless
Sigstore signature, and GitHub provenance attestations.

See [RELEASING.md](RELEASING.md) for the operational process.
