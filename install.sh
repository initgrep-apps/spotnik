#!/bin/bash
set -euo pipefail

# spotnik installer — macOS and Linux
# Usage: curl -fsSL https://raw.githubusercontent.com/initgrep-apps/spotnik/main/install.sh | bash
# Env:   SPOTNIK_VERSION=v0.1.0   pin a release (default: latest)
#        SPOTNIK_INSTALL_DIR=/path override install destination
#        SPOTNIK_NO_MODIFY_PATH=1  suppress PATH warning

BOLD='\033[1m'
SUCCESS='\033[38;2;0;229;204m'
WARN='\033[38;2;255;176;32m'
ERROR='\033[38;2;230;57;70m'
MUTED='\033[38;2;90;100;128m'
NC='\033[0m'

ui_banner()  { echo -e "${BOLD}  spotnik installer${NC}"; echo ""; }
ui_success() { echo -e "${SUCCESS}✓${NC} $*"; }
ui_warn()    { echo -e "${WARN}!${NC} $*"; }
ui_error()   { echo -e "${ERROR}✗${NC} $*" >&2; }
ui_info()    { echo -e "${MUTED}·${NC} $*"; }

TMPDIRS=()
cleanup() { local d; for d in "${TMPDIRS[@]:-}"; do rm -rf "$d" 2>/dev/null || true; done; }
trap cleanup EXIT

mktmpdir() { local d; d="$(mktemp -d)"; TMPDIRS+=("$d"); echo "$d"; }

detect_os() {
    case "$(uname -s)" in
        Darwin) echo "darwin" ;;
        Linux)  echo "linux"  ;;
        *)
            ui_error "Unsupported OS: $(uname -s)"
            echo "For Windows use: powershell -c \"irm https://raw.githubusercontent.com/initgrep-apps/spotnik/main/install.ps1 | iex\"" >&2
            exit 1
            ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)  echo "amd64" ;;
        arm64|aarch64) echo "arm64" ;;
        *)
            ui_error "Unsupported architecture: $(uname -m)"
            exit 1
            ;;
    esac
}

resolve_version() {
    if [[ -n "${SPOTNIK_VERSION:-}" ]]; then
        echo "$SPOTNIK_VERSION"
        return
    fi
    local response
    if ! response="$(curl -fsSL "https://api.github.com/repos/initgrep-apps/spotnik/releases/latest" 2>&1)"; then
        ui_error "Failed to query GitHub API: $response"
        ui_info "Workaround: pin a version, e.g. SPOTNIK_VERSION=v0.1.0 curl ... | bash"
        exit 1
    fi
    local version
    version="$(echo "$response" | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')"
    if [[ -z "$version" ]]; then
        ui_error "Could not parse tag_name from GitHub API response"
        exit 1
    fi
    echo "$version"
}

verify_checksum() {
    local dir="$1" checksums="$2"
    if command -v sha256sum >/dev/null 2>&1; then
        (cd "$dir" && sha256sum --ignore-missing -c "$checksums")
    elif command -v shasum >/dev/null 2>&1; then
        (cd "$dir" && shasum -a 256 --ignore-missing -c "$checksums")
    else
        ui_error "Neither sha256sum nor shasum found — refusing to install without verification"
        return 1
    fi
}

path_contains() { case ":${PATH}:" in *":${1}:"*) return 0 ;; *) return 1 ;; esac; }

resolve_install_dir() {
    if [[ -n "${SPOTNIK_INSTALL_DIR:-}" ]]; then
        mkdir -p "$SPOTNIK_INSTALL_DIR"
        echo "$SPOTNIK_INSTALL_DIR"
        return
    fi
    local user_bin="$HOME/.local/bin"
    mkdir -p "$user_bin" 2>/dev/null || true
    if [[ -w "$user_bin" ]]; then
        echo "$user_bin"
    else
        echo "/usr/local/bin"
    fi
}

main() {
    ui_banner

    local os arch version
    os="$(detect_os)";       ui_success "OS: $os"
    arch="$(detect_arch)";   ui_success "Arch: $arch"

    ui_info "Resolving version..."
    version="$(resolve_version)"; ui_success "Version: $version"

    # GoReleaser strips the leading 'v' from {{.Version}} in artifact names,
    # but the GitHub release tag (and download URL path) keeps it.
    local version_num="${version#v}"
    local tarball="spotnik_${version_num}_${os}_${arch}.tar.gz"
    local checksums="spotnik_${version_num}_checksums.txt"
    local base_url="https://github.com/initgrep-apps/spotnik/releases/download/${version}"

    local tmpdir; tmpdir="$(mktmpdir)"

    ui_info "Downloading $tarball..."
    curl -fsSL --retry 3 -o "$tmpdir/$tarball"   "$base_url/$tarball"
    curl -fsSL --retry 3 -o "$tmpdir/$checksums" "$base_url/$checksums"
    ui_success "Downloaded"

    ui_info "Verifying checksum..."
    if ! verify_checksum "$tmpdir" "$checksums"; then
        ui_error "Checksum mismatch — aborting"
        exit 1
    fi
    ui_success "Checksum OK"

    ui_info "Extracting..."
    tar -xzf "$tmpdir/$tarball" -C "$tmpdir"
    ui_success "Extracted"

    if [[ ! -f "$tmpdir/spotnik" ]]; then
        ui_error "spotnik binary not found in $tarball after extraction"
        ui_info "The release artifact may be corrupt — please file an issue"
        exit 1
    fi

    local install_dir; install_dir="$(resolve_install_dir)"
    local need_sudo=false
    [[ -w "$install_dir" ]] || need_sudo=true

    ui_info "Installing to $install_dir..."
    chmod +x "$tmpdir/spotnik"
    if [[ "$need_sudo" == "true" ]]; then
        ui_info "sudo required for $install_dir"
        if ! sudo mv "$tmpdir/spotnik" "$install_dir/spotnik" </dev/tty; then
            ui_error "Failed to install to $install_dir (sudo cancelled or unavailable)"
            ui_info "Override the destination: SPOTNIK_INSTALL_DIR=\$HOME/.local/bin curl ... | bash"
            exit 1
        fi
    else
        mv "$tmpdir/spotnik" "$install_dir/spotnik"
    fi
    ui_success "Installed $install_dir/spotnik"

    if [[ "${SPOTNIK_NO_MODIFY_PATH:-0}" != "1" ]] && ! path_contains "$install_dir"; then
        echo ""
        ui_warn "$install_dir is not in your PATH. Add it:"
        echo -e "  export PATH=\"$install_dir:\$PATH\""
        echo "  Append this to ~/.bashrc or ~/.zshrc, then restart your shell."
    fi

    echo ""
    local installed_version
    installed_version="$("$install_dir/spotnik" --version 2>/dev/null || echo "spotnik ${version}")"
    ui_success "$installed_version"
    echo -e "\n${BOLD}  Run: spotnik${NC}\n"
}

main
