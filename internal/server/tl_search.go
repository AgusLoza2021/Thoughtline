package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// searchArgs is the typed shape of a tl_search call.
type searchArgs struct {
	Query    string
	Type     string
	Scope    string
	Project  string
	TopicKey string
	Limit    int
	Offset   int
}

func registerTLSearch(srv *server.MCPServer, s *storage.Storage, cfg Config) {
	srv.AddTool(
		mcp.NewTool("tl_search",
			mcp.WithTitleAnnotation("Search Memory"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(false),
			mcp.WithDescription(tlSearchDescription),
			mcp.WithString("query",
				mcp.Required(),
				mcp.Description("Keyword query. Each whitespace-delimited token is matched as a literal phrase (implicit AND). Queries containing '/' are first matched against topic_key as a GLOB pattern; if any rows match, FTS does not run."),
			),
			mcp.WithString("type",
				mcp.Description("Optional filter — one of: game-design-decision, scene-pattern, asset-reference, perf-gotcha, pipeline-step, script-pattern, bugfix, convention, preference."),
			),
			mcp.WithString("scope",
				mcp.Description("Optional filter — 'project' or 'personal'."),
			),
			mcp.WithString("project",
				mcp.Description("Project identifier. Defaults to the working directory basename if omitted."),
			),
			mcp.WithString("topic_key",
				mcp.Description("Optional GLOB filter on topic_key (e.g. 'design/auth/*')."),
			),
			mcp.WithNumber("limit",
				mcp.Description("Max results (default 10, hard cap 50)."),
			),
			mcp.WithNumber("offset",
				mcp.Description("Skip the first N matches; pair with limit for pagination."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := decodeSearchArgs(req)
			return doSearch(ctx, s, cfg, args)
		},
	)
}

const tlSearchDescription = `Search the local Thoughtline store. Returns up to 'limit' matches (default 10) ranked by SQLite FTS5 BM25, lowest score first.

Use this PROACTIVELY when starting work on something that might already be documented — avoid re-deriving knowledge that's already in memory. Snippets are previews (≤300 chars); use tl_get_observation(id) to read the full content of a specific match.

Topic-key shortcut: a query containing '/' is treated as a topic_key GLOB lookup first. If any rows match, FTS does not run. This makes 'design/auth/*' or an exact 'scene/playcanvas/inn-cellar' lookup O(1).`

func decodeSearchArgs(req mcp.CallToolRequest) searchArgs {
	a := req.GetArguments()
	out := searchArgs{
		Query:    asString(a, "query"),
		Type:     asString(a, "type"),
		Scope:    asString(a, "scope"),
		Project:  asString(a, "project"),
		TopicKey: asString(a, "topic_key"),
		Limit:    asInt(a, "limit"),
		Offset:   asInt(a, "offset"),
	}
	out.Query = strings.TrimSpace(out.Query)
	out.Type = strings.TrimSpace(out.Type)
	out.Scope = strings.TrimSpace(out.Scope)
	out.Project = strings.TrimSpace(out.Project)
	out.TopicKey = strings.TrimSpace(out.TopicKey)
	return out
}

func doSearch(ctx context.Context, s *storage.Storage, cfg Config, args searchArgs) (*mcp.CallToolResult, error) {
	args.Query = strings.TrimSpace(args.Query)
	if args.Query == "" {
		return mcp.NewToolResultError("'query' is required and must contain non-whitespace characters"), nil
	}

	project := args.Project
	if project == "" {
		project = cfg.DefaultProject
	}

	opts := storage.SearchOptions{
		Project:  project,
		Scope:    args.Scope,
		Type:     args.Type,
		TopicKey: args.TopicKey,
		Limit:    args.Limit,
		Offset:   args.Offset,
	}

	results, err := s.Search(ctx, args.Query, opts)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
	}

	return mcp.NewToolResultText(formatSearchResults(args.Query, results)), nil
}

// formatSearchResults builds the natural-language response. The structure is
// designed so an AI can re-parse fields (ID, Topic, Score) without regex
// gymnastics, while still being readable to a human glancing at the output.
func formatSearchResults(query string, results []storage.SearchResult) string {
	if len(results) == 0 {
		return fmt.Sprintf("No matches for %q.", query)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Found %d result(s) for %q:\n\n", len(results), query)

	for i, r := range results {
		if i > 0 {
			b.WriteString("---\n")
		}
		writeResultBlock(&b, r, true)
	}

	b.WriteString("\n---\n")
	b.WriteString("Snippets above are previews (≤300 chars, may include '…' ellipses from FTS5). Call tl_get_observation(id: <ID>) to read the full untruncated content of a specific match.")
	return strings.TrimRight(b.String(), "\n")
}

// writeResultBlock renders a single SearchResult as a labelled block. Shared
// by tl_search and tl_context so both tools produce identically-shaped
// per-result output and the AI can parse them with the same logic. The
// includeScore flag controls whether the BM25 line is emitted — score is
// meaningful for tl_search (FTS rank) but not for tl_context (recency).
func writeResultBlock(b *strings.Builder, r storage.SearchResult, includeScore bool) {
	fmt.Fprintf(b, "Title: %s\n", r.Title)
	fmt.Fprintf(b, "ID: %d\n", r.ID)
	fmt.Fprintf(b, "Sync ID: %s\n", r.SyncID)
	fmt.Fprintf(b, "Project: %s\n", r.Project)
	fmt.Fprintf(b, "Type: %s\n", r.Type)
	fmt.Fprintf(b, "Scope: %s\n", r.Scope)
	if r.TopicKey != "" {
		fmt.Fprintf(b, "Topic: %s\n", r.TopicKey)
	}
	fmt.Fprintf(b, "Revision: %d\n", r.RevisionCount)
	if includeScore {
		fmt.Fprintf(b, "Score: %.4f\n", r.Score)
	}
	if !r.UpdatedAt.IsZero() {
		fmt.Fprintf(b, "Updated: %s\n", r.UpdatedAt.UTC().Format("2006-01-02 15:04:05 UTC"))
	}
	if len(r.Tags) > 0 {
		fmt.Fprintf(b, "Tags: %s\n", strings.Join(r.Tags, ", "))
	}
	fmt.Fprintf(b, "Snippet: %s\n", r.Snippet)
}

func asInt(a map[string]any, k string) int {
	switch v := a[k].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	default:
		return 0
	}
}

func asInt64(a map[string]any, k string) int64 {
	switch v := a[k].(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	default:
		return 0
	}
}
