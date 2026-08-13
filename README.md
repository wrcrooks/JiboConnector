# JiboConnector

Worker service for the [OpenJibo](https://github.com/wrcrooks/JiboExperiments) self-hosted cloud. Runs as a separate Docker container alongside `jibo-api`.

## Purpose

When Jibo recognizes an enrolled person and (with their consent) takes a photo, `jibo-api` tags the resulting media record with the recognized person's id and stores it via the `Media_20160725` API. JiboConnector is responsible for the next step: picking up newly captured, person-tagged photos and delivering them to that person's configured notification contacts (text and/or email), as recorded via the `jibo-api` portal's photo-contacts endpoints.

## Stack

**Go**, standard library only (no web framework) — chosen for performance (compiled, static binary, goroutine-based concurrency well suited to a poll-then-fan-out worker), a minimal attack surface (the final Docker image is `distroless/static`, non-root, no shell — this service holds notification-provider credentials and handles household PII), and low ceremony for iterating on a small, focused service.

## Status

Scaffolding only:
- `cmd/worker/main.go` — entrypoint; runs a poll loop on a timer plus a `/health` HTTP endpoint. The poll loop doesn't do anything yet.
- `internal/config` — env-based config (`JIBOCONNECTOR_` prefix).
- `internal/jiboapi` — client stub for `jibo-api`'s `Media_20160725` and photo-contacts endpoints. `ListRecentPersonTaggedMedia` and `PhotoContactsForPerson` are not implemented yet — see the comments in `client.go` for what's still undecided (`jibo-api` doesn't yet support filtering media by "has a PersonId" or "created since", and this worker's auth to `jibo-api`'s portal API isn't designed yet).
- `internal/notify` — a `Notifier` interface with only a `NoopNotifier` (logs what it would send). No SMS/email provider has been chosen yet.

Not yet decided: SMS/email provider (Twilio, SendGrid, plain SMTP, ...), how this worker authenticates to `jibo-api`, and how it tracks which photos it's already processed.

## Build & run

```
go build -o bin/worker ./cmd/worker
./bin/worker
```

or via Docker:

```
docker build -t jibo-connector .
docker run --rm -p 8090:8090 jibo-connector
```

## Relationship to jibo-api

This repository is intentionally separate from [`JiboExperiments`](https://github.com/wrcrooks/JiboExperiments) (the `jibo-api` cloud server). `jibo-api` never sends SMS/email directly — it only tags and stores photos and stores contact preferences. JiboConnector is the piece that actually reaches out to a phone/email provider.
