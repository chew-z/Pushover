# Pushover

`pushover` sends Pushover notifications from a CLI or exposes the same operations to MCP clients.
The binary provides the default notification command and the `mcp`, `cancel`, and `glance` subcommands.

For implementation and HTTP-authentication design, see [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).
For deployment, environment variables, nginx, systemd, and connector setup, see [docs/OPERATIONS.md](docs/OPERATIONS.md).

## Build and install

```sh
git clone <repository-url>
cd Pushover
task build
```

`task build` writes `bin/pushover` with a version injected from Git.
Use `task build --force` before deployment when Task considers the output current.

| Command | Purpose |
| --- | --- |
| `task build` | Build `bin/pushover`. |
| `task check` | Run `fmt:check`, `vet`, `lint:ci`, and `test:short`. |
| `task test` | Run the full test suite with verbose output. |
| `task test:short` | Run tests without verbose output. |
| `task lint` | Run golangci-lint with automatic fixes. |
| `task lint:ci` | Run golangci-lint without changes. |
| `task fmt` | Format Go files. |
| `task lint:md` | Lint Markdown. |
| `task install` | Copy the binary to `/usr/local/bin`. |
| `task clean` | Remove build artifacts. |

## Configuration

`APP_KEY` and `RECIPIENT_KEY` are required for notification, cancellation, Glance, and MCP server operation.
The default HTTP address is `127.0.0.1:8080`, and the default MCP path is `/mcp`.
The default CORS-origin list is empty, so no origin is allowed unless `PUSHOVER_HTTP_CORS_ORIGINS` is set.

```env
APP_KEY=<pushover-application-key>
RECIPIENT_KEY=<pushover-recipient-or-group-key>
```

