# `hub` — in-process pub/sub for repository events

This package provides a **small, in-memory event hub** used to push **repository-scoped** notifications from HTTP handlers / services to **long-lived SSE connections** (the public portal’s `/v1/portal/:org/:repo/events` stream).

## Role in the app

- **Publishers** (e.g. `FeaturesService`) call `Publish(repoID, event)` when something about that repository’s features changes.
- **Subscribers** (the portal `Subscribe` handler) call `Subscribe(repoID)` to receive a channel of `EventType` values, then map those to SSE frames for browsers.

A **single shared `Hub`** is created in `handlers.NewHandlerCollection` and injected into both the features service and the portal handlers so publishes and SSE subscriptions see the same broker.

## API

| Method | Purpose |
|--------|---------|
| `Subscribe(repoID int64) chan EventType` | Register interest in one repository; returns a receive channel. |
| `Unsubscribe(repoID int64, ch chan EventType)` | Remove a subscription (call when the SSE request ends). |
| `Publish(repoID int64, event EventType)` | Notify all subscribers for that `repoID`. |

`EventType` is a string alias; constants such as `EventFeatureCreated` and `EventFeatureUpdated` live in `hub.go`.

## Semantics

- **Process-local only** — not shared across multiple server instances or restarts.
- **Best-effort delivery** — each subscriber gets a **buffered channel (capacity 1)**. If a client is slow and the buffer is full, `Publish` **drops** that notification for that subscriber (`select` with `default`) so publishers never block.
- **Not a message queue** — no persistence, ordering guarantees beyond “usually FIFO per channel”, or replay.

## When to use / not use

- **Use** for “something changed for repo X; connected portal tabs should refresh.”
- **Do not use** for cross-process fan-out, guaranteed delivery, or high-volume streaming; use a real broker or a DB-backed outbox if you need that later.
