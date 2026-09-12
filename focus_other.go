//go:build !linux

package main

func focusLauncher()                       {}
func cancelLauncherFocus()                 {}
func nativeFocusState() (bool, bool, bool) { return false, false, false }
