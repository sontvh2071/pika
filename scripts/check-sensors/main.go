// Read the same unprivileged provider as Pika, without opening a window.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"pika/internal/sensors"
)

func main() {
	snapshot := sensors.New().Read(context.Background())
	if err := json.NewEncoder(os.Stdout).Encode(snapshot); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if snapshot.UpdatedAt == 0 {
		os.Exit(1)
	}
}
