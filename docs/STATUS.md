---
title: Pushover status
date: 2026-10-18
status: active
tags: [status]
---

# Status

> Where the project stands today.
> Design lives in [ARCHITECTURE.md](ARCHITECTURE.md), and unresolved work lives in [OPEN_ITEMS.md](OPEN_ITEMS.md).

## Toolchain

- Go directive: `1.27.0` (`go.mod:3`), upgraded from `1.26.2` on 2026-10-17 with `go fix` applied.
- Installed Go: `go1.27.2 darwin/arm64`.
- `golangci-lint`: `2.14.0` (built with go1.27.1).
- `markdownlint-cli2`: `0.23.3` (markdownlint `0.41.1`).
- `task`: `3.54.0`.
- MCP transport: `github.com/mark3labs/mcp-go v1.3.0` (`go.mod:9`).
- JWT validation: `github.com/golang-jwt/jwt/v5 v5.3.1` (`go.mod:6`).
- Pushover API client: `github.com/gregdel/pushover v1.4.0` (`go.mod:7`).
- Environment loader: `github.com/joho/godotenv v1.5.1` (`go.mod:8`).
- Terminal password input: `golang.org/x/term v0.47.0` (`go.mod:10`).

## Health

- `task check` (fmt:check + vet + lint:ci + lint:md + test:short with `-race`) passes as of 2026-10-18.
- `task build` produces `bin/pushover` with the version injected from `git describe` and rebuilds when that version changes (`Taskfile.yml:13`, `Taskfile.yml:30-47`).
  No tag exists yet, so the version is a commit hash until `v1.0.0` is created.

## Deployment

karma.rrj.pl runs Pushover on `127.0.0.1:8092` behind nginx, redeployed 2026-10-18 from `77f04a9`.
Port 8092 is also closed by the firewall with stateless HTTP and OAuth.
The issuer is `https://karma.rrj.pl/pushover`, and the resource is `https://karma.rrj.pl/pushover/mcp`.
The systemd unit is `/etc/systemd/system/pushover.service`.
The claude.ai connector was verified on 2026-10-17 by logging in and calling `send_notification`.
The static Trading token still works.

## Recent changes

- 2026-10-18 — Week-in-review fixes. `d505362` reports delivery honestly when Pushover omits quota headers;
  `5fc5e59` makes configuration fail closed, validates it on the CLI paths, and refuses misspelled subcommands;
  `e731dec` adds rate-limit and wrong-secret tests, bounds login CPU with a 4-slot semaphore, and fixes the consent text;
  `6e69f6b` puts `-race` and `lint:md` in the gate, aligns CI build flags, and generates help from the flag set.
  Behavior change: a malformed `.env` value now stops startup, and an explicit `PUSHOVER_EXPIRE`/`PUSHOVER_RETRY` is no longer replaced by a default.
- 2026-10-18 — `8bd47eb` added Pushover Glances through the `send_glance` MCP tool and `glance` CLI subcommand.
- 2026-10-17 — `1ea1849`, `0a11331`, `f2aa2ef`, and `43ea5f4` added OAuth connector authentication and nginx deployment configuration.
- 2026-10-17 — `94dcdd3`, `84361f2`, `2da81b3`, `e5a30af`, and `e50ebfe` added cancel, attachments, schemas, expiry caps, and priority zero.
- 2026-10-17 — `8d96809`, `4ee8c6a`, `5d4ffa4`, `7f26dfc`, and `dc3941b` completed notification results, timestamp, TTL, formatting, and URL controls.
- 2026-10-17 — `5956a06` renamed the module and upgraded mcp-go to v1.3.0.
- 2026-10-17 — Scaffold re-initialization finished: Taskfile merged, lint and Markdown configuration refreshed,
  docs filled from the code, and `.gitignore` corrected so gate configuration and rules are versioned.
  Fixed version injection by making `main.version` a variable (`main.go:11-13`).
- 2026-10-17 — Go 1.27 upgrade: directive bumped to `1.27.0`, `go fix` applied until stable, and dependencies tidied.
- 2026-10-17 — Release flow folded into `task release`, `run_release.sh` removed, and the stray root nginx snippet removed.
- 2026-10-17 — The Trading integration handoff was answered in [USING.md](USING.md#reply-to-the-trading-handoff-2026-10-17) and retired.
- 2026-07-25 — Unit tests for auth, CORS, and CLI handlers; `errorlint`; structured JSON logging; tracked dependency checksums;
  CI and release automation; and Makefile replacement by Taskfile.
- 2026-07-11 — HTTP transport can disable mcp-go localhost protection for reverse-proxy deployments.
- 2026-05-05 — `retry` parameter support for emergency messages.

## Notes

- `go.sum` stays tracked.
  The scaffold template ignores it, but this project keeps module checksums in Git so a fresh clone verifies dependencies without a network round-trip.
