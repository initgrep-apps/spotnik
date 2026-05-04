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
    # Ubuntu image creates .bashrc by default for the tester user.
    [ -f "$HOME/.bashrc" ]
    run_install_pinned
    assert_marker_block "$HOME/.bashrc"
    grep -q '\. "\$HOME/.config/spotnik/env"' "$HOME/.bashrc"
}

@test "idempotent reinstall does not duplicate marker block" {
    run_install_pinned
    cp "$HOME/.bashrc" "$HOME/.bashrc.snapshot"
    run_install_pinned
    diff -u "$HOME/.bashrc.snapshot" "$HOME/.bashrc"
}
