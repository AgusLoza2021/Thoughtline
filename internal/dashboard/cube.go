package dashboard

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// CubeMode picks the glyph set used by the cube. Plain is pure ASCII
// (`+ - | / \`) — safe in Windows CMD, conhost, and other legacy terminals.
// Fancy uses box-drawing characters for sharper edges; requires a font
// that supports them, which is most modern terminals.
type CubeMode int

const (
	CubePlain CubeMode = iota
	CubeFancy
)

// CubeConfig is the set of knobs an embedder can tune. Zero value =
// disabled cube, no animation, no draw cost.
type CubeConfig struct {
	Enabled    bool
	Mode       CubeMode
	FrameDelay time.Duration // default 200ms when zero
}

// cubeTickMsg is the per-frame advance signal. It is dispatched on the
// configured FrameDelay cadence and consumed by CubeModel.Update.
type cubeTickMsg time.Time

// CubeModel is a Bubbletea sub-model that renders an animated 3D
// wireframe cube. It oscillates through the frames forward and back to
// give the impression of gentle rotation between -15° and +15°.
//
// The component is self-contained: it owns its own tick command, its
// own frame buffer, and its own forward/backward direction. Embed it as
// a field on the parent Model and forward cubeTickMsg in your Update.
type CubeModel struct {
	cfg       CubeConfig
	frames    []string
	idx       int
	direction int // +1 forward, -1 backward
}

// NewCube builds a CubeModel from cfg. If cfg.FrameDelay is zero we
// fall back to 200ms which feels meditative without being distracting.
func NewCube(cfg CubeConfig) CubeModel {
	if cfg.FrameDelay <= 0 {
		cfg.FrameDelay = 200 * time.Millisecond
	}
	return CubeModel{
		cfg:       cfg,
		frames:    cubeFrames(cfg.Mode),
		idx:       2, // start centered so the first paint is calm
		direction: +1,
	}
}

// Init returns the first tick command. If the cube is disabled, no
// command is returned and no work is scheduled.
func (m CubeModel) Init() tea.Cmd {
	if !m.cfg.Enabled {
		return nil
	}
	return cubeTick(m.cfg.FrameDelay)
}

func cubeTick(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg { return cubeTickMsg(t) })
}

// Update advances the frame index. The cube oscillates: it walks
// forward to the last frame, flips direction, walks back to frame 0,
// flips direction, and repeats. That mimics a gentle back-and-forth
// rotation rather than a one-way spin.
func (m CubeModel) Update(msg tea.Msg) (CubeModel, tea.Cmd) {
	switch msg.(type) {
	case cubeTickMsg:
		if !m.cfg.Enabled || len(m.frames) == 0 {
			return m, nil
		}
		next := m.idx + m.direction
		if next >= len(m.frames) {
			m.direction = -1
			next = m.idx - 1
		} else if next < 0 {
			m.direction = +1
			next = m.idx + 1
		}
		m.idx = next
		return m, cubeTick(m.cfg.FrameDelay)
	}
	return m, nil
}

// View returns the current frame, optionally colorized with the brand
// palette so the cube reads as part of the dashboard's identity rather
// than a foreign sprite.
func (m CubeModel) View() string {
	if !m.cfg.Enabled || len(m.frames) == 0 {
		return ""
	}
	return cubeStyle.Render(m.frames[m.idx])
}

// cubeFrames returns the oscillation frames for the requested mode.
// Frames are kept identical in width and height (18 cols × 9 rows) so
// surrounding UI does not shift when the index advances.
func cubeFrames(mode CubeMode) []string {
	if mode == CubeFancy {
		return cubeFramesFancy()
	}
	return cubeFramesPlain()
}

// cubeFramesPlain — pure ASCII, safe in Windows CMD.
//
// Order: hard left → soft left → center → soft right → hard right.
// The Update oscillation walks the index back down after reaching the
// end, so the perceived loop is 0,1,2,3,4,3,2,1,0,1,2,3,4,…
func cubeFramesPlain() []string {
	return []string{
		// 0 — hard left lean
		joinLines(
			"        +------+  ",
			"       /      /|  ",
			"      /      / |  ",
			"     +------+  |  ",
			"    /|      |  |  ",
			"   / |      |  +  ",
			"  +  |      | /   ",
			"   \\ |      |/    ",
			"    \\+------+     ",
		),
		// 1 — soft left
		joinLines(
			"       +------+   ",
			"      /      /|   ",
			"     /      / |   ",
			"    +------+  |   ",
			"   /|      |  |   ",
			"  / |      |  +   ",
			" +  |      | /    ",
			"  \\ |      |/     ",
			"   \\+------+      ",
		),
		// 2 — center (no lean)
		joinLines(
			"     +------+     ",
			"    /      /|     ",
			"   /      / |     ",
			"  +------+  |     ",
			"  |      |  |     ",
			"  |      |  +     ",
			"  |      | /      ",
			"  |      |/       ",
			"  +------+        ",
		),
		// 3 — soft right
		joinLines(
			"    +------+      ",
			"    |\\      \\     ",
			"    | \\      \\    ",
			"    |  +------+   ",
			"    |  |      |\\  ",
			"    +  |      | \\ ",
			"     \\ |      |  +",
			"      \\|      | / ",
			"       +------+   ",
		),
		// 4 — hard right lean
		joinLines(
			"   +------+       ",
			"   |\\      \\      ",
			"   | \\      \\     ",
			"   |  +------+    ",
			"   |  |      |\\   ",
			"   +  |      | \\  ",
			"    \\ |      |  + ",
			"     \\|      | /  ",
			"      +------+    ",
		),
	}
}

// cubeFramesFancy — same shapes with box-drawing glyphs. Looks sharper
// on modern terminals but requires a Unicode-capable font. Falls back
// to plain frames for the diagonals (`╱`, `╲`) for which there is no
// box-drawing equivalent that lines up cleanly at this scale.
func cubeFramesFancy() []string {
	repl := func(s string) string {
		s = strings.ReplaceAll(s, "+", "■")
		s = strings.ReplaceAll(s, "-", "─")
		s = strings.ReplaceAll(s, "|", "│")
		return s
	}
	plain := cubeFramesPlain()
	out := make([]string, len(plain))
	for i, f := range plain {
		out[i] = repl(f)
	}
	return out
}

func joinLines(lines ...string) string {
	return strings.Join(lines, "\n")
}
