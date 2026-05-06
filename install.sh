#!/bin/bash
set -euo pipefail

# spotnik installer — macOS and Linux
# Usage:
#   Latest stable:  curl -fsSL https://raw.githubusercontent.com/initgrep-apps/spotnik/main/install.sh | bash
#   Pinned:         curl -fsSL https://raw.githubusercontent.com/initgrep-apps/spotnik/main/install.sh | bash -s v0.1.0
# Env:
#   SPOTNIK_VERSION=v0.1.0    pin a release (alternative to positional arg)
#   SPOTNIK_INSTALL_DIR=/path override install destination
#   SPOTNIK_NO_MODIFY_PATH=1  do not write env file or modify shell init files
#
# Positional arg wins over env var. Default = latest stable (skips pre-releases).

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
    local arg="${1:-}"
    if [[ -n "$arg" ]]; then
        echo "$arg"
        return
    fi
    if [[ -n "${SPOTNIK_VERSION:-}" ]]; then
        echo "$SPOTNIK_VERSION"
        return
    fi
    local response matched version
    if ! response="$(curl -fsSL "https://api.github.com/repos/initgrep-apps/spotnik/releases/latest" 2>&1)"; then
        ui_error "Failed to query GitHub API: $response"
        ui_info "Workaround: pin a version, e.g. SPOTNIK_VERSION=v0.1.0 curl ... | bash"
        exit 1
    fi
    matched="$(printf '%s' "$response" | grep '"tag_name"' || true)"
    if [[ -z "$matched" ]]; then
        ui_error "Could not find tag_name in GitHub API response"
        ui_info "Response (first 200 chars): ${response:0:200}"
        ui_info "Workaround: pin a version, e.g. SPOTNIK_VERSION=v0.1.0 curl ... | bash"
        exit 1
    fi
    version="$(printf '%s' "$matched" | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')"
    if [[ -z "$version" ]]; then
        ui_error "Could not parse tag_name value from: $matched"
        exit 1
    fi
    echo "$version"
}

