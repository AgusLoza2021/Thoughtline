package server

import (
	"context"
	"errors"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// deleteArgs is the typed shape of a tl_delete call.
type deleteArgs struct {
	ID int64
}

func registerTLDelete(srv *server.MCPServer, s *storage.Storage) {
	srv.AddTool(
		mcp.NewTool("tl_delete",
			mcp.WithTitleAnnotation("Soft-Delete Memory"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(false),
			mcp.WithDescription(tlDeleteDescription),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The local id of the memory to soft-delete."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := decodeDeleteArgs(req)
			return doDelete(ctx, s, args)
		},
	)
}

const tlDeleteDescription = `Soft-delete a memory by id: sets deleted_at, hides the row from tl_search / tl_context / tl_get_observation. The topic_key (if any) becomes available for a fresh tl_save. Repeating tl_delete on an already-deleted row returns "not found".

This is destructive from the AI's point of view (the row stops being visible). It is NOT a hard delete — the row remains on disk and can be recovered manually if needed.`

func decodeDeleteArgs(req mcp.CallToolRequest) deleteArgs {
	a := req.GetArguments()
	return deleteArgs{ID: asInt64(a, "id")}
}

func doDelete(ctx context.Context, s *storage.Storage, args deleteArgs) (*mcp.CallToolResult, error) {
	if args.ID <= 0 {
		return mcp.NewToolResultError("'id' is required and must be a positive integer"), nil
	}

	err := s.SoftDelete(ctx, args.ID)
	if err != nil {
		if errors.Is(err, storage.ErrMemoryNotFound) {
			return mcp.NewToolResultError(fmt.Sprintf("no memory found with id=%d", args.ID)), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("delete failed: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Deleted memory id=%d (soft delete — row hidden from search/context, topic_key freed).", args.ID)), nil
}
