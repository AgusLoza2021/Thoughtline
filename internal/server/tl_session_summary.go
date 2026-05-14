package server

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// uuidv7Re matches UUIDv7 strings: version nibble is 7.
var uuidv7Re = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[0-9a-f]{4}-[0-9a-f]{12}`)

// sessionSummaryArgs is the typed shape of a tl_session_summary call.
type sessionSummaryArgs struct {
	ID              string
	Summary         string
	CompactionBlock string
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
			mcp.WithString("compaction_block",
				mcp.Description("Optional. Paste the compaction summary block here when context was compacted and the session ID is no longer known. The tool will extract the session ID automatically."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := decodeSessionSummaryArgs(req)
			return doSessionSummary(ctx, s, args)
		},
	)
}

const tlSessionSummaryDescription = `Close a session opened by tl_session_start, persisting an end-of-session digest. A session can only be closed once — repeating tl_session_summary on a closed session returns an "already ended" error.

Call this MANDATORILY before saying "done" / "listo" / "that's it" so the next session can recover what was being worked on. The summary is durable plain text — write it for a future session that has no other context.

If context was compacted and the session ID is no longer known, pass the compaction summary text in compaction_block — the session ID will be extracted automatically.`

func decodeSessionSummaryArgs(req mcp.CallToolRequest) sessionSummaryArgs {
	a := req.GetArguments()
	return sessionSummaryArgs{
		ID:              strings.TrimSpace(asString(a, "id")),
		Summary:         strings.TrimSpace(asString(a, "summary")),
		CompactionBlock: strings.TrimSpace(asString(a, "compaction_block")),
	}
}

// extractUUIDv7 returns the last UUIDv7 found in s, or "" if none.
func extractUUIDv7(s string) string {
	matches := uuidv7Re.FindAllString(strings.ToLower(s), -1)
	if len(matches) == 0 {
		return ""
	}
	return matches[len(matches)-1]
}

func doSessionSummary(ctx context.Context, s *storage.Storage, args sessionSummaryArgs) (*mcp.CallToolResult, error) {
	args.ID = strings.TrimSpace(args.ID)
	args.Summary = strings.TrimSpace(args.Summary)

	compactionRecovered := false
	if args.ID == "" && args.CompactionBlock != "" {
		extracted := extractUUIDv7(args.CompactionBlock)
		if extracted != "" {
			args.ID = extracted
			compactionRecovered = true
		}
	}

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

	return mcp.NewToolResultText(formatSessionSummary(sess, compactionRecovered)), nil
}

func formatSessionSummary(sess memory.Session, compactionRecovered bool) string {
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
	if compactionRecovered {
		b.WriteString("compaction_recovered: true\n")
	}
	b.WriteString("\nNext session can recover this digest via tl_search or by reading sessions directly. Open a new session with tl_session_start when ready.")
	return b.String()
}
