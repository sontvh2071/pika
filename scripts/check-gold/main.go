// Read the same public quote/history provider as Pika, without a GUI.
package main

import (
	"context"
	"encoding/json"
	"os"
	"pika/internal/gold"
)

func main() {
	s := gold.New().Read(context.Background(), false)
	json.NewEncoder(os.Stdout).Encode(s)
	if s.FetchedAt == 0 {
		os.Exit(1)
	}
}
