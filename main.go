// Spotnik — a terminal Spotify client for developers.
// This file is the entry point only — no logic lives here.
package main

import "github.com/initgrep-apps/spotnik/cmd"

// version and buildTime are injected at build time via LDFLAGS:
//
//	-X main.version={{.Version}} -X main.buildTime={{.Date}}
//
// They fall back to "dev" and "" when running without a release build.
var (
	version   = "dev"
	buildTime = ""
)

func main() {
	cmd.Execute(version, buildTime)
}
