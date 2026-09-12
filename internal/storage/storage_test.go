package storage

import (
	"path/filepath"
	"pika/internal/catalog"
	"testing"
)

func TestPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	s, e := Open(path)
	if e != nil {
		t.Fatal(e)
	}
	want := map[string]catalog.Usage{"app:code.desktop": {Count: 3, LastUsed: 123}}
	if e = s.Save(want); e != nil {
		t.Fatal(e)
	}
	s.Close()
	s, e = Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	got, e := s.Load()
	if e != nil || got["app:code.desktop"] != want["app:code.desktop"] {
		t.Fatal(got, e)
	}
}
