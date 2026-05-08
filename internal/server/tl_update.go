package server

import (
	"context"
	"errors"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// updateArgs is the typed shape of a tl_update call. The Has* flags carry
// "field was present in the JSON request" so we can distinguish "leave field
// unchanged" from "set field to its zero value" — JSON has no other natural
// way to express this without sentinel values.
type updateArgs struct {
	ID         int64
	Title      string
	HasTitle   bool
	Content    string
	HasContent bool
	Tags       []string
	HasTags    bool
}

func registerTLUpdate(srv *server.MCPServer, s *storage.Storage) {
	srv.AddTool(
		mcp.NewTool("tl_update",
			mcp.WithTitleAnnotation("Update Memory"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(false),
			mcp.WithDescription(tlUpdateDescription),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The local id of the memory to update (returned by tl_save / tl_search / tl_context)."),
			),
			mcp.WithString("title",
				mcp.Description("New title (≤200 chars). Omit to leave unchanged."),
			),
			mcp.WithString("content",
				mcp.Description("New markdown body. Omit to leave unchanged."),
			),
			mcp.WithArray("tags",
				mcp.Description("New tag list. Pass an empty array to clear all tags. Omit to leave unchanged."),
				mcp.Items(map[string]any{"type": "string"}),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := decodeUpdateArgs(req)
			return doUpdate(ctx, s, args)
		},
	)
}

const tlUpdateDescription = `Patch an existing memory by id. Only title, content, and tags are mutable — type, topic_key, project, and scope are identity-defining and cannot be changed (delete + tl_save if those need to move).

Empty patch (no title/content/tags supplied) is a noop: returns the row unchanged without bumping revision_count or updated_at. Real changes bump revision_count and refresh updated_at while preserving id, sync_id, and created_at. The merged memory is re-validated before persisting.`

func decodeUpdateArgs(req mcp.CallToolRequest) updateArgs {
	a := req.GetArguments()
	out := updateArgs{ID: asInt64(a, "id")}
	if v, ok := a["title"]; ok {
		if s, ok := v.(string); ok {
			out.Title = s
			out.HasTitle = true
		}
	}
	if v, ok := a["content"]; ok {
		if s, ok := v.(string); ok {
			out.Content = s
			out.HasContent = true
		}
	}
	if _, ok := a["tags"]; ok {
		out.Tags = asStringSlice(a, "tags")
		if out.Tags == nil {
			out.Tags = []string{} // present but empty array → explicit clear
		}
		out.HasTags = true
	}
	return out
}

func doUpdate(ctx context.Context, s *storage.Storage, args updateArgs) (*mcp.CallToolResult, error) {
	if args.ID <= 0 {
		return mcp.NewToolResultError("'id' is required and must be a positive integer"), nil
	}

	// Discover which brain owns this memory so we can pass brainID to UpdateByID.
	m, err := s.GetByIDUnscoped(ctx, args.ID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("no memory found with id=%d", args.ID)), nil
	}
	if m.DeletedAt != nil {
		return mcp.NewToolResultError(fmt.Sprintf("no memory found with id=%d", args.ID)), nil
	}

	var patch storage.UpdatePatch
	if args.HasTitle {
		patch.Title = &args.Title
	}
	if args.HasContent {
		patch.Content = &args.Content
	}
	if args.HasTags {
		patch.Tags = &args.Tags
	}

	updated, err := s.UpdateByID(ctx, m.BrainID, args.ID, patch)
	if err != nil {
		if errors.Is(err, storage.ErrMemoryNotFound) {
			return mcp.NewToolResultError(fmt.Sprintf("no memory found with id=%d", args.ID)), nil
		}
		// memory.Validate errors: surface via the same formatter tl_save uses.
		return mcp.NewToolResultError(formatValidationError(err)), nil
	}

	action := storage.ActionUpdated
	if patch.IsEmpty() {
		action = storage.ActionNoop
	}
	return mcp.NewToolResultText(formatSaveResult(updated, action)), nil
}
