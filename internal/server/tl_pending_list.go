package server

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

const snippetMaxChars = 200

func registerTLPendingList(srv *server.MCPServer, s *storage.Storage, cfg Config) {
	srv.AddTool(
		mcp.NewTool("tl_pending_list",
			mcp.WithTitleAnnotation("List Pending Events"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDescription(`List raw hook events waiting for promotion.

Returns a paginated list of pending events captured by thoughtline hook. Use
tl_pending_get to inspect the full payload of a specific event, and tl_promote
to convert selected events into typed memories.`),
			mcp.WithString("project",
				mcp.Description("Project identifier. Defaults to the server's default project."),
			),
			mcp.WithString("status",
				mcp.Description("Filter by status: pending (default), promoted, archived, or empty for all."),
			),
			mcp.WithString("event_type",
				mcp.Description("Filter by event type, e.g. PreToolUse, PostToolUse, SessionStart."),
			),
			mcp.WithString("since",
				mcp.Description("ISO-8601 timestamp lower bound for captured_at filter."),
			),
			mcp.WithNumber("limit",
				mcp.Description("Maximum number of results (default 50, max 200)."),
			),
			mcp.WithNumber("offset",
				mcp.Description("Pagination offset (default 0)."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return doTLPendingList(ctx, s, cfg, req.GetArguments())
		},
	)
}

func doTLPendingList(ctx context.Context, s *storage.Storage, cfg Config, args map[string]any) (*mcp.CallToolResult, error) {
	project := asString(args, "project")
	if project == "" {
		project = cfg.DefaultProject
	}

	statusFilter := asString(args, "status")
	if statusFilter == "" {
		statusFilter = "pending"
	}

	eventType := asString(args, "event_type")
	sinceStr := asString(args, "since")

	limit := int(asFloat(args, "limit"))
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	offset := int(asFloat(args, "offset"))

	var since time.Time
	if sinceStr != "" {
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02"} {
			if t, err := time.Parse(layout, sinceStr); err == nil {
				since = t
				break
			}
		}
	}

	events, err := s.ListPending(ctx, storage.ListPendingParams{
		Project:   project,
		Status:    statusFilter,
		EventType: eventType,
		Since:     since,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("list pending: %v", err)), nil
	}

	if len(events) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf(
			"No pending events for project %q (status=%s).", project, statusFilter,
		)), nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Pending events for project %q (status=%s) — %d result(s):\n\n",
		project, statusFilter, len(events))

	for _, ev := range events {
		snippet := snippetOf(ev.Payload, snippetMaxChars)
		fmt.Fprintf(&b, "ID: %d | %s | session=%s | %s\n  Payload snippet: %s\n\n",
			ev.ID, ev.EventType, ev.SessionID,
			ev.CapturedAt.UTC().Format(time.RFC3339),
			snippet,
		)
	}

	return mcp.NewToolResultText(strings.TrimRight(b.String(), "\n")), nil
}

// snippetOf returns the first maxChars runes of s.
func snippetOf(s string, maxChars int) string {
	if utf8.RuneCountInString(s) <= maxChars {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxChars]) + "…"
}

func asFloat(a map[string]any, k string) float64 {
	v, _ := a[k].(float64)
	return v
}
