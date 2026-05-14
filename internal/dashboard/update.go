package dashboard

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// Update is the dashboard message dispatcher. The shape is:
//
//   1. Window resize: distribute width/height to all child components.
//   2. Async results (stats, browse, search, tags, detail, tick): fold into
//      Model state; surface errors via the err field.
//   3. Key events: handled by handleKey, which routes to per-tab logic.
//
// Side effects (re-fetching, ticking, focused list scrolling) come back as
// returned tea.Cmds so Update stays pure and unit-testable.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.applyLayout()
		return m, nil

	case cubeTickMsg:
		// Idle animation. Only Overview shows the cube, but we keep the
		// tick running on every tab so re-entering Overview shows the
		// next frame instead of a frozen one.
		var cmd tea.Cmd
		m.cube, cmd = m.cube.Update(msg)
		return m, cmd

	case tickMsg:
		// Auto-refresh overview stats. The ticker keeps firing; we only
		// re-issue the load when overview is the active tab to avoid
		// spinning the SQLite query when the user is elsewhere.
		var cmds []tea.Cmd
		cmds = append(cmds, tickCmd())
		if m.tab == tabOverview {
			m.loading = true
			cmds = append(cmds, loadStatsCmd(m.storage, m.cfg.Project))
		}
		return m, tea.Batch(cmds...)

	case statsLoadedMsg:
		m.loading = false
		m.loaded = true
		m.err = msg.err
		if msg.err == nil {
			m.stats = msg.stats
			// Keep Sessions tab in sync — Stats already returns RecentSessions.
			items := make([]list.Item, 0, len(m.stats.RecentSessions))
			for _, s := range m.stats.RecentSessions {
				items = append(items, sessionItem{s: s})
			}
			m.sessionList.SetItems(items)
			m.tabsLoaded[tabSessions] = true
		}
		// Start the ticker exactly once, after the first successful load.
		// Same pass also kicks off the cube animation so the test harness
		// (which only drains a single Init cmd) keeps working.
		if !m.tickActive {
			m.tickActive = true
			cmds := []tea.Cmd{tickCmd(), m.cube.Init()}
			if m.cfg.CheckUpdates {
				cmds = append(cmds, checkUpdateCmd(m.cfg.Version))
			}
			if m.splashActive {
				cmds = append(cmds, splashTimer(m.cfg.SplashDuration))
			}
			return m, tea.Batch(cmds...)
		}
		return m, nil

	case updateInfoMsg:
		m.latestVersion = msg.Latest
		m.updateAvailable = msg.Available
		m.updateURL = msg.URL
		return m, nil

	case splashDoneMsg:
		m.splashActive = false
		return m, nil

	case browseLoadedMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.browseItems = msg.items
			items := make([]list.Item, 0, len(msg.items))
			for _, r := range msg.items {
				items = append(items, memoryItem{r: r})
			}
			m.browseList.SetItems(items)
			m.tabsLoaded[tabBrowse] = true
		}
		return m, nil

	case searchLoadedMsg:
		m.searchLoading = false
		m.err = msg.err
		if msg.err == nil {
			m.lastQuery = msg.query
			m.resultItems = msg.items
			items := make([]list.Item, 0, len(msg.items))
			for _, r := range msg.items {
				items = append(items, memoryItem{r: r})
			}
			m.resultList.SetItems(items)
			m.tabsLoaded[legacyTabSearch] = true
		}
		return m, nil

	case tagsLoadedMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			items := make([]list.Item, 0, len(msg.items))
			for _, t := range msg.items {
				items = append(items, tagItem{tc: t})
			}
			m.tagList.SetItems(items)
			m.tabsLoaded[tabTags] = true
		}
		return m, nil

	case detailLoadedMsg:
		m.detailLoading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		env := msg.envelope
		m.detail = &env
		m.detailContent = msg.content
		m.detailVP.SetContent(renderDetailBody(env, msg.content))
		m.detailVP.GotoTop()
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

