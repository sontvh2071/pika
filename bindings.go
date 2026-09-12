//go:build bindings

package main

import (
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
)

// Generate Wails bindings without starting IPC, touching user config, or opening SQLite.
func main() { _ = wails.Run(&options.App{Bind: []interface{}{&App{}}}) }
