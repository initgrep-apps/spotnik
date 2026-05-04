# Pinned release used for all deterministic tests.
export SPOTNIK_TEST_VERSION="v0.1.0-rc1"

# Default install dir for all tests; can be overridden per-test.
export TEST_INSTALL_DIR="$HOME/.local/bin"

# Marker text — single source of truth for the test layer. If install.sh's
# RC_MARKER_OPEN/RC_MARKER_CLOSE ever change, update these to match.
export SPOTNIK_RC_MARKER_OPEN_RE='^# >>> spotnik installer >>>$'
export SPOTNIK_RC_MARKER_CLOSE_RE='^# <<< spotnik installer <<<$'

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
strip_marker_block() {
    local rc="$1"
    [ -f "$rc" ] || return 0
    sed -i "/${SPOTNIK_RC_MARKER_OPEN_RE}/,/${SPOTNIK_RC_MARKER_CLOSE_RE}/d" "$rc"
}

# Assert spotnik resolves on PATH after sourcing the env file.
assert_spotnik_on_path_after_source() {
    [ -f "$HOME/.config/spotnik/env" ] || { echo "env file missing" >&2; return 1; }
    # shellcheck source=/dev/null
    . "$HOME/.config/spotnik/env"
    command -v spotnik >/dev/null || { echo "spotnik not on PATH" >&2; return 1; }
}
