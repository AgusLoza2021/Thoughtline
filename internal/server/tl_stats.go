// tl_stats.go — stats MCP tool.
// Stats is a cross-brain aggregate view (queries by project string, not brainID).
// Per-brain breakdown is deferred to a future change; see storage/stats.go.
package server

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// statsArgs is the typed shape of a tl_stats call.
type statsArgs struct {
	// Project: empty → use cfg.DefaultProject; "*" → ignore filter, show all.
	Project string
}

// AllProjectsSentinel is the special Project value that opts out of the
// default-project filter. Documented in the tool description.
const AllProjectsSentinel = "*"

func registerTLStats(srv *server.MCPServer, s *storage.Storage, cfg Config) {
	srv.AddTool(
		mcp.NewTool("tl_stats",
			mcp.WithTitleAnnotation("Memory Database Stats"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(false),
			mcp.WithDescription(tlStatsDescription),
			mcp.WithString("project",
				mcp.Description("Project filter. Default: working directory project. Pass '*' to see counts across ALL projects."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := decodeStatsArgs(req)
			return doStats(ctx, s, cfg, args)
		},
	)
}

const tlStatsDescription = `Return a snapshot of the local Thoughtline database: memory counts (total + by type / project / scope), session counts (open / closed), and the most recent memories. Useful for "how many memories do we have?" or "what types are we tracking most?".

By default the snapshot is scoped to the active project. Pass project='*' to ignore the project filter and see counts across the whole DB.`

func decodeStatsArgs(req mcp.CallToolRequest) statsArgs {
	a := req.GetArguments()
	return statsArgs{
		Project: strings.TrimSpace(asString(a, "project")),
	}
}

func doStats(ctx context.Context, s *storage.Storage, cfg Config, args statsArgs) (*mcp.CallToolResult, error) {
	project := args.Project
	switch {
	case project == AllProjectsSentinel:
		project = "" // wildcard → no filter
	case project == "":
		project = cfg.DefaultProject
	}

	stats, err := s.Stats(ctx, storage.StatsOptions{Project: project})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("stats lookup failed: %v", err)), nil
	}
	return mcp.NewToolResultText(formatStats(project, stats)), nil
}

// formatStats renders the Stats snapshot as a readable text response. Same
// shape the dashboard TUI uses internally so the AI's view of the data
// matches what a human sees on screen.
func formatStats(scope string, stats storage.Stats) string {
	var b strings.Builder

	if scope == "" {
		b.WriteString("Thoughtline stats — across ALL projects\n")
	} else {
		fmt.Fprintf(&b, "Thoughtline stats — project: %s\n", scope)
	}
	fmt.Fprintf(&b, "Generated: %s\n\n",
		stats.GeneratedAt.UTC().Format("2006-01-02 15:04:05 UTC"))

	fmt.Fprintf(&b, "Memories: %d active", stats.TotalMemories)
	if stats.DeletedMemories > 0 {
		fmt.Fprintf(&b, " (%d soft-deleted)", stats.DeletedMemories)
	}
	b.WriteString("\n\n")

	// By type — only emit lines for types that actually have entries.
	if len(stats.ByType) > 0 {
		b.WriteString("By type:\n")
		for _, t := range memory.AllTypes() {
			if n, ok := stats.ByType[t]; ok && n > 0 {
				fmt.Fprintf(&b, "  %-22s %d\n", string(t), n)
			}
		}
		b.WriteString("\n")
	}

	// By project — only meaningful when not already filtered to one.
	if scope == "" && len(stats.ByProject) > 0 {
		b.WriteString("By project:\n")
		// Stable ordering: alphabetical so output is deterministic across runs.
		projects := make([]string, 0, len(stats.ByProject))
		for p := range stats.ByProject {
			projects = append(projects, p)
		}
		sort.Strings(projects)
		for _, p := range projects {
			fmt.Fprintf(&b, "  %-22s %d\n", p, stats.ByProject[p])
		}
		b.WriteString("\n")
	}

	// By scope.
	if len(stats.ByScope) > 0 {
		b.WriteString("By scope:\n")
		for _, sc := range []memory.Scope{memory.ScopeProject, memory.ScopePersonal} {
			if n, ok := stats.ByScope[sc]; ok && n > 0 {
				fmt.Fprintf(&b, "  %-22s %d\n", string(sc), n)
			}
		}
		b.WriteString("\n")
	}

	fmt.Fprintf(&b, "Sessions: %d open, %d closed\n\n",
		stats.OpenSessions, stats.ClosedSessions)

	if len(stats.RecentMemories) > 0 {
		b.WriteString("Recent memories (newest first):\n")
		for _, r := range stats.RecentMemories {
			fmt.Fprintf(&b, "  - [%s] %s", r.Type, r.Title)
			if r.TopicKey != "" {
				fmt.Fprintf(&b, "  (%s)", r.TopicKey)
			}
			b.WriteString("\n")
		}
	}

	return strings.TrimRight(b.String(), "\n")
}
