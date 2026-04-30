#!/usr/bin/env bash
set -euo pipefail
# Catalogue characters are defined in internal/uikit/glyph.go and must not
# be used as raw string literals in any other production Go source.
#
# Exempt files (rendering implementation layers that cannot import uikit
# due to import-cycle constraints):
#   - internal/uikit/glyph.go           (canonical glyph catalogue definition)
#   - internal/uikit/spinner_frames.go  (canonical spinner-frame backing slices)
#   - internal/ui/layout/border.go      (owns fallback glyph defaults)
#   - internal/ui/layout/truncate.go    (uses "…" constant; layout cannot import uikit)
#   - internal/app/splash.go            (figlet "ANSI Shadow" banner art; see docs/system/tui.md §4.1)
#
# Comment-only hits are filtered: a match is ignored when the glyph first
# appears after "//" on the same line (i.e. only in a comment, not in code).
CHARS=(
  # Structural / borders
  "╭" "╮" "╰" "╯" "─" "│" "×"
  # Intent / feedback
  "✓" "✗" "◬" "→" "⧖" "⚡" "◷" "⏸" "⊘"
  # State / availability
  "◉" "◎" "○" "●" "□" "■" "◌" "★" "☆" "•"
  # Navigation / scroll
  "▼" "▲" "►" "◄" "…" "›" "‹" "←" "↑" "↓" "↔"
  # Playback controls
  "▶" "▷" "⏭" "⏮" "⏩" "⏪" "⇄" "↻" "↻¹" "⟳" "≡" "⏏"
  # Domain / music / identity
  "♪" "♫" "♛" "☁" "▤" "·"
  # Device-type icons
  "⊡" "⊞" "⊟" "⊠"
  # Keyboard chords
  "⏎" "⎋" "⇥" "⌫" "␣"
  # Superscripts
  "⁰" "¹" "²" "³" "⁴" "⁵" "⁶" "⁷" "⁸" "⁹" "⁺" "⁻"
  # Graphical fills / bars
  "█" "▉" "▊" "▋" "▌" "▍" "▎" "▏" "░" "▒" "▓"
  # Spinner braille frames — dispatched via SpinnerFrames(mode), not GlyphFor,
  # but raw uses elsewhere should still be caught by this guard.
  "⠋" "⠙" "⠹" "⠸" "⠼" "⠴" "⠦" "⠧" "⠇" "⠏"
)

LEAKS=""
for c in "${CHARS[@]}"; do
    found=$(grep -rn --include="*.go" "$c" internal/ cmd/ 2>/dev/null \
        | grep -v "internal/uikit/glyph.go" \
        | grep -v "internal/uikit/spinner_frames.go" \
        | grep -v "internal/ui/layout/border.go" \
        | grep -v "internal/ui/layout/truncate.go" \
        | grep -v "internal/app/splash.go" \
        | grep -v "_test.go" \
        | perl -ne 'print unless m{//.*'"$c"'}' \
        || true)
    if [ -n "$found" ]; then
        LEAKS="$LEAKS\n$found"
    fi
done

if [ -n "$LEAKS" ]; then
    echo "ERROR: catalogue characters leaked outside internal/uikit/glyph.go:"
    printf "%b\n" "$LEAKS"
    exit 1
fi
echo "OK: no catalogue leaks"
