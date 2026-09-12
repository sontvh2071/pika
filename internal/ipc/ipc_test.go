package ipc

import (
	"path/filepath"
	"sync"
	"testing"
)

func TestSingleInstanceAndIdempotency(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "pika")
	s, e := Acquire(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if other, e := Acquire(dir); e == nil {
		other.Close()
		t.Fatal("second lock acquired")
	}
	count := 0
	s.Serve(func(cmd string) Response { count++; return Response{OK: true} })
	r := NewRequest("toggle")
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, e := Send(dir, r)
			if e != nil || !v.OK {
				t.Errorf("%+v %v", v, e)
			}
		}()
	}
	wg.Wait()
	s.mu.Lock()
	defer s.mu.Unlock()
	if count != 1 {
		t.Fatalf("toggle ran %d times", count)
	}
}
