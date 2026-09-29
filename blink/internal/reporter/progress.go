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
	mu     sync.Mutex
}

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
	_, _ = fmt.Fprintf(p.w, "\r%s\r", strings.Repeat(" ", p.width+20))
}

func (p *Progress) render() {
	if p.total <= 0 {
		return
	}
	pct := p.cur * 100 / p.total
	filled := p.cur * p.width / p.total
	var bar string
	switch {
	case filled >= p.width:
		bar = strings.Repeat("=", p.width)
	case filled == 0:
		bar = ">" + strings.Repeat(" ", p.width-1)
	default:
		bar = strings.Repeat("=", filled-1) + ">" + strings.Repeat(" ", p.width-filled)
	}
	_, _ = fmt.Fprintf(p.w, "\r %d/%d [%s] %3d%%", p.cur, p.total, bar, pct)
}
