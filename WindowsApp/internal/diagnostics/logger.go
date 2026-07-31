package diagnostics

import (
	"log"
	"os"
	"path/filepath"
	"sync"
)

type Logger struct {
	mu   sync.Mutex
	file *os.File
	log  *log.Logger
}

func New() (*Logger, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	dir = filepath.Join(dir, "ZhengshiWMS", "logs")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(filepath.Join(dir, "windowsapp.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}
	return &Logger{file: file, log: log.New(file, "", log.Ldate|log.Ltime|log.Lmicroseconds)}, nil
}

func (l *Logger) Printf(format string, args ...any) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.log.Printf(format, args...)
}

func (l *Logger) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	return l.file.Close()
}
