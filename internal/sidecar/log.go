package sidecar

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
)

// A noisy long-lived gateway must not fill a small VPS's disk. Each
// supervised provider gets one bounded private log, with no extra process,
// polling goroutine, or unbounded in-memory queue. At the cap we start a new
// segment inside the same file; recent logs remain available for diagnosis.
const maxSidecarLogBytes int64 = 4 << 20

type cappedLogFile struct {
	mu   sync.Mutex
	file *os.File
	size int64
}

func openCappedLogFile(path string) (*cappedLogFile, error) {
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("refusing non-regular gateway log %q", path)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := f.Chmod(0600); err != nil {
		f.Close()
		return nil, err
	}
	stat, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	log := &cappedLogFile{file: f, size: stat.Size()}
	if log.size >= maxSidecarLogBytes {
		if err := log.resetLocked(); err != nil {
			f.Close()
			return nil, err
		}
	} else if _, err := f.Seek(0, io.SeekEnd); err != nil {
		f.Close()
		return nil, err
	}
	return log, nil
}

func (l *cappedLogFile) resetLocked() error {
	if err := l.file.Truncate(0); err != nil {
		return err
	}
	if _, err := l.file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	l.size = 0
	return nil
}

func (l *cappedLogFile) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file == nil {
		return 0, os.ErrClosed
	}
	written := 0
	for len(p) > 0 {
		if l.size >= maxSidecarLogBytes {
			if err := l.resetLocked(); err != nil {
				return written, err
			}
		}
		remaining := int(maxSidecarLogBytes - l.size)
		n := len(p)
		if n > remaining {
			n = remaining
		}
		count, err := l.file.Write(p[:n])
		written += count
		l.size += int64(count)
		p = p[count:]
		if err != nil {
			return written, err
		}
		if count != n {
			return written, io.ErrShortWrite
		}
	}
	return written, nil
}

func (l *cappedLogFile) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file == nil {
		return nil
	}
	err := l.file.Close()
	l.file = nil
	return err
}
