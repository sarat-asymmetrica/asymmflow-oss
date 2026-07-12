package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

func appDataDirForDiagnostics() string {
	if runtime.GOOS == "windows" {
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, "AsymmFlow")
		}
		return ""
	}
	if homeDir, err := os.UserHomeDir(); err == nil {
		return filepath.Join(homeDir, ".local", "share", "AsymmFlow")
	}
	return ""
}

func appDebugLogPath() string {
	if dataDir := appDataDirForDiagnostics(); dataDir != "" {
		return filepath.Join(dataDir, "logs", "app_debug.log")
	}
	return "app_debug.log"
}

func startupDiagnosticPaths() []string {
	paths := []string{filepath.Join(os.TempDir(), "asymmflow_startup.log")}
	if dataDir := appDataDirForDiagnostics(); dataDir != "" {
		paths = append(paths, filepath.Join(dataDir, "logs", "startup.log"))
	}
	return paths
}

func resetStartupDiagnostics() {
	for _, path := range startupDiagnosticPaths() {
		_ = os.MkdirAll(filepath.Dir(path), 0700)
		_ = os.Remove(path)
	}
	appendStartupDiagnostic("MAIN: diagnostic log reset")
}

func appendStartupDiagnostic(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	line := fmt.Sprintf("%s %s\n", time.Now().Format("2006-01-02 15:04:05.000"), msg)
	for _, path := range startupDiagnosticPaths() {
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			continue
		}
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			continue
		}
		_, _ = f.WriteString(line)
		_ = f.Close()
	}
}
