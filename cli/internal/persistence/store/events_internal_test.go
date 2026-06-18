package store

import (
	"testing"
	"time"
)

func TestParseEventDuration(t *testing.T) {
	cases := map[string]time.Duration{
		"1h00m": time.Hour,
		"30m":   30 * time.Minute,
		"2h":    2 * time.Hour,
		"1h30m": 90 * time.Minute,
		"":      0,
		"abc":   0,
	}
	for in, want := range cases {
		if got := parseEventDuration(in); got != want {
			t.Errorf("parseEventDuration(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestParseEventTimes(t *testing.T) {
	start, dur, ok := parseEventTimes("2026-06-18", "09:15 (30m)")
	if !ok {
		t.Fatal("expected ok for valid detail")
	}
	want := time.Date(2026, 6, 18, 9, 15, 0, 0, time.Local)
	if !start.Equal(want) {
		t.Errorf("start = %v, want %v", start, want)
	}
	if dur != 30*time.Minute {
		t.Errorf("dur = %v, want 30m", dur)
	}

	if _, _, ok := parseEventTimes("2026-06-18", "all day"); ok {
		t.Error("expected not ok for detail without time")
	}

	_, dur, ok = parseEventTimes("2026-06-18", "09:00")
	if !ok || dur != 0 {
		t.Errorf("expected ok with zero duration, got ok=%v dur=%v", ok, dur)
	}
}

func TestNextEventFrom(t *testing.T) {
	now := time.Date(2026, 6, 18, 9, 0, 0, 0, time.Local)
	today := "2026-06-18"
	tomorrow := "2026-06-19"

	ev := func(title, date, detail string) Item {
		return Item{Title: title, EntryDate: date, Detail: detail, Category: "event"}
	}

	t.Run("in-progress wins as Now", func(t *testing.T) {
		events := []Item{
			ev("ended", today, "07:00 (30m)"),
			ev("ongoing", today, "08:30 (1h00m)"),
			ev("later", today, "09:15 (30m)"),
		}
		got := nextEventFrom(events, now)
		if got == nil {
			t.Fatal("expected an event")
		}
		if got.Item.Title != "ongoing" {
			t.Fatalf("got %q, want ongoing", got.Item.Title)
		}
		if !got.InProgress {
			t.Error("expected in-progress")
		}
		if got.MinutesUntil != -30 {
			t.Errorf("minutesUntil = %d, want -30", got.MinutesUntil)
		}
	})

	t.Run("soonest upcoming", func(t *testing.T) {
		events := []Item{
			ev("far", tomorrow, "10:00 (1h00m)"),
			ev("soon", today, "09:14 (30m)"),
		}
		got := nextEventFrom(events, now)
		if got == nil || got.Item.Title != "soon" {
			t.Fatalf("got %v, want soon", got)
		}
		if got.InProgress {
			t.Error("did not expect in-progress")
		}
		if got.MinutesUntil != 14 {
			t.Errorf("minutesUntil = %d, want 14", got.MinutesUntil)
		}
	})

	t.Run("all ended returns nil", func(t *testing.T) {
		events := []Item{
			ev("a", today, "07:00 (30m)"),
			ev("b", today, "08:00 (30m)"),
		}
		if got := nextEventFrom(events, now); got != nil {
			t.Fatalf("expected nil, got %q", got.Item.Title)
		}
	})

	t.Run("ignores unparseable", func(t *testing.T) {
		events := []Item{
			ev("notime", today, "all day"),
			ev("ok", today, "09:30 (15m)"),
		}
		got := nextEventFrom(events, now)
		if got == nil || got.Item.Title != "ok" {
			t.Fatalf("got %v, want ok", got)
		}
	})
}

func TestImminentCount(t *testing.T) {
	now := time.Date(2026, 6, 18, 9, 0, 0, 0, time.Local)
	today := "2026-06-18"
	tomorrow := "2026-06-19"
	ev := func(date, detail string) Item {
		return Item{EntryDate: date, Detail: detail, Category: "event"}
	}

	events := []Item{
		ev(today, "07:00 (30m)"),   // ended, ignored
		ev(today, "08:45 (1h00m)"), // in progress
		ev(today, "09:05 (30m)"),   // within 10m
		ev(today, "09:30 (30m)"),   // outside 10m
		ev(tomorrow, "10:00 (1h)"), // far
		ev(today, "all day"),       // unparseable, ignored
	}

	if got := imminentCount(events, now, 10*time.Minute); got != 2 {
		t.Errorf("imminentCount = %d, want 2", got)
	}
	if got := imminentCount(events, now, 60*time.Minute); got != 3 {
		t.Errorf("imminentCount(60m) = %d, want 3", got)
	}
}
