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

// saveArgs is the typed shape of a tl_save call. Decoding the MCP arguments
// into this struct (decodeSaveArgs) keeps the handler straightforward and
// makes unit testing the save logic possible without spinning up the MCP
// transport.
type saveArgs struct {
	Title    string
	Content  string
	Type     string
	Scope    string
	TopicKey string
	Project  string
	Tags     []string
}

func registerTLSave(srv *server.MCPServer, s *storage.Storage, cfg Config) {
	srv.AddTool(
		mcp.NewTool("tl_save",
			mcp.WithTitleAnnotation("Save Memory"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(false),
			mcp.WithDescription(tlSaveDescription),
			mcp.WithString("title",
				mcp.Required(),
				mcp.Description("Short, searchable headline. Imperative form preferred ('Lock loot UI to 4x6 grid')."),
			),
			mcp.WithString("content",
				mcp.Required(),
				mcp.Description("Markdown body. Use the **What**/**Why**/**Where**/**Learned** structure when applicable."),
			),
			mcp.WithString("type",
				mcp.Required(),
				mcp.Description("One of: game-design-decision, scene-pattern, asset-reference, perf-gotcha, pipeline-step, script-pattern, bugfix, convention, preference."),
			),
			mcp.WithString("scope",
				mcp.Description("project (default) or personal. The 'preference' type REQUIRES personal."),
			),
			mcp.WithString("topic_key",
				mcp.Description("Optional stable key for evolving topics. Re-saving with the same project+topic_key upserts. Pattern: <category>/<subject>, lowercase."),
			),
			mcp.WithString("project",
				mcp.Description("Project identifier. Defaults to the working directory basename if omitted."),
			),
			mcp.WithArray("tags",
				mcp.Description("Lowercase tags, optionally key:value (e.g. 'engine:playcanvas', 'platform:android')."),
				mcp.Items(map[string]any{"type": "string"}),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := decodeSaveArgs(req)
			return doSave(ctx, s, cfg, args)
		},
	)
}

const tlSaveDescription = `Persist a memory to the local Thoughtline store.

Call this PROACTIVELY after any of these:
  - A game-design decision is made (and especially after the rationale is settled)
  - A scene/component/script pattern is discovered worth reusing
  - An asset's import settings or origin matter for reproducibility
  - A performance gotcha (drawcalls, GC, batching) is hit and debugged
  - A bug is fixed (root cause + fix + what misled you)
  - A pipeline step worth documenting is run
  - A naming/structural convention is agreed

If the topic is likely to evolve, set topic_key — re-saves on the same key will upsert (preserving id and creation time, bumping revision_count). Identical re-saves are noops.`

// decodeSaveArgs extracts and lightly cleans the typed arguments from an MCP
// CallToolRequest.
func decodeSaveArgs(req mcp.CallToolRequest) saveArgs {
	a := req.GetArguments()
	out := saveArgs{
		Title:    asString(a, "title"),
		Content:  asString(a, "content"),
		Type:     asString(a, "type"),
		Scope:    asString(a, "scope"),
		TopicKey: asString(a, "topic_key"),
		Project:  asString(a, "project"),
		Tags:     asStringSlice(a, "tags"),
	}
	out.Title = strings.TrimSpace(out.Title)
	out.Type = strings.TrimSpace(out.Type)
	out.Scope = strings.TrimSpace(out.Scope)
	out.TopicKey = strings.TrimSpace(out.TopicKey)
	out.Project = strings.TrimSpace(out.Project)
	return out
}

// doSave is the testable core of tl_save. It is pure with respect to the
// MCP transport — given a *storage.Storage and decoded args, it produces a
// CallToolResult or returns an error.
func doSave(ctx context.Context, s *storage.Storage, cfg Config, args saveArgs) (*mcp.CallToolResult, error) {
	scope := args.Scope
	if scope == "" {
		if args.Type == string(memory.TypePreference) {
			scope = string(memory.ScopePersonal)
		} else {
			scope = string(memory.ScopeProject)
		}
	}

	project := args.Project
	if project == "" {
		project = cfg.DefaultProject
	}

	m := memory.Memory{
		Project:  project,
		Scope:    memory.Scope(scope),
		Type:     memory.Type(args.Type),
		TopicKey: args.TopicKey,
		Title:    args.Title,
		Content:  args.Content,
		Tags:     args.Tags,
	}

	if err := memory.Validate(m); err != nil {
		return mcp.NewToolResultError(formatValidationError(err)), nil
	}

	saved, action, err := s.Save(ctx, m)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("save failed: %v", err)), nil
	}

	return mcp.NewToolResultText(formatSaveResult(saved, action)), nil
}

