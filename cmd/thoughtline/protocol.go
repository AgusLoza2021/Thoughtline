package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// protocolVersion identifies the shape of the markdown emitted by the
// `thoughtline protocol` subcommand. Bump this when the active-protocol
// content changes meaningfully so plugins / hooks can detect drift.
const protocolVersion = "1"

// runProtocol implements `thoughtline protocol`. It emits the active
// protocol markdown to stdout, ready to be consumed by Claude Code
// hooks (SessionStart, PreCompact, etc.) as additionalContext.
//
// One source of truth: this replaces the duplicated bash heredocs in
// plugin/claude-code/scripts/*.sh. The binary is cross-platform, so
// Windows users no longer need WSL/Git Bash to install the plugin.
func runProtocol(args []string) error {
	fs := flag.NewFlagSet("protocol", flag.ContinueOnError)
	event := fs.String("event", "session-start", "session-start | post-compaction")
	project := fs.String("project", "", "project identifier (auto-detected from CWD when empty)")
	out := fs.String("o", "", "write to file instead of stdout")
	if err := fs.Parse(args); err != nil {
		return err
	}

	proj := *project
	if proj == "" {
		proj = detectProject()
	}

	var body string
	switch *event {
	case "session-start", "startup", "clear", "resume":
		body = renderProtocol(proj, false)
	case "post-compaction", "compact":
		body = renderProtocol(proj, true)
	default:
		return fmt.Errorf("unknown --event %q (expected session-start or post-compaction)", *event)
	}

	w := io.Writer(os.Stdout)
	if *out != "" {
		f, err := os.Create(*out)
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}

	_, err := io.WriteString(w, body)
	return err
}

// renderProtocol returns the protocol markdown with the project name
// substituted in. When postCompaction is true the recovery 4-step is
// appended at the end.
func renderProtocol(project string, postCompaction bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## Thoughtline Persistent Memory — ACTIVE PROTOCOL (v%s)\n\n", protocolVersion)
	fmt.Fprintf(&b, "You have thoughtline memory tools (namespace: `tl`). This protocol is MANDATORY and ALWAYS ACTIVE.\n")
	fmt.Fprintf(&b, "Active project: `%s`\n\n", project)

	b.WriteString(`### CORE TOOLS — always available, no ToolSearch needed
tl_save, tl_search, tl_context, tl_get_observation, tl_session_start, tl_session_summary

Other tools via ToolSearch: tl_update, tl_delete, tl_stats

### PROACTIVE SAVE — do NOT wait for the user to ask
Call ` + "`tl_save`" + ` IMMEDIATELY after ANY of these:
- Architecture or design decision made
- Team convention or workflow established / updated
- Tool or library choice made with tradeoffs
- Bug fixed (include root cause)
- Feature implemented with non-obvious approach
- Configuration change or environment setup done
- Non-obvious discovery, gotcha, or edge case found
- Pattern established (naming, structure, convention)
- User preference or constraint learned
- User confirms a recommendation ("dale", "go with that", "sí, esa")
- User rejects an approach or expresses a preference

**Self-check after EVERY task**: "Did I or the user just decide, confirm, prefer, fix, learn, or establish a convention? If yes → tl_save NOW."

Recommended content shape (What / Why / Where / Learned):
` + "```" + `
**What**: [one sentence — what was done or decided]
**Why**: [the reason — user request, bug, perf, etc.]
**Where**: [files or paths affected]
**Learned**: [gotchas, edge cases, surprises — omit if none]
` + "```" + `

### TOPIC KEY CONVENTION
Use ` + "`category/subject`" + ` or ` + "`category/area/subject`" + `. Re-saving the same topic_key upserts (same sync_id, bumped revision_count).
Examples: ` + "`decision/sqlite-wal`" + `, ` + "`architecture/storage-layer`" + `, ` + "`bugfix/fts5-shared-cache`" + `, ` + "`convention/script-naming`" + `.

### SEARCH MEMORY when:
- User asks to recall ("remember", "what did we do", "acordate", "qué hicimos")
- Starting work on something that might have been done before
- User mentions a topic you have no context on
- User's FIRST message references the project, a feature, or a problem — call tl_search first
- A query containing "/" matches topic_key first (GLOB shortcut)

### SEARCH WORKFLOW
1. tl_search returns id, title, snippet (≤300 chars), score
2. If a snippet looks promising → tl_get_observation with the id for full untruncated content

### SESSION CLOSE — before saying "done"/"listo":
Call tl_session_summary with: Goal, Discoveries, Accomplished, Next Steps, Relevant Files.
`)

	if postCompaction {
		fmt.Fprintf(&b, "\n---\n\n")
		b.WriteString("CRITICAL INSTRUCTION POST-COMPACTION — follow these steps IN ORDER:\n\n")
		fmt.Fprintf(&b, "1. FIRST: Call tl_session_summary with the content of the compacted summary above. Use project: '%s'.\n", project)
		b.WriteString("   This preserves what was accomplished before compaction.\n\n")
		fmt.Fprintf(&b, "2. THEN: Call tl_context with project: '%s' to recover recent session history and observations.\n", project)
		b.WriteString("   Read the returned context carefully — it tells you what was being worked on.\n\n")
		b.WriteString("3. If you need detail on a specific topic, call tl_search with relevant keywords, then tl_get_observation for the full content.\n\n")
		b.WriteString("4. Only THEN continue with what the user asked.\n\n")
		b.WriteString("All 4 steps are MANDATORY. Without them you lose context and start blind.\n")
	}

	return b.String()
}

// detectProject mirrors the bash detect_project helper. Order:
// 1. THOUGHTLINE_PROJECT env var
// 2. git remote origin repo name
// 3. git toplevel basename
// 4. CWD basename
// 5. "default"
func detectProject() string {
	if p := os.Getenv("THOUGHTLINE_PROJECT"); p != "" {
		return p
	}

	if name := gitRemoteRepoName(); name != "" {
		return name
	}
	if root := gitToplevel(); root != "" {
		return filepath.Base(root)
	}

	cwd, err := os.Getwd()
	if err == nil && cwd != "" {
		base := filepath.Base(cwd)
		if base != "." && base != string(filepath.Separator) {
			return base
		}
	}
	return "default"
}

func gitRemoteRepoName() string {
	out, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return ""
	}
	url := strings.TrimSpace(string(out))
	url = strings.TrimSuffix(url, ".git")
	// Take the part after the last / or :
	if i := strings.LastIndexAny(url, "/:"); i >= 0 {
		return url[i+1:]
	}
	return url
}

func gitToplevel() string {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
