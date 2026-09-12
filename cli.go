//go:build !bindings

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"pika/internal/config"
	"pika/internal/ipc"
	"pika/internal/launcher"
	"syscall"
	"time"
)

func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, "Pika:", e)
		os.Exit(1)
	}
}
func run() error {
	command := "show"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	if command == "--help" || command == "help" {
		fmt.Println("Pika — personal Linux launcher\n\nUsage: pika [show|toggle|hide|--background|reindex|reload-config|stats|focus-state|quit]\n       pika --check-config\n       pika --version\n\nConfig: " + config.Path())
		return nil
	}
	if command == "--version" {
		fmt.Println("Pika 0.1.0")
		return nil
	}
	if command == "--check-config" {
		_, e := config.Load(config.Path(), true)
		if e == nil {
			fmt.Println("Configuration valid:", config.Path())
		}
		return e
	}
	valid := map[string]bool{"show": true, "hide": true, "toggle": true, "--background": true, "reindex": true, "reload-config": true, "stats": true, "focus-state": true, "quit": true, "--serve-background": true}
	if !valid[command] {
		return fmt.Errorf("unknown command %q; use --help", command)
	}
	dir, e := ipc.Directory()
	if e != nil {
		return e
	}
	r := ipc.NewRequest(command)
	if command != "--serve-background" {
		if reply, e := ipc.Send(dir, r); e == nil {
			if reply.Message == "starting" {
				return waitFor(dir, r)
			}
			return printReply(reply)
		}
		if command == "hide" || command == "quit" {
			return nil
		}
		if command == "focus-state" || command == "stats" || command == "reindex" || command == "reload-config" {
			return fmt.Errorf("Pika is not running. Start it with pika show")
		}
		if len(os.Args) > 1 && command != "--background" {
			exe, e := os.Executable()
			if e != nil {
				return e
			}
			logDir := filepath.Join(config.HomePath("XDG_STATE_HOME", ".local/state"), "pika")
			if e = os.MkdirAll(logDir, 0700); e != nil {
				return e
			}
			logPath := filepath.Join(logDir, "pika.log")
			if info, e := os.Stat(logPath); e == nil && info.Size() > 2*1024*1024 {
				_ = os.Rename(logPath, logPath+".old")
			}
			f, e := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
			if e != nil {
				return e
			}
			defer f.Close()
			child := exec.Command(exe, "--serve-background")
			child.Stdout = f
			child.Stderr = f
			child.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
			if e = child.Start(); e != nil {
				return e
			}
			go child.Wait()
			return waitFor(dir, r)
		}
	}
	server, e := ipc.Acquire(dir)
	if e != nil {
		if command == "--serve-background" {
			return nil
		}
		return waitFor(dir, r)
	}
	cfg, e := config.Load(config.Path(), true)
	if e != nil {
		server.Close()
		return e
	}
	svc := launcher.New(cfg, config.Path(), config.DataPath())
	app := &App{service: svc, server: server, initialShow: command != "--background" && command != "--serve-background"}
	return runWindow(app, cfg)
}
func waitFor(dir string, r ipc.Request) error {
	until := time.Now().Add(5 * time.Second)
	for time.Now().Before(until) {
		reply, e := ipc.Send(dir, r)
		if e == nil && reply.Message != "starting" {
			return printReply(reply)
		}
		time.Sleep(80 * time.Millisecond)
	}
	return fmt.Errorf("Pika did not become ready in 5 seconds. Run pika in a terminal; check ~/.local/state/pika/pika.log")
}
func printReply(r ipc.Response) error {
	if !r.OK {
		return fmt.Errorf("%s", r.Message)
	}
	if r.Data != nil {
		b, _ := json.MarshalIndent(r.Data, "", "  ")
		fmt.Println(string(b))
	} else if r.Message != "" {
		fmt.Println(r.Message)
	}
	return nil
}
