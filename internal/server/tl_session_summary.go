package server

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// sessionSummaryArgs is the typed shape of a tl_session_summary call.
type sessionSummaryArgs struct {
	ID      string
	Summary string
}

func registerTLSessionSummary(srv *server.MCPServer, s *storage.Storage) {
	srv.AddTool(
		mcp.NewTool("tl_session_summary",
			mcp.WithTitleAnnotation("Close Memory Session"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(false),
			mcp.WithDescription(tlSessionSummaryDescription),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("The session id returned by tl_session_start."),
			),
			mcp.WithString("summary",
				mcp.Required(),
				mcp.Description("Structured end-of-session digest. Recommended sections: ## Goal / ## Discoveries / ## Accomplished / ## Next Steps / ## Relevant Files."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := decodeSessionSummaryArgs(req)
			return doSessionSummary(ctx, s, args)
		},
	)
}

const tlSessionSummaryDescription = `Close a session opened by tl_session_start, persisting an end-of-session digest. A session can only be closed once — repeating tl_session_summary on a closed session returns an "already ended" error.

Call this MANDATORILY before saying "done" / "listo" / "that's it" so the next session can recover what was being worked on. The summary is durable plain text — write it for a future session that has no other context.`

func decodeSessionSummaryArgs(req mcp.CallToolRequest) sessionSummaryArgs {
	a := req.GetArguments()
	return sessionSummaryArgs{
		ID:      strings.TrimSpace(asString(a, "id")),
		Summary: strings.TrimSpace(asString(a, "summary")),
	}
}

func doSessionSummary(ctx context.Context, s *storage.Storage, args sessionSummaryArgs) (*mcp.CallToolResult, error) {
	args.ID = strings.TrimSpace(args.ID)
	args.Summary = strings.TrimSpace(args.Summary)

	if args.ID == "" {
		return mcp.NewToolResultError("'id' is required (the UUIDv7 returned by tl_session_start)."), nil
	}
	if args.Summary == "" {
		return mcp.NewToolResultError("'summary' is required and must contain non-whitespace characters. Write a structured digest of the session."), nil
	}

	sess, err := s.EndSession(ctx, args.ID, args.Summary)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrSessionNotFound):
			return mcp.NewToolResultError(fmt.Sprintf("session not found: id=%q", args.ID)), nil
		case errors.Is(err, storage.ErrSessionAlreadyEnded):
			return mcp.NewToolResultError(fmt.Sprintf("session %q is already ended. A session can only be closed once; start a new one with tl_session_start.", args.ID)), nil
		case errors.Is(err, memory.ErrSessionSummaryTooLong):
			return mcp.NewToolResultError(fmt.Sprintf("'summary' too long (limit %d bytes). Trim or split into smaller sessions.", memory.MaxSessionSummaryBytes)), nil
		case errors.Is(err, memory.ErrInvalidSessionID):
			return mcp.NewToolResultError("'id' must be a valid UUIDv7 (returned by tl_session_start)."), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("close session failed: %v", err)), nil
	}

	return mcp.NewToolResultText(formatSessionSummary(sess)), nil
}

func formatSessionSummary(sess memory.Session) string {
	var b strings.Builder
	b.WriteString("Session closed.\n")
	fmt.Fprintf(&b, "Session ID: %s\n", sess.ID)
	fmt.Fprintf(&b, "Project: %s\n", sess.Project)
	if sess.AgentLabel != "" {
		fmt.Fprintf(&b, "Agent: %s\n", sess.AgentLabel)
	}
	fmt.Fprintf(&b, "Started: %s\n", sess.StartedAt.UTC().Format("2006-01-02 15:04:05 UTC"))
	if sess.EndedAt != nil {
		fmt.Fprintf(&b, "Ended:   %s\n", sess.EndedAt.UTC().Format("2006-01-02 15:04:05 UTC"))
	}
	fmt.Fprintf(&b, "Duration: %s\n", sess.Duration().Round(time.Second))
	b.WriteString("\nNext session can recover this digest via tl_search or by reading sessions directly. Open a new session with tl_session_start when ready.")
	return b.String()
}
