# Weather App — Cache-Aside + Read Replica Showcase

A backend portfolio project demonstrating **latency reduction** and **read scaling** through the Cache-Aside pattern (Redis) and PostgreSQL read replicas. The entire stack — including benchmarking and observability — runs with a single command.

---

## The Scenario

Millions of users or smart-home devices request current weather conditions for major cities simultaneously. The underlying data only changes every 15 minutes. Without caching, every request hits the database directly — redundant, slow, and expensive at scale.

**The solution:** serve reads from Redis. On a cache miss, fetch from a read replica, populate the cache, and return. Subsequent requests for the same city cost one Redis round-trip instead of a full database query.

---

## What It Showcases

| Concept | Implementation |
|---|---|
| Cache-Aside pattern | Redis with 15-minute TTL; app is responsible for cache population |
| Read replica scaling | 1 primary + 2 replicas; reads distributed via round-robin |
| Latency measurement | Two endpoints in one app — cached vs. no-cache — benchmarked simultaneously |
| Observability | Prometheus metrics + Grafana dashboard (cache hit rate, p50/p95/p99 latency) |
| Container orchestration | Full stack via Docker Compose; single `docker-compose up` |

---

## Architecture

```
                        ┌─────────────────────────────┐
                        │         Go HTTP App          │
                        │  Chi router · Prometheus     │
                        └────────────┬────────────────-┘
                                     │
              ┌──────────────────────┴──────────────────────┐
              │                                             │
    GET /v1/weather/{city}                   GET /v1/weather/{city}/no-cache
    (Cache-Aside path)                       (Direct replica path)
              │                                             │
              ▼                                             │
       ┌─────────────┐   miss                               │
       │    Redis    │ ─────────────────────────────────────┤
       │   (cache)   │                                      │
       └──────┬──────┘                                      │
              │ hit                                         │
              ▼                                             ▼
         Return immediately              ┌─────────────────────────────┐
                                         │      Round-Robin Selector   │
                                         └────────┬──────────┬─────────┘
                                                  │          │
                                         ┌────────▼─┐   ┌────▼─────┐
                                         │Replica 1 │   │ Replica 2│
                                         └────────┬─┘   └────┬─────┘
                                                  │          │
                                         ┌────────▼──────────▼──────┐
                                         │      Primary (writes)    │
                                         └──────────────────────────┘
```

---

## Tech Stack

| Layer | Technology |
|---|---|
| Language | Go 1.22 |
| HTTP router | Chi v5 |
| Cache | Redis 7 |
| Database | PostgreSQL (bitnami, streaming replication) |
| Metrics | Prometheus + Grafana |
| Benchmarking | k6 |
| Infrastructure | Docker Compose |

---

## Project Structure

```
weather-app/
├── cmd/server/             # Application entrypoint
├── internal/
│   ├── cache/              # Redis client (Get, Set, ErrCacheMiss)
│   ├── db/                 # pgx pool + atomic round-robin replica selector
│   ├── handler/            # HTTP handlers + Prometheus instrumentation
│   ├── model/              # WeatherRecord struct
│   └── repository/         # SQL query layer
├── docker/
│   ├── postgres/seed.sql   # 20 cities seeded on primary startup
│   ├── prometheus/         # Scrape config (5s interval)
│   └── grafana/            # Auto-provisioned datasource + dashboard
├── benchmark/benchmark.js  # k6 script: warmup → cached + no-cache simultaneously
├── docs/adr/               # Architecture Decision Records
├── docker-compose.yml
└── Dockerfile              # Multi-stage Go build → alpine
```

---

## Quick Start

**Prerequisites:** Docker, Docker Compose

```bash
# Clone and start the full stack
git clone https://github.com/riorhezaharris/weather-app.git
cd weather-app
docker compose up --build
```

All services start in dependency order. The app waits for healthy replicas before accepting traffic.

