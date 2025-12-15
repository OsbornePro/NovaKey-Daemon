// cmd/novakey/logfile.go
package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type logOutputs struct {
	writer   *rotatingFileWriter
	toStderr bool
}

func selectLogOutputs() logOutputs {
	toStderr := true
	if cfg.LogStderr != nil {
		toStderr = *cfg.LogStderr
	}

	logFile := strings.TrimSpace(cfg.LogFile)
	logDir := strings.TrimSpace(cfg.LogDir)

	if logFile == "" && logDir != "" {
		logFile = filepath.Join(logDir, "novakey.log")
	}
	if logFile == "" {
		return logOutputs{writer: nil, toStderr: toStderr}
	}

	rotateMB := cfg.LogRotateMB
	if rotateMB < 1 {
		rotateMB = 10
	}
	keep := cfg.LogKeep
	if keep < 1 {
		keep = 10
	}

	_ = os.MkdirAll(filepath.Dir(logFile), 0755)

	w := &rotatingFileWriter{
		path:      logFile,
		maxBytes:  int64(rotateMB) * 1024 * 1024,
		keepFiles: keep,
	}
	// open now (best effort)
	_ = w.openIfNeeded()
	return logOutputs{writer: w, toStderr: toStderr}
}

type rotatingFileWriter struct {
	mu        sync.Mutex
	path      string
	maxBytes  int64
	keepFiles int
	f         *os.File
}

func (w *rotatingFileWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.openIfNeeded(); err != nil {
		return 0, err
	}

	if w.maxBytes > 0 {
		if fi, err := w.f.Stat(); err == nil {
			if fi.Size()+int64(len(p)) > w.maxBytes {
				_ = w.rotateLocked()
			}
		}
	}
	return w.f.Write(p)
}

func (w *rotatingFileWriter) openIfNeeded() error {
	if w.f != nil {
		return nil
	}
	_ = os.MkdirAll(filepath.Dir(w.path), 0755)
	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	w.f = f
	return nil
}

func (w *rotatingFileWriter) rotateLocked() error {
	if w.f != nil {
		_ = w.f.Close()
		w.f = nil
	}

	fi, err := os.Stat(w.path)
	if err != nil || fi.Size() == 0 {
		return w.openIfNeeded()
	}

	// Rename current file to timestamped rotated file.
	ts := time.Now().Format("20060102-150405")
	rotated := w.path + "." + ts
	_ = os.Rename(w.path, rotated)

	if err := w.openIfNeeded(); err != nil {
		return err
	}
	w.cleanupOldLocked()
	return nil
}

func (w *rotatingFileWriter) cleanupOldLocked() {
	dir := filepath.Dir(w.path)
	base := filepath.Base(w.path)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	type cand struct {
		path string
		mod  time.Time
	}
	var cands []cand

	for _, e := range entries {
		name := e.Name()
		if name == base {
			continue
		}
		if !strings.HasPrefix(name, base+".") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		cands = append(cands, cand{
			path: filepath.Join(dir, name),
			mod:  info.ModTime(),
		})
	}

	sort.Slice(cands, func(i, j int) bool { return cands[i].mod.After(cands[j].mod) })

	for i := w.keepFiles; i < len(cands); i++ {
		_ = os.Remove(cands[i].path)
	}
}

