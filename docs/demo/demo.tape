# MediaConv terminal demo, recorded with VHS (https://github.com/charmbracelet/vhs).
#
# Regenerate with:
#   docs/demo/setup.sh && vhs docs/demo/demo.tape
#
# setup.sh builds the CLI and generates the fixtures into /tmp/mediaconv-demo.
# The PATH below is pinned so the recording always uses that build and the
# system FFmpeg, which keeps `doctor` output identical between machines.

Output site/demo.gif

Require bash
Require ffmpeg

Set Shell bash
Set FontSize 16
Set Width 1280
# Tall enough for the whole session to stay on screen at the end, so the last
# frame doubles as a summary. Raise this if the commands below grow.
Set Height 520
Set Padding 24
Set BorderRadius 10
Set TypingSpeed 55ms
Set Framerate 24

# The site's dark code panel, so the demo and the landing page look related.
Set Theme { "background": "#0b1220", "foreground": "#e5f7f3", "cursor": "#5eead4", "selection": "#1e3a5f", "black": "#0b1220", "red": "#f87171", "green": "#34d399", "yellow": "#facc15", "blue": "#60a5fa", "magenta": "#c084fc", "cyan": "#5eead4", "white": "#e5f7f3", "brightBlack": "#64748b", "brightRed": "#fca5a5", "brightGreen": "#6ee7b7", "brightYellow": "#fde047", "brightBlue": "#93c5fd", "brightMagenta": "#d8b4fe", "brightCyan": "#99f6e4", "brightWhite": "#ffffff" }

Hide
Type "cd /tmp/mediaconv-demo" Enter
Type "export PATH=/tmp/mediaconv-demo/bin:/usr/bin:/bin" Enter
Type "export PS1='$ '" Enter
Type "clear" Enter
Show

Sleep 700ms

# 1. Check FFmpeg and the codecs the profiles need.
Type "mediaconv doctor" Sleep 500ms Enter
Sleep 2500ms

# 2. Convert, with live progress while FFmpeg works.
Type "mediaconv convert recording.webm" Sleep 500ms Enter
Sleep 7s

# 3. Run it again: existing output is never replaced silently.
Type "mediaconv convert recording.webm" Sleep 500ms Enter
Sleep 2500ms

# 4. Convert a whole folder, several files at a time.
Type "mediaconv batch ./clips --to mp3 --jobs 4" Sleep 500ms Enter
Sleep 3s

# 5. The royalty-free profile: VP9 and Opus in a WebM container.
Type "mediaconv convert clipe.mp4 --to webm" Sleep 500ms Enter
Sleep 4s
