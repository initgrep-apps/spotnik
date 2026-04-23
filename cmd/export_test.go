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

// ValidateClientID is a test-only export of validateClientID.
var ValidateClientID = validateClientID
