package dashboard

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/AgusLoza2021/Thoughtline/internal/pending"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// pendingEmptyMsg is the exact copy from design decision E.
const pendingEmptyMsg = "No pending events. Passive capture is OFF — enable via tl_capture_passive."

// pendingLoadedMsg carries the list of pending events from storage.
type pendingLoadedMsg struct {
	events []pending.Event
	err    error
}

// PendingScreen lists pending events for the current project.
type PendingScreen struct {
	storage *storage.Storage
	project string
	events  []pending.Event
	cursor  int
}

func newPendingScreenFull(st *storage.Storage, project string) *PendingScreen {
	return &PendingScreen{storage: st, project: project}
}

func (p *PendingScreen) Title() string { return "Pending Events" }

func (p *PendingScreen) Init() tea.Cmd { return p.loadCmd() }

func (p *PendingScreen) OnFocus() tea.Cmd { return p.loadCmd() }

func (p *PendingScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			return p, func() tea.Msg { return popScreenCmd{} }
		case tea.KeyEnter:
			if len(p.events) > 0 && p.cursor < len(p.events) {
				ev := p.events[p.cursor]
				detail := fmt.Sprintf("[%s] %s\n\n%s", ev.EventType, ev.Hash, ev.Payload)
				return p, func() tea.Msg {
					return pushScreenCmd{screen: newDetailScreen(detail)}
				}
			}
		}
		switch msg.String() {
		case "q":
			return p, func() tea.Msg { return popScreenCmd{} }
		case "j", "down":
			if p.cursor < len(p.events)-1 {
				p.cursor++
			}
		case "k", "up":
			if p.cursor > 0 {
				p.cursor--
			}
		}
	case pendingLoadedMsg:
		if msg.err == nil {
			p.events = msg.events
		}
	}
	return p, nil
}

func (p *PendingScreen) View(width, height int, pal palette) string {
	var b strings.Builder
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(pal.Foreground)
	b.WriteString(titleStyle.Render("Pending Events"))
	b.WriteString("\n\n")

	if len(p.events) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(pal.Muted).Render(pendingEmptyMsg))
		return b.String()
	}

	mutedStyle := lipgloss.NewStyle().Foreground(pal.Muted)
	for i, ev := range p.events {
		prefix := "  "
		if i == p.cursor {
			prefix = lipgloss.NewStyle().Foreground(pal.Cursor).Render("▸") + " "
		}
		// 8-char hash prefix
		hashPfx := ev.Hash
		if len(hashPfx) > 8 {
			hashPfx = hashPfx[:8]
		}
		// Human-readable captured_at
		ts := ev.CapturedAt.Format("2006-01-02 15:04")
		// Payload preview (first 60 chars)
		payload := ev.Payload
		runes := []rune(payload)
		if len(runes) > 60 {
			payload = string(runes[:60]) + "…"
		}
		line := fmt.Sprintf("%s %s [%s] %s  %s",
			prefix,
			ev.EventType,
			hashPfx,
			mutedStyle.Render(ts),
			payload,
		)
		b.WriteString(line + "\n")
	}
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render("esc/q back • enter open full payload"))
	return b.String()
}

func (p *PendingScreen) loadCmd() tea.Cmd {
	st := p.storage
	project := p.project
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		events, err := st.ListPending(ctx, storage.ListPendingParams{
			Project: project,
			Status:  "pending",
			Limit:   50,
		})
		return pendingLoadedMsg{events: events, err: err}
	}
}
