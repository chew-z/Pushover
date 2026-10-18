---
title: Pushover architecture
date: 2026-10-17
status: active
tags: [architecture]
---

# Pushover architecture

> How the system is built, as verified against the code.
> Current health lives in [STATUS.md](STATUS.md); unresolved work in [OPEN_ITEMS.md](OPEN_ITEMS.md).

## What the system does

One Go binary, `pushover`, with two modes.
The default mode sends a single Pushover notification and exits — the CLI used by shell runners.
The `mcp` subcommand runs a Model Context Protocol server over STDIO or streamable HTTP,
exposing `send_notification`, `cancel_emergency`, and `send_glance` to an MCP client (`main.go:28`, `mcp_server.go:13`).
There is no database, no queue and no persistent state; every path ends in a call to the Pushover API.

## Components

All code is one flat `package main` at the repository root; files split by concern.

| File | Owns |
| --- | --- |
| `main.go` | Entry point and `mcp` subcommand routing (`main.go:14`, `main.go:28`) |
| `cli.go` | CLI flag parsing, help text, `Run` orchestration (`cli.go:72`, `cli.go:131`) |
| `config.go` | Environment configuration, defaults, validation (`config.go:74`, `config.go:111`, `config.go:167`) |
| `pushover.go` | `PushoverClient` interface, message construction, send (`pushover.go:11`, `pushover.go:47`) |
| `mcp_cli.go` | `mcp` flags, token generation, transport selection (`mcp_cli.go:53`) |
| `mcp_server.go` | MCP server setup, notification, emergency-cancellation, and Glance tool schemas, auth wrapper (`mcp_server.go:13`, `mcp_server.go:123`) |
| `http_server.go` | Streamable HTTP transport, `/health`, `/capabilities`, CORS (`http_server.go:33`) |
| `auth.go` | JWT issue and validation, auth context helpers (`auth.go:42`, `auth.go:102`) |
| `logging.go` | JSON `slog` setup on stderr (`logging.go:25`) |
| `main_test.go` | Unit and end-to-end tests, including in-process MCP calls (`main_test.go:484`) |

## Data flow

CLI: `main` → `hasSubcommand` → `Run` (`main.go:14`, `main.go:43`) → `LoadConfig` (`cli.go:133`)
→ `ParseArgs` (`cli.go:136`) → `NewPushoverClient` (`cli.go:149`) → `CreateMessage` (`cli.go:159`)
→ `SendMessage` (`cli.go:162`).
Flags override environment values, which override built-in defaults (`pushover.go:55`).
Positional arguments remain a fallback: `pushover <message> [title]` (`cli.go:115`).

MCP: `main` → `parseMCPArgs` (`main.go:34`) → `runMCPServerWithTransport` (`mcp_server.go:62`)
→ `setupMCPServer` registers `send_notification` with schema validation (`mcp_server.go:13`)
→ either `server.ServeStdio` (`mcp_server.go:77`) or `HTTPServerManager.Start` (`mcp_server.go:81`, `http_server.go:33`).
Each tool call passes through its logging wrapper, which records the authenticated actor when present; the HTTP guard enforces
JWT only when the request arrived over HTTP. `handleSendNotification` builds and sends notifications, while the other tools
cancel emergency retries or send Glance updates. The notification result carries the response text and, for emergency priority,
the receipt.

The HTTP transport mounts the MCP handler at `/` (`http_server.go:66`) with the configured endpoint path handled by mcp-go
(`config.go:62`), plus `/health` (`http_server.go:69`) and `/capabilities`, which derives advertised tool metadata from registered tools (`http_server.go:123`).
JWT is validated by `AuthMiddleware.HTTPContextFunc` (`auth.go:50`) and stored in the request context.
The handler for `/generate-token` exists (`http_server.go:189`) but is not registered on the mux.

## Data model

No database, no files, no queues.
The shapes that matter are `Config` and `MCPConfig` (`config.go:15`, `config.go:27`),
`CLIArgs` (`cli.go:13`), the Pushover message built in `CreateMessage` (`pushover.go:47`),
and JWT `Claims` (`auth.go:21`).
Message fields set by the project: priority, sound, expire, retry, device, title, url, url_title, html/monospace and ttl; timestamp is set only when supplied (`pushover.go:69`).

## Configuration

Configuration is environment-first; `.env` is loaded at startup by `godotenv/autoload` (`main.go:8`).
Flags override environment values where both exist.

| Variable | Meaning | Default |
| --- | --- | --- |
| `APP_KEY` | Pushover application token | required (`pushover.go:31`) |
| `RECIPIENT_KEY` | Pushover user or group key | required (`pushover.go:35`) |
| `DEVICE_NAME` | Target device | empty |
| `DEFAULT_TITLE` | Title when none is given | empty |
| `PUSHOVER_PRIORITY` | Default priority, -2 to 2 | `-1` (`config.go:76`) |
| `PUSHOVER_SOUND` | Default sound | `vibrate` (`config.go:83`) |
| `PUSHOVER_EXPIRE` | Emergency expire seconds | `180` (`config.go:87`) |
| `PUSHOVER_RETRY` | Emergency retry seconds, minimum 30 | `60` (`config.go:93`) |
| `PUSHOVER_HTTP_ADDRESS` | HTTP bind address | `127.0.0.1:8080` (`config.go:61`) |
| `PUSHOVER_HTTP_PATH` | MCP endpoint path | `/mcp` (`config.go:62`) |
| `PUSHOVER_HTTP_STATELESS` | Stateless sessions | `false` |
| `PUSHOVER_HTTP_HEARTBEAT` | Heartbeat interval | `30s` |
| `PUSHOVER_HTTP_TIMEOUT` | Read and write timeout | `30s` |
| `PUSHOVER_HTTP_CORS_ENABLED` | CORS middleware | `true` |
| `PUSHOVER_HTTP_CORS_ORIGINS` | Allowed origins, comma-separated | empty |
| `PUSHOVER_HTTP_DISABLE_LOCALHOST_PROTECTION` | mcp-go DNS-rebinding guard | `true` (`config.go:67`) |
| `PUSHOVER_AUTH_ENABLED` | Require JWT on HTTP | `false` |
| `PUSHOVER_AUTH_SECRET_KEY` | HS256 signing secret | required when auth is on (`config.go:191`) |
| `PUSHOVER_LOG_LEVEL` | `debug`, `info`, `warn`, `error` | `info` (`logging.go:26`) |

Environment parsing helpers tolerate bad values by falling back to defaults (`config.go:215`).
