package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// Logger with rotation support
type RotatingLogger struct {
	mu          sync.Mutex
	file        *os.File
	filename    string
	maxSizeMB   int
	maxFiles    int
	currentSize int64
}

var (
	appLogger    *RotatingLogger
	requestCount int64
	lastLogTime  = time.Now()
	logInterval  = time.Minute // Summary log every minute
)

// InitLogger initializes the rotating logger
func InitLogger(filename string, maxSizeMB, maxFiles int) error {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	appLogger = &RotatingLogger{
		filename:  filename,
		maxSizeMB: maxSizeMB,
		maxFiles:  maxFiles,
	}

	return appLogger.openFile()
}

func (l *RotatingLogger) openFile() error {
	f, err := os.OpenFile(l.filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	l.file = f

	info, err := f.Stat()
	if err == nil {
		l.currentSize = info.Size()
	}

	// Also write to stdout
	log.SetOutput(io.MultiWriter(os.Stdout, f))
	return nil
}

func (l *RotatingLogger) Write(p []byte) (n int, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if rotation needed
	if l.currentSize+int64(len(p)) > int64(l.maxSizeMB)*1024*1024 {
		l.rotate()
	}

	n, err = l.file.Write(p)
	l.currentSize += int64(n)
	return
}

func (l *RotatingLogger) rotate() {
	l.file.Close()

	// Remove oldest files
	for i := l.maxFiles - 1; i >= 1; i-- {
		oldName := fmt.Sprintf("%s.%d", l.filename, i)
		newName := fmt.Sprintf("%s.%d", l.filename, i+1)
		os.Rename(oldName, newName)
	}

	// Rename current to .1
	os.Rename(l.filename, l.filename+".1")

	// Remove files beyond maxFiles
	for i := l.maxFiles + 1; i <= l.maxFiles+5; i++ {
		os.Remove(fmt.Sprintf("%s.%d", l.filename, i))
	}

	// Open new file
	l.openFile()
	l.currentSize = 0
	log.Println("Log rotated")
}

func (l *RotatingLogger) Close() {
	if l.file != nil {
		l.file.Close()
	}
}

// LogRequest increments counter (called per request)
func LogRequest() {
	atomic.AddInt64(&requestCount, 1)
}

// LogSummary logs summary periodically (call from goroutine)
// Pass a context to enable graceful shutdown
func LogSummary(ctx context.Context) {
	ticker := time.NewTicker(logInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			count := atomic.SwapInt64(&requestCount, 0)
			if count > 0 {
				log.Printf("[SUMMARY] Received %d reports in last minute", count)
			}
		case <-ctx.Done():
			log.Println("[INFO] LogSummary shutting down")
			return
		}
	}
}

// LogError logs errors immediately
func LogError(format string, v ...interface{}) {
	log.Printf("[ERROR] "+format, v...)
}

// LogInfo logs info (startup, shutdown, important events)
func LogInfo(format string, v ...interface{}) {
	log.Printf("[INFO] "+format, v...)
}
