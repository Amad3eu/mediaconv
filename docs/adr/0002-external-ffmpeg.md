# ADR 0002: Keep FFmpeg external

- Status: accepted
- Date: 2026-08-29

## Context

Bundling FFmpeg would make initial installation more convenient, but each platform
needs a trustworthy binary, updates, source provenance, and license compliance. The
initial profile uses `libx264`, which changes the license of an FFmpeg build that
enables it.

## Decision

Locate user-installed `ffmpeg` and `ffprobe` in `PATH`, with explicit override flags
for controlled environments. Provide `mediaconv doctor` to check paths and required
capabilities. Do not download or redistribute FFmpeg.

## Consequences

- MediaConv release archives remain small and contain only project artifacts.
- Users are responsible for installing a compatible FFmpeg distribution.
- The exact FFmpeg license remains associated with the user's chosen build.
- A bundled distribution requires a future legal, supply-chain, and maintenance review.
