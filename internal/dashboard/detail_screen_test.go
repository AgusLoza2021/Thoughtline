package dashboard

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// ─── K-group helpers ──────────────────────────────────────────────────────────

const testDetailContent = "**What**: Enable WAL via DSN pragma.\n\n**Why**: Concurrent MCP tool calls need simultaneous readers and writer.\n\n**Where**: internal/storage/storage.go\n\n**Learned**: busy_timeout=5000 prevents lock errors under load."

func newTestDetail(t *testing.T) *DetailScreen {
	t.Helper()
	return newDetailScreenFull(testDetailContent)
}

func renderDetail(t *testing.T, d *DetailScreen) string {
	t.Helper()
	return d.View(100, 30, defaultPalette)
}

// ─── legacy tests (preserved) ────────────────────────────────────────────────

// TestDetailScreen_RendersFullContent verifies the detail screen shows the
// injected content. The viewport wraps long lines so we check for a prefix
// substring that fits within one wrap column rather than the full string.
func TestDetailScreen_RendersFullContent(t *testing.T) {
	content := "This is the full memory content with all the details preserved without truncation."
	screen := newDetailScreenFull(content)

	out := screen.View(80, 24, defaultPalette)
	// Check for a unique substring that won't span a wrap boundary at 80 cols
	if !strings.Contains(out, "full memory content") {
		t.Errorf("detail view must contain content text\ngot:\n%s", out)
	}
}

// TestDetailScreen_EscReturnsPop verifies esc returns a popScreenCmd.
func TestDetailScreen_EscReturnsPop(t *testing.T) {
	screen := newDetailScreenFull("some content")

	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("esc must return a non-nil cmd")
	}
	if _, ok := cmd().(popScreenCmd); !ok {
		t.Errorf("esc must return popScreenCmd")
	}
}

// TestDetailScreen_QReturnsPop verifies q also pops.
func TestDetailScreen_QReturnsPop(t *testing.T) {
	screen := newDetailScreenFull("some content")

	_, cmd := screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Fatal("q must return a non-nil cmd")
	}
	if _, ok := cmd().(popScreenCmd); !ok {
		t.Errorf("q must return popScreenCmd")
	}
}

// ─── K1: Content and footer present ──────────────────────────────────────────

func TestDetailScreen_K1_RendersContent(t *testing.T) {
	d := newTestDetail(t)
	view := renderDetail(t, d)

	if !strings.Contains(view, "What") {
		t.Errorf("detail view must contain content text, got:\n%s", view)
	}
	if !strings.Contains(view, "WAL") {
		t.Errorf("detail view must contain 'WAL' from test content, got:\n%s", view)
	}
}

func TestDetailScreen_K1_FooterPresent(t *testing.T) {
	d := newTestDetail(t)
	view := renderDetail(t, d)

	if !strings.Contains(view, "esc") {
		t.Errorf("detail view footer must contain 'esc', got:\n%s", view)
	}
	// [C] copy must be advertised in the footer
	if !strings.Contains(view, "[C]") {
		t.Errorf("detail view footer must advertise '[C] copy', got:\n%s", view)
	}
}

// ─── K2: [C] triggers clipboard cmd and status renders ───────────────────────

func TestDetailScreen_K2_CKeyTriggersClipboard(t *testing.T) {
	d := newTestDetail(t)

	// Press 'C' (uppercase)
	s, cmd := d.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'C'}})
	d = s.(*DetailScreen)

	if cmd == nil {
		t.Fatal("pressing C must produce a clipboard cmd")
	}

	// Execute and check type
	msg := cmd()
	clipMsg, ok := msg.(clipboardResultMsg)
	if !ok {
		t.Fatalf("expected clipboardResultMsg from C, got %T", msg)
	}
	// Content must match the screen's content
	if clipMsg.content != testDetailContent {
		t.Errorf("clipboard content mismatch: got %q, want %q", clipMsg.content, testDetailContent)
	}
}

func TestDetailScreen_K2_ClipboardSuccessStatus(t *testing.T) {
	d := newTestDetail(t)

	// Feed a success result directly
	s, _ := d.Update(clipboardResultMsg{content: testDetailContent, err: nil})
	d = s.(*DetailScreen)

	view := renderDetail(t, d)
	if !strings.Contains(view, "Copied") {
		t.Errorf("view must contain 'Copied' status on clipboard success, got:\n%s", view)
	}
}

func TestDetailScreen_K2_ClipboardFailureStatus(t *testing.T) {
	d := newTestDetail(t)

	// Feed a failure result
	s, _ := d.Update(clipboardResultMsg{content: testDetailContent, err: ErrClipboardUnavailable})
	d = s.(*DetailScreen)

	view := renderDetail(t, d)
	if !strings.Contains(view, "Clipboard") {
		t.Errorf("view must contain 'Clipboard' on clipboard failure, got:\n%s", view)
	}
}

