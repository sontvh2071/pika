// Package codexusage reads account quotas through the official Codex app-server
// protocol. It never reads tokens itself, creates a turn, or consumes a reset.
package codexusage

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"
)

type Window struct {
	RemainingPercent float64 `json:"remaining_percent"`
	ResetsAt         *int64  `json:"resets_at"`
}
type Snapshot struct {
	FiveHour     *Window `json:"five_hour"`
	Weekly       *Window `json:"weekly"`
	ResetCredits *int    `json:"reset_credits"`
	UpdatedAt    int64   `json:"updated_at"`
	Stale        bool    `json:"stale"`
	Message      string  `json:"message"`
}
type rawWindow struct {
	Used     *float64 `json:"usedPercent"`
	Duration *int     `json:"windowDurationMins"`
	ResetsAt *int64   `json:"resetsAt"`
}
type rawLimit struct {
	ID        string     `json:"limitId"`
	Primary   *rawWindow `json:"primary"`
	Secondary *rawWindow `json:"secondary"`
}

func Parse(data []byte) (Snapshot, error) {
	var response struct {
		Legacy  *rawLimit            `json:"rateLimits"`
		Buckets map[string]*rawLimit `json:"rateLimitsByLimitId"`
		Resets  *struct {
			Available *int `json:"availableCount"`
		} `json:"rateLimitResetCredits"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return Snapshot{}, fmt.Errorf("Invalid Codex quota response")
	}
	limit, exists := response.Buckets["codex"]
	if !exists && (response.Legacy != nil && (response.Legacy.ID == "" || response.Legacy.ID == "codex")) {
		limit = response.Legacy
	}
	result := Snapshot{}
	if limit != nil {
		for _, w := range []*rawWindow{limit.Primary, limit.Secondary} {
			if w == nil || w.Used == nil || w.Duration == nil {
				continue
			}
			value := &Window{RemainingPercent: math.Max(0, math.Min(100, 100-*w.Used)), ResetsAt: w.ResetsAt}
			switch *w.Duration {
			case 300:
				result.FiveHour = value
			case 10080:
				result.Weekly = value
			}
		}
	}
	if response.Resets != nil && response.Resets.Available != nil && *response.Resets.Available >= 0 {
		result.ResetCredits = response.Resets.Available
	}
	if result.FiveHour == nil && result.Weekly == nil && result.ResetCredits == nil {
		result.Message = "Codex has not reported quota for this account."
	}
	return result, nil
}

type Client struct {
	mu        sync.Mutex
	cached    Snapshot
	attempted time.Time
	fetch     func(context.Context) ([]byte, error)
	now       func() time.Time
}

func New() *Client { return &Client{fetch: fetchUsage, now: time.Now} }
func (c *Client) Read(ctx context.Context, force bool) Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	age := c.now().Sub(c.attempted)
	if !c.attempted.IsZero() && age < 30*time.Second && (!force || age < 5*time.Second) {
		return c.cached
	}
	c.attempted = c.now()
	data, err := c.fetch(ctx)
	if err == nil {
		var current Snapshot
		current, err = Parse(data)
		if err == nil {
			current.UpdatedAt = c.now().Unix()
			c.cached = current
			return current
		}
	}
	c.cached.Stale = c.cached.UpdatedAt != 0
	c.cached.Message = "Cannot refresh usage. Open Codex and check your sign-in or connection."
	return c.cached
}
