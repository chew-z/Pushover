---
title: Using Pushover from other projects
date: 2026-10-18
status: active
tags: [usage, integration, mcp, cli]
---

# Using Pushover from other projects

> A contract for projects that send Pushover notifications through the shared MCP connector or CLI.

This document is self-contained so it can be copied into a project skill or rule.

## Reply to the Trading handoff (2026-10-17)

All requested notification features are available to MCP clients and the CLI.

| Request | Status | What was built |
| --- | --- | --- |
| 1. `url` and `url_title` | Complete | URL limit: 512 characters. Title limit: 100 characters. `url_title` requires `url`. |
| 2. Attachment | Complete: limit changed | One PNG, JPEG, GIF, or WebP image. Limit: 2,621,440 bytes (2.5 MiB), not 5 MB. HTTP needs base64 and MIME type. |
| 3. `html` and `monospace` | Complete | Both boolean options are available and mutually exclusive. |
| 4. `ttl` | Complete | A positive number of seconds deletes a transient notification after that time. Pushover ignores it for priority 2. |
| 5. `timestamp` | Complete | A positive Unix timestamp controls the displayed event time. |
| 6. Quota and receipt | Complete | Notification results include a receipt when Pushover returns one and quota information when response headers provide it. |
| 6b. `cancel_emergency` | Complete | Cancel a priority-2 notification by its 30-character receipt. |
| 7. `send_glance` | Complete; unthrottled | Watch updates. Caller keeps a 20-minute minimum and 50-per-day maximum. Library has `SendGlanceUpdate`. |
| Expiry cap | Complete | Emergency expiry is capped at 10,800 seconds (three hours). |
| Priority-2 description | Complete | The contract states that priority 2 requires both `retry` and `expire`. |
| Explicit priority 0 | Complete | An explicit `priority: 0` is honored instead of being replaced by the server default. |

## How to connect

Choose the transport based on where the caller runs and whether it needs local file access.

| Caller | Use | Why |
| --- | --- | --- |
| Interactive Claude in claude.ai, Claude Desktop, or mobile | Custom connector | OAuth login is handled in the client. |
| Claude Code in a repository | HTTP MCP with OAuth or a static bearer token | OAuth avoids token handling. A static header suits existing configuration. |
| Unattended shell runner or cron job | CLI | It needs only Pushover credentials in its environment. |
| A caller attaching a local image file | CLI or STDIO MCP | HTTP connectors cannot use `attachment_path`. |

### Claude.ai, Claude Desktop, and mobile

Create a custom connector with this URL:

```text
https://karma.rrj.pl/pushover/mcp
```

Leave Client ID and Client Secret empty.
Log in with the user name and password supplied by the operator.
The server stores OAuth state in memory, so a server restart can require another login.

### Claude Code

Add this to the project `.mcp.json` without headers, then run `/mcp` to complete OAuth login:

```json
{
  "mcpServers": {
    "pushover": {
      "type": "http",
      "url": "https://karma.rrj.pl/pushover/mcp"
    }
  }
}
```

Alternatively, ask the operator for a static bearer token and configure it as a header:

```json
{
  "mcpServers": {
    "pushover": {
      "type": "http",
      "url": "https://karma.rrj.pl/pushover/mcp",
      "headers": {
        "Authorization": "Bearer <operator-issued-token>"
      }
    }
  }
}
```

If Claude Code reports “Conflicting scopes”, remove its stale local registration and run `/mcp` again:

```sh
claude mcp remove pushover -s local
```

### CLI

Use `~/.local/bin/pushover` for shell runners and cron jobs.
Set `APP_KEY` and `RECIPIENT_KEY` in the environment, or place them in `.env` in the command's working directory.
Never place real keys or bearer tokens in a repository, skill, prompt, or command history.

## Tool reference

“Server default” means the configured value at the running service.
Unless an operator overrides it, the defaults are priority `-1`, sound `vibrate`, emergency expiry `180` seconds, and emergency retry `60` seconds.

### `send_notification`

Send one Pushover notification.

| Parameter | Type and default | Limits and rules |
| --- | --- | --- |
| `message` | string; required | At most 1,024 characters. |
| `title` | string; the configured default title when empty | At most 250 characters. |
| `priority` | integer; server default when omitted | `-2` through `2`. Priority 2 requires `retry` and `expire`. |
| `sound` | string; server default when empty | A Pushover sound name. |
| `device` | string; configured device when empty | Each comma-separated device name must be 1–25 ASCII letters, digits, `_`, or `-`. |
| `expire` | integer; server default when omitted | Emergency retry lifetime from 1 through 10,800 seconds. Used for priority 2. |
| `retry` | integer; server default when omitted | Emergency retry interval from 30 through 86,400 seconds. Used for priority 2. |
| `url` | string; empty | At most 512 characters. |
| `url_title` | string; empty | At most 100 characters and requires `url`. |
| `html` | boolean; `false` | Mutually exclusive with `monospace`. |
| `monospace` | boolean; `false` | Mutually exclusive with `html`. |
| `ttl` | integer; omitted | At least 1 second. Pushover ignores it for priority 2. |
| `timestamp` | integer; omitted | A positive Unix timestamp in seconds. |
| `attachment_path` | string; empty | Nonempty regular image file. STDIO and CLI only. Mutually exclusive with `attachment_base64`. |
| `attachment_base64` | standard-base64 string; empty | Nonempty image. Requires `attachment_type`. Mutually exclusive with `attachment_path`. |
| `attachment_type` | string; empty | Required with base64. Optional with a path. Must match the detected image type. |

