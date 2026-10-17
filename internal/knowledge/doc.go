// Package knowledge parses the owner's hand-written Markdown recipe corpora
// (for example ~/.claude/skills/unity-knowledge: a SKILL.md index plus
// entries/*.md) into typed Entries so Thoughtline can consult them before code
// gets written.
//
// The Markdown files stay the source of truth; this package never writes.
// Parsing is deliberately tolerant: corpora are hand-edited, so a missing
// frontmatter block, an empty tag, or an unexpected "##" heading must degrade
// into something usable instead of failing the whole scan.
package knowledge
