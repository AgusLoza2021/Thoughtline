# Security Policy

## Project status

**Thoughtline ships its own memory server**, and this repository is the project: the Go source
under `cmd/` and `internal/`, the Claude Code plugin under `plugin/claude-code/`, and a gamedev
memory vocabulary that runs on that server or on
[Engram](https://github.com/Gentleman-Programming/engram). Builds tagged `v*.*.*` are published
by CI. See the [README](README.md) for what the published tag contains and where `main`
currently sits.

**No response time is promised.** This is a small project with one maintainer, so treat it that
way: reports are read, and a valid one is acted on as time allows. There is no LTS branch and no
guaranteed fix window.

## What is still worth reporting

The Go source on `main` is public and buildable, so two classes of report are real:

- **A vulnerability in the server itself**, on any path a user could reach. Threats in scope: path traversal or arbitrary-file-read via tool
  arguments, SQL injection through tool parameters, crash-on-malformed-input on the local stdio
  loop, secret leakage through logs or error messages, and privilege escalation through the
  `migrate` subcommand or schema upgrades.
- **A supply-chain problem in this repository rather than in its code** — a workflow, an install
  script, or a plugin manifest that could be abused to run something on your machine. The
  automation is part of the shipped surface, so this class counts.

Out of scope: anything that requires the attacker to already have shell access as the user
running Thoughtline.

## Reporting a vulnerability

Please **do not** open a public GitHub issue for these.

- Use GitHub's [private vulnerability reporting](https://github.com/AgusLoza2021/Thoughtline/security/advisories/new) (preferred), or
- Email the maintainer at the address listed in the most recent commit's `Author` field.

Include a minimal reproducer and the platform.

## What to expect

**There is no response deadline, and no fix deadline.** Reports will be read, and a valid and
material one will be published as an advisory. A previous version of this file promised
acknowledgement within 5 business days and a fix within 30 days; that promise was withdrawn and
is not being reinstated.

If what you need is a memory server with a team behind it, use
[Engram](https://github.com/Gentleman-Programming/engram). A vulnerability report there is worth
more of your time than one here.
