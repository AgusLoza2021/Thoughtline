package dashboard

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
	"github.com/AgusLoza2021/Thoughtline/internal/pending"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// ─── InboxScreen ─────────────────────────────────────────────────────────────

// inboxLoadedMsg carries the list of pending events from storage.
type inboxLoadedMsg struct {
	events []pending.Event
	err    error
}

// inboxActionMsg signals an inbox action completed (accept or reject).
type inboxActionMsg struct {
	success bool
	err     error
}

// InboxScreen is tabs[TabInbox]. It lists pending capture events and provides:
//   - [A] accept as-is — calls MarkPromoted after saving the memory
//   - [E] edit — pushes InboxEditScreen pre-filled with the pending payload
//   - [R] reject — calls MarkRejected
//   - enter — preview in DetailScreen (read-only)
type InboxScreen struct {
	storage *storage.Storage
	events  []pending.Event
	cursor  int
	status  string
	loaded  bool
}

// NewInboxScreen constructs an unloaded InboxScreen.
func NewInboxScreen(st *storage.Storage) *InboxScreen {
	return &InboxScreen{storage: st}
}

// ─── Screen interface ─────────────────────────────────────────────────────────

func (is *InboxScreen) Title() string { return "Inbox" }

func (is *InboxScreen) Init() tea.Cmd { return is.loadCmd() }

func (is *InboxScreen) OnFocus() tea.Cmd { return is.loadCmd() }

func (is *InboxScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return is.handleKey(msg)
	case inboxLoadedMsg:
		if msg.err == nil {
			is.events = msg.events
		}
		is.loaded = true
		is.cursor = clampCursor(is.cursor, len(is.events))
		return is, nil
	case inboxActionMsg:
		if msg.success {
			is.status = "Done"
		} else if msg.err != nil {
			is.status = fmt.Sprintf("Error: %v", msg.err)
		}
		return is, is.loadCmd()
	}
	return is, nil
}

func (is *InboxScreen) handleKey(msg tea.KeyMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if is.cursor < len(is.events)-1 {
			is.cursor++
		}
		return is, nil

	case "k", "up":
		if is.cursor > 0 {
			is.cursor--
		}
		return is, nil

	case "A":
		// Accept as-is: save to memories, then MarkPromoted
		if len(is.events) > 0 && is.cursor < len(is.events) {
			ev := is.events[is.cursor]
			return is, is.acceptCmd(ev)
		}
		return is, nil

	case "E":
		// Edit: push InboxEditScreen with pre-filled payload. Pass the full
		// event so the submit path can preserve Project / SessionID /
		// CapturedAt from the pending row (fidelity is contract per spec
		// Req 24 — see fix(dashboard): Inbox promotion fidelity).
		if len(is.events) > 0 && is.cursor < len(is.events) {
			ev := is.events[is.cursor]
			editScreen := NewInboxEditScreen(ev)
			editScreen.storage = is.storage
			return is, func() tea.Msg {
				return pushScreenCmd{screen: editScreen}
			}
		}
		return is, nil

	case "R":
		// Reject: call MarkRejected
		if len(is.events) > 0 && is.cursor < len(is.events) {
			ev := is.events[is.cursor]
			return is, is.rejectCmd(ev.ID)
		}
		return is, nil

	case "enter":
		// Preview in DetailScreen
		if len(is.events) > 0 && is.cursor < len(is.events) {
			ev := is.events[is.cursor]
			content := fmt.Sprintf("[%s] %s\n\n%s", ev.EventType, ev.Hash, ev.Payload)
			return is, func() tea.Msg {
				return pushScreenCmd{screen: newDetailScreenFull(content)}
			}
		}
		return is, nil
	}
	return is, nil
}

// ─── View ─────────────────────────────────────────────────────────────────────

