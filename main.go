// Spotnik — a terminal Spotify client for developers.
// This file is the entry point only — no logic lives here.
package main

import "github.com/initgrep-apps/spotnik/cmd"

// version is injected at build time via LDFLAGS:
//
//	-X main.version={{.Tag}}
//
// It falls back to "dev" when running without a release build.
var version = "dev"

func main() {
	cmd.Execute(version)
}
