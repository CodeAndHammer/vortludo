package main

import (
	"os"
	"testing"
	"time"

	util "vortludo/internal/util"
)

func TestDirExists(t *testing.T) {
	dir := t.TempDir()
	if !util.DirExists(dir) {
		t.Errorf("Expected DirExists to return true for existing dir")
	}
	if util.DirExists(dir + "-notfound") {
		t.Errorf("Expected DirExists to return false for non-existent dir")
	}
}

func TestFormatUptime(t *testing.T) {
	cases := []struct {
		dur      time.Duration
		expected string
	}{
		{time.Second * 5, "5 seconds"},
		{time.Second * 65, "1 minute, 5 seconds"},
		{time.Second * 3665, "1 hour, 1 minute, 5 seconds"},
		{time.Second * 3600, "1 hour, 0 minutes, 0 seconds"},
		{time.Second * 60, "1 minute, 0 seconds"},
		{time.Second * 1, "1 second"},
	}
	for _, c := range cases {
		got := util.FormatUptime(c.dur)
		if got != c.expected {
			t.Errorf("FormatUptime(%v) = %q, want %q", c.dur, got, c.expected)
		}
	}
}

func TestPlural(t *testing.T) {
	if plural(1) != "" {
		t.Errorf("plural(1) = %q, want \"\"", plural(1))
	}
	if plural(2) != "s" {
		t.Errorf("plural(2) = %q, want \"s\"", plural(2))
	}
	if plural(0) != "s" {
		t.Errorf("plural(0) = %q, want \"s\"", plural(0))
	}
}

func TestGetEnvDuration(t *testing.T) {
	os.Setenv("TEST_DURATION", "2s")
	defer os.Unsetenv("TEST_DURATION")
	if got := util.GetEnvDuration("TEST_DURATION", time.Second); got != 2*time.Second {
		t.Errorf("GetEnvDuration = %v, want 2s", got)
	}
	os.Setenv("TEST_DURATION", "notaduration")
	if got := util.GetEnvDuration("TEST_DURATION", 3*time.Second); got != 3*time.Second {
		t.Errorf("GetEnvDuration fallback = %v, want 3s", got)
	}
	os.Unsetenv("TEST_DURATION")
	if got := util.GetEnvDuration("TEST_DURATION", 4*time.Second); got != 4*time.Second {
		t.Errorf("GetEnvDuration fallback unset = %v, want 4s", got)
	}
}

func TestGetEnvInt(t *testing.T) {
	os.Setenv("TEST_INT", "42")
	defer os.Unsetenv("TEST_INT")
	if got := util.GetEnvInt("TEST_INT", 7); got != 42 {
		t.Errorf("GetEnvInt = %d, want 42", got)
	}
	os.Setenv("TEST_INT", "notanint")
	if got := util.GetEnvInt("TEST_INT", 8); got != 8 {
		t.Errorf("GetEnvInt fallback = %d, want 8", got)
	}
	os.Unsetenv("TEST_INT")
	if got := util.GetEnvInt("TEST_INT", 9); got != 9 {
		t.Errorf("GetEnvInt fallback unset = %d, want 9", got)
	}
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
