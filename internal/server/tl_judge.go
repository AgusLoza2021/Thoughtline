package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

type judgeArgs struct {
	ExistingID      int64
	IncomingTitle   string
	IncomingContent string
	Relation        string
	Note            string
}

var validRelations = []string{
	"supersedes",
	"compatible",
	"conflicts_with",
	"scoped",
	"not_conflict",
}

var recommendedActions = map[string]string{
	"supersedes":    "save with same topic_key to replace",
	"compatible":    "save as new memory (different topic_key or no topic_key)",
	"conflicts_with": "resolve manually before saving — consider updating existing or creating supersedes link",
	"scoped":        "save as new memory, both are valid in their scope",
	"not_conflict":  "safe to save",
}

func registerTLJudge(srv *server.MCPServer, s *storage.Storage, cfg Config) {
	srv.AddTool(
		mcp.NewTool("tl_judge",
			mcp.WithTitleAnnotation("Judge Memory Conflict"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(false),
			mcp.WithDescription(tlJudgeDescription),
			mcp.WithNumber("existing_id",
				mcp.Required(),
				mcp.Description("ID of the memory that might conflict with the incoming one."),
			),
			mcp.WithString("incoming_title",
				mcp.Required(),
				mcp.Description("Title of the new memory being considered."),
			),
			mcp.WithString("incoming_content",
				mcp.Required(),
				mcp.Description("Content of the new memory being considered."),
			),
			mcp.WithString("relation",
				mcp.Required(),
				mcp.Description("Declared semantic relation — one of: supersedes, compatible, conflicts_with, scoped, not_conflict."),
			),
			mcp.WithString("note",
				mcp.Description("Optional free-form reason for the judgment."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			a := req.GetArguments()
			args := judgeArgs{
				ExistingID:      asInt64(a, "existing_id"),
				IncomingTitle:   strings.TrimSpace(asString(a, "incoming_title")),
				IncomingContent: strings.TrimSpace(asString(a, "incoming_content")),
				Relation:        strings.TrimSpace(asString(a, "relation")),
				Note:            strings.TrimSpace(asString(a, "note")),
			}
			return doJudge(ctx, s, cfg, args)
		},
	)
}

const tlJudgeDescription = `Compare an incoming memory against an existing one and declare a semantic relation before deciding to write.

This tool does NOT write to the database — it is a pure read + judgment formatting tool. Use it to surface semantic conflicts before calling tl_save.

Relations:
  supersedes    — incoming replaces existing; save with same topic_key
  compatible    — both can coexist; save with a different topic_key
  conflicts_with — direct contradiction; resolve manually before saving
  scoped        — both are valid but in different scopes
  not_conflict  — no semantic overlap; safe to save`

func doJudge(ctx context.Context, s *storage.Storage, cfg Config, args judgeArgs) (*mcp.CallToolResult, error) {
	if args.ExistingID <= 0 {
		return mcp.NewToolResultError("'existing_id' is required and must be a positive integer"), nil
	}
	if args.IncomingTitle == "" {
		return mcp.NewToolResultError("'incoming_title' is required and must contain non-whitespace characters"), nil
	}
	if args.IncomingContent == "" {
		return mcp.NewToolResultError("'incoming_content' is required and must contain non-whitespace characters"), nil
	}
	if !isValidRelation(args.Relation) {
		return mcp.NewToolResultError(fmt.Sprintf(
			"invalid 'relation' %q — must be one of: %s",
			args.Relation, strings.Join(validRelations, ", "),
		)), nil
	}

	existing, err := s.GetByIDUnscoped(ctx, args.ExistingID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("no memory found with existing_id=%d", args.ExistingID)), nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Judgment:\n\n")

	fmt.Fprintf(&b, "Existing Memory:\n")
	fmt.Fprintf(&b, "  ID: %d\n", existing.ID)
	fmt.Fprintf(&b, "  Title: %s\n", existing.Title)
	fmt.Fprintf(&b, "  Type: %s\n", existing.Type)
	if existing.TopicKey != "" {
		fmt.Fprintf(&b, "  Topic: %s\n", existing.TopicKey)
	}
	if !existing.UpdatedAt.IsZero() {
		fmt.Fprintf(&b, "  Updated: %s\n", existing.UpdatedAt.UTC().Format("2006-01-02 15:04:05 UTC"))
	}

	fmt.Fprintf(&b, "\nIncoming Memory:\n")
	fmt.Fprintf(&b, "  Title: %s\n", args.IncomingTitle)
	fmt.Fprintf(&b, "  Content: %s\n", args.IncomingContent)

	fmt.Fprintf(&b, "\nRelation: %s\n", args.Relation)
	if args.Note != "" {
		fmt.Fprintf(&b, "Note: %s\n", args.Note)
	}
	fmt.Fprintf(&b, "Recommended Action: %s\n", recommendedActions[args.Relation])

	return mcp.NewToolResultText(strings.TrimRight(b.String(), "\n")), nil
}

func isValidRelation(r string) bool {
	for _, v := range validRelations {
		if r == v {
			return true
		}
	}
	return false
}
