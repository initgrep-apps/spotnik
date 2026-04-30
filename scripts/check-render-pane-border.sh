#!/usr/bin/env bash
set -euo pipefail
# Flags any file outside the rendering layer that calls layout.RenderPaneBorder
# directly. Only uikit/ (which wraps it into PaneChrome/OverlayChrome) and
# layout/ itself should reference this function. Comment-only references are
# excluded — they appear in doc comments that describe the architecture.

OFFENDERS=$(grep -rn --include="*.go" "layout\.RenderPaneBorder\|RenderPaneBorder(" internal/ \
    | grep -v "internal/uikit/" \
    | grep -v "internal/ui/layout/" \
    | grep -v "_test.go" \
    | perl -ne 'print unless m{//.*RenderPaneBorder}' \
    || true)

if [ -n "$OFFENDERS" ]; then
    echo "ERROR: layout.RenderPaneBorder called outside internal/uikit/ — use uikit.PaneChrome / OverlayChrome instead."
    echo "$OFFENDERS"
    exit 1
fi
echo "OK: no direct RenderPaneBorder callers outside uikit"
