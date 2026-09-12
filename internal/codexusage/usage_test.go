package codexusage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const fixture = `{"rateLimits":{"limitId":"codex","primary":{"usedPercent":99,"windowDurationMins":300}},"rateLimitsByLimitId":{"codex":{"primary":{"usedPercent":8,"windowDurationMins":10080,"resetsAt":2000},"secondary":{"usedPercent":50,"windowDurationMins":300,"resetsAt":1000}}},"rateLimitResetCredits":{"availableCount":2,"credits":[]}}`

func TestQuotaMappingAndAuthoritativeResetCount(t *testing.T) {
	s, err := Parse([]byte(fixture))
	if err != nil || s.FiveHour == nil || s.FiveHour.RemainingPercent != 50 || s.Weekly == nil || s.Weekly.RemainingPercent != 92 || s.ResetCredits == nil || *s.ResetCredits != 2 {
		t.Fatalf("%+v %v", s, err)
	}
	if *s.FiveHour.ResetsAt != 1000 || *s.Weekly.ResetsAt != 2000 {
		t.Fatal(s)
	}
}
func TestUnknownIsNotZeroAndWindowLabelsAreHonest(t *testing.T) {
	for _, raw := range []string{`{}`, `{"rateLimits":{"primary":{"usedPercent":0,"windowDurationMins":15},"secondary":{"usedPercent":null,"windowDurationMins":10080}},"rateLimitResetCredits":null}`} {
		s, err := Parse([]byte(raw))
		if err != nil || s.FiveHour != nil || s.Weekly != nil || s.ResetCredits != nil {
			t.Fatalf("%+v %v", s, err)
		}
	}
	s, err := Parse([]byte(`{"rateLimits":{"primary":{"usedPercent":110,"windowDurationMins":300},"secondary":{"usedPercent":-5,"windowDurationMins":10080}},"rateLimitResetCredits":{"availableCount":0}}`))
	if err != nil || s.FiveHour.RemainingPercent != 0 || s.Weekly.RemainingPercent != 100 || *s.ResetCredits != 0 {
		t.Fatal(s, err)
	}
}
func TestCacheDeduplicatesAndMarksOldData(t *testing.T) {
	now := time.Unix(1000, 0)
	var calls atomic.Int32
	fail := false
	c := &Client{now: func() time.Time { return now }, fetch: func(context.Context) ([]byte, error) {
		calls.Add(1)
		if fail {
			return nil, errors.New("offline")
		}
		return []byte(fixture), nil
	}}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); c.Read(context.Background(), false) }()
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatal(calls.Load())
	}
	c.Read(context.Background(), true)
	if calls.Load() != 1 {
		t.Fatal("refresh must have a minimum interval")
	}
	now = now.Add(31 * time.Second)
	fail = true
	s := c.Read(context.Background(), false)
	if calls.Load() != 2 || !s.Stale || s.UpdatedAt != 1000 || s.FiveHour.RemainingPercent != 50 || s.Message == "" {
		t.Fatal(s)
	}
	now = now.Add(31 * time.Second)
	fail = false
	s = c.Read(context.Background(), false)
	if s.Stale || s.Message != "" || s.UpdatedAt != 1062 {
		t.Fatal(s)
	}
}
func TestProtocolOnlyInitializesAndReads(t *testing.T) {
	var sent bytes.Buffer
	received := `{"method":"account/updated","params":{}}
{"id":1,"result":{"userAgent":"test"}}
{"id":2,"result":` + fixture + `}\n`
	// Use a real newline rather than a JSON string escape at the stream boundary.
	received = strings.TrimSuffix(received, `\n`) + "\n"
	result, err := exchange(&sent, strings.NewReader(received))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Parse(result); err != nil {
		t.Fatal(err)
	}
	messages := strings.Split(strings.TrimSpace(sent.String()), "\n")
	want := []string{"initialize", "initialized", "account/rateLimits/read"}
	if len(messages) != len(want) {
		t.Fatal(sent.String())
	}
	for i, line := range messages {
		var msg map[string]any
		if err = json.Unmarshal([]byte(line), &msg); err != nil || msg["method"] != want[i] {
			t.Fatal(line, err)
		}
	}
}
func TestProtocolRejectsTruncatedAndErrorResponses(t *testing.T) {
	for _, raw := range []string{"", `{"id":1,"error":{"message":"private error must not reach UI"}}`, "{not json}"} {
		var sent bytes.Buffer
		_, err := exchange(&sent, strings.NewReader(raw))
		if err == nil || strings.Contains(err.Error(), "private error") {
			t.Fatal(err)
		}
	}
}
