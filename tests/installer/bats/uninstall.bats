#!/usr/bin/env bats

load 'helpers'

@test "uninstall round-trip leaves no spotnik traces in rc files" {
    # bats does not isolate $HOME between test files; clear any residue from
    # install.bats so the round-trip assertions are meaningful.
    strip_marker_block "$HOME/.bashrc"
    strip_marker_block "$HOME/.zshrc"
    strip_marker_block "$HOME/.bash_profile"
    strip_marker_block "$HOME/.profile"
    rm -rf "$HOME/.config/spotnik"

    run_install_pinned
    [ -f "$HOME/.config/spotnik/env" ]
    run_uninstall
    [ ! -f "$HOME/.config/spotnik/env" ]
    for rc in "$HOME/.bashrc" "$HOME/.zshrc" "$HOME/.bash_profile" "$HOME/.profile"; do
        [ -f "$rc" ] || continue
        ! grep -i 'spotnik' "$rc"
    done
}

@test "uninstall is idempotent (safe second run)" {
    strip_marker_block "$HOME/.bashrc"
    strip_marker_block "$HOME/.zshrc"
    strip_marker_block "$HOME/.bash_profile"
    strip_marker_block "$HOME/.profile"
    rm -rf "$HOME/.config/spotnik"

    run_install_pinned
    run_uninstall
    run_uninstall  # must not error
}

@test "uninstall fish-only env removes conf.d/spotnik.fish" {
    [ -d "$HOME/.config/fish" ] || skip "not a fish image"
    rm -rf "$HOME/.config/spotnik"
    rm -f "$HOME/.config/fish/conf.d/spotnik.fish"

    run_install_pinned
    [ -f "$HOME/.config/fish/conf.d/spotnik.fish" ]
    run_uninstall
    [ ! -f "$HOME/.config/fish/conf.d/spotnik.fish" ]
}
