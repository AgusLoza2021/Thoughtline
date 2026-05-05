# Security Policy

## Scope

Thoughtline is a **local-first, single-user MCP server**. It runs on your machine, owns a single SQLite file, and does not phone home or open network ports. Threats in scope:

- Path traversal or arbitrary-file-read via tool arguments
- SQL injection through tool parameters
- Crash-on-malformed-input (denial of service) on the local stdio MCP loop
- Secret leakage through logs, error messages, or the `tl_stats` / dashboard surfaces
- Privilege escalation through the `migrate` subcommand or schema upgrades

Out of scope: anything that requires the attacker to already have shell access as the user running Thoughtline.

## Supported versions

Only the latest minor release on the `main` branch receives fixes. There is no LTS.

## Reporting a vulnerability

Please **do not** open a public GitHub issue for security problems.

- Use GitHub's [private vulnerability reporting](https://github.com/AgusLoza2021/Thoughtline/security/advisories/new) (preferred), or
- Email the maintainer at the address listed in the most recent commit's `Author` field.

Include: a minimal reproducer, the version (`thoughtline --version`), and the platform. We aim to acknowledge within 5 business days and ship a fix or mitigation within 30 days for high-severity issues.

Coordinated disclosure is appreciated — we will credit you in the release notes unless you prefer otherwise.