Priority values are `-2` lowest, `-1` low, `0` normal, `1` high, and `2` emergency.
An explicit `0` is normal priority, even when a different server default is configured.

An attachment must be PNG, JPEG, GIF, or WebP and no more than 2,621,440 bytes (2.5 MiB).
Base64 input is limited to 3,495,256 encoded bytes.
`attachment_type` must be `image/png`, `image/jpeg`, `image/gif`, or `image/webp`.
`attachment_path` is refused over HTTP, so connectors must send `attachment_base64` and `attachment_type`.

The success result is assembled from the parts Pushover returns.
Its complete form is:

```text
Notification sent. Receipt: <receipt>. Quota: <remaining>/<total> remaining, resets <RFC3339-UTC> (sent by <username>)
```

`Receipt: <receipt>` is omitted when Pushover does not return a receipt.
`Quota: ...` is omitted when the response has no quota headers, and `, resets <RFC3339-UTC>` is omitted when no reset time is present.
`(sent by <username>)` appears only for an authenticated HTTP call and names the authenticated user.
With none of those optional parts, the result is exactly `Notification sent.`.

### `cancel_emergency`

Stop retries for a priority-2 notification.

| Parameter | Type and default | Limits and rules |
| --- | --- | --- |
| `receipt` | string; required | Exactly 30 alphanumeric characters. |

The success result is `Emergency notification <receipt> canceled`.
Authenticated HTTP calls append `(sent by <username>)` after a space.

### `send_glance`

Send an Apple Watch complication Glance update.
At least one of `title`, `text`, `subtext`, `count`, or `percent` is required.
An explicitly supplied empty string clears the corresponding text field.

| Parameter | Type and default | Limits and rules |
| --- | --- | --- |
| `title` | string; omitted | At most 100 characters. |
| `text` | string; omitted | At most 100 characters. |
| `subtext` | string; omitted | At most 100 characters. |
| `count` | number; omitted | Must be an integer. Negative values are allowed. |
| `percent` | number; omitted | Must be an integer from 0 through 100. |
| `device` | string; empty | Optional target device. It does not inherit the configured notification device. |

The success result is `Glance sent. Quota: <remaining>/<total> remaining`.
It adds `, resets <RFC3339-UTC>` when a reset time is available.
When quota headers are unavailable, it is exactly `Glance sent. Quota: unknown`.
Authenticated HTTP calls append `(sent by <username>)` after a space.

The service does not throttle or count Glance requests.
Keep at least 20 minutes between updates, send at most 50 per day, and allow up to 10 minutes for an update to appear.

## CLI reference

The default command, `cancel`, and `glance` require `APP_KEY` and `RECIPIENT_KEY`.
The CLI prints the same successful result text as its matching MCP tool, without an authenticated-user suffix.

### `pushover`

Send a notification:

```sh
~/.local/bin/pushover \
  -message "Build completed" \
  -title "CI" \
  -priority 1 \
  -sound siren
```

`pushover <message> [title]` is a positional alternative.

| Flag | Default | Meaning |
| --- | --- | --- |
| `-m`, `-message` | empty | Message; required unless the first positional argument supplies it. |
| `-t`, `-title` | empty | Title; an empty value uses the configured default title when one exists. |
| `-p`, `-priority` | `0` | Priority; when omitted, the configured default is used, while an explicit `0` is honored. |
| `-s`, `-sound` | empty | Sound; an empty value uses the configured default. |
| `-d`, `-device` | empty | Target device; an empty value uses the configured default. |
| `-e`, `-expire` | `0` | Emergency expiry; zero uses the configured default; maximum 10,800. |
| `-r`, `-retry` | `0` | Emergency retry interval; zero uses the configured default; minimum 30. |
| `-attachment` | empty | Local image attachment. |
| `-attachment-type` | empty | Expected attachment MIME type. |
| `-url` | empty | Supplementary URL. |
| `-url-title` | empty | URL title. |
| `-html` | `false` | Render HTML. |
| `-monospace` | `false` | Render monospace text. |
| `-ttl` | `0` | Notification lifetime; zero is unset. |
| `-timestamp` | `0` | Displayed Unix event time; zero is unset. |
| `-version` | `false` | Print the binary version. |
| `-h`, `-help` | `false` | Print help. |

The notification constraints in the `send_notification` reference apply to the corresponding CLI flags.

### `pushover cancel`

Cancel retries with the receipt from a priority-2 notification:

