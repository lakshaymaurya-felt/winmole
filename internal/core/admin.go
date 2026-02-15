package core

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/windows"
)

// IsElevated returns true if the current process is running with
// administrator privileges.
func IsElevated() bool {
	token := windows.GetCurrentProcessToken()
	return token.IsElevated()
}

// RequireAdmin returns an error if the current process is not elevated.
// The operation parameter is included in the error message for context.
func RequireAdmin(operation string) error {
	if IsElevated() {
		return nil
	}
	return fmt.Errorf(
		"operation %q requires administrator privileges\n"+
			"  → Re-run WinMole in an elevated terminal:\n"+
			"    Right-click Terminal → Run as Administrator\n"+
			"    Or: gsudo wm %s",
		operation, operation,
	)
}

// RunElevated is a placeholder for future UAC elevation support.
// It will re-launch the current process elevated via ShellExecuteEx.
// For now it returns an instructional error — WinMole does not
// auto-elevate per design decision.
func RunElevated(args []string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine executable path: %w", err)
	}

	// NOTE: Actual ShellExecuteEx("runas") implementation deferred.
	// This avoids surprise UAC prompts; the user should explicitly
	// open an admin terminal.
	return fmt.Errorf(
		"auto-elevation is not yet implemented\n"+
			"  → Please re-run as administrator:\n"+
			"    %s %s",
		exe, strings.Join(args, " "),
	)
}
