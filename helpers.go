package main

import "os"

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func truncateText(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
