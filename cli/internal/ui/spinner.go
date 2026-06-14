package ui

import (
	"fmt"
	"sync"
	"time"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Spinner shows an animated label on a TTY and degrades to a single static
// line when output is not a terminal (or color is disabled).
type Spinner struct {
	label   string
	mu      sync.Mutex
	stop    chan struct{}
	done    chan struct{}
	running bool
}

func NewSpinner(label string) *Spinner {
	return &Spinner{label: label}
}

func (s *Spinner) Start() {
	if !colorEnabled || !isTerminal(Out) {
		fmt.Fprintf(Out, "%s…\n", s.label)
		return
	}
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.stop = make(chan struct{})
	s.done = make(chan struct{})
	s.mu.Unlock()

	go func() {
		defer close(s.done)
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()
		i := 0
		for {
			select {
			case <-s.stop:
				return
			case <-ticker.C:
				fmt.Fprintf(Out, "\r%s %s", paint(styleInfo, spinnerFrames[i%len(spinnerFrames)]), s.label)
				i++
			}
		}
	}()
}

func (s *Spinner) halt() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	close(s.stop)
	s.mu.Unlock()
	<-s.done
	fmt.Fprint(Out, "\r\033[K")
}

// Succeed stops the spinner and prints a success line.
func (s *Spinner) Succeed(format string, args ...any) {
	s.halt()
	PrintSuccess(format, args...)
}

// Fail stops the spinner and prints an error line to stderr.
func (s *Spinner) Fail(format string, args ...any) {
	s.halt()
	PrintError(format, args...)
}

// Stop clears the spinner without printing a result line.
func (s *Spinner) Stop() { s.halt() }

// Progress prints a determinate progress bar (TTY) or periodic percentages.
func Progress(label string, current, total int) {
	if total <= 0 {
		return
	}
	pct := current * 100 / total
	if !colorEnabled || !isTerminal(Out) {
		if current == total {
			fmt.Fprintf(Out, "%s: 100%%\n", label)
		}
		return
	}
	const width = 24
	filled := pct * width / 100
	bar := ""
	for i := 0; i < width; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	fmt.Fprintf(Out, "\r%s %s %3d%%", label, paint(styleInfo, bar), pct)
	if current >= total {
		fmt.Fprint(Out, "\n")
	}
}
