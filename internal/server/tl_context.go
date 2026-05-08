package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// contextArgs is the typed shape of a tl_context call.
type contextArgs struct {
	Project string
	Limit   int
}

func registerTLContext(srv *server.MCPServer, s *storage.Storage, cfg Config) {
	srv.AddTool(
		mcp.NewTool("tl_context",
			mcp.WithTitleAnnotation("Recent Memory Context"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(false),
			mcp.WithDescription(tlContextDescription),
			mcp.WithString("project",
				mcp.Description("Project identifier. Defaults to the working directory basename if omitted."),
			),
			mcp.WithNumber("limit",
				mcp.Description("Max recent memories (default 10, hard cap 50)."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := decodeContextArgs(req)
			return doContext(ctx, s, cfg, args)
		},
	)
}

const tlContextDescription = `Return the most recently updated memories for the active project, ordered by updated_at DESC. Soft-deleted memories are excluded.

Use this PROACTIVELY at the start of a session, or after a context compaction, to recover what was being worked on. Returns the same per-result envelope as tl_search (Title / ID / Topic / Snippet / ...) so it can be parsed identically.`

func decodeContextArgs(req mcp.CallToolRequest) contextArgs {
	a := req.GetArguments()
	out := contextArgs{
		Project: asString(a, "project"),
		Limit:   asInt(a, "limit"),
	}
	out.Project = strings.TrimSpace(out.Project)
	return out
}

func doContext(ctx context.Context, s *storage.Storage, cfg Config, args contextArgs) (*mcp.CallToolResult, error) {
	project := args.Project
	if project == "" {
		project = cfg.DefaultProject
	}
	if project == "" {
		return mcp.NewToolResultError("'project' is required (and could not be auto-detected). Pass it explicitly or restart the server in the project's working directory."), nil
	}

	// Read path: resolve brain, return empty on not-found (never auto-create on reads).
	brainID, err := s.ResolveBrainID(ctx, project)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Brain doesn't exist yet — no memories possible, return empty.
			return mcp.NewToolResultText(formatContextResults(project, nil)), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("resolve brain: %v", err)), nil
	}

	results, err := s.Recent(ctx, brainID, args.Limit)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("context lookup failed: %v", err)), nil
	}

	return mcp.NewToolResultText(formatContextResults(project, results)), nil
}

func formatContextResults(project string, results []storage.SearchResult) string {
	if len(results) == 0 {
		return fmt.Sprintf("No recent memories for project %q.", project)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Recent %d memorie(s) for project %q (newest first):\n\n", len(results), project)

	for i, r := range results {
		if i > 0 {
			b.WriteString("---\n")
		}
		writeResultBlock(&b, r, false) // false: no BM25 score, recency only
	}

	b.WriteString("\n---\n")
	b.WriteString("Snippets above are content previews (≤300 chars). Call tl_get_observation(id: <ID>) to read the full untruncated content of a specific entry.")
	return strings.TrimRight(b.String(), "\n")
}
