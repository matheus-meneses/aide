package agent

import (
	"testing"
	"time"
)

type fakeClock struct{ t time.Time }

func (c *fakeClock) Now() time.Time { return c.t }

func (c *fakeClock) advance(d time.Duration) { c.t = c.t.Add(d) }

func TestDueBriefings(t *testing.T) {
	base := time.Date(2026, 6, 15, 9, 0, 0, 0, time.Local)

	tests := []struct {
		name  string
		now   time.Time
		times []string
		fired map[string]bool
		want  []string
	}{
		{
			name:  "fires at configured minute",
			now:   base,
			times: []string{"09:00", "17:00"},
			fired: map[string]bool{},
			want:  []string{"09:00"},
		},
		{
			name:  "skips already fired",
			now:   base,
			times: []string{"09:00"},
			fired: map[string]bool{"09:00": true},
			want:  nil,
		},
		{
			name:  "nothing due at other time",
			now:   base,
			times: []string{"08:00", "17:00"},
			fired: map[string]bool{},
			want:  nil,
		},
		{
			name:  "fires within the catch-up window",
			now:   base.Add(20 * time.Second),
			times: []string{"09:00"},
			fired: map[string]bool{},
			want:  []string{"09:00"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := dueBriefings(tc.now, tc.times, tc.fired)
			if len(got) != len(tc.want) {
				t.Fatalf("dueBriefings = %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("dueBriefings[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestSessionEviction(t *testing.T) {
	clk := &fakeClock{t: time.Date(2026, 6, 15, 9, 0, 0, 0, time.Local)}
	m := newSessionManager(time.Hour, clk)

	m.getOrCreate("stale")

	clk.advance(90 * time.Minute)
	m.getOrCreate("fresh")

	m.evictExpired()

	if _, ok := m.sessions["stale"]; ok {
		t.Fatalf("expected stale session to be evicted")
	}
	if _, ok := m.sessions["fresh"]; !ok {
		t.Fatalf("expected fresh session to survive eviction")
	}
}
