package dashboard

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// refreshInterval is the cadence of the auto-refresh ticker on Overview.
// Other tabs are loaded lazily on first visit and re-fetched only on demand
// (via [r] or via re-entering the tab after [R]).
const refreshInterval = 10 * time.Second

// tickMsg is delivered every refreshInterval. The Update loop uses it to
// re-fetch overview stats so the dashboard doesn't go stale during long
// sessions, while keeping the rest of the model untouched.
type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(refreshInterval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// statsLoadedMsg carries the result of a Stats() call.
type statsLoadedMsg struct {
	stats storage.Stats
	err   error
}

func loadStatsCmd(st *storage.Storage, project string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		stats, err := st.Stats(ctx, storage.StatsOptions{Project: project, RecentLimit: 50})
		return statsLoadedMsg{stats: stats, err: err}
	}
}

type browseLoadedMsg struct {
	items []storage.SearchResult
	err   error
}

func loadBrowseCmd(st *storage.Storage, project string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		items, err := st.RecentAll(ctx, project, 50)
		return browseLoadedMsg{items: items, err: err}
	}
}

type searchLoadedMsg struct {
	query string
	items []storage.SearchResult
	err   error
}

func runSearchCmd(st *storage.Storage, project, query string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		items, err := st.Search(ctx, query, storage.SearchOptions{Project: project, Limit: 50})
		return searchLoadedMsg{query: query, items: items, err: err}
	}
}

type tagsLoadedMsg struct {
	items []storage.TagCount
	err   error
}

func loadTagsCmd(st *storage.Storage, project string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		items, err := st.TopTags(ctx, project, 30)
		return tagsLoadedMsg{items: items, err: err}
	}
}

type detailLoadedMsg struct {
	id      int64
	envelope storage.SearchResult
	content string
	err     error
}

func loadDetailCmd(st *storage.Storage, id int64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		m, err := st.GetByID(ctx, id)
		if err != nil {
			return detailLoadedMsg{id: id, err: err}
		}
		env := storage.SearchResult{
			ID:            m.ID,
			SyncID:        m.SyncID,
			Project:       m.Project,
			Scope:         m.Scope,
			Type:          m.Type,
			TopicKey:      m.TopicKey,
			Title:         m.Title,
			Tags:          m.Tags,
			RevisionCount: m.RevisionCount,
			UpdatedAt:     m.UpdatedAt,
		}
		return detailLoadedMsg{id: id, envelope: env, content: m.Content}
	}
}
