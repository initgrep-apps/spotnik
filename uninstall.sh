#!/bin/bash
set -euo pipefail

# spotnik uninstaller — macOS and Linux
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/initgrep-apps/spotnik/main/uninstall.sh | bash
# Env:
#   SPOTNIK_PURGE_CONFIG=1   also delete ~/.config/spotnik (default: prompt)
#   SPOTNIK_KEEP_CONFIG=1    skip config deletion (default: prompt)

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
    ui_info "Wiping tokens and client ID from keychain (spotnik auth forget)..."
    if "$bin" auth forget </dev/tty 2>/dev/null; then
        ui_success "Credentials wiped"
    else
        ui_warn "spotnik auth forget exited non-zero (already forgotten?). Continuing."
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

main() {
    ui_banner

    local bin
    if ! bin="$(find_binary)"; then
        ui_warn "spotnik binary not found in PATH or common install locations"
        ui_info "Nothing to uninstall."
        handle_config
        exit 0
    fi
    ui_success "Found: $bin"

    forget_credentials "$bin"
    remove_binary "$bin"
    handle_config

    echo ""
    ui_success "Uninstall complete."
}

main "$@"
