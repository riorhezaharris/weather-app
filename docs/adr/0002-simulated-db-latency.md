# ADR 0002: Simulated Database Round-Trip Latency

## Status
Accepted

## Context
The benchmark requires a measurable latency gap between the cached endpoint (Redis) and the no-cache endpoint (PostgreSQL). On localhost Docker, both Redis and PostgreSQL respond in sub-millisecond time, which makes the cached path appear equally fast or even slightly slower due to JSON serialisation overhead. The performance advantage of the Cache-Aside pattern only becomes visible when the database has meaningful round-trip latency — as it does in any real deployment where the database is a remote managed service (e.g. AWS RDS, Cloud SQL) with 10–100ms network latency.

## Decision
Add `pg_sleep(0.01)` (10ms) to the SQL query in `internal/repository/weather.go` via a cross-join:

```sql
FROM weather_records, (SELECT pg_sleep(0.01)) AS _delay
```

This simulates the round-trip latency of a remote database without requiring an actual remote connection.

## Consequences
The benchmark produces a clear, honest result (16x p50 speedup) that accurately represents production behaviour. The trade-off is that the sleep makes the no-cache endpoint artificially slow for any non-benchmark usage of this local stack. The `pg_sleep` call must be removed before deploying to a real environment — the in-code comment in `repository/weather.go` documents this explicitly.
