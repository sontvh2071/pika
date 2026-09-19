// Package sensors reads lm-sensors without privilege escalation or hardware probing.
package sensors

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

type Reading struct {
	Chip     string   `json:"chip"`
	Label    string   `json:"label"`
	Kind     string   `json:"kind"`
	Unit     string   `json:"unit"`
	Value    *float64 `json:"value"`
	High     *float64 `json:"high,omitempty"`
	Critical *float64 `json:"critical,omitempty"`
}
type Snapshot struct {
	Readings  []Reading `json:"readings"`
	UpdatedAt int64     `json:"updated_at"`
	Stale     bool      `json:"stale"`
	Message   string    `json:"message"`
}
type Client struct {
	mu      sync.Mutex
	last    Snapshot
	checked time.Time
	read    func(context.Context) ([]Reading, error)
}

func New() *Client { return &Client{read: readCommand} }

// Coalesce polling and manual refresh, including failures. No background worker.
func (c *Client) Read(ctx context.Context) Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.checked.IsZero() && time.Since(c.checked) < 1500*time.Millisecond {
		return c.last
	}
	readings, err := c.read(ctx)
	c.checked = time.Now()
	if err != nil {
		c.last.Stale = c.last.UpdatedAt != 0
		c.last.Message = err.Error()
		return c.last
	}
	c.last = Snapshot{Readings: readings, UpdatedAt: time.Now().Unix()}
	if len(readings) == 0 {
		c.last.Message = "No readable sensors found. Check lm-sensors and hardware driver setup."
	}
	return c.last
}
func readCommand(ctx context.Context) ([]Reading, error) {
	path, err := exec.LookPath("sensors")
	if err != nil {
		return nil, fmt.Errorf("lm-sensors is not installed. Install it to view hardware readings.")
	}
	ctx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "-j")
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	out, err := cmd.Output()
	if ctx.Err() != nil {
		return nil, fmt.Errorf("Sensor read timed out or was cancelled. Try refreshing.")
	}
	if len(out) > 1024*1024 {
		return nil, fmt.Errorf("Sensor output is too large.")
	}
	if err != nil {
		return nil, fmt.Errorf("Cannot read sensors. Check lm-sensors and hardware driver setup.")
	}
	return Parse(out)
}

var input = regexp.MustCompile(`^(temp|fan|in|curr|power|humidity)[0-9]+_(input|average)$`)

func Parse(data []byte) ([]Reading, error) {
	var chips map[string]map[string]json.RawMessage
	if err := json.Unmarshal(data, &chips); err != nil || chips == nil {
		return nil, fmt.Errorf("Invalid sensor response.")
	}
	readings := []Reading{}
	for chip, features := range chips {
		for label, raw := range features {
			var values map[string]*float64
			if json.Unmarshal(raw, &values) != nil {
				continue
			} // Adapter is text.
			for key, value := range values {
				match := input.FindStringSubmatch(key)
				if match == nil || value == nil {
					continue
				}
				if match[2] == "average" && (match[1] != "power" || values[strings.TrimSuffix(key, "average")+"input"] != nil) {
					continue
				}
				units := map[string]string{"temp": "°C", "fan": "RPM", "in": "V", "curr": "A", "power": "W", "humidity": "%"}
				prefix := key[:strings.LastIndexByte(key, '_')]
				r := Reading{Chip: chip, Label: label, Kind: match[1], Unit: units[match[1]], Value: value}
				if fault := values[prefix+"_fault"]; fault != nil && *fault != 0 {
					r.Value = nil
				}
				if r.Kind == "temp" {
					// Zero/unset limits are common; do not present them as meaningful thresholds.
					if high := values[prefix+"_max"]; high != nil && *high > 0 {
						r.High = high
					}
					if crit := values[prefix+"_crit"]; crit != nil && *crit > 0 {
						r.Critical = crit
					}
				}
				readings = append(readings, r)
			}
		}
	}
	rank := map[string]int{"temp": 0, "fan": 1, "in": 2, "curr": 3, "power": 4, "humidity": 5}
	sort.Slice(readings, func(i, j int) bool {
		a, b := readings[i], readings[j]
		cpuA, cpuB := strings.HasPrefix(a.Chip, "coretemp-") || strings.HasPrefix(a.Chip, "k10temp-"), strings.HasPrefix(b.Chip, "coretemp-") || strings.HasPrefix(b.Chip, "k10temp-")
		if cpuA != cpuB {
			return cpuA
		}
		if a.Chip != b.Chip {
			return a.Chip < b.Chip
		}
		if rank[a.Kind] != rank[b.Kind] {
			return rank[a.Kind] < rank[b.Kind]
		}
		return a.Label < b.Label
	})
	return readings, nil
}
