// Read the same unprivileged df -h provider used by Pika.
package main

import (
	"context"
	"encoding/json"
	"os"
	"pika/internal/diskfree"
)

func main() {
	s := diskfree.New().Read(context.Background())
	json.NewEncoder(os.Stdout).Encode(s)
	if s.UpdatedAt == 0 || s.Stale {
		os.Exit(1)
	}
}
