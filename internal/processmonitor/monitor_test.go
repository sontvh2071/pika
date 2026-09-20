package processmonitor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestCPUAndMemory(t *testing.T) {
	total, idle, cpus, err := parseCPU("cpu  100 20 30 400 10 2 3 5 50 10\ncpu0 1\ncpu1 1\n")
	if err != nil || total != 570 || idle != 410 || cpus != 2 {
		t.Fatalf("CPU %d %d %d %v", total, idle, cpus, err)
	}
	used, total, swapUsed, swapTotal, err := parseMemory("MemTotal: 1000 kB\nMemAvailable: 400 kB\nSwapTotal: 200 kB\nSwapFree: 150 kB\n")
	if err != nil || used != 600*1024 || total != 1000*1024 || swapUsed != 50*1024 || swapTotal != 200*1024 {
		t.Fatal(used, total, swapUsed, swapTotal, err)
	}
	for _, bad := range []string{"", "MemTotal: 1 kB\nMemAvailable: 2 kB"} {
		if _, _, _, _, err := parseMemory(bad); err == nil {
			t.Fatal("invalid memory accepted")
		}
	}
	if _, _, _, err := parseCPU("cpu 1 2"); err == nil {
		t.Fatal("short CPU accepted")
	}
}
func processStat(pid int, name string) string {
	f := strings.Fields("S 1 2 3 4 5 6 7 8 9 10 120 30 0 0 20 0 4 0 900 5000 12")
	return fmt.Sprintf("%d (%s) %s", pid, name, strings.Join(f, " "))
}
func TestProcessNamesAndFields(t *testing.T) {
	p, err := parseProcess(processStat(12, "name with ) spaces"), 4096)
	if err != nil || p.PID != 12 || p.Name != "name with ) spaces" || p.ticks != 150 || p.start != 900 || p.Memory != 12*4096 {
		t.Fatal(p, err)
	}
	for _, bad := range []string{"", "1 (name) S 0", "not a process"} {
		if _, err := parseProcess(bad, 4096); err == nil {
			t.Fatal("malformed process accepted")
		}
	}
}
func TestIntervalCPUAndPIDReuse(t *testing.T) {
	prev := sample{total: 1000, idle: 400, cpus: 4, processes: map[int]Process{1: {ticks: 10, start: 1}, 2: {ticks: 80, start: 1}}}
	cur := sample{total: 1400, idle: 700, cpus: 4, processes: map[int]Process{1: {PID: 1, ticks: 160, start: 1}, 2: {PID: 2, ticks: 180, start: 2}, 3: {PID: 3, ticks: 10, start: 1}}}
	s := compare(cur, prev)
	if s.CPU == nil || *s.CPU != 25 {
		t.Fatal(s)
	}
	if s.Processes[0].PID != 1 || s.Processes[0].CPU == nil || *s.Processes[0].CPU != 150 {
		t.Fatal(s.Processes)
	}
	for _, p := range s.Processes[1:] {
		if p.CPU != nil {
			t.Fatal("new/reused PID must be unknown", p)
		}
	}
	first := compare(cur, sample{})
	if first.CPU != nil || first.Message == "" {
		t.Fatal(first)
	}
	// A reset/decreased counter must not underflow to a huge percentage.
	cur.total = 900
	if s := compare(cur, prev); s.CPU != nil {
		t.Fatal(s)
	}
}
func TestTopLimitAndRSSOrder(t *testing.T) {
	cur := sample{processes: map[int]Process{}}
	for i := 1; i <= 20; i++ {
		cur.processes[i] = Process{PID: i, Memory: uint64(i)}
	}
	s := compare(cur, sample{})
	if s.ProcessCount != 20 || len(s.Processes) != 8 || s.Processes[0].PID != 20 {
		t.Fatal(s)
	}
}
func TestCacheFailureAndResume(t *testing.T) {
	calls := 0
	c := &Client{read: func(context.Context) (sample, error) { calls++; return sample{total: 100, idle: 40, cpus: 1}, nil }}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); c.Read(context.Background()) }()
	}
	wg.Wait()
	if calls != 1 {
		t.Fatal(calls)
	}
	stamp := c.last.UpdatedAt
	c.checked = time.Now().Add(-2 * time.Second)
	c.read = func(context.Context) (sample, error) { return sample{}, errors.New("unavailable") }
	stale := c.Read(context.Background())
	if !stale.Stale || stale.UpdatedAt != stamp || stale.Message == "" {
		t.Fatal(stale)
	}
	c.checked = time.Now().Add(-10 * time.Second)
	c.read = func(context.Context) (sample, error) { return sample{total: 900, idle: 50, cpus: 1}, nil }
	resumed := c.Read(context.Background())
	if resumed.Stale || resumed.CPU != nil {
		t.Fatal(resumed)
	}
}
func TestProcReadSkipsVanishedAndCancellation(t *testing.T) {
	root := t.TempDir()
	for name, data := range map[string]string{"stat": "cpu 100 0 0 900 0 0 0 0\ncpu0 0\n", "meminfo": "MemTotal: 1000 kB\nMemAvailable: 400 kB\nSwapTotal: 0 kB\nSwapFree: 0 kB\n", "12/stat": processStat(12, "worker")} {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	os.Mkdir(filepath.Join(root, "99"), 0700)
	s, err := readProc(context.Background(), root)
	if err != nil || len(s.processes) != 1 || s.skipped != 1 {
		t.Fatal(s, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := readProc(ctx, root); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
