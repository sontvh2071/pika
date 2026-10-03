// Package diskfree reads GNU df in human-readable mode without modifying storage.
package diskfree

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Filesystem struct {
	Source    string `json:"source"`
	Type      string `json:"type"`
	Total     string `json:"total"`
	Used      string `json:"used"`
	Available string `json:"available"`
	Percent   *int   `json:"percent"`
	Mount     string `json:"mount"`
	Virtual   bool   `json:"virtual"`
}
type Snapshot struct {
	Filesystems []Filesystem `json:"filesystems"`
	UpdatedAt   int64        `json:"updated_at"`
	Stale       bool         `json:"stale"`
	Message     string       `json:"message"`
}
type Client struct {
	mu      sync.Mutex
	last    Snapshot
	checked time.Time
	read    func(context.Context) ([]Filesystem, error)
}

func New() *Client { return &Client{read: readCommand} }
func (c *Client) Read(ctx context.Context) Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.checked.IsZero() && time.Since(c.checked) < 1500*time.Millisecond {
		return c.last
	}
	rows, err := c.read(ctx)
	c.checked = time.Now()
	if err != nil {
		c.last.Stale = c.last.UpdatedAt != 0
		c.last.Message = err.Error()
		return c.last
	}
	c.last = Snapshot{Filesystems: rows, UpdatedAt: c.checked.Unix()}
	if len(rows) == 0 {
		c.last.Message = "No mounted filesystems reported by df."
	}
	return c.last
}
func readCommand(ctx context.Context) ([]Filesystem, error) {
	path, err := exec.LookPath("df")
	if err != nil {
		return nil, fmt.Errorf("GNU df is not installed. Install coreutils to view disk space.")
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "-h", "--output=source,fstype,size,used,avail,pcent,target")
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	cmd.WaitDelay = 200 * time.Millisecond
	data, err := cmd.Output()
	if ctx.Err() != nil {
		return nil, fmt.Errorf("Disk read timed out or was cancelled. A mounted drive may be unavailable.")
	}
	if err != nil {
		return nil, fmt.Errorf("Cannot read disk space. Check mounted drives and GNU df, then refresh.")
	}
	if len(data) > 1024*1024 {
		return nil, fmt.Errorf("Disk response is too large.")
	}
	return Parse(data)
}

// Numeric columns delimit source/mount names which can themselves contain spaces.
const amount = `(-?[0-9]+(?:\.[0-9]+)?[KMGTPEZYRQ]?|-)`

var rowPattern = regexp.MustCompile(`^(.+?)\s+(\S+)\s+` + amount + `\s+` + amount + `\s+` + amount + `\s+([0-9]+%|-)\s+(.+)$`)

func Parse(data []byte) ([]Filesystem, error) {
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 0 || !strings.HasPrefix(lines[0], "Filesystem") || !strings.Contains(lines[0], "Mounted on") {
		return nil, fmt.Errorf("Invalid df response.")
	}
	rows := []Filesystem{}
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m := rowPattern.FindStringSubmatch(line)
		if m == nil {
			return nil, fmt.Errorf("Unrecognized df output. Disk readings were not updated.")
		}
		r := Filesystem{Source: m[1], Type: m[2], Total: m[3], Used: m[4], Available: m[5], Mount: m[7]}
		if m[6] != "-" {
			v, err := strconv.Atoi(strings.TrimSuffix(m[6], "%"))
			if err != nil {
				return nil, fmt.Errorf("Invalid disk percentage.")
			}
			r.Percent = &v
		}
		switch r.Type {
		case "tmpfs", "devtmpfs", "efivarfs", "squashfs", "ramfs", "proc", "sysfs", "devpts", "cgroup", "cgroup2":
			r.Virtual = true
		}
		rows = append(rows, r)
	}
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if a.Virtual != b.Virtual {
			return !a.Virtual
		}
		if a.Mount == "/" || b.Mount == "/" {
			return a.Mount == "/" && b.Mount != "/"
		}
		return a.Mount < b.Mount
	})
	return rows, nil
}