```sh
~/.local/bin/pushover cancel -receipt <30-character-receipt>
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `-receipt` | empty | Required 30-character alphanumeric emergency receipt. |
| `-h`, `-help` | `false` | Print help. |

### `pushover glance`

Send a watch update:

```sh
~/.local/bin/pushover glance -title "Open positions" -count 3 -percent 75
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `-title` | empty | Glance title; only sent when the flag is supplied. |
| `-text` | empty | Main text; only sent when the flag is supplied. |
| `-subtext` | empty | Secondary text; only sent when the flag is supplied. |
| `-count` | `0` | Integer count; only sent when the flag is supplied. |
| `-percent` | `0` | Integer percentage; only sent when the flag is supplied. |
| `-device` | empty | Target device. |
| `-h`, `-help` | `false` | Print help. |

Use `-text ""` to clear text and `-count 0` to send an explicit zero.
The Glance constraints in the `send_glance` reference apply to these flags.

### `pushover mcp`

`pushover mcp` runs the server and is normally operated by the shared service, not by a consuming project.
Its relevant flags are `-transport stdio` or `-transport http`, `-auth-enabled`, `-generate-token`, `-token-username`, `-token-ttl`, and `-hash-password`.
Only an operator should issue static tokens or create password hashes.

## Recipes

### Decision with a link and bold ask

```json
{
  "message": "<b>Decision needed:</b> approve the hedge before London opens.",
  "title": "Trading",
  "url": "https://example.invalid/chart",
  "url_title": "Open chart",
  "html": true,
  "priority": 1
}
```

### Chart attachment

Over the HTTP connector, base64-encode the chart and send the content with its MIME type:

```json
{
  "message": "Chart attached.",
  "attachment_base64": "<standard-base64-png>",
  "attachment_type": "image/png"
}
```

From a shell runner, pass the local path instead:

```sh
~/.local/bin/pushover -message "Chart attached." -attachment ./chart.png
```

### Emergency alert and cancellation

Send priority 2 with both retry controls:

```json
{
  "message": "<b>Risk limit breached.</b>",
  "title": "Emergency",
  "html": true,
  "priority": 2,
  "retry": 60,
  "expire": 1800
}
```

Read `Receipt: <receipt>` from the result, preserve that receipt, and cancel once the condition clears:

```json
{
  "receipt": "<receipt-from-notification-result>"
}
```

### Watch Glance with caller-side throttling

Before sending, record the previous Glance time and the current day's count in the caller's own state.
Skip the update if the previous one is less than 20 minutes old or the daily count is already 50.

```json
{
  "title": "Open positions",
  "text": "3 positions",
  "count": 3,
  "percent": 75
}
```

## Rules of thumb for skills

- Use priority `-1` for background information, `0` for normal decisions, `1` for urgent attention, and `2` only for emergencies that need acknowledgement and cancellation.
- Choose a sound appropriate to urgency, and avoid repeated high-priority alerts.
- Read the `Quota:` part of every result and reduce nonessential notifications when the remaining quota is low.
- Never put secrets, application keys, recipient keys, passwords, or bearer tokens in a notification, prompt, source file, or documentation.
- An explicit priority `0` is normal priority and is honored.
- Use `timestamp` only when the event time differs meaningfully from the send time.
- Use `ttl` for transient heads-ups, knowing that Pushover ignores it for priority 2.
- Choose either `html` or `monospace`, never both.

## Errors and what they mean

| Symptom | Meaning and action |
| --- | --- |
| HTTP 401 | Not logged in, or bearer token expired or invalid. Run `/mcp`, or obtain a new token from the operator. |
| HTTP 403 with `insufficient_scope` | The token is valid but lacks the required scope. Re-run OAuth login or ask the operator to correct token configuration. |
| `Message parameter is required` | `send_notification` needs a nonempty `message`. |
| `Invalid notification: html and monospace are mutually exclusive` | Choose one rendering mode. |
| `Invalid notification: url_title requires url` | Supply `url` whenever supplying `url_title`. |
| `Invalid notification: expire must be at most 10800 seconds` | Reduce emergency expiry to three hours or less. |
| `Invalid attachment: attachment_path is not allowed over HTTP; use attachment_base64` | Use base64 with a MIME type, or a local CLI or STDIO call. |
| `Invalid attachment: attachment_type is required with attachment_base64` | Supply an allowed MIME type that matches the decoded image. |
| `Invalid Glance: at least one of title, text, subtext, count, or percent must be provided` | Supply a Glance data field. |
| `Invalid Glance: percent must be between 0 and 100` | Use an integer percentage from 0 through 100. |
| `Invalid receipt: receipt must be 30 alphanumeric characters` | Use the receipt returned by the emergency notification result without modification. |
| `Failed to send notification: <error>` | Pushover or the network rejected the notification. The Pushover error follows. |
| `Failed to send Glance: <error>` | Pushover or the network rejected the Glance update. The Pushover error follows. |
| `Failed to cancel emergency notification: <error>` | Pushover or the network rejected the cancellation. The Pushover error follows. |
| `Failed to create Pushover client: <error>` | The server lacks `APP_KEY` or `RECIPIENT_KEY`. Tell the operator. |

The CLI reports the same failures in lowercase on stderr and exits nonzero, such as `failed to send message: ...`,
`send Glance update: ...`, and `cancel emergency notification: ...`.
Pushover errors can report an invalid user, application key, or device, or an exhausted quota.
