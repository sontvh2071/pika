//go:build !linux

package desktop

import "fmt"

func Launch(path string) error { return fmt.Errorf("Desktop launching requires Linux") }
