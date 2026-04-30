#!/usr/bin/env bash
set -euo pipefail
# Flags any file that calls lipgloss.RoundedBorder() (or similar inline border
# constructors) outside of internal/uikit/border.go. Such direct calls leak
# unicode glyphs into ASCII mode because they bypass uikit.RoundedBorder()
# which honours uikit.ActiveMode().
#
# Only internal/uikit/border.go is allowed to call lipgloss.RoundedBorder()
# (it is the canonical wrapper). Test files are excluded.

PATTERN='lipgloss\.(RoundedBorder|NormalBorder|ThickBorder|DoubleBorder|MarkdownBorder|ASCIIBorder)\b'

OFFENDERS=$(grep -rEn --include="*.go" "$PATTERN" internal/ cmd/ \
    | grep -v "internal/uikit/border.go:" \
    | grep -v "_test.go" \
    | perl -ne "print unless m{//.*lipgloss\\.}" \
    || true)

if [ -n "$OFFENDERS" ]; then
    echo "ERROR: direct lipgloss border constructor outside uikit — use uikit.RoundedBorder() instead."
    echo "$OFFENDERS"
    exit 1
fi
echo "OK: no direct lipgloss border callers outside uikit/border.go"