// handleKey routes key events. Detail overlay is handled first (it shadows
// most tab-level keys); then global keys; then tab-specific keys.
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Detail overlay: arrow keys / pgup / pgdn scroll the viewport. Esc / q
	// closes the overlay and returns to the underlying tab.
	if m.detail != nil {
		switch key {
		case "esc", "q":
			m.detail = nil
			m.detailContent = ""
			return m, nil
		}
		var cmd tea.Cmd
		m.detailVP, cmd = m.detailVP.Update(msg)
		return m, cmd
	}

	// Search input is focused: most keys go to the textinput; enter submits.
	if m.tab == legacyTabSearch && m.searchFocused {
		switch key {
		case "esc":
			m.searchFocused = false
			m.searchInput.Blur()
			return m, nil
		case "enter":
			q := m.searchInput.Value()
			if q == "" {
				return m, nil
			}
			m.searchFocused = false
			m.searchInput.Blur()
			m.searchLoading = true
			return m, runSearchCmd(m.storage, m.cfg.Project, q)
		}
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		return m, cmd
	}

	// Global keys.
	switch key {
	case "ctrl+c":
		m.Quitting = true
		return m, tea.Quit
	case "q":
		m.Quitting = true
		return m, tea.Quit
	case "esc":
		// Esc on any tab quits at the top level when nothing else captured it.
		m.Quitting = true
		return m, tea.Quit
	case "?":
		// Toggle Help: jump to Help tab, or back to Overview if already there.
		if m.tab == legacyTabHelp {
			m.tab = tabOverview
		} else {
			m.tab = legacyTabHelp
		}
		return m, nil
	case "tab":
		m.tab = nextTab(m.tab, +1)
		return m, m.ensureTabLoaded()
	case "shift+tab":
		m.tab = nextTab(m.tab, -1)
		return m, m.ensureTabLoaded()
	case "1":
		m.tab = tabOverview
		return m, m.ensureTabLoaded()
	case "2":
		m.tab = tabBrowse
		return m, m.ensureTabLoaded()
	case "3":
		m.tab = legacyTabSearch
		return m, m.ensureTabLoaded()
	case "4":
		m.tab = tabSessions
		return m, m.ensureTabLoaded()
	case "5":
		m.tab = tabTags
		return m, m.ensureTabLoaded()
	case "6":
		m.tab = legacyTabHelp
		return m, nil
	case "r":
		// Refresh: re-fetch the data the active tab needs.
		return m, m.refreshActive()
	case "t":
		// Cycle through palettes (brand → zbrush → mono → brand). Affects
		// every styled element on next render via ApplyTheme.
		m.theme = nextTheme(m.theme)
		ApplyTheme(m.theme)
		return m, nil
	case "u":
		// Open the latest-release URL in the user's browser when the
		// status bar is showing an update pill. No-op otherwise.
		if m.updateAvailable && m.updateURL != "" {
			return m, openURLCmd(m.updateURL)
		}
		return m, nil
	}

	// Tab-specific routing.
	switch m.tab {
	case tabBrowse:
		return m.handleBrowseKey(msg)
	case legacyTabSearch:
		return m.handleSearchKey(msg)
	case tabSessions:
		var cmd tea.Cmd
		m.sessionList, cmd = m.sessionList.Update(msg)
		return m, cmd
	case tabTags:
		var cmd tea.Cmd
		m.tagList, cmd = m.tagList.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) handleBrowseKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if it, ok := m.browseList.SelectedItem().(memoryItem); ok {
			m.detailLoading = true
			return m, loadDetailCmd(m.storage, it.r.ID)
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.browseList, cmd = m.browseList.Update(msg)
	return m, cmd
}

func (m Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "/":
		// Focus the textinput.
		m.searchFocused = true
		m.searchInput.Focus()
		return m, nil
	case "enter":
		if it, ok := m.resultList.SelectedItem().(memoryItem); ok {
			m.detailLoading = true
			return m, loadDetailCmd(m.storage, it.r.ID)
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.resultList, cmd = m.resultList.Update(msg)
	return m, cmd
}

// ensureTabLoaded triggers a lazy load the first time a non-overview tab is
// activated, so navigation feels instant after the initial fetch.
func (m *Model) ensureTabLoaded() tea.Cmd {
	if m.tabsLoaded[m.tab] {
		return nil
	}
	switch m.tab {
	case tabBrowse:
		m.loading = true
		return loadBrowseCmd(m.storage, m.cfg.Project)
	case tabTags:
		m.loading = true
		return loadTagsCmd(m.storage, m.cfg.Project)
	case legacyTabSearch:
		// Search is opt-in — only fires on the user's first query.
		m.tabsLoaded[legacyTabSearch] = true
	}
	return nil
}

// refreshActive re-fetches whatever the current tab is showing.
func (m *Model) refreshActive() tea.Cmd {
	switch m.tab {
	case tabOverview, tabSessions:
		m.loading = true
		return loadStatsCmd(m.storage, m.cfg.Project)
	case tabBrowse:
		m.loading = true
		return loadBrowseCmd(m.storage, m.cfg.Project)
	case tabTags:
		m.loading = true
		return loadTagsCmd(m.storage, m.cfg.Project)
	case legacyTabSearch:
		if m.lastQuery != "" {
			m.searchLoading = true
			return runSearchCmd(m.storage, m.cfg.Project, m.lastQuery)
		}
	}
	return nil
}

func nextTab(t legacyTabKey, delta int) legacyTabKey {
	idx := int(t) + delta
	n := len(allTabs)
	idx = ((idx % n) + n) % n
	return allTabs[idx]
}
