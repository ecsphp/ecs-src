package reporter

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

// Progress is an ECS-style progress bar rendered on a terminal. It is a no-op
// when the writer is not a terminal (piped output, CI), so scripts stay clean.
type Progress struct {
	w      *os.File
	width  int
	active bool
	total  int
	cur    int
	frame  int
	mu     sync.Mutex
}

// iconFrames is a pulsing bloom that grows from a dot to a star and back, advanced one step per render.
var iconFrames = []string{"·", "∘", "○", "◯", "✦", "✧", "✦", "◯", "○", "∘"}

// NewProgress returns a progress bar; it only draws when w is a terminal.
func NewProgress(w *os.File) *Progress {
	p := &Progress{w: w, width: 28}
	if _, ok := winsize(w); ok {
		p.active = true
	}
	return p
}

// Start records the total and draws the empty bar.
func (p *Progress) Start(total int) {
	if p == nil {
		return
	}
	p.total = total
	if p.active {
		p.render()
	}
}

// Advance moves the bar one step; safe to call from multiple workers.
func (p *Progress) Advance() {
	if p == nil || !p.active {
		return
	}
	p.mu.Lock()
	if p.cur < p.total {
		p.cur++
	}
	p.render()
	p.mu.Unlock()
}

// Total is the number of files the bar was started with.
func (p *Progress) Total() int {
	if p == nil {
		return 0
	}
	return p.total
}

// Finish clears the bar line.
func (p *Progress) Finish() {
	if p == nil || !p.active {
		return
	}
	_, _ = fmt.Fprintf(p.w, "\r%s\r", strings.Repeat(" ", p.width+24))
}

func (p *Progress) render() {
	if p.total <= 0 {
		return
	}
	icon := iconFrames[p.frame%len(iconFrames)]
	p.frame++
	_, _ = fmt.Fprintf(p.w, "\r %s", icon)
}
