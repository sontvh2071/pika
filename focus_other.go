//go:build !linux

package main

func installLauncherPresentation() {}
func presentLauncher()             {}

func focusLauncher()                             {}
func cancelLauncherFocus()                       {}
func nativeFocusState() (bool, bool, bool, bool) { return false, false, false, true }
