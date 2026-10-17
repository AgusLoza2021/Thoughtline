// tl_check.go — knowledge-check MCP tool.
//
// The tool answers one question for an agent that is about to write code:
// "which of the owner's hand-written recipes, and which rules of this project,
// apply to this change?" It is a read-only tool: a check never saves, never
// creates a brain and never mutates the corpus.
package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/AgusLoza2021/Thoughtline/internal/knowledge"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// checkArgs is the typed shape of a tl_check call.
type checkArgs struct {
	Task             string
	Paths            []string
	Project          string
	KnowledgeProject string
	ExcludeStacks    []string
	Limit            int
}

func registerTLCheck(srv *server.MCPServer, s *storage.Storage, cfg Config) {
	srv.AddTool(
		mcp.NewTool("tl_check",
			mcp.WithTitleAnnotation("Knowledge Check"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(false),
			mcp.WithDescription(tlCheckDescription),
			mcp.WithString("task",
				mcp.Description("What you are about to build or fix, in one sentence (e.g. 'add camera shake when the player gets hit')."),
			),
			mcp.WithArray("paths",
				mcp.Description("Files the change will touch. Their names are part of the query, so 'Assets/EnemyHitFlashSystem.cs' finds the hit-flash recipes."),
				mcp.Items(map[string]any{"type": "string"}),
			),
			mcp.WithString("project",
				mcp.Description("Project whose saved conventions/decisions/architecture should be checked first. Defaults to the working directory project."),
			),
			mcp.WithString("knowledge_project",
				mcp.Description("Brain holding the recipe corpus. Default: the configured knowledge project (THOUGHTLINE_KNOWLEDGE_PROJECT, else '"+knowledge.DefaultSyncProject+"')."),
			),
			mcp.WithArray("exclude_stacks",
				mcp.Description("Foreign technology stacks to skip (e.g. ['svn','fiber']). Entries tagged stack:<name> are dropped and counted in the 'excluded' line."),
				mcp.Items(map[string]any{"type": "string"}),
			),
			mcp.WithNumber("limit",
				mcp.Description(fmt.Sprintf("Max recipe hits (default %d, hard cap %d). Project rules are separate: at most 3 are listed first.", knowledge.DefaultCheckLimit, knowledge.MaxCheckLimit)),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := decodeCheckArgs(req)
			return doCheck(ctx, s, cfg, args)
		},
	)
}

const tlCheckDescription = `Check what the owner already wrote about this change BEFORE you write or edit code for a feature.

Call this FIRST — before the first edit of a task, not after it — with a one-sentence 'task' and the files you are about to touch. It searches the hand-written recipe corpus (Unity camera/UI/VFX/physics/editor recipes and similar) plus the current project's saved conventions, decisions and architecture, and returns for each hit its 'Applies when', 'Validate' and 'Limits' sections.

Project rules are listed first on purpose: a rule saved for this project always wins over a generic recipe.

Rules for using the answer:
- If a hit's "Applies when" matches what you are doing, follow its procedure; read the full entry with tl_search/tl_get_observation (topic_key is shown) when the sections are not enough.
- Validate the result as the hit says, not as you would have done it.
- Ignore hits whose "Applies when" does not match; the score is relative relevance, not a promise of fit.

An empty answer is normal and means nothing applies: proceed. Entries from foreign stacks (SVN, Fiber, other projects) are excluded and counted on the last line.`

func decodeCheckArgs(req mcp.CallToolRequest) checkArgs {
	a := req.GetArguments()
	out := checkArgs{
		Task:             strings.TrimSpace(asString(a, "task")),
		Paths:            asStringSlice(a, "paths"),
		Project:          strings.TrimSpace(asString(a, "project")),
		KnowledgeProject: strings.TrimSpace(asString(a, "knowledge_project")),
		ExcludeStacks:    asStringSlice(a, "exclude_stacks"),
		Limit:            asInt(a, "limit"),
	}
	return out
}

func doCheck(ctx context.Context, s *storage.Storage, cfg Config, args checkArgs) (*mcp.CallToolResult, error) {
	if args.Task == "" && len(args.Paths) == 0 {
		return mcp.NewToolResultError("at least one of 'task' or 'paths' is required"), nil
	}

	project := args.Project
	if project == "" {
		project = cfg.DefaultProject
	}
	// The knowledge project and the excluded stacks are operator config: an
	// agent that omits them must get the configured corpus, not an empty one.
	knowledgeProject := args.KnowledgeProject
	if knowledgeProject == "" {
		knowledgeProject = cfg.KnowledgeProject
	}
	exclude := args.ExcludeStacks
	if len(exclude) == 0 {
		exclude = cfg.KnowledgeExclude
	}

	res, err := knowledge.Check(ctx, s, knowledge.CheckRequest{
		Task:             args.Task,
		Paths:            args.Paths,
		Project:          project,
		KnowledgeProject: knowledgeProject,
		ExcludeStacks:    exclude,
		Limit:            args.Limit,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("knowledge check failed: %v", err)), nil
	}
	return mcp.NewToolResultText(formatCheckResult(res)), nil
}

// formatCheckResult renders the compact Markdown an agent reads. Sections that
// the entry does not have are omitted rather than shown empty: an empty
// "Limits" block reads as "there are no limits", which is a lie.
func formatCheckResult(res knowledge.CheckResult) string {
	var b strings.Builder
	if len(res.Hits) == 0 {
		fmt.Fprintf(&b, "No applicable knowledge for %q.\n", res.Query)
		fmt.Fprintf(&b, "excluded %d foreign-stack entries", res.Excluded)
		return b.String()
	}

	fmt.Fprintf(&b, "Knowledge check for %q — %d hit(s). Project rules come first and override generic recipes.\n",
		res.Query, len(res.Hits))
	for _, h := range res.Hits {
		fmt.Fprintf(&b, "\n### [%s] %s (%s, score %.2f)\n", h.Source, h.Title, h.TopicKey, h.Score)
		writeCheckSection(&b, "Applies when", h.AppliesWhen)
		writeCheckSection(&b, "Validate", h.Validate)
		writeCheckSection(&b, "Limits", h.Limits)
		if len(h.Stacks) > 0 {
			fmt.Fprintf(&b, "**Stacks**: %s\n", strings.Join(h.Stacks, ", "))
		}
	}
	fmt.Fprintf(&b, "\nexcluded %d foreign-stack entries", res.Excluded)
	return b.String()
}

func writeCheckSection(b *strings.Builder, label, body string) {
	body = strings.TrimSpace(body)
	if body == "" {
		return
	}
	fmt.Fprintf(b, "**%s**\n%s\n", label, body)
}