// formatSaveResult builds the natural-language text the AI sees in its tool
// result. Compact, structured-ish lines so the model can parse fields back
// out (id, sync_id, action) when it needs to reference them.
func formatSaveResult(m memory.Memory, action storage.UpsertAction) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Saved (action=%s): %q\n", action, m.Title)
	fmt.Fprintf(&b, "ID: %d\n", m.ID)
	fmt.Fprintf(&b, "Sync ID: %s\n", m.SyncID)
	fmt.Fprintf(&b, "Project: %s\n", m.Project)
	fmt.Fprintf(&b, "Type: %s\n", m.Type)
	fmt.Fprintf(&b, "Scope: %s\n", m.Scope)
	if m.TopicKey != "" {
		fmt.Fprintf(&b, "Topic: %s\n", m.TopicKey)
	}
	fmt.Fprintf(&b, "Revision: %d\n", m.RevisionCount)
	if action == storage.ActionNoop {
		b.WriteString("Note: identical content — no changes applied.\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// formatValidationError converts a domain validation error into a message
// the AI can act on (and the user can read).
func formatValidationError(err error) string {
	switch {
	case errors.Is(err, memory.ErrInvalidType):
		return "invalid 'type' — must be one of: game-design-decision, scene-pattern, asset-reference, perf-gotcha, pipeline-step, script-pattern, bugfix, convention, preference"
	case errors.Is(err, memory.ErrInvalidScope):
		return "invalid 'scope' — must be 'project' or 'personal'"
	case errors.Is(err, memory.ErrPreferenceMustBePersonal):
		return "type 'preference' requires scope='personal'"
	case errors.Is(err, memory.ErrNonPreferenceMustBeProject):
		return "non-preference types require scope='project'"
	case errors.Is(err, memory.ErrEmptyProject):
		return "'project' is required (and could not be auto-detected). Pass it explicitly or restart the server in the project's working directory."
	case errors.Is(err, memory.ErrEmptyTitle):
		return "'title' is required and must contain non-whitespace characters"
	case errors.Is(err, memory.ErrTitleTooLong):
		return fmt.Sprintf("'title' too long (limit %d characters)", memory.MaxTitleChars)
	case errors.Is(err, memory.ErrEmptyContent):
		return "'content' is required and must contain non-whitespace characters"
	case errors.Is(err, memory.ErrContentTooLong):
		return fmt.Sprintf("'content' too long (limit %d bytes). Split into smaller observations.", memory.MaxContentBytes)
	case errors.Is(err, memory.ErrInvalidTopicKey):
		return "'topic_key' format invalid. Use lowercase letters/digits/'/'/'_'/'-', start with a letter or digit, max 129 chars."
	case errors.Is(err, memory.ErrInvalidTag):
		return "one of the 'tags' is invalid. Lowercase only, optional ':' for key:value, max 41 chars."
	default:
		return fmt.Sprintf("validation failed: %v", err)
	}
}

func asString(a map[string]any, k string) string {
	v, _ := a[k].(string)
	return v
}

func asStringSlice(a map[string]any, k string) []string {
	raw, ok := a[k].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
