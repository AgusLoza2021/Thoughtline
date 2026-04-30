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

// sessionStartArgs is the typed shape of a tl_session_start call.
type sessionStartArgs struct {
	Project    string
	AgentLabel string
}

func registerTLSessionStart(srv *server.MCPServer, s *storage.Storage, cfg Config) {
	srv.AddTool(
		mcp.NewTool("tl_session_start",
			mcp.WithTitleAnnotation("Start Memory Session"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(false),
			mcp.WithDescription(tlSessionStartDescription),
			mcp.WithString("project",
				mcp.Description("Project identifier. Defaults to the working directory basename if omitted."),
			),
			mcp.WithString("agent_label",
				mcp.Description("Optional client tag (e.g. 'claude-code', 'cursor', 'zed'). Useful for cross-session forensics. Max 64 chars."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := decodeSessionStartArgs(req)
			return doSessionStart(ctx, s, cfg, args)
		},
	)
}

const tlSessionStartDescription = `Open a new memory session and return its UUIDv7 id. Subsequent tl_save calls can pass session_id to attach memories to this session, enabling cross-session forensics ("what did we work on yesterday?").

Multiple open sessions per project are allowed — the server is stateless and does not track an "active" session. The AI is expected to thread the returned session_id through subsequent calls until tl_session_summary closes it.`

func decodeSessionStartArgs(req mcp.CallToolRequest) sessionStartArgs {
	a := req.GetArguments()
	out := sessionStartArgs{
		Project:    asString(a, "project"),
		AgentLabel: asString(a, "agent_label"),
	}
	out.Project = strings.TrimSpace(out.Project)
	out.AgentLabel = strings.TrimSpace(out.AgentLabel)
	return out
}

func doSessionStart(ctx context.Context, s *storage.Storage, cfg Config, args sessionStartArgs) (*mcp.CallToolResult, error) {
	project := args.Project
	if project == "" {
		project = cfg.DefaultProject
	}
	if project == "" {
		return mcp.NewToolResultError("'project' is required (and could not be auto-detected). Pass it explicitly or restart the server in the project's working directory."), nil
	}

	sess, err := s.StartSession(ctx, project, args.AgentLabel)
	if err != nil {
		// Surface domain validation errors with friendly messages.
		switch {
		case errors.Is(err, memory.ErrAgentLabelTooLong):
			return mcp.NewToolResultError(fmt.Sprintf("'agent_label' too long (limit %d characters).", memory.MaxAgentLabelChars)), nil
		case errors.Is(err, memory.ErrEmptySessionProject):
			return mcp.NewToolResultError("'project' must not be empty."), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("start session failed: %v", err)), nil
	}

	return mcp.NewToolResultText(formatSessionStart(sess)), nil
}

func formatSessionStart(sess memory.Session) string {
	var b strings.Builder
	b.WriteString("Session opened.\n")
	fmt.Fprintf(&b, "Session ID: %s\n", sess.ID)
	fmt.Fprintf(&b, "Project: %s\n", sess.Project)
	if sess.AgentLabel != "" {
		fmt.Fprintf(&b, "Agent: %s\n", sess.AgentLabel)
	}
	fmt.Fprintf(&b, "Started: %s\n", sess.StartedAt.UTC().Format("2006-01-02 15:04:05 UTC"))
	b.WriteString("\nThread this Session ID through subsequent tl_save calls (session_id arg) to attach memories. Close the session with tl_session_summary when work is done.")
	return b.String()
}
