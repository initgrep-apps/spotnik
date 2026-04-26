package cmd

import (
	"io"

	"github.com/spf13/cobra"
)

// RunRegister is a test-only shim that exposes runRegister to the cmd_test
// package so tests can exercise the prompt and validator wiring without going
// through full Cobra dispatch.
func RunRegister(c *cobra.Command, r io.Reader) error {
	return runRegister(c, r)
}

// ValidateClientID is a test-only shim that exposes validateClientID.
var ValidateClientID = validateClientID

// LoadConfigFromPath is a test-only shim that exposes loadConfigFromPath
// so tests can verify config loading behaviour without going through the
// full runApp path.
var LoadConfigFromPath = loadConfigFromPath
