package ipc

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

type Request struct {
	Version int    `json:"version"`
	ID      string `json:"id"`
	Command string `json:"command"`
}
type Response struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}
type Server struct {
	listener net.Listener
	lock     *os.File
	dir      string
	mu       sync.Mutex
	cache    map[string]Response
	order    []string
	wg       sync.WaitGroup
}

func Directory() (string, error) {
	p := os.Getenv("XDG_RUNTIME_DIR")
	if !filepath.IsAbs(p) {
		return "", fmt.Errorf("XDG_RUNTIME_DIR must be set to your session runtime directory")
	}
	st, e := os.Stat(p)
	if e != nil {
		return "", e
	}
	sys, ok := st.Sys().(*syscall.Stat_t)
	if !ok || int(sys.Uid) != os.Getuid() || st.Mode().Perm()&0077 != 0 {
		return "", fmt.Errorf("XDG_RUNTIME_DIR must be owned by you with mode 0700")
	}
	return filepath.Join(p, "pika"), nil
}
func NewRequest(command string) Request {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return Request{1, hex.EncodeToString(b[:]), command}
}
func Send(dir string, r Request) (Response, error) {
	c, e := net.DialTimeout("unix", filepath.Join(dir, "control.sock"), 300*time.Millisecond)
	if e != nil {
		return Response{}, e
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(3 * time.Second))
	if e = json.NewEncoder(c).Encode(r); e != nil {
		return Response{}, e
	}
	var v Response
	e = json.NewDecoder(io.LimitReader(c, 1024*1024)).Decode(&v)
	return v, e
}
func Acquire(dir string) (*Server, error) {
	if e := os.MkdirAll(dir, 0700); e != nil {
		return nil, e
	}
	st, e := os.Lstat(dir)
	if e != nil {
		return nil, e
	}
	sys, ok := st.Sys().(*syscall.Stat_t)
	if !st.IsDir() || st.Mode().Perm()&0077 != 0 || !ok || int(sys.Uid) != os.Getuid() {
		return nil, fmt.Errorf("unsafe IPC directory")
	}
	f, e := os.OpenFile(filepath.Join(dir, "instance.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return nil, e
	}
	if e = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); e != nil {
		f.Close()
		return nil, e
	}
	socket := filepath.Join(dir, "control.sock")
	if st, e := os.Lstat(socket); e == nil {
		if st.Mode()&os.ModeSocket == 0 {
			f.Close()
			return nil, fmt.Errorf("refusing to remove non-socket %s", socket)
		}
		if e = os.Remove(socket); e != nil {
			f.Close()
			return nil, e
		}
	}
	l, e := net.Listen("unix", socket)
	if e != nil {
		f.Close()
		return nil, e
	}
	if e = os.Chmod(socket, 0600); e != nil {
		l.Close()
		f.Close()
		return nil, e
	}
	return &Server{listener: l, lock: f, dir: dir, cache: map[string]Response{}}, nil
}
func (s *Server) Serve(handle func(string) Response) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			c, e := s.listener.Accept()
			if e != nil {
				return
			}
			s.wg.Add(1)
			go func() {
				defer s.wg.Done()
				defer c.Close()
				_ = c.SetDeadline(time.Now().Add(3 * time.Second))
				scanner := bufio.NewScanner(io.LimitReader(c, 4097))
				scanner.Buffer(make([]byte, 4096), 4096)
				if !scanner.Scan() {
					return
				}
				var r Request
				var v Response
				if e := json.Unmarshal(scanner.Bytes(), &r); e != nil || r.Version != 1 || r.ID == "" {
					v = Response{Message: "Invalid IPC request"}
				} else {
					s.mu.Lock()
					cached, ok := s.cache[r.ID]
					if ok {
						v = cached
					} else {
						v = handle(r.Command)
						if v.Message != "starting" {
							s.cache[r.ID] = v
							s.order = append(s.order, r.ID)
							if len(s.order) > 128 {
								delete(s.cache, s.order[0])
								s.order = s.order[1:]
							}
						}
					}
					s.mu.Unlock()
				}
				_ = json.NewEncoder(c).Encode(v)
			}()
		}
	}()
}
func (s *Server) Close() { _ = s.listener.Close(); s.wg.Wait(); _ = s.lock.Close() }
