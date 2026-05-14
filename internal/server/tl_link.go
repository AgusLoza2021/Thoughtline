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

type linkArgs struct {
	FromID   int64
	ToID     int64
	Relation string
	Note     string
	Project  string
}

var validLinkRelations = map[string]bool{
	"supersedes":   true,
	"contradicts":  true,
	"refines":      true,
	"depends_on":   true,
	"references":   true,
	"related":      true,
	"derived_from": true,
}

func registerTLLink(srv *server.MCPServer, s *storage.Storage, cfg Config) {
	srv.AddTool(
		mcp.NewTool("tl_link",
			mcp.WithTitleAnnotation("Link Memories"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(false),
			mcp.WithDescription(tlLinkDescription),
			mcp.WithNumber("from_id",
				mcp.Required(),
				mcp.Description("ID of the source memory."),
			),
			mcp.WithNumber("to_id",
				mcp.Required(),
				mcp.Description("ID of the target memory."),
			),
			mcp.WithString("relation",
				mcp.Required(),
				mcp.Description("Relationship type: supersedes | contradicts | refines | depends_on | references | related | derived_from."),
			),
			mcp.WithString("note",
				mcp.Description("Optional free-text annotation about why this link exists."),
			),
			mcp.WithString("project",
				mcp.Description("Project identifier. Defaults to the working directory basename if omitted."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := decodeLinkArgs(req)
			return doLink(ctx, s, cfg, args)
		},
	)
}

const tlLinkDescription = `Create a directed link between two memories in the graph.

Relations: supersedes, contradicts, refines, depends_on, references, related, derived_from.

Re-linking two memories with the same relation is a no-op (returns "link already exists"). Different relations on the same pair are independent links. Use tl_related to inspect all links for a memory.`

func decodeLinkArgs(req mcp.CallToolRequest) linkArgs {
	a := req.GetArguments()
	return linkArgs{
		FromID:   asInt64(a, "from_id"),
		ToID:     asInt64(a, "to_id"),
		Relation: strings.TrimSpace(asString(a, "relation")),
		Note:     strings.TrimSpace(asString(a, "note")),
		Project:  strings.TrimSpace(asString(a, "project")),
	}
}

func doLink(ctx context.Context, s *storage.Storage, cfg Config, args linkArgs) (*mcp.CallToolResult, error) {
	if args.FromID <= 0 {
		return mcp.NewToolResultError("'from_id' is required and must be a positive integer"), nil
	}
	if args.ToID <= 0 {
		return mcp.NewToolResultError("'to_id' is required and must be a positive integer"), nil
	}
	if args.FromID == args.ToID {
		return mcp.NewToolResultError("'from_id' and 'to_id' must be different — a memory cannot link to itself"), nil
	}
	if !validLinkRelations[args.Relation] {
		return mcp.NewToolResultError("'relation' must be one of: supersedes, contradicts, refines, depends_on, references, related, derived_from"), nil
	}

	project := args.Project
	if project == "" {
		project = cfg.DefaultProject
	}

	brainID, err := s.ResolveBrainID(ctx, project)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return mcp.NewToolResultError(fmt.Sprintf("no brain found for project %q", project)), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("resolve brain: %v", err)), nil
	}

	from, err := s.GetByIDUnscoped(ctx, args.FromID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("no memory found with from_id=%d", args.FromID)), nil
	}
	if from.BrainID != brainID {
		return mcp.NewToolResultError(fmt.Sprintf("memory id=%d does not belong to project %q", args.FromID, project)), nil
	}
	if from.DeletedAt != nil {
		return mcp.NewToolResultError(fmt.Sprintf("no memory found with from_id=%d", args.FromID)), nil
	}

	to, err := s.GetByIDUnscoped(ctx, args.ToID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("no memory found with to_id=%d", args.ToID)), nil
	}
	if to.BrainID != brainID {
		return mcp.NewToolResultError(fmt.Sprintf("memory id=%d does not belong to project %q", args.ToID, project)), nil
	}
	if to.DeletedAt != nil {
		return mcp.NewToolResultError(fmt.Sprintf("no memory found with to_id=%d", args.ToID)), nil
	}

	link, noop, err := s.CreateLink(ctx, brainID, args.FromID, args.ToID, args.Relation, args.Note)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("create link: %v", err)), nil
	}
	if noop {
		return mcp.NewToolResultText(fmt.Sprintf("link already exists: %q --[%s]--> %q", from.Title, args.Relation, to.Title)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf(
		"Linked: %q --[%s]--> %q (link_id: %d)",
		from.Title, args.Relation, to.Title, link.ID,
	)), nil
}
