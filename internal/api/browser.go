package api

import (
	"fmt"
	"os/exec"
	"runtime"
)

// openBrowserPlatform opens the URL in the system browser using platform-native commands.
// It is a best-effort operation — failures are logged but do not abort the auth flow.
func openBrowserPlatform(urlStr string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{urlStr}
	case "linux":
		cmd = "xdg-open"
		args = []string{urlStr}
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", urlStr}
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return exec.Command(cmd, args...).Start()
}
