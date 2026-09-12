package launcher

import (
	"fmt"
	"os/exec"
)

// Use Cinnamon's own confirmation dialogs, never a direct power/session action.
func SystemCommand(action string) (*exec.Cmd, error) {
	switch action {
	case "lock":
		return exec.Command("cinnamon-screensaver-command", "--lock"), nil
	case "logout":
		return exec.Command("cinnamon-session-quit", "--logout"), nil
	case "shutdown":
		return exec.Command("cinnamon-session-quit", "--power-off"), nil
	default:
		return nil, fmt.Errorf("Unknown system action")
	}
}
