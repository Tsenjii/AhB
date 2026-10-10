package sidecar

import (
	"bytes"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestCappedLogFileBoundsExistingAndNewOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "noisy.log")
	if err := os.WriteFile(path, bytes.Repeat([]byte("x"), int(maxSidecarLogBytes)+23), 0644); err != nil {
		t.Fatal(err)
	}
	log, err := openCappedLogFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if st, err := os.Stat(path); err != nil || st.Size() != 0 || st.Mode().Perm() != 0600 {
		t.Fatalf("oversized existing log must reset privately: stat=%v err=%v", st, err)
	}

	payload := bytes.Repeat([]byte("A"), int(maxSidecarLogBytes)+35)
	if n, err := log.Write(payload); err != nil || n != len(payload) {
		t.Fatalf("writing across cap: n=%d err=%v", n, err)
	}
	if err := log.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(payload[len(payload)-35:]) {
		t.Fatalf("wanted only latest 35 bytes after wrap, got %d", len(data))
	}
}

func TestCappedLogFileConcurrentWritersStayBounded(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gateway.log")
	log, err := openCappedLogFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			payload := bytes.Repeat([]byte("z"), 256<<10)
			for j := 0; j < 5; j++ {
				if n, err := log.Write(payload); n != len(payload) || err != nil {
					t.Errorf("concurrent write: n=%d err=%v", n, err)
				}
			}
		}()
	}
	wg.Wait()
	if err := log.Close(); err != nil {
		t.Fatal(err)
	}
	stat, err := os.Stat(path)
	if err != nil || stat.Size() > maxSidecarLogBytes {
		t.Fatalf("log exceeded cap: stat=%v err=%v", stat, err)
	}
	if _, err := log.Write([]byte("closed")); !os.IsNotExist(err) && err != os.ErrClosed {
		t.Fatalf("closed writer must reject output: %v", err)
	}
}

func TestCappedLogRefusesSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "private.txt")
	if err := os.WriteFile(target, []byte("unchanged"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "gateway.log")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}
	if log, err := openCappedLogFile(link); err == nil {
		log.Close()
		t.Fatal("refused to open symlinked log")
	}
	if body, err := os.ReadFile(target); err != nil || string(body) != "unchanged" {
		t.Fatalf("symlink target changed: %q, %v", body, err)
	}
}
