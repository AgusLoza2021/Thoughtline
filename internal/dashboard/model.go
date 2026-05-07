package dashboard

import (
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// Config carries the wiring the dashboard needs from main(). DBPath and
// Version come from the binary's runtime; Project is the auto-detected
// default ("" means "show all projects").
type Config struct {
	Version string
	DBPath  string
	Project string

	// ThemeName is preserved on Config for backwards compatibility with
	// the existing CLI flag --theme. The new screen-stack dashboard
	// ignores it and renders a single palette; the old tab-based view
	// is still wired through this field until the migration completes.
	ThemeName string

	// Splash + SplashDuration drive the v1 intro animation that lives in
	// the old tab-based view. The new flat dashboard skips the splash.
	// These fields stay so --no-splash and --splash-ms keep parsing while
	// the migration is in flight.
	Splash         bool
	SplashDuration time.Duration

	// CheckUpdates toggles the GitHub release lookup. Default true in
	// the binary, false in tests so we don't hit the network.
	CheckUpdates bool
}

// tabKey enumerates the dashboard tabs. Order in this enum is the order in
// which tabs render in the header bar.
type tabKey int

const (
	tabOverview tabKey = iota
	tabBrowse
	tabSearch
	tabSessions
	tabTags
	tabHelp
)

func (t tabKey) String() string {
	switch t {
	case tabOverview:
		return "Overview"
	case tabBrowse:
		return "Browse"
	case tabSearch:
		return "Search"
	case tabSessions:
		return "Sessions"
	case tabTags:
		return "Tags"
	case tabHelp:
		return "Help"
	}
	return "?"
}

var allTabs = []tabKey{tabOverview, tabBrowse, tabSearch, tabSessions, tabTags, tabHelp}

// Model is the Bubbletea Model for the dashboard. It owns the per-tab
// bubbles components (lists, textinput, viewport) and a coarse "loaded"
// map so each non-overview tab fetches its data lazily on first visit.
//
// The fields stats, loaded, loading, err, width, height, Quitting are kept
// at the top level (not inside a sub-struct) so the v1 dashboard tests in
// model_test.go that read them directly still compile unchanged.
type Model struct {
	storage *storage.Storage
	cfg     Config

	// Overview state — kept verbatim from v1 so existing tests keep passing.
	stats   storage.Stats
	loaded  bool
	loading bool
	err     error

	// Layout
	width    int
	height   int
	Quitting bool

	// New: tab navigation + lazy load tracking.
	tab        tabKey
	tabsLoaded map[tabKey]bool
	tickActive bool

	// Browse tab.
	browseItems []storage.SearchResult
	browseList  list.Model

	// Search tab.
	searchInput   textinput.Model
	searchFocused bool
	lastQuery     string
	searchLoading bool
	resultItems   []storage.SearchResult
	resultList    list.Model

	// Sessions tab.
	sessionList list.Model

	// Tags tab.
	tagList list.Model

	// Detail overlay (active in Browse / Search when the user hits enter).
	detail        *storage.SearchResult
	detailContent string
	detailVP      viewport.Model
	detailLoading bool

	// Screen stack — new flat-screen navigation (I1+).
	// The root is always DashboardScreen; child screens are pushed on top.
	screens []Screen
	pal     palette

	// Animated header cube (Overview tab only).
	cube CubeModel

	// Update check (filled by checkUpdateCmd, dispatched after first
	// stats load). Both empty == "no info yet"; updateAvailable == false
	// means we checked and we are up to date.
	latestVersion   string
	updateAvailable bool
	updateURL       string

	// Splash overlay. When splashActive is true, View() renders the
	// intro animation instead of the dashboard. A splashDoneMsg flips it
	// off after Config.SplashDuration.
	splashActive bool

	// Theme cycling (driven by [t] hotkey).
	theme Theme
}

// stackModel is a minimal struct that owns only the screen stack. It is used
// by unit tests (C3) that need Push/Pop/Peek without spinning up a full Model.
type stackModel struct {
	stack []Screen
}

// newStackModel returns an empty stackModel for testing.
func newStackModel() *stackModel { return &stackModel{} }

// Push adds s to the top of the stack and calls s.Init().
func (m *stackModel) Push(s Screen) {
	m.stack = append(m.stack, s)
}

// Pop removes the top screen. It is a no-op when the stack has ≤1 item
// (the dashboard root is always present).
func (m *stackModel) Pop() {
	if len(m.stack) <= 1 {
		return
	}
	m.stack = m.stack[:len(m.stack)-1]
}

// Peek returns the top screen without removing it. Returns nil on empty stack.
func (m *stackModel) Peek() Screen {
	if len(m.stack) == 0 {
		return nil
	}
	return m.stack[len(m.stack)-1]
}

// New constructs a dashboard Model. The Init() command triggers the first
// stats load. All bubbles widgets are created in a default-sized state and
// resized to fit on the first WindowSizeMsg.
func New(st *storage.Storage, cfg Config) Model {
	// Apply the default theme so all styles render consistently.
	ApplyTheme(ThemeBrand)

	delegate := list.NewDefaultDelegate()
	delegate.SetSpacing(0)

	mk := func(title string) list.Model {
		l := list.New(nil, delegate, 0, 0)
		l.Title = title
		l.SetShowHelp(false)
		l.SetShowStatusBar(true)
		l.SetFilteringEnabled(true)
		l.DisableQuitKeybindings()
		return l
	}

	browseList := mk("All memories")
	resultList := mk("Search results")
	sessList := mk("Sessions")
	tagList := mk("Top tags")

	ti := textinput.New()
	ti.Placeholder = "search memories — query, or topic_key/* GLOB"
	ti.CharLimit = 200
	ti.Prompt = "› "

	vp := viewport.New(0, 0)

	// Root screen: the new flat dashboard. The old tab-based view continues
	// to render via View()/Update() until I3 removes it; the screen stack
	// drives the new navigation path in parallel.
	rootScreen := NewDashboardScreen(st, cfg.Project, cfg.Version)

	return Model{
		storage:     st,
		cfg:         cfg,
		screens:     []Screen{rootScreen},
		pal:         defaultPalette,
		tab:         tabOverview,
		tabsLoaded:  make(map[tabKey]bool),
		browseList:  browseList,
		resultList:  resultList,
		sessionList: sessList,
		tagList:     tagList,
		searchInput: ti,
		detailVP:    vp,
		cube: NewCube(CubeConfig{
			Enabled:    true,
			Mode:       CubePlain,
			FrameDelay: 220 * time.Millisecond,
		}),
		theme:        ThemeBrand,
		splashActive: cfg.Splash,
	}
}

// Init kicks off the first stats fetch. The ticker starts when the first
// statsLoadedMsg arrives so the test harness in model_test.go (which calls
// cmd() once and feeds the message into Update) keeps working without
// having to handle tea.BatchMsg.
func (m Model) Init() tea.Cmd {
	// Init returns a single cmd so the model_test drainInit harness keeps
	// working. Cube, ticker, update check and (optional) splash are all
	// scheduled lazily from later messages.
	return loadStatsCmd(m.storage, m.cfg.Project)
}
