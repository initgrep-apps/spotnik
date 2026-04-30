#!/usr/bin/env bash
set -euo pipefail
# Banned glyphs must not appear in production Go source. Exemptions:
#   - *_test.go: may reference banned chars in assert.NotContains assertions.
#   - internal/app/splash.go: the SPOTNIK figlet "ANSI Shadow" banner uses
#     ╔╗╚╝ as letter-shape glyphs, not as chrome corners. See docs/system/tui.md §4.1.
BANNED=( "⚠" "ᐅ" "┌" "┐" "└" "┘" "╔" "╗" "╚" "╝" "✅" "❌" "❗" )
for g in "${BANNED[@]}"; do
    if grep -rn --include="*.go" "$g" internal/ cmd/ 2>/dev/null \
           | grep -v "_test.go" \
           | grep -v "internal/app/splash.go"; then
        echo "ERROR: banned glyph '$g' present in source"
        exit 1
    fi
done
echo "OK: no banned glyphs"
