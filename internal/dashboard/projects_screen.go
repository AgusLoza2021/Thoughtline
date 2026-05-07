package dashboard

import (
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// BrowseProjectsScreen shows all distinct projects, sorted alphabetically.
// Pressing enter drills into a project-scoped RecentScreen.
type BrowseProjectsScreen struct {
	storage  *storage.Storage
	projects []string // sorted alphabetically
	cursor   int
}

func newBrowseProjectsScreenFull(st *storage.Storage, stats storage.Stats) *BrowseProjectsScreen {
	projects := make([]string, 0, len(stats.ByProject))
	for p := range stats.ByProject {
		projects = append(projects, p)
	}
	sort.Strings(projects)
	return &BrowseProjectsScreen{storage: st, projects: projects}
}

func (b *BrowseProjectsScreen) Title() string { return "Browse Projects" }

func (b *BrowseProjectsScreen) Init() tea.Cmd { return nil }

func (b *BrowseProjectsScreen) OnFocus() tea.Cmd { return nil }

func (b *BrowseProjectsScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	if kMsg, ok := msg.(tea.KeyMsg); ok {
		switch kMsg.Type {
		case tea.KeyEsc:
			return b, func() tea.Msg { return popScreenCmd{} }
		case tea.KeyEnter:
			if len(b.projects) > 0 && b.cursor < len(b.projects) {
				proj := b.projects[b.cursor]
				st := b.storage
				return b, func() tea.Msg {
					return pushScreenCmd{screen: newRecentScreen(st, proj, proj)}
				}
			}
		}
		switch kMsg.String() {
		case "q":
			return b, func() tea.Msg { return popScreenCmd{} }
		case "j", "down":
			if b.cursor < len(b.projects)-1 {
				b.cursor++
			}
		case "k", "up":
			if b.cursor > 0 {
				b.cursor--
			}
		}
	}
	return b, nil
}

func (b *BrowseProjectsScreen) View(width, height int, p palette) string {
	var sb strings.Builder
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(p.Foreground)
	sb.WriteString(titleStyle.Render("Browse Projects"))
	sb.WriteString("\n\n")

	if len(b.projects) == 0 {
		sb.WriteString(lipgloss.NewStyle().Foreground(p.Muted).Render("(no projects yet)"))
		return sb.String()
	}
	for i, proj := range b.projects {
		prefix := "  "
		if i == b.cursor {
			prefix = lipgloss.NewStyle().Foreground(p.Cursor).Render("▸") + " "
		}
		sb.WriteString(prefix + proj + "\n")
	}
	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(p.Muted).Render("esc/q back • enter drill in"))
	return sb.String()
}
