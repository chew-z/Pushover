---
title: Pushover architecture
date: 2026-10-18
status: active
tags: [architecture]
---

# Pushover architecture

> How the system is built, verified against the current code.
> Current health lives in [STATUS.md](STATUS.md), and unresolved work lives in [OPEN_ITEMS.md](OPEN_ITEMS.md).

## System shape

`pushover` is one Go binary with a notification CLI plus `mcp`, `cancel`, and `glance` subcommands (`main.go:15-58`).
The MCP subcommand serves the same three Pushover operations over STDIO or streamable HTTP (`mcp_server.go:15-165`).
When OAuth is enabled, its client, authorization-request, authorization-code, refresh-token, and rate-limit state is in an in-memory `Store` (`internal/oauth/oauth_store.go:133-164`).

## Components

The repository root is one `main` package, and `internal/oauth` is its only subpackage (`main.go:1`, `internal/oauth/config.go:1-2`).

| File | Role |
| --- | --- |
| `main.go` | Loads `.env` and dispatches commands (`main.go:3-9`, `main.go:15-58`). |
| `cli.go` | Parses the notification CLI and sends it (`cli.go:111-217`). |
| `cancel_cli.go` | Validates receipts and runs `cancel` (`cancel_cli.go:11-70`). |
| `glance_cli.go` | Parses and sends `glance` updates (`glance_cli.go:30-85`). |
| `config.go` | Loads and validates application, MCP, and OAuth settings (`config.go:30-306`). |
| `attachment.go` | Validates path or base64 image attachments (`attachment.go:16-105`). |
| `pushover.go` | Defines the client seam, messages, and response formatting (`pushover.go:18-190`). |
| `auth.go` | Implements JWT profiles, guard, and actor context (`auth.go:15-201`). |
| `mcp_cli.go` | Parses MCP flags and starts the selected transport (`mcp_cli.go:62-116`). |
| `mcp_server.go` | Registers, logs, and handles MCP tools (`mcp_server.go:15-324`). |
| `http_server.go` | Builds and runs the HTTP server, routes, and CORS (`http_server.go:20-208`). |
| `glance.go` | Validates Glance values and formats results (`glance.go:10-120`). |
| `logging.go` | Directs JSON `slog` to stderr (`logging.go:23-29`). |
| `internal/oauth/config.go` | Defines and validates OAuth configuration (`internal/oauth/config.go:11-76`). |
| `internal/oauth/mount.go` | Mounts discovery and OAuth endpoint handlers (`internal/oauth/mount.go:8-31`). |
| `internal/oauth/oauth_authorize.go` | Runs authorization, login, and code issuance (`internal/oauth/oauth_authorize.go:17-298`). |
| `internal/oauth/oauth_clients.go` | Resolves clients and implements registration (`internal/oauth/oauth_clients.go:74-478`). |
| `internal/oauth/oauth_metadata.go` | Builds and serves OAuth discovery metadata (`internal/oauth/oauth_metadata.go:10-179`). |
| `internal/oauth/oauth_store.go` | Holds clients, grants, tokens, and limiters (`internal/oauth/oauth_store.go:39-478`). |
| `internal/oauth/oauth_token.go` | Exchanges grants for access and refresh tokens (`internal/oauth/oauth_token.go:11-259`). |
| `internal/oauth/password.go` | Hashes and verifies login passwords (`internal/oauth/password.go:14-100`). |

`internal/oauth` was copied from the Alerts project in commit `0a11331` and is intentionally maintained as a copy rather than a shared dependency.
Its exported API is retained.
It includes `Signer`, `Config`, and `Mount` (`internal/oauth/config.go:11-25`, `internal/oauth/mount.go:8-31`).
It also includes discovery and password helpers (`internal/oauth/oauth_metadata.go:97-140`, `internal/oauth/password.go:29-31`, `internal/oauth/password.go:81-84`).

## Request flow

### STDIO MCP

`main` dispatches `pushover mcp` to `runMCPCommand`, which parses flags and calls `runMCPServerWithTransport` (`main.go:30-50`, `mcp_cli.go:62-116`).
For `-transport stdio`, it validates configuration, registers tools, logs to stderr, and calls `server.ServeStdio` (`mcp_server.go:139-160`, `logging.go:23-29`).
STDIO does not traverse `protectMCP`, so HTTP bearer authentication does not apply (`mcp_server.go:152-160`, `http_server.go:56-60`).

### HTTP MCP

For `-transport http`, the same setup creates `HTTPServerManager` with the endpoint path and `copyActor` context hook (`mcp_server.go:157-160`, `http_server.go:34-46`).
The mux registers that endpoint behind `protectMCP`, registers open health and capability handlers, and mounts OAuth handlers only when OAuth is enabled (`http_server.go:47-70`).
The HTTP server applies the configured timeout to reads and writes and twice that value to idle connections (`http_server.go:79`),
and its CORS wrapper exposes `WWW-Authenticate` to browsers (`http_server.go:73-100`, `http_server.go:180-198`).
The nginx configuration forwards `/pushover/` and preserves the OAuth well-known paths (`deploy/nginx-pushover.conf:19-53`).

## HTTP routes and modes

Open means no bearer check in this service.
Static means the legacy static JWT profile is accepted.
OAuth means either the static profile or the configured OAuth profile is accepted.

