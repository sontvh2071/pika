package catalog

import "os/exec"

// Built-in session actions are available even when personal commands are off.
func CollectSystemActions() []Candidate {
	out := []Candidate{}
	for _, c := range []Candidate{
		{ID: "system:lock", Kind: "system", Name: "Lock", Target: "lock", Subtitle: "Lock the screen", Aliases: []string{"lock screen", "khoa", "khoa man hinh"}, Icon: "system-lock-screen"},
		{ID: "system:logout", Kind: "system", Name: "Log Out", Target: "logout", Subtitle: "Choose Log Out, Switch User or Cancel", Aliases: []string{"logout", "log out", "dang xuat", "switch user"}, Icon: "system-log-out"},
		{ID: "system:shutdown", Kind: "system", Name: "Shut Down", Target: "shutdown", Subtitle: "Choose Suspend, Restart, Shut Down or Cancel", Aliases: []string{"shutdown", "shut down", "tat may", "power", "restart", "suspend"}, Icon: "system-shutdown"},
	} {
		executable := "cinnamon-session-quit"
		if c.Target == "lock" {
			executable = "cinnamon-screensaver-command"
		}
		path, err := exec.LookPath(executable)
		if err != nil {
			continue
		}
		c.Path = path
		out = append(out, Prepare(c))
	}
	return out
}
