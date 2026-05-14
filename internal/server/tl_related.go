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

type relatedArgs struct {
	ID        int64
	Direction string
	Project   string
}

func registerTLRelated(srv *server.MCPServer, s *storage.Storage, cfg Config) {
	srv.AddTool(
		mcp.NewTool("tl_related",
			mcp.WithTitleAnnotation("Get Related Memories"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(false),
			mcp.WithDescription(tlRelatedDescription),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("ID of the memory to get links for."),
			),
			mcp.WithString("direction",
				mcp.Description("Filter: 'from' (links originating here), 'to' (links pointing here), or omit for both."),
			),
			mcp.WithString("project",
				mcp.Description("Project identifier. Defaults to the working directory basename if omitted."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := decodeRelatedArgs(req)
			return doRelated(ctx, s, cfg, args)
		},
	)
}

const tlRelatedDescription = `Fetch all memory graph links for a given memory (both directions by default).

Returns each link with: direction indicator (→ or ←), relation type, linked memory title/id/type, and optional note. Use 'direction' to filter to outgoing ('from') or incoming ('to') links only.`

func decodeRelatedArgs(req mcp.CallToolRequest) relatedArgs {
	a := req.GetArguments()
	return relatedArgs{
		ID:        asInt64(a, "id"),
		Direction: strings.TrimSpace(asString(a, "direction")),
		Project:   strings.TrimSpace(asString(a, "project")),
	}
}

func doRelated(ctx context.Context, s *storage.Storage, cfg Config, args relatedArgs) (*mcp.CallToolResult, error) {
	if args.ID <= 0 {
		return mcp.NewToolResultError("'id' is required and must be a positive integer"), nil
	}
	if args.Direction != "" && args.Direction != "from" && args.Direction != "to" {
		return mcp.NewToolResultError("'direction' must be 'from', 'to', or omitted"), nil
	}

	project := args.Project
	if project == "" {
		project = cfg.DefaultProject
	}

	brainID, err := s.ResolveBrainID(ctx, project)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return mcp.NewToolResultText(fmt.Sprintf("No links found for memory id=%d.", args.ID)), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("resolve brain: %v", err)), nil
	}

	links, err := s.GetLinks(ctx, brainID, args.ID, args.Direction)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("get links: %v", err)), nil
	}

	if len(links) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No links found for memory id=%d.", args.ID)), nil
	}

	// Fetch titles for all linked memories.
	type linkRow struct {
		link      storage.MemoryLink
		otherID   int64
		direction string // "→" or "←"
	}
	rows := make([]linkRow, 0, len(links))
	for _, l := range links {
		r := linkRow{link: l}
		if l.FromID == args.ID {
			r.otherID = l.ToID
			r.direction = "→"
		} else {
			r.otherID = l.FromID
			r.direction = "←"
		}
		rows = append(rows, r)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Links for memory id=%d (%d found):\n\n", args.ID, len(links))

	for i, r := range rows {
		if i > 0 {
			b.WriteString("---\n")
		}
		other, err := s.GetByIDUnscoped(ctx, r.otherID)
		otherTitle := fmt.Sprintf("id=%d", r.otherID)
		otherType := "unknown"
		if err == nil {
			otherTitle = other.Title
			otherType = string(other.Type)
		}

		fmt.Fprintf(&b, "Direction: %s\n", r.direction)
		fmt.Fprintf(&b, "Relation: %s\n", r.link.Relation)
		fmt.Fprintf(&b, "Other ID: %d\n", r.otherID)
		fmt.Fprintf(&b, "Other Title: %s\n", otherTitle)
		fmt.Fprintf(&b, "Other Type: %s\n", otherType)
		if r.link.Note != "" {
			fmt.Fprintf(&b, "Note: %s\n", r.link.Note)
		}
		fmt.Fprintf(&b, "Link ID: %d\n", r.link.ID)
	}

	return mcp.NewToolResultText(strings.TrimRight(b.String(), "\n")), nil
}
