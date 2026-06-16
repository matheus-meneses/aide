package agent

import "time"

// Clock abstracts wall-clock reads so time-dependent behaviour (scheduling,
// session eviction, data-freshness checks) can be driven deterministically in
// tests. Production code uses realClock; tests inject a fake.
type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }
