// Read-only quota diagnostic. Uses the same provider as Pika's details panel.
package main

import (
	"context"
	"encoding/json"
	"os"
	"pika/internal/codexusage"
)

func main() {
	state := codexusage.New().Read(context.Background(), false)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(state)
	if state.UpdatedAt == 0 {
		os.Exit(1)
	}
}
