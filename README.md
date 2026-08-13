# JiboConnector

Worker service for the [OpenJibo](https://github.com/wrcrooks/JiboExperiments) self-hosted cloud. Runs as a separate Docker container alongside `jibo-api`.

## Purpose

When Jibo recognizes an enrolled person and (with their consent) takes a photo, `jibo-api` tags the resulting media record with the recognized person's id and stores it via the `Media_20160725` API. JiboConnector is responsible for the next step: picking up newly captured, person-tagged photos and delivering them to that person's configured notification contacts (text and/or email), as recorded via the `jibo-api` portal's photo-contacts endpoints.

## Status

Scaffolding only — implementation not yet started. See `jibo-api`'s `PhotoNotificationContactRecord` / `MediaRecord.PersonId` / `Media_20160725` API for the data this service will consume.

## Relationship to jibo-api

This repository is intentionally separate from [`JiboExperiments`](https://github.com/wrcrooks/JiboExperiments) (the `jibo-api` cloud server). `jibo-api` never sends SMS/email directly — it only tags and stores photos and stores contact preferences. JiboConnector is the piece that actually reaches out to a phone/email provider.
