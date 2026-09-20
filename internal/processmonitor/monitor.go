// Package processmonitor provides a small, read-only Linux /proc overview.
package processmonitor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Process struct {
	PID          int      `json:"pid"`
	Name         string   `json:"name"`
	CPU          *float64 `json:"cpu"`
	Memory       uint64   `json:"memory"`
	ticks, start uint64
}
type Snapshot struct {
	CPU          *float64  `json:"cpu"`
	CPUs         int       `json:"cpus"`
	MemoryUsed   uint64    `json:"memory_used"`
	MemoryTotal  uint64    `json:"memory_total"`
	SwapUsed     uint64    `json:"swap_used"`
	SwapTotal    uint64    `json:"swap_total"`
	ProcessCount int       `json:"process_count"`
	Skipped      int       `json:"skipped"`
	Processes    []Process `json:"processes"`
	UpdatedAt    int64     `json:"updated_at"`
	Stale        bool      `json:"stale"`
	Message      string    `json:"message"`
}
type sample struct {
	total, idle                                  uint64
	cpus                                         int
	memoryUsed, memoryTotal, swapUsed, swapTotal uint64
	processes                                    map[int]Process
	skipped                                      int
}
type Client struct {
	mu       sync.Mutex
	previous sample
	checked  time.Time
	last     Snapshot
	read     func(context.Context) (sample, error)
}

func New() *Client {
	return &Client{read: func(ctx context.Context) (sample, error) { return readProc(ctx, "/proc") }}
}

