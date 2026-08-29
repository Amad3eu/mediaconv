# ADR 0001: Go and Cobra for the CLI

- Status: accepted
- Date: 2026-08-29

## Context

MediaConv must be straightforward to install on Windows, Linux, and macOS and must
grow from one conversion into multiple commands and profiles. FFmpeg performs the
CPU-intensive work, so the wrapper language is primarily a distribution and
maintainability decision.

## Decision

Use Go for a native CLI and Cobra for command routing, help, flags, and shell
completion. Keep Cobra inside `internal/cli` so application and media logic do not
depend on a presentation framework. Build with `CGO_ENABLED=0`.

## Consequences

- Users can download a native MediaConv executable without installing Go.
- Developers can use `go install` and the standard Go test toolchain.
- Release cross-compilation is simple for the initial platform matrix.
- FFmpeg remains a separate runtime dependency.
- The project accepts the maintenance cost of Cobra and its transitive dependencies.