func (is *InboxScreen) View(width, height int, p palette) string {
	var b strings.Builder
	mutedStyle := lipgloss.NewStyle().Foreground(p.Muted)
	brandStyle := lipgloss.NewStyle().Foreground(p.Brand)
	warnStyle := lipgloss.NewStyle().Foreground(p.Warning)
	cursorStyle := lipgloss.NewStyle().Foreground(p.NavActive)
	focusedBg := lipgloss.NewStyle().Background(p.FocusedBg)

	// Header
	title := lipgloss.NewStyle().Foreground(p.Foreground).Bold(true).Render("Inbox")
	b.WriteString(title)
	b.WriteString("\n")

	if !is.loaded || len(is.events) == 0 {
		b.WriteString(mutedStyle.Render("No pending captures yet."))
		b.WriteString("\n")
	} else {
		// Summary line
		summary := warnStyle.Render(fmt.Sprintf("%d pending captures", len(is.events)))
		b.WriteString(summary)
		b.WriteString("\n\n")

		// Rows
		for i, ev := range is.events {
			prefix := "  "
			if i == is.cursor {
				prefix = cursorStyle.Render("▸") + " "
			}

			typeTag := truncate(ev.EventType, 10)
			badge := "[" + brandStyle.Render(typeTag) + "]"
			// Show payload preview
			payload := ev.Payload
			runes := []rune(payload)
			if len(runes) > 60 {
				payload = string(runes[:60]) + "…"
			}
			timeStr := mutedStyle.Render(relTime(ev.CapturedAt))
			line := fmt.Sprintf("%s%-14s %-60s  %s", prefix, badge, payload, timeStr)
			if i == is.cursor {
				line = focusedBg.Render(line)
			}
			b.WriteString(line + "\n")
		}
	}

	b.WriteString("\n")

	// Status
	if is.status != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(p.Success).Render(is.status))
		b.WriteString("\n")
	}

	// Footer
	footer := mutedStyle.Render("↑↓ select · [A] accept · [E] edit · [R] reject · enter preview · q quit")
	b.WriteString(footer)
	return b.String()
}

// ─── Commands ─────────────────────────────────────────────────────────────────

func (is *InboxScreen) loadCmd() tea.Cmd {
	st := is.storage
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		// The Inbox is workspace-wide (per spec Req 24) — query unscoped so
		// pending captures from every project show up. Filtering by project
		// is a future UI concern, not a load-time invariant.
		events, err := st.ListPending(ctx, storage.ListPendingParams{
			Status: "pending",
			Limit:  50,
		})
		return inboxLoadedMsg{events: events, err: err}
	}
}

func (is *InboxScreen) acceptCmd(ev pending.Event) tea.Cmd {
	st := is.storage
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Parse the optional proposed_* fields out of the payload, falling
		// back to sensible defaults when the payload is unstructured. This
		// preserves the capture even when the hook didn't shape the fields,
		// and — critically — keeps the memory.Type as a real memory type
		// rather than a raw hook EventType string (e.g. "PostToolUse").
		proposedType, proposedTitle, proposedContent := parseProposedMemory(ev.Payload, ev.EventType)

		brainID, err := st.ResolveOrCreateBrainID(ctx, ev.Project)
		if err != nil {
			return inboxActionMsg{err: err}
		}
		m := memory.Memory{
			Project:   ev.Project,
			Scope:     memory.ScopeProject,
			Type:      memory.Type(proposedType),
			Title:     proposedTitle,
			Content:   proposedContent,
			TopicKey:  fmt.Sprintf("inbox/%s/%d", ev.EventType, ev.ID),
			SessionID: ev.SessionID,
			CreatedAt: ev.CapturedAt,
		}
		saved, _, err := st.Save(ctx, brainID, m)
		if err != nil {
			// If the pending event references a session that no longer
			// exists (or was never persisted as a Session row), retry
			// without the link rather than losing the capture. Audit
			// fidelity matters less than not dropping the promoted memory.
			if errors.Is(err, storage.ErrSessionNotFound) && m.SessionID != "" {
				m.SessionID = ""
				saved, _, err = st.Save(ctx, brainID, m)
			}
			if err != nil {
				return inboxActionMsg{err: err}
			}
		}
		if err := st.MarkPromoted(ctx, ev.ID, saved.ID); err != nil {
			return inboxActionMsg{err: err}
		}
		return inboxActionMsg{success: true}
	}
}

func (is *InboxScreen) rejectCmd(id int64) tea.Cmd {
	st := is.storage
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := st.MarkRejected(ctx, id); err != nil {
			return inboxActionMsg{err: err}
		}
		return inboxActionMsg{success: true}
	}
}
