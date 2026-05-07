package server

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

func registerTLPendingGet(srv *server.MCPServer, s *storage.Storage, cfg Config) {
	srv.AddTool(
		mcp.NewTool("tl_pending_get",
			mcp.WithTitleAnnotation("Get Pending Event"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDescription(`Fetch the full payload of a single pending event by ID.

Use tl_pending_list first to find the ID, then call this to inspect the raw
payload before deciding whether to promote it via tl_promote.`),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The integer ID of the pending event (from tl_pending_list)."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return doTLPendingGet(ctx, s, cfg, req.GetArguments())
		},
	)
}

func doTLPendingGet(ctx context.Context, s *storage.Storage, cfg Config, args map[string]any) (*mcp.CallToolResult, error) {
	id := int64(asFloat(args, "id"))
	if id == 0 {
		return mcp.NewToolResultError("'id' is required and must be a non-zero integer"), nil
	}

	ev, err := s.GetPendingByID(ctx, id)
	if errors.Is(err, storage.ErrPendingNotFound) {
		return mcp.NewToolResultError(fmt.Sprintf("pending event %d not found", id)), nil
	}
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("get pending event: %v", err)), nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Pending Event ID: %d\n", ev.ID)
	fmt.Fprintf(&b, "SyncID: %s\n", ev.SyncID)
	fmt.Fprintf(&b, "Project: %s\n", ev.Project)
	fmt.Fprintf(&b, "EventType: %s\n", ev.EventType)
	if ev.SessionID != "" {
		fmt.Fprintf(&b, "SessionID: %s\n", ev.SessionID)
	}
	if ev.ToolName != "" {
		fmt.Fprintf(&b, "ToolName: %s\n", ev.ToolName)
	}
	if ev.ToolUseID != "" {
		fmt.Fprintf(&b, "ToolUseID: %s\n", ev.ToolUseID)
	}
	fmt.Fprintf(&b, "Status: %s\n", ev.Status)
	fmt.Fprintf(&b, "CapturedAt: %s\n", ev.CapturedAt.UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "CreatedAt: %s\n", ev.CreatedAt.UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "\nPayload:\n%s\n", ev.Payload)

	return mcp.NewToolResultText(strings.TrimRight(b.String(), "\n")), nil
}
