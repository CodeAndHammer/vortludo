package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		logWarn("Error checking directory existence: %v", err)
		return false
	}
	return info.IsDir()
}

func formatUptime(d time.Duration) string {
	seconds := int(d.Seconds()) % 60
	minutes := int(d.Minutes()) % 60
	hours := int(d.Hours())
	switch {
	case hours > 0:
		return fmt.Sprintf("%d hour%s, %d minute%s, %d second%s",
			hours, plural(hours),
			minutes, plural(minutes),
			seconds, plural(seconds))
	case minutes > 0:
		return fmt.Sprintf("%d minute%s, %d second%s",
			minutes, plural(minutes),
			seconds, plural(seconds))
	default:
		return fmt.Sprintf("%d second%s", seconds, plural(seconds))
	}
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		logWarn("Invalid duration for %s: %v, using default %v", key, err, fallback)
		return fallback
	}
	return d
}

func getEnvInt(key string, fallback int) int {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	i, err := parseInt(val)
	if err != nil {
		logWarn("Invalid int for %s: %v, using default %d", key, err, fallback)
		return fallback
	}
	return i
}

func parseInt(val string) (int, error) {
	return strconv.Atoi(val)
}

func logInfo(format string, v ...any) {
	log.Printf("[INFO] "+format, v...)
}

func logWarn(format string, v ...any) {
	log.Printf("[WARN] "+format, v...)
}

func logFatal(format string, v ...any) {
	log.Fatalf("[FATAL] "+format, v...)
}
