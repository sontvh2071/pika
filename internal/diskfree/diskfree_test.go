package diskfree

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const header = "Filesystem Type Size Used Avail Use% Mounted on\n"

func TestParseHumanReadableMounts(t *testing.T) {
	rows, err := Parse([]byte(header + `tmpfs tmpfs 7.8G 0 7.8G 0% /run
/dev/sda1 vfat 511M 6.2M 505M 2% /boot/efi
/dev/sda2 ext4 219G 22G 186G 11% /
server:/Backup disk nfs4 1.8T 1.7T 80G 96% /media/My Backup
/dev/full ext4 10G 11G -1G 110% /full
unknown fuse.example - - - - /unknown
`))
	if err != nil || len(rows) != 6 {
		t.Fatal(rows, err)
	}
	if rows[0].Mount != "/" || rows[0].Available != "186G" || *rows[0].Percent != 11 {
		t.Fatal(rows[0])
	}
	byMount := map[string]Filesystem{}
	for _, r := range rows {
		byMount[r.Mount] = r
	}
	if r := byMount["/media/My Backup"]; r.Source != "server:/Backup disk" || r.Type != "nfs4" || r.Total != "1.8T" || r.Virtual {
		t.Fatal(r)
	}
	if r := byMount["/full"]; *r.Percent != 110 || r.Available != "-1G" {
		t.Fatal(r)
	}
	if r := byMount["/unknown"]; r.Percent != nil || r.Total != "-" {
		t.Fatal(r)
	}
	if !rows[len(rows)-1].Virtual || rows[len(rows)-1].Mount != "/run" {
		t.Fatal(rows)
	}
}
func TestInvalidAndEmptyOutput(t *testing.T) {
	for _, data := range []string{"", "unexpected", header + "bad row", header + "/dev/a ext4 100G invalid 20G 80% /"} {
		if _, err := Parse([]byte(data)); err == nil {
			t.Fatalf("invalid response accepted %q", data)
		}
	}
	rows, err := Parse([]byte(header))
	if err != nil || len(rows) != 0 {
		t.Fatal(rows, err)
	}
}
func TestCacheAndStaleRecovery(t *testing.T) {
	var calls atomic.Int32
	c := &Client{read: func(context.Context) ([]Filesystem, error) {
		calls.Add(1)
		return []Filesystem{{Mount: "/", Available: "10G"}}, nil
	}}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); c.Read(context.Background()) }()
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatal(calls.Load())
	}
	stamp := c.last.UpdatedAt
	c.checked = time.Time{}
	c.read = func(context.Context) ([]Filesystem, error) { return nil, errors.New("offline") }
	s := c.Read(context.Background())
	if !s.Stale || s.UpdatedAt != stamp || s.Filesystems[0].Available != "10G" || s.Message != "offline" {
		t.Fatal(s)
	}
	c.checked = time.Time{}
	c.read = func(context.Context) ([]Filesystem, error) { return []Filesystem{{Mount: "/", Available: "9G"}}, nil }
	s = c.Read(context.Background())
	if s.Stale || s.Message != "" || s.Filesystems[0].Available != "9G" {
		t.Fatal(s)
	}
}
func TestCommandUsesHumanReadableNoShellOrSudo(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "df")
	script := "#!/bin/sh\n[ \"$#\" = 2 ] && [ \"$1\" = -h ] && [ \"$2\" = --output=source,fstype,size,used,avail,pcent,target ] && [ \"$LC_ALL\" = C ] || exit 9\nprintf '%s\\n' 'Filesystem Type Size Used Avail Use% Mounted on' '/dev/a ext4 100G 20G 80G 20% /'\n"
	if err := os.WriteFile(path, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	rows, err := readCommand(context.Background())
	if err != nil || len(rows) != 1 || rows[0].Available != "80G" {
		t.Fatal(rows, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := readCommand(ctx); err == nil {
		t.Fatal("ignored cancellation")
	}
	// Partial df output plus nonzero exit must not masquerade as a complete snapshot.
	os.WriteFile(path, []byte(script+"exit 1\n"), 0700)
	if _, err := readCommand(context.Background()); err == nil {
		t.Fatal("ignored df failure")
	}
	os.Remove(path)
	if _, err := readCommand(context.Background()); err == nil {
		t.Fatal("missing df not reported")
	}
}