| Route | Open mode | Static mode | OAuth mode | Notes |
| --- | --- | --- | --- | --- |
| `PUSHOVER_HTTP_PATH` (normally `/mcp`) | Open | Static | OAuth | Guarded route (`config.go:96-97`, `http_server.go:56-60`). |
| `/health` | Open | Open | Open | Health document (`http_server.go:60-62`, `http_server.go:111-120`). |
| `/capabilities` | Open | Open | Open | Tool schemas and auth state (`http_server.go:60-62`, `http_server.go:122-177`). |
| Protected-resource metadata, bare and resource-path forms | Absent | Absent | Open | Both forms mount (`internal/oauth/mount.go:15-21`). |
| Authorization-server metadata, bare and issuer-path forms | Absent | Absent | Open | Both forms mount (`internal/oauth/mount.go:22-26`). |
| `/authorize`, `/login`, `/token`, `/register` | Absent | Absent | Open | OAuth endpoints (`internal/oauth/mount.go:27-30`). |

There is no HTTP `/generate-token` route.
`pushover mcp -generate-token` generates a static token locally and exits (`mcp_cli.go:73-99`).

## Authentication and actor propagation

`MCPConfig` enables authentication from `-auth-enabled` or `PUSHOVER_AUTH_ENABLED`.
OAuth forces authentication on when `PUSHOVER_MCP_OAUTH` is true (`config.go:157-241`).
Authentication uses the configured secret to parse an HS256 bearer JWT with a required expiration and non-empty username (`auth.go:73-99`).

| Profile | Required issuer | Required audience | Required scope |
| --- | --- | --- | --- |
| Static | `pushover-mcp` | `pushover-mcp-user` | None |
| OAuth | Configured OAuth issuer | Configured OAuth resource | `pushover` by default |

The profile rows above are selected by `newAuthGuard` (`auth.go:113-119`, `internal/oauth/config.go:27-41`).
`protectMCP` accepts a token only when issuer and audience match a configured pair, and requires exact scope for the OAuth pair (`auth.go:121-150`).

Validation is stateless: `protectMCP` never consults the store, so an access token or a static token cannot be revoked before it expires.
Rotating `PUSHOVER_AUTH_SECRET_KEY` and restarting is the only kill switch, and it invalidates every token at once.
The static token lifetime defaults to 744 hours (`mcp_cli.go:70`); keep it short or accept that a leaked token is valid until then.
Missing, malformed, expired, or unmatched tokens receive HTTP 401 with `WWW-Authenticate: Bearer` and `error="invalid_token"` (`auth.go:123-150`, `auth.go:163-176`).
When OAuth is configured, that challenge also names the RFC 9728 `resource_metadata` URL and required scope (`auth.go:163-169`, `internal/oauth/oauth_metadata.go:97-140`).
A token matching the OAuth issuer and resource but lacking the required scope receives HTTP 403 and `error="insufficient_scope"` (`auth.go:136-143`).

On acceptance, `protectMCP` puts the token username in the HTTP request context (`auth.go:136-146`).
`copyActor` marks the tool context as HTTP and copies that actor because mcp-go supplies the original request to its context hook (`auth.go:188-201`, `http_server.go:34-46`).
The logging wrapper reads `actorFrom` and tool result formatters include it (`mcp_server.go:167-191`, `mcp_server.go:270-324`, `pushover.go:159-177`, `glance.go:110-120`).

## MCP tools

`send_notification` requires `message` and accepts notification controls, presentation, timing, targeting, and an image attachment (`mcp_server.go:24-91`, `mcp_server.go:194-271`).
`cancel_emergency` requires a 30-character alphanumeric receipt and cancels emergency retries (`mcp_server.go:94-104`, `mcp_server.go:274-293`, `cancel_cli.go:11-23`).
`send_glance` accepts one or more Glance fields plus an optional device and sends a complication update (`mcp_server.go:107-134`, `mcp_server.go:296-314`, `glance.go:10-59`).
The full user-facing argument reference belongs in README and USING documentation.

## Pushover client seam and attachments

`PushoverClient` is the narrow interface for sending messages, canceling emergencies, and sending Glances (`pushover.go:18-23`).
`newPushoverClient` defaults to `NewPushoverClient` and is replaceable by tests, so CLI and MCP code share the same Pushover boundary (`pushover.go:45-67`).

Attachments accept only PNG, JPEG, GIF, or WebP content (`attachment.go:16-22`, `attachment.go:97-105`).
The Pushover library's `MessageMaxAttachmentByte` limit is 2.5 MiB in the MCP schema (`attachment.go:39-94`, `mcp_server.go:36-45`).
Decoded base64 and local file size are checked before sending (`attachment.go:39-94`).
`attachment_path` is allowed only outside HTTP, while HTTP callers must send `attachment_base64` with an attachment type (`attachment.go:41-44`, `attachment.go:76-82`).
The HTTP guard separately caps the entire MCP request body at 4 MiB (`auth.go:15-20`, `auth.go:123-126`).

## Configuration boundaries

`.env` is autoloaded during process initialization (`main.go:3-9`).
`LoadConfig` reads application and notification defaults (`config.go:111-142`).
Every `PUSHOVER_*` value either parses or stops startup, and `Config.Validate` range-checks it on the CLI paths too (`config.go:44-52`).
`NewMCPConfig` validates HTTP, authentication, OAuth, and Pushover values before transport starts (`config.go:145-278`).
OAuth requires an enabled authentication secret of at least 32 bytes (`config.go:265-275`).
OAuth validates issuer, resource, login, password hash, signer, and refresh lifetime (`internal/oauth/config.go:27-63`).
