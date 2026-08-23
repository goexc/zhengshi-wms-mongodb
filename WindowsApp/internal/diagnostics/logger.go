package diagnostics

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

const (
	defaultMaxLogBytes = int64(5 << 20)
	defaultLogBackups  = 3
)

type Logger struct {
	mu       sync.Mutex
	file     *os.File
	log      *log.Logger
	name     string
	maxBytes int64
	backups  int
}

var (
	defaultLoggerMu sync.RWMutex
	defaultLogger   *Logger
)

func New() (*Logger, error) {
	name, err := Path()
	if err != nil {
		return nil, err
	}
	return newAt(name, defaultMaxLogBytes, defaultLogBackups)
}

func newAt(name string, maxBytes int64, backups int) (*Logger, error) {
	dir := filepath.Dir(name)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	if err := rotateIfNeeded(name, maxBytes, backups); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(name, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}
	return &Logger{
		file: file, log: log.New(file, "", log.Ldate|log.Ltime|log.Lmicroseconds),
		name: name, maxBytes: maxBytes, backups: backups,
	}, nil
}

func rotateIfNeeded(name string, maxBytes int64, backups int) error {
	if maxBytes <= 0 || backups <= 0 {
		return nil
	}
	info, err := os.Stat(name)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.Size() < maxBytes {
		return nil
	}
	oldest := fmt.Sprintf("%s.%d", name, backups)
	if err := os.Remove(oldest); err != nil && !os.IsNotExist(err) {
		return err
	}
	for index := backups - 1; index >= 1; index-- {
		source := fmt.Sprintf("%s.%d", name, index)
		target := fmt.Sprintf("%s.%d", name, index+1)
		if err := os.Rename(source, target); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return os.Rename(name, name+".1")
}

func Path() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "ZhengshiWMS", "logs", "windowsapp.log"), nil
}

func ReadTail(maxBytes int64) (string, error) {
	name, err := Path()
	if err != nil {
		return "", err
	}
	file, err := os.Open(name)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	defer file.Close()
	if maxBytes <= 0 {
		maxBytes = 64 << 10
	}
	info, err := file.Stat()
	if err != nil {
		return "", err
	}
	start := info.Size() - maxBytes
	if start < 0 {
		start = 0
	}
	if _, err = file.Seek(start, io.SeekStart); err != nil {
		return "", err
	}
	data, err := io.ReadAll(io.LimitReader(file, maxBytes))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (l *Logger) Printf(format string, args ...any) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.rotateDuringRun()
	l.log.Printf(format, args...)
}

func (l *Logger) rotateDuringRun() {
	if l.file == nil || l.maxBytes <= 0 || l.backups <= 0 {
		return
	}
	info, err := l.file.Stat()
	if err != nil || info.Size() < l.maxBytes {
		return
	}
	_ = l.file.Close()
	if err := rotateIfNeeded(l.name, l.maxBytes, l.backups); err != nil {
		// Reopen the current path even when rotation fails so diagnostics never
		// turn a recoverable file-system problem into an application failure.
	}
	file, err := os.OpenFile(l.name, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		l.file = nil
		l.log.SetOutput(io.Discard)
		return
	}
	l.file = file
	l.log.SetOutput(file)
}

func (l *Logger) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	return l.file.Close()
}

func IncidentID() string {
	return fmt.Sprintf("WAPP-%s", time.Now().UTC().Format("20060102T150405.000000000Z"))
}

func (l *Logger) RecordPanic(incidentID string, recovered any) {
	l.RecordTaskPanic(incidentID, "main", recovered)
}

func (l *Logger) RecordTaskPanic(incidentID, task string, recovered any) {
	if l == nil {
		return
	}
	task = strings.TrimSpace(task)
	if task == "" {
		task = "background"
	}
	if len(task) > 120 {
		task = task[:120]
	}
	l.Printf("incident_id=%s task=%q result=panic recovered_type=%T\n%s", incidentID, task, recovered, debug.Stack())
}

// SetDefaultLogger makes the process logger available to guarded background
// tasks without giving UI packages ownership of the log file lifecycle.
func SetDefaultLogger(logger *Logger) {
	defaultLoggerMu.Lock()
	defaultLogger = logger
	defaultLoggerMu.Unlock()
}

// RecordBackgroundPanic records only a stable task origin and the recovered
// value type. The recovered value text is deliberately excluded because it
// may contain business data returned by a dependency.
func RecordBackgroundPanic(task string, recovered any) string {
	incidentID := IncidentID()
	defaultLoggerMu.RLock()
	logger := defaultLogger
	defaultLoggerMu.RUnlock()
	if logger != nil {
		logger.RecordTaskPanic(incidentID, task, recovered)
	}
	return incidentID
}
