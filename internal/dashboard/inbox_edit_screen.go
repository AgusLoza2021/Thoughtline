package dashboard

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// ─── InboxEditScreen ─────────────────────────────────────────────────────────

// inboxEditSubmitMsg signals a successful edit+promote operation.
type inboxEditSubmitMsg struct {
	err error
}

// InboxEditScreen is pushed by InboxScreen when the user presses [E].
// It shows three editable fields:
//
//	focus=0 — type  (textinput)
//	focus=1 — title (textinput)
//	focus=2 — body  (textarea)
//
// Tab cycles through fields. ctrl+s submits (save memory + MarkPromoted).
// esc cancels without any DB changes (pushes popScreenCmd).
type InboxEditScreen struct {
	pendingID  int64
	typeField  textinput.Model
	titleField textinput.Model
	bodyField  textarea.Model
	focus      int // 0=type 1=title 2=body
	storage    *storage.Storage
	err        error
	done       bool
}

// NewInboxEditScreen constructs an InboxEditScreen pre-filled with the
// supplied values. focus starts at 0 (type field).
func NewInboxEditScreen(pendingID int64, typeName, title, body string) *InboxEditScreen {
	tf := textinput.New()
	tf.Placeholder = "memory type (e.g. decision)"
	tf.CharLimit = 50
	tf.SetValue(typeName)
	tf.Focus()

	ti := textinput.New()
	ti.Placeholder = "title"
	ti.CharLimit = 200
	ti.SetValue(title)

	ta := textarea.New()
	ta.Placeholder = "memory body…"
	ta.SetValue(body)
	ta.SetWidth(60)
	ta.SetHeight(6)

	return &InboxEditScreen{
		pendingID:  pendingID,
		typeField:  tf,
		titleField: ti,
		bodyField:  ta,
		focus:      0,
	}
}

// ─── Screen interface ─────────────────────────────────────────────────────────

func (s *InboxEditScreen) Title() string { return "Edit Capture" }

func (s *InboxEditScreen) Init() tea.Cmd { return textinput.Blink }

func (s *InboxEditScreen) OnFocus() tea.Cmd { return textinput.Blink }

// InputFocused always returns true so the flatModel InputFocused guard routes
// all keys to this screen (no global hotkey interference while editing).
func (s *InboxEditScreen) InputFocused() bool { return true }

func (s *InboxEditScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return s.handleKey(msg)
	case inboxEditSubmitMsg:
		if msg.err != nil {
			s.err = msg.err
		} else {
			s.done = true
		}
		return s, func() tea.Msg { return popScreenCmd{} }
	}
	// Delegate to the focused field.
	return s, s.updateFocusedField(msg)
}

func (s *InboxEditScreen) handleKey(msg tea.KeyMsg) (Screen, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		return s, func() tea.Msg { return popScreenCmd{} }

	case tea.KeyTab:
		s.cycleFocus(+1)
		return s, textinput.Blink

	case tea.KeyShiftTab:
		s.cycleFocus(-1)
		return s, textinput.Blink

	case tea.KeyCtrlS:
		if s.storage != nil {
			return s, s.promoteCmd()
		}
		// No storage (unit tests without storage) — just pop.
		return s, func() tea.Msg { return popScreenCmd{} }
	}

	// Delegate typing to the active field.
	return s, s.updateFocusedField(msg)
}

// cycleFocus moves focus by delta (-1 or +1), wrapping at the 3 fields.
func (s *InboxEditScreen) cycleFocus(delta int) {
	n := 3
	s.focus = ((s.focus + delta) % n + n) % n
	// Update focus state on each widget.
	if s.focus == 0 {
		s.typeField.Focus()
		s.titleField.Blur()
		s.bodyField.Blur()
	} else if s.focus == 1 {
		s.typeField.Blur()
		s.titleField.Focus()
		s.bodyField.Blur()
	} else {
		s.typeField.Blur()
		s.titleField.Blur()
		s.bodyField.Focus()
	}
}

// updateFocusedField passes the message to the currently-focused widget and
// syncs its model back.
func (s *InboxEditScreen) updateFocusedField(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch s.focus {
	case 0:
		s.typeField, cmd = s.typeField.Update(msg)
	case 1:
		s.titleField, cmd = s.titleField.Update(msg)
	case 2:
		s.bodyField, cmd = s.bodyField.Update(msg)
	}
	return cmd
}

// ─── View ─────────────────────────────────────────────────────────────────────

func (s *InboxEditScreen) View(width, height int, p palette) string {
	var b strings.Builder

	mutedStyle := lipgloss.NewStyle().Foreground(p.Muted)
	labelStyle := lipgloss.NewStyle().Foreground(p.Brand)
	errStyle := lipgloss.NewStyle().Foreground(p.Error)

	b.WriteString(lipgloss.NewStyle().Foreground(p.Foreground).Bold(true).Render("Edit Capture"))
	b.WriteString("\n\n")

	// Type field
	focused0 := s.focus == 0
	label0 := labelStyle.Render("Type:")
	if focused0 {
		label0 = lipgloss.NewStyle().Foreground(p.NavActive).Bold(true).Render("Type:")
	}
	b.WriteString(fmt.Sprintf("%s %s\n", label0, s.typeField.View()))

	// Title field
	focused1 := s.focus == 1
	label1 := labelStyle.Render("Title:")
	if focused1 {
		label1 = lipgloss.NewStyle().Foreground(p.NavActive).Bold(true).Render("Title:")
	}
	b.WriteString(fmt.Sprintf("%s %s\n", label1, s.titleField.View()))

	// Body field
	focused2 := s.focus == 2
	label2 := labelStyle.Render("Body:")
	if focused2 {
		label2 = lipgloss.NewStyle().Foreground(p.NavActive).Bold(true).Render("Body:")
	}
	b.WriteString(fmt.Sprintf("%s\n%s\n", label2, s.bodyField.View()))

	if s.err != nil {
		b.WriteString(errStyle.Render(fmt.Sprintf("Error: %v", s.err)))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(mutedStyle.Render("tab cycle fields · ctrl+s save · esc cancel"))
	return b.String()
}

// ─── Commands ─────────────────────────────────────────────────────────────────

func (s *InboxEditScreen) promoteCmd() tea.Cmd {
	st := s.storage
	pendingID := s.pendingID
	typeName := s.typeField.Value()
	title := s.titleField.Value()
	body := s.bodyField.Value()

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Resolve or create brain from project — use empty project (global).
		brainID, err := st.ResolveOrCreateBrainID(ctx, "test-workspace")
		if err != nil {
			return inboxEditSubmitMsg{err: err}
		}

		m := memory.Memory{
			Project:  "test-workspace",
			Scope:    memory.ScopeProject,
			Type:     memory.Type(typeName),
			Title:    title,
			Content:  body,
			TopicKey: fmt.Sprintf("inbox/edit/%s/%d", typeName, pendingID),
		}
		saved, _, err := st.Save(ctx, brainID, m)
		if err != nil {
			return inboxEditSubmitMsg{err: err}
		}
		if err := st.MarkPromoted(ctx, pendingID, saved.ID); err != nil {
			return inboxEditSubmitMsg{err: err}
		}
		return inboxEditSubmitMsg{}
	}
}
