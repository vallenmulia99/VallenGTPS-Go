package ui

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Colors
var (
	ColorPink   = tcell.ColorHotPink
	ColorRed    = tcell.ColorRed
	ColorGreen  = tcell.ColorGreen
	ColorYellow = tcell.ColorYellow
	ColorWhite  = tcell.ColorWhite
)

// LogPanel represents a single scrollable log panel
type LogPanel struct {
	tui        *TerminalUI
	textView   *tview.TextView
	mu         sync.Mutex
	autoScroll bool
}

func NewLogPanel(tui *TerminalUI, maxLines int) *LogPanel {
	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetMaxLines(maxLines)
	tv.SetBorder(false)

	return &LogPanel{
		tui:        tui,
		textView:   tv,
		autoScroll: true,
	}
}

func (lp *LogPanel) Write(p []byte) (n int, err error) {
	lp.mu.Lock()
	defer lp.mu.Unlock()

	n, err = lp.textView.Write(p)
	if lp.autoScroll {
		lp.textView.ScrollToEnd()
	}
	if lp.tui != nil {
		lp.tui.RequestDraw()
	}
	return n, err
}

func (lp *LogPanel) GetView() *tview.TextView {
	return lp.textView
}

// TerminalUI is the main TUI controller
type TerminalUI struct {
	app          *tview.Application
	httpsPanel   *LogPanel
	gtpsPanel    *LogPanel
	statusText   *tview.TextView
	grid         *tview.Grid
	serverStatus string
	statusMu     sync.Mutex
	running      atomic.Bool
	redrawCh     chan struct{}
	stopCh       chan struct{}
	stopOnce     sync.Once
}

func NewTerminalUI() *TerminalUI {
	tui := &TerminalUI{
		app:          tview.NewApplication(),
		serverStatus: "STARTING",
		redrawCh:     make(chan struct{}, 1),
		stopCh:       make(chan struct{}),
	}
	tui.httpsPanel = NewLogPanel(tui, 1000)
	tui.gtpsPanel = NewLogPanel(tui, 1000)
	return tui
}

func (tui *TerminalUI) Build() {
	// Header
	header := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	fmt.Fprintf(header, "[#FF69B4::b]╔════════════════════════════════════════════════════════════════════╗[-]\n")
	fmt.Fprintf(header, "[#FF69B4::b]║                           VALL-GTPS-GO                             ║[-]\n")
	fmt.Fprintf(header, "[#FF69B4::b]╚════════════════════════════════════════════════════════════════════╝[-]")
	header.SetBorder(false)

	// Left panel - HTTPS
	httpsBox := tview.NewFrame(tui.httpsPanel.GetView()).
		SetBorders(0, 0, 1, 0, 0, 0).
		AddText("[red::b]HTTPS REQUEST[-]", true, tview.AlignCenter, tcell.ColorRed)
	httpsBox.SetBorder(true)
	httpsBox.SetBorderColor(tcell.ColorRed)

	// Right panel - GTPS
	gtpsBox := tview.NewFrame(tui.gtpsPanel.GetView()).
		SetBorders(0, 0, 1, 0, 0, 0).
		AddText("[#FF69B4::b]VALLEN GTPS REQUEST[-]", true, tview.AlignCenter, ColorPink)
	gtpsBox.SetBorder(true)
	gtpsBox.SetBorderColor(ColorPink)

	// Status bar
	tui.statusText = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	tui.updateStatusBar()
	tui.statusText.SetBorder(false)

	// Grid layout
	tui.grid = tview.NewGrid().
		SetRows(3, 0, 1).                         // Header: 3 rows, Content: fill, Status: 1 row
		SetColumns(0, 0).                         // Two equal columns
		AddItem(header, 0, 0, 1, 2, 0, 0, false). // Header spans both columns
		AddItem(httpsBox, 1, 0, 1, 1, 0, 0, false). // Left panel
		AddItem(gtpsBox, 1, 1, 1, 1, 0, 0, true).  // Right panel (focused by default)
		AddItem(tui.statusText, 2, 0, 1, 2, 0, 0, false) // Status spans both columns

	tui.grid.SetBorder(false)
	tui.app.SetRoot(tui.grid, true)
	tui.app.EnableMouse(true)

	// Tab to toggle focus between panels
	tui.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			if tui.app.GetFocus() == tui.gtpsPanel.GetView() {
				tui.app.SetFocus(tui.httpsPanel.GetView())
			} else {
				tui.app.SetFocus(tui.gtpsPanel.GetView())
			}
			return nil
		}
		return event
	})
}

