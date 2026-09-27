#!/usr/bin/env sh
# Prepares the sandbox used by demo.tape: a freshly built mediaconv plus the
# synthetic fixtures it converts on screen.
#
# Usage:
#   docs/demo/setup.sh && vhs docs/demo/demo.tape
#
# Requires Go and FFmpeg. Everything is written under DEMO_DIR, which is
# removed and recreated on every run so the recording always starts clean.

set -eu

DEMO_DIR="${DEMO_DIR:-/tmp/mediaconv-demo}"
REPO_ROOT="$(CDPATH='' cd -- "$(dirname -- "$0")/../.." && pwd)"

rm -rf "$DEMO_DIR"
mkdir -p "$DEMO_DIR/bin" "$DEMO_DIR/clips"

cd "$REPO_ROOT"
go build -trimpath -o "$DEMO_DIR/bin/mediaconv" ./cmd/mediaconv

# The demo ships no third-party media: the video is FFmpeg's test pattern and
# the audio is a generated sine tone. 1080p for twelve seconds is heavy enough
# to keep the progress display on screen for about five seconds, which is what
# makes the recording worth watching.
ffmpeg -v error -y \
	-f lavfi -i "testsrc2=size=1920x1080:rate=30:duration=12" \
	-f lavfi -i "sine=frequency=440:duration=12" \
	-c:v libvpx-vp9 -b:v 2M -deadline realtime -cpu-used 8 \
	-c:a libopus \
	"$DEMO_DIR/recording.webm"

# Um clipe curto e pequeno: VP9 e lento, e o demo nao deve virar uma espera.
ffmpeg -v error -y \
	-f lavfi -i "testsrc2=size=480x270:rate=24:duration=2" \
	-f lavfi -i "sine=frequency=440:duration=2" \
	-c:v libx264 -crf 23 -c:a aac -pix_fmt yuv420p \
	"$DEMO_DIR/clipe.mp4"

for name in intro outro; do
	ffmpeg -v error -y \
		-f lavfi -i "sine=frequency=330:duration=3" \
		"$DEMO_DIR/clips/$name.wav"
done

printf 'Demo sandbox ready: %s\n' "$DEMO_DIR"
