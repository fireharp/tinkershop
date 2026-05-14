package config

import (
	"testing"
	"time"
)

func TestSinceTimeDate(t *testing.T) {
	cfg := Config{Since: "2026-05-01"}
	since, err := cfg.SinceTime(time.Date(2026, 5, 14, 12, 0, 0, 0, time.Local))
	if err != nil {
		t.Fatal(err)
	}
	if since == nil {
		t.Fatal("since is nil")
	}
	if since.In(time.Local).Format("2006-01-02 15:04") != "2026-05-01 00:00" {
		t.Fatalf("since = %s", since.In(time.Local).Format("2006-01-02 15:04"))
	}
}

func TestSinceTimeDayWindow(t *testing.T) {
	now := time.Date(2026, 5, 14, 12, 30, 0, 0, time.Local)
	cfg := Config{Since: "14d"}
	since, err := cfg.SinceTime(now)
	if err != nil {
		t.Fatal(err)
	}
	if since == nil {
		t.Fatal("since is nil")
	}
	if since.In(time.Local).Format("2006-01-02 15:04") != "2026-04-30 12:30" {
		t.Fatalf("since = %s", since.In(time.Local).Format("2006-01-02 15:04"))
	}
}
