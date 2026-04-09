package logger

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"foundry/backend/appdata"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const maxLogFiles = 16

type Logger struct {
	verbose bool
	file    *os.File
	ctx     context.Context
}

func New(verbose bool) (*Logger, error) {
	filename := fmt.Sprintf("%s.log", time.Now().Format("2006-01-02"))
	logPath := filepath.Join(appdata.LogsPath(), filename)

	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	pruneOldLogs(appdata.LogsPath())

	return &Logger{
		verbose: verbose,
		file:    file,
	}, nil
}

func pruneOldLogs(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	var logFiles []os.DirEntry
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".log" {
			logFiles = append(logFiles, e)
		}
	}

	if len(logFiles) <= maxLogFiles {
		return
	}

	sort.Slice(logFiles, func(i, j int) bool {
		ii, _ := logFiles[i].Info()
		jj, _ := logFiles[j].Info()
		return ii.ModTime().Before(jj.ModTime())
	})

	for _, f := range logFiles[:len(logFiles)-maxLogFiles] {
		os.Remove(filepath.Join(dir, f.Name()))
	}
}

func (l *Logger) SetContext(ctx context.Context) {
	l.ctx = ctx
}

func (l *Logger) Error(format string, args ...interface{}) {
	l.log("ERROR", format, args...)
}

func (l *Logger) Info(format string, args ...interface{}) {
	if !l.verbose {
		return
	}
	l.log("INFO", format, args...)
}

func (l *Logger) Debug(format string, args ...interface{}) {
	if !l.verbose {
		return
	}
	l.log("DEBUG", format, args...)
}

func (l *Logger) log(level, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	timestamp := time.Now().Format("15:04:05")
	line := fmt.Sprintf("[%s] %s: %s", timestamp, level, msg)

	fmt.Fprintln(l.file, line)

	if l.ctx != nil {
		runtime.EventsEmit(l.ctx, "log", map[string]string{
			"level":   level,
			"message": msg,
		})
	}
}

func (l *Logger) Close() {
	if l.file != nil {
		l.file.Close()
	}
}
