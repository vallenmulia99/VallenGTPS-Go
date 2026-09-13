package ui

import (
	"sync"
)

// Panel represents an independent scrollable log panel
type Panel struct {
	mu          sync.RWMutex
	buffer      []LogEntry
	maxLines    int
	autoScroll  bool
	scrollPos   int
	name        string
}

// NewPanel creates a new panel with specified max lines
func NewPanel(name string, maxLines int) *Panel {
	return &Panel{
		buffer:     make([]LogEntry, 0, maxLines),
		maxLines:   maxLines,
		autoScroll: true,
		scrollPos:  0,
		name:       name,
	}
}

// AddLog adds a log entry to the panel (ring buffer implementation)
func (p *Panel) AddLog(entry LogEntry) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Add to buffer
	p.buffer = append(p.buffer, entry)

	// Maintain max lines (ring buffer behavior)
	if len(p.buffer) > p.maxLines {
		p.buffer = p.buffer[len(p.buffer)-p.maxLines:]
	}

	// Auto-scroll to bottom if enabled
	if p.autoScroll {
		p.scrollPos = len(p.buffer)
	}
}

// GetLogs returns visible logs based on scroll position
func (p *Panel) GetLogs(visibleLines int) []LogEntry {
	p.mu.RLock()
	defer p.mu.RUnlock()

	totalLogs := len(p.buffer)
	if totalLogs == 0 {
		return []LogEntry{}
	}

	// Calculate visible range
	end := p.scrollPos
	if p.autoScroll || end > totalLogs {
		end = totalLogs
	}

	start := end - visibleLines
	if start < 0 {
		start = 0
	}

	if end > totalLogs {
		end = totalLogs
	}

	return p.buffer[start:end]
}

// GetAllLogs returns all logs in buffer
func (p *Panel) GetAllLogs() []LogEntry {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	result := make([]LogEntry, len(p.buffer))
	copy(result, p.buffer)
	return result
}

// ScrollUp scrolls the panel up
func (p *Panel) ScrollUp(lines int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.autoScroll = false
	p.scrollPos -= lines
	if p.scrollPos < 0 {
		p.scrollPos = 0
	}
}

// ScrollDown scrolls the panel down
func (p *Panel) ScrollDown(lines int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.scrollPos += lines
	totalLogs := len(p.buffer)
	
	if p.scrollPos >= totalLogs {
		p.scrollPos = totalLogs
		p.autoScroll = true
	}
}

// EnableAutoScroll enables automatic scrolling to bottom
func (p *Panel) EnableAutoScroll() {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.autoScroll = true
	p.scrollPos = len(p.buffer)
}

// DisableAutoScroll disables automatic scrolling
func (p *Panel) DisableAutoScroll() {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.autoScroll = false
}

// IsAutoScroll returns whether auto-scroll is enabled
func (p *Panel) IsAutoScroll() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	return p.autoScroll
}

// GetBufferSize returns current buffer size
func (p *Panel) GetBufferSize() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	return len(p.buffer)
}

// Clear clears the panel buffer
func (p *Panel) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.buffer = make([]LogEntry, 0, p.maxLines)
	p.scrollPos = 0
	p.autoScroll = true
}