func (tui *TerminalUI) updateStatusBar() {
	tui.statusMu.Lock()
	defer tui.statusMu.Unlock()

	if tui.statusText == nil {
		return
	}

	var statusColor string
	var statusSymbol string

	switch tui.serverStatus {
	case "ONLINE":
		statusColor = "green"
		statusSymbol = "●"
	case "OFFLINE":
		statusColor = "red"
		statusSymbol = "●"
	case "STARTING":
		statusColor = "yellow"
		statusSymbol = "◐"
	default:
		statusColor = "white"
		statusSymbol = "○"
	}

	tui.statusText.Clear()
	fmt.Fprintf(tui.statusText, "[%s::b]%s %s[-]  |  VALL-GTPS-GO SERVER  |  [yellow]Tab[-] Switch Panel  |  [yellow]Ctrl+C[-] Exit", statusColor, statusSymbol, tui.serverStatus)
}

func (tui *TerminalUI) RequestDraw() {
	if !tui.running.Load() {
		return
	}
	select {
	case tui.redrawCh <- struct{}{}:
	default:
	}
}

func (tui *TerminalUI) startRedrawLoop() {
	go func() {
		ticker := time.NewTicker(25 * time.Millisecond) // Max ~40 FPS refresh
		defer ticker.Stop()

		var needsDraw bool
		for {
			select {
			case <-tui.stopCh:
				return
			case <-tui.redrawCh:
				needsDraw = true
			case <-ticker.C:
				if needsDraw && tui.running.Load() {
					needsDraw = false
					tui.app.QueueUpdateDraw(func() {})
				}
			}
		}
	}()
}

func (tui *TerminalUI) SetServerStatus(status string) {
	tui.statusMu.Lock()
	tui.serverStatus = status
	tui.statusMu.Unlock()

	if tui.running.Load() {
		tui.app.QueueUpdateDraw(func() {
			tui.updateStatusBar()
		})
	}
}

func (tui *TerminalUI) Start() error {
	tui.Build()
	tui.running.Store(true)
	tui.startRedrawLoop()
	defer func() {
		tui.running.Store(false)
		tui.stopOnce.Do(func() {
			close(tui.stopCh)
		})
	}()
	return tui.app.Run()
}

func (tui *TerminalUI) Stop() {
	if !tui.running.Swap(false) {
		return
	}
	tui.stopOnce.Do(func() {
		close(tui.stopCh)
	})
	tui.app.Stop()
}

func (tui *TerminalUI) GetHTTPSWriter() io.Writer {
	return tui.httpsPanel
}

func (tui *TerminalUI) GetGTPSWriter() io.Writer {
	return tui.gtpsPanel
}

// LogRouter routes log messages to appropriate panel based on prefix
type LogRouter struct {
	httpsWriter io.Writer
	gtpsWriter  io.Writer
	mu          sync.Mutex
}

func NewLogRouter(httpsWriter, gtpsWriter io.Writer) *LogRouter {
	return &LogRouter{
		httpsWriter: httpsWriter,
		gtpsWriter:  gtpsWriter,
	}
}

func (lr *LogRouter) Write(p []byte) (n int, err error) {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	line := strings.TrimRight(string(p), "\r\n")
	timestamp := time.Now().Format("15:04:05")

	// Route based on log prefix
	isHTTPS := false
	if len(line) > 0 {
		if containsAny(line, []string{"[HTTPS]", "HTTPS server", "server_data", "POST /growtopia", "GET /"}) {
			isHTTPS = true
		}
	}

	var formattedLine string
	if isHTTPS {
		formattedLine = fmt.Sprintf("[red][%s][-] %s\n", timestamp, line)
		lr.httpsWriter.Write([]byte(formattedLine))
	} else {
		formattedLine = fmt.Sprintf("[#FF69B4][%s][-] %s\n", timestamp, line)
		lr.gtpsWriter.Write([]byte(formattedLine))
	}

	return len(p), nil
}

func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}
