# Weather App — Domain Glossary

## Weather Record
A snapshot of atmospheric conditions for a single city at a point in time. Fields: city name, temperature (°C), humidity (%), condition (e.g. "sunny"), wind speed (km/h), last_updated timestamp. The unit of data that is cached and served.

## City
The primary identifier for a Weather Record. Fixed set of ~20 major cities seeded into the database at startup.

## Cached Endpoint
`GET /v1/weather/{city}` — serves a Weather Record from Redis if present (cache hit); on miss, fetches from a Read Replica, writes to Redis, then returns the record.

## No-Cache Endpoint
`GET /v1/weather/{city}/no-cache` — bypasses Redis entirely. Always fetches directly from a Read Replica via round-robin. Used to demonstrate baseline latency without caching.

## Cache-Aside Pattern
The application is responsible for populating the cache. On a cache miss the app reads from the database, writes the result to Redis, and returns it. The cache is never written to directly by the database.

## TTL (Time-To-Live)
15 minutes. The duration a Weather Record lives in Redis before expiry. Chosen because the underlying data only changes every 15 minutes.

## Primary
The single PostgreSQL write node. Handles seed data ingestion and any future weather data refresh jobs. Never queried by the read path.

## Read Replica
One of two PostgreSQL replicas that stream changes from the Primary. All read queries — both cached-miss fallback and no-cache — are distributed across the two replicas via round-robin.

## Round-Robin
The strategy used to distribute read queries across the two Read Replicas. Each incoming read request is routed to the next replica in sequence.

## Benchmark
A `k6` script included in the repo that hammers both endpoints concurrently and reports p50/p95/p99 latency and requests/second. The primary artefact for demonstrating the cache vs. no-cache performance difference.

## Observability Stack
Prometheus (metrics scraping from the Go app) + Grafana (dashboard). Both run as Docker Compose services. Provides visual proof of latency difference alongside the k6 hard numbers.
