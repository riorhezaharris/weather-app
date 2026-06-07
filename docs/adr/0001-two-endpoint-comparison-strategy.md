# ADR 0001: Two-Endpoint Comparison Strategy

## Status
Accepted

## Context
The primary goal of this project is to demonstrate the latency impact of the Cache-Aside pattern against a no-cache baseline. Two approaches were considered: separate applications (v1/v2) or two endpoints within a single application.

## Decision
Expose two endpoints within a single Go application:
- `GET /v1/weather/{city}` — Cache-Aside path (Redis → Read Replica fallback)
- `GET /v1/weather/{city}/no-cache` — Direct Read Replica path, no Redis

## Consequences
A single `k6` benchmark script can hammer both endpoints in one run and produce a direct side-by-side latency comparison (p50/p95/p99). The codebase stays unified so the only meaningful difference between the two paths is the cache layer — which is exactly the variable being showcased. The trade-off is that the no-cache endpoint would never exist in a production system; it is a deliberate demo artefact.
