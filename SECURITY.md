# Security Policy

## Project status

**Thoughtline is retired.** The v0.1.0 MCP server this repository was built around is no longer
maintained, and this repository no longer publishes binaries, release archives or an installable
Claude Code plugin. What remains here is a record of that server plus a gamedev memory vocabulary
that runs on [Engram](https://github.com/Gentleman-Programming/engram).

**No version receives security fixes.** That is the honest version of what used to be written
here as a support policy: there is no maintained branch, no latest-minor release line, no LTS,
and no release process to credit anyone in.

## What is still worth reporting

The Go source on `main` stays public and buildable, so two classes of report are still real:

- **A vulnerability in the retired code itself**, severe enough to matter to someone who builds
  it from source anyway. Threats in scope: path traversal or arbitrary-file-read via tool
  arguments, SQL injection through tool parameters, crash-on-malformed-input on the local stdio
  loop, secret leakage through logs or error messages, and privilege escalation through the
  `migrate` subcommand or schema upgrades.
- **A supply-chain problem in this repository rather than in its code** — a workflow, an install
  script, or a plugin manifest that could be abused to run something on your machine. This is
  the class that matters most here now, because the code is frozen and the automation is not.

Out of scope: anything that requires the attacker to already have shell access as the user
running Thoughtline.

## Reporting a vulnerability

Please **do not** open a public GitHub issue for these.

- Use GitHub's [private vulnerability reporting](https://github.com/AgusLoza2021/Thoughtline/security/advisories/new) (preferred), or
- Email the maintainer at the address listed in the most recent commit's `Author` field.

Include a minimal reproducer and the platform.

## What to expect

**There is no response deadline, and no fix deadline.** Reports will be read, and a valid and
material one will be published as an advisory, but this project is unmaintained — a previous
version of this file promised acknowledgement within 5 business days and a fix within 30 days,
and that promise is not kept anymore.

If you need a maintained memory server for a coding agent, use
[Engram](https://github.com/Gentleman-Programming/engram). A vulnerability report there is worth
more of your time than one here.
