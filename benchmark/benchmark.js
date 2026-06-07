import http from 'k6/http';
import { check } from 'k6';
import { Trend, Counter } from 'k6/metrics';

const cachedDuration    = new Trend('cached_duration_ms',    true);
const noCacheDuration   = new Trend('no_cache_duration_ms',  true);
const cachedRequests    = new Counter('cached_requests_total');
const noCacheRequests   = new Counter('no_cache_requests_total');

const CITIES = [
  'jakarta', 'tokyo',       'new_york',    'london',       'paris',
  'sydney',  'dubai',       'singapore',   'mumbai',       'beijing',
  'sao_paulo','cairo',      'lagos',       'toronto',      'berlin',
  'seoul',   'bangkok',     'mexico_city', 'istanbul',     'johannesburg',
];

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export const options = {
  // Tell k6 to compute p50 and p99 in addition to the defaults.
  summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(50)', 'p(90)', 'p(95)', 'p(99)'],

  scenarios: {
    // ── Phase 1: warm every city into Redis before benchmarking ──────────────
    cache_warmup: {
      executor:    'shared-iterations',
      vus:         5,
      iterations:  20,        // 20 VU-iterations × 20 cities = 400 warmup hits
      maxDuration: '30s',
      exec:        'warmup',
    },

    // ── Phase 2: hammer both endpoints simultaneously ─────────────────────────
    cached: {
      executor:  'constant-vus',
      vus:       50,
      duration:  '60s',
      startTime: '30s',
      exec:      'testCached',
    },
    no_cache: {
      executor:  'constant-vus',
      vus:       50,
      duration:  '60s',
      startTime: '30s',
      exec:      'testNoCache',
    },
  },

  thresholds: {
    // On localhost Docker both paths are fast; cached should still win on p95.
    cached_duration_ms:   ['p(95)<50'],
    no_cache_duration_ms: ['p(95)<500'],
    http_req_failed:      ['rate<0.01'],
  },
};

// ── Helpers ──────────────────────────────────────────────────────────────────

function randomCity() {
  return CITIES[Math.floor(Math.random() * CITIES.length)];
}

// ── Scenario functions ────────────────────────────────────────────────────────

export function warmup() {
  for (const city of CITIES) {
    const res = http.get(`${BASE_URL}/v1/weather/${city}`);
    check(res, { 'warmup 200': (r) => r.status === 200 });
  }
}

export function testCached() {
  const city = randomCity();
  const res  = http.get(`${BASE_URL}/v1/weather/${city}`);
  check(res, { 'cached 200': (r) => r.status === 200 });
  cachedDuration.add(res.timings.duration);
  cachedRequests.add(1);
}

export function testNoCache() {
  const city = randomCity();
  const res  = http.get(`${BASE_URL}/v1/weather/${city}/no-cache`);
  check(res, { 'no-cache 200': (r) => r.status === 200 });
  noCacheDuration.add(res.timings.duration);
  noCacheRequests.add(1);
}

// ── Summary ───────────────────────────────────────────────────────────────────

export function handleSummary(data) {
  const c  = data.metrics['cached_duration_ms'];
  const n  = data.metrics['no_cache_duration_ms'];
  const cc = data.metrics['cached_requests_total'];
  const nc = data.metrics['no_cache_requests_total'];

  if (!c || !n) {
    return { stdout: JSON.stringify(data, null, 2) };
  }

  const fmt     = (v) => (v != null ? `${v.toFixed(2)}ms` : 'N/A');
  const speedup = (cv, nv) => (cv && nv ? `${(nv / cv).toFixed(1)}x faster` : 'N/A');

  // k6 exposes median as 'med'; p(50)/p(99) only appear when summaryTrendStats includes them.
  const p50c = c.values['p(50)'] ?? c.values['med'];
  const p50n = n.values['p(50)'] ?? n.values['med'];
  const p95c = c.values['p(95)'];
  const p95n = n.values['p(95)'];
  const p99c = c.values['p(99)'];
  const p99n = n.values['p(99)'];

  const cachedCount   = cc?.values?.count ?? '—';
  const noCacheCount  = nc?.values?.count ?? '—';

  const row = (label, cv, nv) =>
    `  ${label.padEnd(6)}  ${fmt(cv).padEnd(12)}  ${fmt(nv).padEnd(12)}  ${speedup(cv, nv)}`;

  const divider = '  ' + '─'.repeat(54);

  const report = [
    '',
    '  ══════════════════════════════════════════════════════',
    '        WEATHER APP — CACHE vs NO-CACHE BENCHMARK       ',
    '  ══════════════════════════════════════════════════════',
    `  ${''.padEnd(6)}  ${'CACHED'.padEnd(12)}  ${'NO-CACHE'.padEnd(12)}  SPEEDUP`,
    divider,
    row('p50',  p50c, p50n),
    row('p95',  p95c, p95n),
    row('p99',  p99c, p99n),
    divider,
    `  requests   cached: ${String(cachedCount).padEnd(8)}  no-cache: ${noCacheCount}`,
    '  ══════════════════════════════════════════════════════',
    '',
  ].join('\n');

  return { stdout: report };
}
