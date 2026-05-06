#!/bin/bash
set -euo pipefail

# spotnik uninstaller — macOS and Linux
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/initgrep-apps/spotnik/main/uninstall.sh | bash
# Env:
#   SPOTNIK_INSTALL_DIR=/path  prefer this dir when locating the binary
#   SPOTNIK_PURGE_CONFIG=1     also delete ~/.config/spotnik (default: prompt)
#   SPOTNIK_KEEP_CONFIG=1      skip config deletion (default: prompt)

BOLD='\033[1m'
SUCCESS='\033[38;2;0;229;204m'
WARN='\033[38;2;255;176;32m'
ERROR='\033[38;2;230;57;70m'
MUTED='\033[38;2;90;100;128m'
NC='\033[0m'

ui_banner()  { echo -e "${BOLD}  spotnik uninstaller${NC}"; echo ""; }
ui_success() { echo -e "${SUCCESS}✓${NC} $*"; }
ui_warn()    { echo -e "${WARN}!${NC} $*"; }
ui_error()   { echo -e "${ERROR}✗${NC} $*" >&2; }
ui_info()    { echo -e "${MUTED}·${NC} $*"; }

CONFIG_DIR="$HOME/.config/spotnik"

find_binary() {
    if [[ -n "${SPOTNIK_INSTALL_DIR:-}" && -x "$SPOTNIK_INSTALL_DIR/spotnik" ]]; then
        echo "$SPOTNIK_INSTALL_DIR/spotnik"
        return
    fi
    local resolved
    if resolved="$(command -v spotnik 2>/dev/null)"; then
        echo "$resolved"
        return
    fi
    local candidate
    for candidate in "$HOME/.local/bin/spotnik" "/usr/local/bin/spotnik" "/opt/homebrew/bin/spotnik"; do
        if [[ -x "$candidate" ]]; then
            echo "$candidate"
            return
        fi
    done
    return 1
}

forget_credentials() {
    local bin="$1"
    local stderr_capture rc=0
    ui_info "Wiping tokens and client ID from keychain (spotnik auth forget)..."
    stderr_capture="$("$bin" auth forget </dev/tty 2>&1 >/dev/null)" || rc=$?
    if [[ $rc -eq 0 ]]; then
        ui_success "Credentials wiped"
    else
        ui_warn "spotnik auth forget exited $rc. Continuing with uninstall."
        if [[ -n "$stderr_capture" ]]; then
            ui_info "stderr: $stderr_capture"
        fi
    fi
}

remove_binary() {
    local bin="$1"
    local dir; dir="$(dirname "$bin")"
    ui_info "Removing $bin..."
    if [[ -w "$dir" ]]; then
        rm -f "$bin"
    else
        ui_info "sudo required for $dir"
        sudo rm -f "$bin" </dev/tty
    fi
    ui_success "Removed $bin"
}

handle_config() {
    if [[ ! -d "$CONFIG_DIR" ]]; then
        ui_info "No config dir at $CONFIG_DIR"
        return
    fi
    if [[ "${SPOTNIK_KEEP_CONFIG:-0}" == "1" ]]; then
        ui_info "Keeping config dir: $CONFIG_DIR"
        return
    fi
    if [[ "${SPOTNIK_PURGE_CONFIG:-0}" == "1" ]]; then
        rm -rf "$CONFIG_DIR"
        ui_success "Removed $CONFIG_DIR"
        return
    fi
    if [[ ! -t 0 && ! -e /dev/tty ]]; then
        ui_warn "Config dir kept ($CONFIG_DIR). Re-run with SPOTNIK_PURGE_CONFIG=1 to delete it."
        return
    fi
    echo ""
    read -r -p "  Also remove $CONFIG_DIR? [y/N] " ans </dev/tty || ans="n"
    case "${ans:-n}" in
        y|Y|yes|YES)
            rm -rf "$CONFIG_DIR"
            ui_success "Removed $CONFIG_DIR"
            ;;
        *)
            ui_info "Kept $CONFIG_DIR"
            ;;
    esac
}

remove_env_file() {
    local env_file="$HOME/.config/spotnik/env"
    if [ -f "$env_file" ]; then
        rm -f "$env_file"
        ui_success "Removed $env_file"
    else
        ui_info "No env file at $env_file"
    fi
    rmdir "$HOME/.config/spotnik" 2>/dev/null || true
}

strip_rc_block() {
    local rc="$1"
    [ -f "$rc" ] || return 0
    grep -qF '# >>> spotnik installer >>>' "$rc" || return 0
    # Use awk to drop lines between markers (inclusive). Drop a single
    # leading blank line if it precedes the marker.
    local tmp; tmp="$(mktemp)"
    awk '
        BEGIN { skip = 0; held_blank = 0 }
        /^# >>> spotnik installer >>>$/ { skip = 1; held_blank = 0; next }
        /^# <<< spotnik installer <<<$/ { skip = 0; next }
        skip == 1 { next }
        /^$/ { held_blank = 1; next }
        { if (held_blank) { print ""; held_blank = 0 } print }
    ' "$rc" > "$tmp"
    # Sanity: the awk output must not contain either marker. Protects
    # against an awk regression that fails to strip the block while
    # producing output. False-positives an installer-only rc (legitimately
    # produces empty output) would have hit the byte-count check we tried
    # earlier — this marker-presence check correctly accepts that case.
    if grep -qF '# >>> spotnik installer >>>' "$tmp" 2>/dev/null \
        || grep -qF '# <<< spotnik installer <<<' "$tmp" 2>/dev/null; then
        ui_error "Refusing to overwrite $rc — markers still present after strip (awk anomaly)"
        rm -f "$tmp"
        return 1
    fi
    mv "$tmp" "$rc"
    ui_success "Cleaned $rc"
}

strip_all_rc_files() {
    local rc cleaned=0 rc_present=0
    for rc in "$HOME/.bashrc" "$HOME/.zshrc" "$HOME/.bash_profile" "$HOME/.profile"; do
        [ -f "$rc" ] || continue
        rc_present=$((rc_present + 1))
        if grep -qF '# >>> spotnik installer >>>' "$rc" 2>/dev/null; then
            strip_rc_block "$rc"
            cleaned=$((cleaned + 1))
        fi
    done
    if [ "$rc_present" -eq 0 ]; then
        ui_info "No POSIX rc files present"
    elif [ "$cleaned" -eq 0 ]; then
        ui_info "No installer-managed lines in rc files (checked $rc_present)"
    fi
}

remove_fish_conf() {
    local conf="$HOME/.config/fish/conf.d/spotnik.fish"
    if [ -f "$conf" ]; then
        rm -f "$conf"
        ui_success "Removed $conf"
    else
        ui_info "No fish conf at $conf"
    fi
}

main() {
    ui_banner

    local bin
    if bin="$(find_binary)"; then
        ui_success "Found: $bin"
        forget_credentials "$bin"
        remove_binary "$bin"
    else
        ui_warn "spotnik binary not found in PATH or common install locations"
        ui_info "Continuing with config + rc cleanup."
    fi

    remove_env_file
    strip_all_rc_files
    remove_fish_conf
    handle_config

    echo ""
    ui_success "Uninstall complete."
}

main "$@"
