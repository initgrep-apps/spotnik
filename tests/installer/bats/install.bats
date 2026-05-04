#!/usr/bin/env bats

load 'helpers'

@test "fresh install (pinned) lands binary and it is executable" {
    run_install_pinned
    [ -x "$TEST_INSTALL_DIR/spotnik" ]
    run "$TEST_INSTALL_DIR/spotnik" --version
    [ "$status" -eq 0 ]
    [[ "$output" == *"v0.1.0-rc1"* ]]
}

@test "env file is created with case-guarded PATH export" {
    run_install_pinned
    [ -f "$HOME/.config/spotnik/env" ]
    grep -q 'case ":\${PATH}:" in' "$HOME/.config/spotnik/env"
    grep -q "export PATH=\"\$HOME/.local/bin:\$PATH\"" "$HOME/.config/spotnik/env"
}

@test "sourcing env file makes spotnik resolvable" {
    run_install_pinned
    assert_spotnik_on_path_after_source
}

@test "marker block written exactly once to ~/.bashrc on Ubuntu" {
    [ ! -d "$HOME/.config/fish" ] || skip "fish image — rc files intentionally not edited"
    # Ubuntu image creates .bashrc by default for the tester user.
    [ -f "$HOME/.bashrc" ]
    run_install_pinned
    assert_marker_block "$HOME/.bashrc"
    grep -q '\. "\$HOME/.config/spotnik/env"' "$HOME/.bashrc"
}

@test "idempotent reinstall does not duplicate marker block" {
    [ ! -d "$HOME/.config/fish" ] || skip "fish image — rc files intentionally not edited"
    run_install_pinned
    cp "$HOME/.bashrc" "$HOME/.bashrc.snapshot"
    run_install_pinned
    diff -u "$HOME/.bashrc.snapshot" "$HOME/.bashrc"
}

@test "fish-only env writes conf.d/spotnik.fish and skips rc files" {
    [ -d "$HOME/.config/fish" ] || skip "not a fish image"
    run_install_pinned
    [ -f "$HOME/.config/fish/conf.d/spotnik.fish" ]
    grep -q 'fish_add_path' "$HOME/.config/fish/conf.d/spotnik.fish"
    refute_marker_block "$HOME/.bashrc"
    refute_marker_block "$HOME/.zshrc"
}

@test "SPOTNIK_NO_MODIFY_PATH=1 skips env file and rc edits" {
    # bats does not isolate $HOME between tests; clear residue from earlier
    # cases so the absence-assertions are meaningful.
    rm -rf "$HOME/.config/spotnik"
    if [ -f "$HOME/.bashrc" ]; then
        sed -i '/^# >>> spotnik installer >>>$/,/^# <<< spotnik installer <<<$/d' "$HOME/.bashrc"
    fi
    SPOTNIK_VERSION="$SPOTNIK_TEST_VERSION" SPOTNIK_NO_MODIFY_PATH=1 \
        bash "$HOME/install.sh"
    [ ! -f "$HOME/.config/spotnik/env" ]
    refute_marker_block "$HOME/.bashrc"
}

@test "SPOTNIK_INSTALL_DIR overrides env file path" {
    local custom="$HOME/altbin"
    SPOTNIK_VERSION="$SPOTNIK_TEST_VERSION" SPOTNIK_INSTALL_DIR="$custom" \
        bash "$HOME/install.sh"
    [ -x "$custom/spotnik" ]
    grep -q "export PATH=\"\$HOME/altbin:\$PATH\"" "$HOME/.config/spotnik/env"
}

@test "latest smoke: no version pin produces an executable binary" {
    # GitHub's /releases/latest skips pre-releases; until a stable tag ships
    # this exercises a 404 path. Skip rather than red.
    curl -fsSL "https://api.github.com/repos/initgrep-apps/spotnik/releases/latest" \
        >/dev/null 2>&1 || skip "no stable (non-pre) release published yet"
    run_install_latest
    [ -x "$TEST_INSTALL_DIR/spotnik" ]
    run "$TEST_INSTALL_DIR/spotnik" --version
    [ "$status" -eq 0 ]
    [[ "$output" == spotnik* ]]
}