The program autoloads `.env` without overwriting existing environment variables.
The complete environment-variable reference, including OAuth configuration, is in [docs/OPERATIONS.md](docs/OPERATIONS.md#configuration).

## CLI

The default command sends one notification.
It accepts `pushover <message> [title]` as a positional fallback.

```sh
bin/pushover -m "Build completed" -t "CI" -p 1 -s siren
bin/pushover cancel -receipt <receipt>
bin/pushover glance -title "Widgets Sold" -count 42 -percent 75
```

### `pushover`

| Flag | Type and default | Meaning |
| --- | --- | --- |
| `-m`, `-message` | string; empty | Required message, unless the first positional argument supplies it. |
| `-t`, `-title` | string; empty | Notification title, or the `DEFAULT_TITLE` value when omitted. |
| `-p`, `-priority` | integer; `0` | `-2` through `2`; when omitted, the configured default is used; an explicit `0` overrides it. |
| `-s`, `-sound` | string; empty | Sound name, or the configured sound when omitted. |
| `-d`, `-device` | string; empty | Target device, or `DEVICE_NAME` when omitted. |
| `-e`, `-expire` | integer; `0` | Emergency retry lifetime in seconds; a zero value uses the configured default; maximum `10800`. |
| `-r`, `-retry` | integer; `0` | Emergency retry interval in seconds; a zero value uses the configured default; minimum `30`. |
| `-attachment` | string; empty | Local PNG, JPEG, GIF, or WebP file; maximum 2.5 MiB. |
| `-attachment-type` | string; empty | Optional expected MIME type; must match detected content when supplied. |
| `-url` | string; empty | Supplementary URL; maximum 512 characters. |
| `-url-title` | string; empty | URL label; maximum 100 characters and requires `-url`. |
| `-html` | boolean; `false` | Render the message as HTML. |
| `-monospace` | boolean; `false` | Render the message in monospace; mutually exclusive with `-html`. |
| `-ttl` | integer; `0` | Delete after this many seconds; zero is unset and other values below 1 are invalid. |
| `-timestamp` | integer; `0` | Display this Unix timestamp; zero is unset and negative values are invalid. |
| `-h`, `-help` | boolean; `false` | Print help. |
| `-version` | boolean; `false` | Print the binary version. |

`PUSHOVER_PRIORITY` defaults to `-1`, `PUSHOVER_SOUND` defaults to `vibrate`, `PUSHOVER_EXPIRE` defaults to 180 seconds, and `PUSHOVER_RETRY` defaults to 60 seconds.
`-html` and `-monospace` are mutually exclusive.
Emergency priority requires appropriate expiry and retry values.

### `pushover cancel`

`cancel` stops retries for an emergency notification.

| Flag | Type and default | Meaning |
| --- | --- | --- |
| `-receipt` | string; empty | Required 30-character alphanumeric emergency receipt. |
| `-h`, `-help` | boolean; `false` | Print help. |

### `pushover glance`

`glance` sends an Apple Watch complication update.
Use it only for Apple Watch complications, wait at least 20 minutes between updates, and send at most 50 each day.
An update can take up to 10 minutes to appear.

| Flag | Type and default | Meaning |
| --- | --- | --- |
| `-title` | string; empty | Title; an explicitly empty value clears it. |
| `-text` | string; empty | Main text; an explicitly empty value clears it. |
| `-subtext` | string; empty | Secondary text; an explicitly empty value clears it. |
| `-count` | integer; `0` | Integer count, including negative values. |
| `-percent` | integer; `0` | Integer percentage from 0 through 100. |
| `-device` | string; empty | Target device. |
| `-h`, `-help` | boolean; `false` | Print help. |

Supply at least one of `-title`, `-text`, `-subtext`, `-count`, or `-percent`.
Only supplied flags are sent, so `-count 0` is distinct from omitting `-count`.

## MCP server

Start an MCP server over STDIO or streamable HTTP:

```sh
bin/pushover mcp
bin/pushover mcp -transport http
```

### `pushover mcp`

| Flag | Type and default | Meaning |
| --- | --- | --- |
| `-transport` | string; `stdio` | Transport: `stdio` or `http`. |
| `-auth-enabled` | boolean; `false` | Require bearer authentication for HTTP. |
| `-generate-token` | boolean; `false` | Print a static bearer token and exit. |
| `-token-username` | string; `admin` | Username embedded in a generated static token. |
| `-token-ttl` | duration; `744h` | Lifetime for a generated static token. |
| `-hash-password` | boolean; `false` | Prompt for and print an OAuth login-password hash. |
| `-h`, `-help` | boolean; `false` | Print help. |

`-generate-token` and `-hash-password` cannot be combined.
`-generate-token` requires `PUSHOVER_AUTH_SECRET_KEY` and supports Go duration syntax such as `24h`.

```sh
PUSHOVER_AUTH_SECRET_KEY=<secret> \
  bin/pushover mcp -generate-token -token-username <user> -token-ttl 24h

bin/pushover mcp -hash-password
```

There is no HTTP token-generation endpoint.

### Authentication modes

- STDIO has no HTTP bearer-authentication layer.
- HTTP is open when authentication is disabled.
- Static bearer mode accepts a token generated with `-generate-token`.
- OAuth mode accepts OAuth tokens and static bearer tokens.

For HTTP routes, token profiles, and challenges, see [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md#authentication-and-actor-propagation).
For production OAuth and client setup, see [docs/OPERATIONS.md](docs/OPERATIONS.md#connect-from-claude).

## MCP tools

### `send_notification`

`send_notification` sends one Pushover notification.

| Parameter | Type and default | Constraints |
| --- | --- | --- |
| `message` | string; required | Maximum 1024 characters. |
| `title` | string; empty | Optional title. |
| `priority` | integer; configured default | `-2` through `2`; priority 2 requires retry and expiry. |
| `sound` | string; configured default | Optional Pushover sound name. |
| `device` | string; configured default | Optional target device. |
| `expire` | integer; configured default | Priority-2 retry lifetime; 1 through 10800 seconds. |
| `retry` | integer; configured default | Priority-2 retry interval; 30 through 86400 seconds. |
| `url` | string; empty | Maximum 512 characters. |
| `url_title` | string; empty | Maximum 100 characters; requires `url`. |
| `html` | boolean; `false` | Mutually exclusive with `monospace`. |
| `monospace` | boolean; `false` | Mutually exclusive with `html`. |
| `ttl` | integer; unset | Minimum 1 second; ignored for priority 2. |
| `timestamp` | integer; unset | Unix timestamp; minimum 1. |
| `attachment_path` | string; empty | STDIO only; mutually exclusive with `attachment_base64`. |
| `attachment_base64` | string; empty | Standard base64; maximum 2.5 MiB decoded or 3495256 encoded bytes. |
| `attachment_type` | string; empty | Required with base64; optional with path; `image/png`, `image/jpeg`, `image/gif`, or `image/webp`. |

An attachment path must be a nonempty regular file within the 2.5 MiB limit.
The detected content type must be one of the allowed image types and must match `attachment_type` when supplied.

### `cancel_emergency`

`cancel_emergency` cancels retries for an emergency notification.

| Parameter | Type and default | Constraints |
| --- | --- | --- |
| `receipt` | string; required | Exactly 30 alphanumeric characters. |

### `send_glance`

`send_glance` sends an Apple Watch complication Glance update.
At least one of `title`, `text`, `subtext`, `count`, or `percent` is required.
An explicitly empty string clears a supplied text field.

| Parameter | Type and default | Constraints |
| --- | --- | --- |
| `title` | string; unset | Maximum 100 characters. |
| `text` | string; unset | Maximum 100 characters. |
| `subtext` | string; unset | Maximum 100 characters. |
| `count` | integer; unset | Negative values are allowed. |
| `percent` | integer; unset | 0 through 100. |
| `device` | string; empty | Optional target device. |

Use Glances only for Apple Watch complications, wait at least 20 minutes between updates, and send at most 50 per day.
Pushover can take up to 10 minutes to display an update.

## MCP client examples

Use STDIO locally:

```json
{
  "mcpServers": {
    "pushover": {
      "command": "/path/to/pushover",
      "args": ["mcp", "-transport", "stdio"],
      "env": {
        "APP_KEY": "<pushover-application-key>",
        "RECIPIENT_KEY": "<pushover-recipient-or-group-key>"
      }
    }
  }
}
```

For remote Claude connectors, OAuth login, nginx paths, and a static-bearer alternative, use [docs/OPERATIONS.md](docs/OPERATIONS.md#connect-from-claude).

## Pushover keys

1. Create an account at [pushover.net](https://pushover.net).
2. Register an application to obtain `APP_KEY`.
3. Find `RECIPIENT_KEY` in the user dashboard.
4. Register devices when targeting a specific device name.
