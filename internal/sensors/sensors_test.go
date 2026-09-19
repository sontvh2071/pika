package sensors

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestParseReadings(t *testing.T) {
	data := []byte(`{"board":{"Adapter":"ISA","fan":{"fan1_input":0},"voltage":{"in0_input":1.056},"bad probe":{"temp1_input":-55,"temp1_max":0,"temp1_crit":0},"fault":{"temp2_input":25,"temp2_fault":1},"empty":{"temp3_input":null},"power":{"power1_input":20,"power1_average":18}},"coretemp-isa-0000":{"Core 0":{"temp2_input":49,"temp2_max":85,"temp2_crit":105}}}`)
	rows, err := Parse(data)
	if err != nil || len(rows) != 6 {
		t.Fatalf("%+v %v", rows, err)
	}
	if rows[0].Chip != "coretemp-isa-0000" || *rows[0].High != 85 || *rows[0].Critical != 105 {
		t.Fatal(rows[0])
	}
	byLabel := map[string]Reading{}
	for _, r := range rows {
		byLabel[r.Label] = r
	}
	if *byLabel["fan"].Value != 0 || byLabel["fan"].Unit != "RPM" {
		t.Fatal("zero fan speed lost")
	}
	if *byLabel["voltage"].Value != 1.056 || byLabel["voltage"].Unit != "V" {
		t.Fatal("sensors JSON units were rescaled")
	}
	if *byLabel["bad probe"].Value != -55 || byLabel["bad probe"].High != nil || byLabel["bad probe"].Critical != nil {
		t.Fatal("raw value or unset limits changed")
	}
	if byLabel["fault"].Value != nil {
		t.Fatal("faulty sensor displayed as valid")
	}
	if *byLabel["power"].Value != 20 {
		t.Fatal("power average duplicated instantaneous value")
	}
	for _, invalid := range []string{"null", "{broken", "[]"} {
		if _, err := Parse([]byte(invalid)); err == nil {
			t.Fatal("invalid data accepted", invalid)
		}
	}
	empty, err := Parse([]byte(`{}`))
	if err != nil || len(empty) != 0 {
		t.Fatal("empty snapshot", empty, err)
	}
}
func TestCacheCoalescingAndStaleFailure(t *testing.T) {
	var calls atomic.Int32
	c := &Client{read: func(context.Context) ([]Reading, error) { calls.Add(1); return []Reading{{Chip: "CPU"}}, nil }}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); c.Read(context.Background()) }()
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatal("duplicate sensor processes", calls.Load())
	}
	c.checked = time.Time{}
	c.read = func(context.Context) ([]Reading, error) { return nil, fmt.Errorf("read failed") }
	stale := c.Read(context.Background())
	if !stale.Stale || stale.UpdatedAt == 0 || len(stale.Readings) != 1 || stale.Message != "read failed" {
		t.Fatal(stale)
	}
}
func TestCommandIsReadOnlyAndBounded(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sensors")
	// Fails unless launched with exactly -j, without any sudo/detect/setup flags.
	script := "#!/bin/sh\n[ \"$#\" = 1 ] && [ \"$1\" = -j ] || exit 9\nprintf '%s' '{\"chip\":{\"Fan\":{\"fan1_input\":1234}}}'\n"
	if err := os.WriteFile(path, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	rows, err := readCommand(context.Background())
	if err != nil || len(rows) != 1 || *rows[0].Value != 1234 {
		t.Fatal(rows, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := readCommand(ctx); err == nil {
		t.Fatal("ignored cancellation")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, err := readCommand(context.Background()); err == nil {
		t.Fatal("missing lm-sensors not reported")
	}
}
