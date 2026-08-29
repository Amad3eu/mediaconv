# ADR 0003: Compiled-in conversion profiles

- Status: accepted
- Date: 2026-08-29

## Context

MediaConv starts with WebM to MP4 but is intended to support more conversions. A
dynamic plugin system would introduce versioning, discovery, trust, and portability
problems before there is a concrete plugin ecosystem.

## Decision

Represent each supported target/preset as a compiled-in profile. Profiles consume
probed media information and detected FFmpeg capabilities, then return a structured
conversion plan. Only the FFmpeg adapter serializes that plan into process arguments.

## Consequences

- Plans can be unit-tested without invoking FFmpeg.
- Adding a format does not require changing the CLI or subprocess security boundary.
- All supported profiles ship together with a MediaConv release.
- External plugins remain out of scope until a real use case justifies a versioned,
  subprocess-based protocol.
