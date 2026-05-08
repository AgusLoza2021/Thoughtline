package server

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// getObservationArgs is the typed shape of a tl_get_observation call.
type getObservationArgs struct {
	ID int64
}

func registerTLGetObservation(srv *server.MCPServer, s *storage.Storage) {
	srv.AddTool(
		mcp.NewTool("tl_get_observation",
			mcp.WithTitleAnnotation("Get Memory Observation"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(false),
			mcp.WithDescription(tlGetObservationDescription),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The local row id returned by tl_save or tl_search."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := decodeGetObservationArgs(req)
			return doGetObservation(ctx, s, args)
		},
	)
}

const tlGetObservationDescription = `Fetch a single memory by id and return its full, untruncated content plus metadata.

Use after tl_search when a result snippet is a preview (truncated) and the full body matters. The id field is the local 'ID' shown in tl_search and tl_save results.`

func decodeGetObservationArgs(req mcp.CallToolRequest) getObservationArgs {
	a := req.GetArguments()
	return getObservationArgs{ID: int64(asInt(a, "id"))}
}

func doGetObservation(ctx context.Context, s *storage.Storage, args getObservationArgs) (*mcp.CallToolResult, error) {
	if args.ID <= 0 {
		return mcp.NewToolResultError("'id' is required and must be a positive integer"), nil
	}

	// Read path: fetch without brain scoping to discover which brain owns the row,
	// then re-fetch brain-scoped so cross-brain isolation is enforced.
	m, err := s.GetByIDUnscoped(ctx, args.ID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("no memory found with id=%d", args.ID)), nil
	}
	if m.DeletedAt != nil {
		return mcp.NewToolResultError(fmt.Sprintf("no memory found with id=%d", args.ID)), nil
	}
	// Enforce brain-scoped access using the row's own brain_id.
	if m.BrainID != 0 {
		scoped, err := s.GetByID(ctx, m.BrainID, args.ID)
		if errors.Is(err, storage.ErrMemoryNotFound) {
			return mcp.NewToolResultError(fmt.Sprintf("no memory found with id=%d", args.ID)), nil
		}
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("get failed: %v", err)), nil
		}
		m = scoped
	}

	return mcp.NewToolResultText(formatObservation(m)), nil
}

func formatObservation(m memory.Memory) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Title: %s\n", m.Title)
	fmt.Fprintf(&b, "ID: %d\n", m.ID)
	fmt.Fprintf(&b, "Sync ID: %s\n", m.SyncID)
	fmt.Fprintf(&b, "Project: %s\n", m.Project)
	fmt.Fprintf(&b, "Type: %s\n", m.Type)
	fmt.Fprintf(&b, "Scope: %s\n", m.Scope)
	if m.TopicKey != "" {
		fmt.Fprintf(&b, "Topic: %s\n", m.TopicKey)
	}
	fmt.Fprintf(&b, "Revision: %d\n", m.RevisionCount)
	if !m.CreatedAt.IsZero() {
		fmt.Fprintf(&b, "Created: %s\n", m.CreatedAt.UTC().Format("2006-01-02 15:04:05 UTC"))
	}
	if !m.UpdatedAt.IsZero() {
		fmt.Fprintf(&b, "Updated: %s\n", m.UpdatedAt.UTC().Format("2006-01-02 15:04:05 UTC"))
	}
	if len(m.Tags) > 0 {
		fmt.Fprintf(&b, "Tags: %s\n", strings.Join(m.Tags, ", "))
	}
	b.WriteString("\nContent:\n")
	b.WriteString(m.Content)
	return strings.TrimRight(b.String(), "\n")
}
