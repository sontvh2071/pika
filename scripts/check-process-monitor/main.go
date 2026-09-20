// Read two unprivileged samples from the same provider as Pika.
package main

import (
	"context"
	"encoding/json"
	"os"
	"pika/internal/processmonitor"
	"time"
)

func main() {
	c := processmonitor.New()
	c.Read(context.Background())
	time.Sleep(2 * time.Second)
	s := c.Read(context.Background())
	json.NewEncoder(os.Stdout).Encode(s)
	if s.UpdatedAt == 0 || s.Stale {
		os.Exit(1)
	}
}