| Service | URL |
|---|---|
| Weather API | http://localhost:8080 |
| Grafana dashboard | http://localhost:3000 (admin / admin) |
| Prometheus | http://localhost:9090 |

---

## API Endpoints

### Cached — Cache-Aside path
```
GET /v1/weather/{city}
```
Checks Redis first. On a cache hit returns immediately (~2ms). On a miss, fetches from a read replica, populates Redis with a 15-minute TTL, then returns.

### No-Cache — Direct replica path
```
GET /v1/weather/{city}/no-cache
```
Bypasses Redis entirely. Always queries a read replica via round-robin. Used as the baseline for latency comparison.

### Other
```
GET /health     — liveness check
GET /metrics    — Prometheus metrics
```

**Example:**
```bash
curl http://localhost:8080/v1/weather/jakarta
curl http://localhost:8080/v1/weather/tokyo/no-cache
```

**Response:**
```json
{
  "city": "jakarta",
  "temperature_celsius": 32.5,
  "humidity_percent": 78,
  "condition": "partly_cloudy",
  "wind_speed_kmh": 14.2,
  "last_updated": "2026-06-07T13:22:26Z"
}
```

**Available cities:** `jakarta`, `tokyo`, `new_york`, `london`, `paris`, `sydney`, `dubai`, `singapore`, `mumbai`, `beijing`, `sao_paulo`, `cairo`, `lagos`, `toronto`, `berlin`, `seoul`, `bangkok`, `mexico_city`, `istanbul`, `johannesburg`

---

## Running the Benchmark

**Prerequisites:** [k6](https://k6.io/docs/get-started/installation/)

```bash
BASE_URL=http://localhost:8080 k6 run benchmark/benchmark.js
```

The script runs in two phases:

| Phase | Duration | What happens |
|---|---|---|
| Warmup | 0–30s | 5 VUs hit every city once to fully populate Redis |
| Benchmark | 30–90s | 50 VUs hammer `/cached` + 50 VUs hammer `/no-cache` simultaneously |

### Results

```
  ══════════════════════════════════════════════════════
        WEATHER APP — CACHE vs NO-CACHE BENCHMARK
  ══════════════════════════════════════════════════════
          CACHED        NO-CACHE      SPEEDUP
  ──────────────────────────────────────────────────────
  p50     2.21ms        36.03ms       16.3x faster
  p95     4.75ms        55.47ms       11.7x faster
  p99     8.29ms        63.02ms        7.6x faster
  ──────────────────────────────────────────────────────
  requests   cached: 1,166,261   no-cache: 81,512
  ══════════════════════════════════════════════════════
```

> **Note on the benchmark setup:** On localhost Docker, both Redis and PostgreSQL respond in sub-millisecond time — which hides the real-world latency gap. To produce a representative result, the no-cache query includes `pg_sleep(0.01)` to simulate the ~10ms round-trip latency of a remote managed database (e.g. AWS RDS, Cloud SQL). This is what makes the benchmark honest rather than artificially optimistic. See [ADR 0002](docs/adr/0002-simulated-db-latency.md) for the full rationale. Remove the `pg_sleep` before deploying to a real environment — the network provides the latency naturally.

---

## Observability

Open **http://localhost:3000** (admin / admin) to view the pre-provisioned Grafana dashboard:

- **Request rate** — cached vs. no-cache requests/sec
- **p50 / p95 / p99 latency** — side-by-side comparison
- **Cache hit rate** — percentage of requests served from Redis
- **Cache hits vs. misses** — time series

Prometheus scrapes `/metrics` every 5 seconds. Raw metrics are available at **http://localhost:9090**.

---

## Architecture Decisions

| ADR | Decision |
|---|---|
| [0001](docs/adr/0001-two-endpoint-comparison-strategy.md) | Two endpoints in one app vs. two separate applications |
| [0002](docs/adr/0002-simulated-db-latency.md) | Simulated DB latency for a representative local benchmark |

---

## Teardown

```bash
docker compose down -v   # stops containers and removes volumes
```
