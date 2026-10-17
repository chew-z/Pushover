# Handoff: what the Trading workplace needs from the Pushover MCP server

From the Trading project (`~/Documents/Trading`, rule `.claude/rules/pushover.md`), 2026-10-17.
Claude sends the user complete decision messages while they are away from the screen;
the daemon `alertsd` sends the level alerts itself. These are the parameters the `send_notification`
tool lacks, in priority order, plus one new tool. The library underneath, `gregdel/pushover` v1.4.0,
already carries everything in 1 to 5; only the MCP tool schema and the handler need to grow.
The CLI should take the same flags, because a shell runner without MCP will use `~/.local/bin/pushover`.

## 1. `url` and `url_title` (first)

A tappable link under the message: a chart, a source article, the Claude session.
Limits from the API: `url` 512 characters, `url_title` 100; `url_title` without `url` is an error.
Map to `Message.URL` and `Message.URLTitle`.

## 2. `attachment`

One image per message, up to 5,242,880 bytes, for the chart PNG the `alerts` connector renders.
Accept either a local file path (the common case from a shell runner or a local MCP) or `attachment_base64`
plus `attachment_type` (MIME, for example `image/png`) for an HTTP MCP client that cannot post multipart.
Map to `Message.AddAttachment(io.Reader)`; the library enforces the size after reading, so check the size
before reading to fail fast. Reject anything that is not an image.

## 3. `html` and `monospace`

Booleans, mutually exclusive (the API refuses both). `html` lets the fact and the ask be bold; `monospace`
aligns a few numbers. Map to `Message.HTML` and `Message.Monospace`.

## 4. `ttl`

Seconds after which the message deletes itself from the devices; positive integer; ignored by Pushover for
priority 2. For Watch-day heads-ups that should not pile up. Map to `Message.TTL` (a duration).

## 5. `timestamp`

Unix time to show instead of the receipt time: the moment the level fired. Map to `Message.Timestamp`.

## 6. Return the quota and the receipt

The API response carries `X-Limit-App-Remaining` and, for priority 2, a `receipt`. Return both in the tool
result text so the caller can see the month's remaining messages and can cancel an emergency
(`CancelEmergencyNotification`) if the situation resolves. A `cancel_emergency` tool taking the receipt
would complete that; low priority.

## 7. A `send_glance` tool

Pushover Glances put standing state on the Apple Watch face without a notification: the attention level,
the open position and its P&L, the distance to the next level, the loss line used.
Verified against `pushover.net/api/glances` on 2026-10-17:

- `POST https://api.pushover.net/1/glances.json` with `token`, `user`, optional `device`, and at least one
  data field: `title` (100 characters), `text` (100), `subtext` (100), `count` (integer, may be negative),
  `percent` (integer 0 to 100). An empty string clears a field. Success is `status: 1` with a `request` id;
  a 400 carries `status: 0` and an `errors` array.
- Pushover does not throttle; the caller must. Apple Watch wants at least 20 minutes between updates and
  watchOS processes at most 50 a day, then nothing until the overnight reset. Updates can take up to
  10 minutes to appear. The tool should refuse, or at least warn, when called under 20 minutes after the
  last call, and should count the day's calls in its result.
- Apple Watch only; the widget has to be registered on the device (dashboard, "Your Devices").

`gregdel/pushover` does not wrap Glances, so this is a direct POST with the same token and user key.

## Fixes seen in passing

- `expire` is capped at 86400 in the tool schema; the API caps it at 10800 (3 hours) and at 50 retries.
  Lower the schema maximum to 10800 so the API does not refuse.
- `priority` 2 without `retry` and `expire` is refused by the library; the tool should say so in its
  description rather than fail at send time.

## Not needed

`device` lists, groups, `callback`, `tags`, encryption. One user, one phone, one watch.

## Validation the Trading side will do

One message at each priority from -1 to 1 with a `url`, one with a chart attachment, one glance,
checked on the phone and the watch; then the rule's table of priorities and sounds is confirmed in use.