verify_checksum() {
    local dir="$1" checksums="$2" tarball="$3"
    # Pre-flight: refuse to install if checksums.txt has no entry for our
    # tarball. Without this, --ignore-missing would silently exit 0 on a
    # malformed/empty checksums file.
    if ! grep -qE "(^| )${tarball}\$" "$dir/$checksums"; then
        ui_error "checksums.txt has no entry for $tarball — refusing to install"
        return 1
    fi
    # Keep --ignore-missing because checksums.txt lists all release artifacts
    # (Linux/macOS/Windows × amd64/arm64), and we only ever have one locally.
    # The trailing grep asserts our tarball's line printed "OK" — defense
    # in depth on top of the pre-flight grep.
    if command -v sha256sum >/dev/null 2>&1; then
        (cd "$dir" && sha256sum --ignore-missing -c "$checksums" 2>&1 | grep -E "^${tarball}: OK\$")
    elif command -v shasum >/dev/null 2>&1; then
        (cd "$dir" && shasum -a 256 --ignore-missing -c "$checksums" 2>&1 | grep -E "^${tarball}: OK\$")
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

write_env_file() {
    local install_dir="$1"
    local env_dir="$HOME/.config/spotnik"
    local env_file="$env_dir/env"
    mkdir -p "$env_dir"
    # Render $HOME-relative paths as a literal $HOME token so the env file
    # stays portable (rustup-style).
    local path_literal="$install_dir"
    if [[ "$install_dir" == "$HOME"/* ]]; then
        path_literal="\$HOME${install_dir#"$HOME"}"
    fi
    # Use printf rather than heredoc to keep the case-guard literal.
    # shellcheck disable=SC2016  # the $PATH/${PATH} tokens MUST stay literal
    {
        printf '%s\n' '# Managed by the spotnik installer.'
        printf '%s\n' '# Edits will be overwritten on reinstall.'
        printf 'case ":${PATH}:" in\n'
        printf '    *":%s:"*) ;;\n' "$path_literal"
        printf '    *) export PATH="%s:$PATH" ;;\n' "$path_literal"
        printf 'esac\n'
    } > "$env_file"
}

RC_MARKER_OPEN='# >>> spotnik installer >>>'
# shellcheck disable=SC2034  # paired with RC_MARKER_OPEN; intent is symmetry
RC_MARKER_CLOSE='# <<< spotnik installer <<<'
# shellcheck disable=SC2016  # $HOME stays literal; expands at source-time
RC_BLOCK='# >>> spotnik installer >>>
. "$HOME/.config/spotnik/env"
# <<< spotnik installer <<<'

# True if rc file already contains a marker block.
rc_has_marker() {
    local rc="$1"
    [ -f "$rc" ] && grep -qF "$RC_MARKER_OPEN" "$rc"
}

# Append the marker block to rc if missing. Idempotent.
update_rc_file() {
    local rc="$1"
    if rc_has_marker "$rc"; then
        return 0
    fi
    # Ensure file ends with a newline before appending.
    if [ -s "$rc" ] && [ "$(tail -c1 "$rc"; echo x)" != $'\nx' ]; then
        printf '\n' >> "$rc"
    fi
    printf '\n%s\n' "$RC_BLOCK" >> "$rc"
    ui_success "Updated $rc"
}

# Edit each existing rc file. If none exist, create one matching $SHELL.
update_rc_files() {
    local edited=0
    local rc
    for rc in "$HOME/.bashrc" "$HOME/.zshrc" "$HOME/.bash_profile" "$HOME/.profile"; do
        if [ -f "$rc" ]; then
            update_rc_file "$rc"
            edited=1
        fi
    done
    if [ "$edited" = "0" ]; then
        local shell_name; shell_name="$(basename "${SHELL:-/bin/bash}")"
        case "$shell_name" in
            zsh)  rc="$HOME/.zshrc" ;;
            *)    rc="$HOME/.bashrc" ;;
        esac
        : > "$rc"
        update_rc_file "$rc"
    fi
}

write_fish_conf() {
    local install_dir="$1"
    local conf_dir="$HOME/.config/fish/conf.d"
    mkdir -p "$conf_dir"
    cat > "$conf_dir/spotnik.fish" <<EOF
# Managed by the spotnik installer.
if not contains -- '$install_dir' \$PATH
    fish_add_path -g '$install_dir'
end
EOF
    ui_success "Added $conf_dir/spotnik.fish"
}

# True if the user has fish configured (directory exists).
has_fish_config() {
    [ -d "$HOME/.config/fish" ]
}

main() {
    ui_banner

    local os arch version
    os="$(detect_os)";       ui_success "OS: $os"
    arch="$(detect_arch)";   ui_success "Arch: $arch"

    ui_info "Resolving version..."
    version="$(resolve_version "${1:-}")"; ui_success "Version: $version"

    # GoReleaser strips the leading 'v' from {{.Version}} in artifact names,
    # but the GitHub release tag (and download URL path) keeps it.
    local version_num="${version#v}"
    local tarball="spotnik_${version_num}_${os}_${arch}.tar.gz"
    local checksums="checksums.txt"
    local base_url="https://github.com/initgrep-apps/spotnik/releases/download/${version}"

    local tmpdir; tmpdir="$(mktmpdir)"

    ui_info "Downloading $tarball..."
    if ! curl -fsSL --retry 3 -o "$tmpdir/$tarball" "$base_url/$tarball"; then
        ui_error "Failed to download $tarball from $base_url"
        ui_info "Check https://github.com/initgrep-apps/spotnik/releases for available versions."
        exit 1
    fi
    ui_info "Downloading $checksums..."
    if ! curl -fsSL --retry 3 -o "$tmpdir/$checksums" "$base_url/$checksums"; then
        ui_error "Failed to download $checksums from $base_url"
        ui_info "The release may be incompletely published. File an issue if persistent."
        exit 1
    fi
    ui_success "Downloaded"

    ui_info "Verifying checksum..."
    if ! verify_checksum "$tmpdir" "$checksums" "$tarball"; then
        ui_error "Checksum mismatch — aborting"
        exit 1
    fi
    ui_success "Checksum OK"

    ui_info "Extracting..."
    if ! tar -xzf "$tmpdir/$tarball" -C "$tmpdir" 2> "$tmpdir/tar.err"; then
        ui_error "Failed to extract $tarball: $(cat "$tmpdir/tar.err" 2>/dev/null)"
        ui_info "The downloaded archive may be corrupt — please retry or file an issue."
        exit 1
    fi
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
            ui_info "Override the destination: SPOTNIK_INSTALL_DIR=\$HOME/.local/bin bash -c \"\$(curl -fsSL ...)\""
            exit 1
        fi
    else
        mv "$tmpdir/spotnik" "$install_dir/spotnik"
    fi
    ui_success "Installed $install_dir/spotnik"

    if [[ "${SPOTNIK_NO_MODIFY_PATH:-0}" == "1" ]]; then
        if ! path_contains "$install_dir"; then
            ui_warn "$install_dir is not in your PATH (SPOTNIK_NO_MODIFY_PATH=1)."
            echo -e "  Add manually: export PATH=\"$install_dir:\$PATH\""
        fi
    else
        write_env_file "$install_dir"
        ui_success "Added $HOME/.config/spotnik/env"
        if has_fish_config; then
            write_fish_conf "$install_dir"
        else
            update_rc_files
        fi
    fi

    echo ""
    local installed_version stderr_capture rc=0
    stderr_capture="$("$install_dir/spotnik" --version 2>&1 >/dev/null)" || rc=$?
    installed_version="$("$install_dir/spotnik" --version 2>/dev/null)" || true
    if [[ $rc -ne 0 || -z "$installed_version" ]]; then
        ui_warn "Installed binary failed to run: $stderr_capture"
        ui_info "The download or your platform may be incompatible. File an issue if persistent."
    else
        ui_success "$installed_version"
    fi

    if [[ "${SPOTNIK_NO_MODIFY_PATH:-0}" != "1" ]] && ! path_contains "$install_dir"; then
        echo ""
        echo -e "${BOLD}  Activate in this shell:${NC}  . \"\$HOME/.config/spotnik/env\""
        echo -e "${MUTED}  (or open a new terminal — new shells inherit automatically)${NC}"
    fi

    echo -e "\n${BOLD}  Run: spotnik${NC}\n"
}

main "$@"
