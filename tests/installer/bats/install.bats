#!/usr/bin/env bats

load 'helpers'

@test "fresh install (pinned) lands binary and it is executable" {
    run_install_pinned
    [ -x "$TEST_INSTALL_DIR/spotnik" ]
    run "$TEST_INSTALL_DIR/spotnik" --version
    [ "$status" -eq 0 ]
    [[ "$output" == *"v0.1.0-rc1"* ]]
}