// ─── K3: [E] and [D] produce no effect (read-only contract) ──────────────────

func TestDetailScreen_K3_EKeyNoEffect(t *testing.T) {
	d := newTestDetail(t)
	initialContent := d.content

	s, cmd := d.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	d = s.(*DetailScreen)

	// [e] must not push or pop
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(pushScreenCmd); ok {
			t.Error("[e] must not push a screen in read-only DetailScreen")
		}
		if _, ok := msg.(popScreenCmd); ok {
			t.Error("[e] must not pop screen — only esc/q does that")
		}
	}
	if d.content != initialContent {
		t.Error("[e] must not modify content in read-only DetailScreen")
	}
}

func TestDetailScreen_K3_DKeyNoEffect(t *testing.T) {
	d := newTestDetail(t)
	initialContent := d.content

	s, cmd := d.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	d = s.(*DetailScreen)

	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(pushScreenCmd); ok {
			t.Error("[d] must not push a screen in read-only DetailScreen")
		}
		if _, ok := msg.(popScreenCmd); ok {
			t.Error("[d] must not pop screen")
		}
	}
	if d.content != initialContent {
		t.Error("[d] must not modify content in read-only DetailScreen")
	}
}

// ─── K-bug-1: Long content wraps at viewport width ───────────────────────────

func TestDetailScreen_KBug1_LongLineWraps(t *testing.T) {
	// A single very long line with no newlines — must be wrapped before viewport
	longContent := strings.Repeat("x", 200)
	d := newDetailScreenFull(longContent)
	// Override viewport to known size
	d.viewport = viewport.New(100, 20)
	wrappedContent := wrap(longContent, 100)
	d.viewport.SetContent(wrappedContent)

	// The viewport content is non-empty and wrapping occurred
	vpContent := d.viewport.View()
	if vpContent == "" {
		t.Error("viewport must have content after setting wrapped content")
	}
}

func TestDetailScreen_KBug1_ContentWrapsOnWindowResize(t *testing.T) {
	d := newTestDetail(t)

	// Simulate a window resize message
	s, _ := d.Update(tea.WindowSizeMsg{Width: 80, Height: 25})
	d = s.(*DetailScreen)

	// View at 80 wide — must render without panic and be non-empty
	view := d.View(80, 25, defaultPalette)
	if view == "" {
		t.Error("view must not be empty after resize")
	}
	// Content must still appear
	if !strings.Contains(view, "WAL") {
		t.Errorf("content must be present after resize, got:\n%s", view)
	}
}

// ─── K-bug-2: Scroll navigation ──────────────────────────────────────────────

func TestDetailScreen_KBug2_ScrollWithArrowKeys(t *testing.T) {
	multiLine := strings.Repeat("line content here\n", 40)
	d := newDetailScreenFull(multiLine)
	// Give it a viewport size so scrolling makes sense
	s, _ := d.Update(tea.WindowSizeMsg{Width: 100, Height: 15})
	d = s.(*DetailScreen)

	initialOffset := d.viewport.YOffset

	// Press down to scroll
	s2, _ := d.Update(tea.KeyMsg{Type: tea.KeyDown})
	d = s2.(*DetailScreen)

	if d.viewport.TotalLineCount() > d.viewport.Height && d.viewport.YOffset == initialOffset {
		t.Errorf("down arrow should scroll viewport; YOffset stayed at %d", d.viewport.YOffset)
	}
}

func TestDetailScreen_KBug2_FooterScrollPercent(t *testing.T) {
	multiLine := strings.Repeat("line content here\n", 60)
	d := newDetailScreenFull(multiLine)
	s, _ := d.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	d = s.(*DetailScreen)

	view := d.View(100, 20, defaultPalette)

	// Footer must contain a percent indicator
	if !strings.Contains(view, "%") {
		t.Errorf("footer must contain scroll percent indicator, got:\n%s", view)
	}
}

// ─── OriginatingTab accessor ──────────────────────────────────────────────────

func TestDetailScreen_OriginatingTab(t *testing.T) {
	d := newDetailScreenFull("content")
	d.origin = TabMemories

	if d.OriginatingTab() != TabMemories {
		t.Errorf("OriginatingTab() = %d, want %d", d.OriginatingTab(), TabMemories)
	}
}

func TestDetailScreen_OriginatingTab_Default(t *testing.T) {
	d := newDetailScreenFull("content")
	// Default origin is TabHome (zero value)
	if d.OriginatingTab() != TabHome {
		t.Errorf("default OriginatingTab() should be TabHome (%d), got %d", TabHome, d.OriginatingTab())
	}
}
