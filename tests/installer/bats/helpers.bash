# Pinned release used for all deterministic tests.
export SPOTNIK_TEST_VERSION="v0.1.0-rc1"

# Default install dir for all tests; can be overridden per-test.
export TEST_INSTALL_DIR="$HOME/.local/bin"

# Marker text — single source of truth for the test layer. If install.sh's
# RC_MARKER_OPEN/RC_MARKER_CLOSE ever change, update these to match.
export SPOTNIK_RC_MARKER_OPEN_RE='^# >>> spotnik installer >>>$'
export SPOTNIK_RC_MARKER_CLOSE_RE='^# <<< spotnik installer <<<$'

# POSIX rc files the installer may edit. Mirrors install.sh's update_rc_files
# loop. If install.sh's list ever changes, update this to match.
SPOTNIK_RC_CANDIDATES=(
    "$HOME/.bashrc"
    "$HOME/.zshrc"
    "$HOME/.bash_profile"
    "$HOME/.profile"
)

# Print every candidate rc file currently present in $HOME, one per line.
# Empty output means no rc files exist in this image.
present_rc_files() {
    local rc
    for rc in "${SPOTNIK_RC_CANDIDATES[@]}"; do
        [ -f "$rc" ] && printf '%s\n' "$rc"
    done
}

# Run install.sh with a pinned version. Captures status + output.
run_install_pinned() {
    SPOTNIK_VERSION="$SPOTNIK_TEST_VERSION" bash "$HOME/install.sh"
}

# Run install.sh with no version (latest path).
run_install_latest() {
    bash "$HOME/install.sh"
}

# Run uninstall.sh (assumes binary exists).
run_uninstall() {
    bash "$HOME/uninstall.sh"
}

# Assert a marker block exists exactly once in the given rc file.
assert_marker_block() {
    local rc="$1"
    [ -f "$rc" ] || { echo "rc file missing: $rc" >&2; return 1; }
    local count
    count="$(grep -c "$SPOTNIK_RC_MARKER_OPEN_RE" "$rc" || true)"
    [ "$count" = "1" ] || { echo "marker count in $rc = $count, want 1" >&2; return 1; }
}

# Assert a marker block does NOT exist in the given rc file.
refute_marker_block() {
    local rc="$1"
    [ -f "$rc" ] || return 0
    grep -q "$SPOTNIK_RC_MARKER_OPEN_RE" "$rc" \
        && { echo "marker unexpectedly present in $rc" >&2; return 1; } || true
}

# Strip the spotnik marker block from an rc file. Idempotent.
# Uses tempfile + mv rather than sed -i for portability across BSD (macOS)
# and GNU sed — BSD requires an extension argument after -i, GNU does not.
strip_marker_block() {
    local rc="$1"
    [ -f "$rc" ] || return 0
    local tmp; tmp="$(mktemp)"
    sed "/${SPOTNIK_RC_MARKER_OPEN_RE}/,/${SPOTNIK_RC_MARKER_CLOSE_RE}/d" "$rc" > "$tmp"
    mv "$tmp" "$rc"
}

# Assert spotnik resolves on PATH after sourcing the env file.
assert_spotnik_on_path_after_source() {
    [ -f "$HOME/.config/spotnik/env" ] || { echo "env file missing" >&2; return 1; }
    # shellcheck source=/dev/null
    . "$HOME/.config/spotnik/env"
    command -v spotnik >/dev/null || { echo "spotnik not on PATH" >&2; return 1; }
}