// No background worker: callers sample only while the preview is visible.
func (c *Client) Read(ctx context.Context) Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.checked.IsZero() && time.Since(c.checked) < 1500*time.Millisecond {
		return c.last
	}
	ctx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()
	current, err := c.read(ctx)
	if err != nil {
		c.checked = time.Now()
		c.previous = sample{}
		c.last.Stale = c.last.UpdatedAt != 0
		c.last.Message = "Cannot read process statistics from /proc. Try refreshing."
		return c.last
	}
	previous := c.previous
	// After a hidden interval, prime again instead of presenting a long-term average.
	if time.Since(c.checked) > 5*time.Second {
		previous = sample{}
	}
	c.last = compare(current, previous)
	c.checked = time.Now()
	c.previous = current
	c.last.UpdatedAt = c.checked.Unix()
	return c.last
}
func compare(cur, prev sample) Snapshot {
	s := Snapshot{CPUs: cur.cpus, MemoryUsed: cur.memoryUsed, MemoryTotal: cur.memoryTotal, SwapUsed: cur.swapUsed, SwapTotal: cur.swapTotal, ProcessCount: len(cur.processes), Skipped: cur.skipped, Processes: []Process{}}
	valid := prev.total > 0 && cur.total > prev.total && cur.idle >= prev.idle && cur.cpus == prev.cpus
	var delta uint64
	if valid {
		delta = cur.total - prev.total
		if idle := cur.idle - prev.idle; idle <= delta {
			v := 100 * float64(delta-idle) / float64(delta)
			s.CPU = &v
		} else {
			valid = false
		}
	}
	for pid, p := range cur.processes {
		p.CPU = nil
		if old, ok := prev.processes[pid]; valid && ok && old.start == p.start && p.ticks >= old.ticks {
			v := 100 * float64(p.ticks-old.ticks) * float64(cur.cpus) / float64(delta)
			// Total is sampled just before processes; bound small sampling skew.
			v = min(v, float64(cur.cpus)*100)
			p.CPU = &v
		}
		s.Processes = append(s.Processes, p)
	}
	sort.Slice(s.Processes, func(i, j int) bool {
		a, b := s.Processes[i], s.Processes[j]
		av, bv := float64(-1), float64(-1)
		if a.CPU != nil {
			av = *a.CPU
		}
		if b.CPU != nil {
			bv = *b.CPU
		}
		if av != bv {
			return av > bv
		}
		if a.Memory != b.Memory {
			return a.Memory > b.Memory
		}
		return a.PID < b.PID
	})
	if len(s.Processes) > 8 {
		s.Processes = s.Processes[:8]
	}
	if s.CPU == nil {
		s.Message = "Measuring CPU… next sample in 2s"
	}
	return s
}
func readProc(ctx context.Context, root string) (sample, error) {
	var s sample
	data, err := os.ReadFile(filepath.Join(root, "stat"))
	if err != nil {
		return s, err
	}
	s.total, s.idle, s.cpus, err = parseCPU(string(data))
	if err != nil {
		return s, err
	}
	data, err = os.ReadFile(filepath.Join(root, "meminfo"))
	if err != nil {
		return s, err
	}
	s.memoryUsed, s.memoryTotal, s.swapUsed, s.swapTotal, err = parseMemory(string(data))
	if err != nil {
		return s, err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return s, err
	}
	s.processes = make(map[int]Process)
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return s, err
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid <= 0 || !entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, entry.Name(), "stat"))
		if err != nil {
			s.skipped++
			continue
		} // A process may exit or deny access between reads.
		p, err := parseProcess(string(data), uint64(os.Getpagesize()))
		if err != nil || p.PID != pid {
			s.skipped++
			continue
		}
		s.processes[pid] = p
	}
	return s, ctx.Err()
}
func parseCPU(data string) (total, idle uint64, cpus int, err error) {
	for _, line := range strings.Split(data, "\n") {
		f := strings.Fields(line)
		if len(f) == 0 {
			continue
		}
		if f[0] == "cpu" {
			if len(f) < 9 {
				return 0, 0, 0, fmt.Errorf("incomplete CPU counters")
			}
			// guest and guest_nice already belong to user and nice; do not count twice.
			for i := 1; i <= 8; i++ {
				v, e := strconv.ParseUint(f[i], 10, 64)
				if e != nil {
					return 0, 0, 0, e
				}
				total += v
				if i == 4 || i == 5 {
					idle += v
				}
			}
		} else if strings.HasPrefix(f[0], "cpu") {
			if _, e := strconv.Atoi(strings.TrimPrefix(f[0], "cpu")); e == nil {
				cpus++
			}
		}
	}
	if total == 0 || cpus == 0 {
		return 0, 0, 0, fmt.Errorf("missing CPU counters")
	}
	return
}
func parseMemory(data string) (used, total, swapUsed, swapTotal uint64, err error) {
	values := map[string]uint64{}
	for _, line := range strings.Split(data, "\n") {
		f := strings.Fields(line)
		if len(f) != 3 || f[2] != "kB" {
			continue
		}
		v, e := strconv.ParseUint(f[1], 10, 64)
		if e == nil {
			values[strings.TrimSuffix(f[0], ":")] = v * 1024
		}
	}
	total = values["MemTotal"]
	available, ok := values["MemAvailable"]
	free, swapOK := values["SwapFree"]
	swapTotal, totalOK := values["SwapTotal"]
	if total == 0 || !ok || available > total || !swapOK || !totalOK || free > swapTotal {
		return 0, 0, 0, 0, fmt.Errorf("invalid memory counters")
	}
	return total - available, total, swapTotal - free, swapTotal, nil
}
func parseProcess(data string, pageSize uint64) (Process, error) {
	var p Process
	open, close := strings.IndexByte(data, '('), strings.LastIndexByte(data, ')')
	if open < 1 || close <= open {
		return p, fmt.Errorf("invalid process stat")
	}
	pid, err := strconv.Atoi(strings.TrimSpace(data[:open]))
	if err != nil {
		return p, err
	}
	f := strings.Fields(data[close+1:])
	if len(f) < 22 {
		return p, fmt.Errorf("short process stat")
	}
	// f starts at field 3 (state); utime=14, stime=15, starttime=22, rss=24.
	nums := make([]uint64, 4)
	for i, index := range []int{11, 12, 19, 21} {
		v, e := strconv.ParseUint(f[index], 10, 64)
		if e != nil {
			return p, e
		}
		nums[i] = v
	}
	return Process{PID: pid, Name: data[open+1 : close], ticks: nums[0] + nums[1], start: nums[2], Memory: nums[3] * pageSize}, nil
}
