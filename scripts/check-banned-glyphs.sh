#!/usr/bin/env bash
set -euo pipefail
# Banned glyphs must not appear in production Go source. Test files are
# exempt because they may reference banned chars in assert.NotContains
# assertions that verify the glyphs have been removed from rendered output.
BANNED=( "⚠" "ᐅ" "┌" "┐" "└" "┘" "╔" "╗" "╚" "╝" "✅" "❌" "❗" )
for g in "${BANNED[@]}"; do
    if grep -rn --include="*.go" "$g" internal/ cmd/ 2>/dev/null \
           | grep -v "_test.go"; then
        echo "ERROR: banned glyph '$g' present in source"
        exit 1
    fi
done
echo "OK: no banned glyphs"
