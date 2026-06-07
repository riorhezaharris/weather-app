package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/riorhezaharris/weather-app/internal/cache"
	"github.com/riorhezaharris/weather-app/internal/repository"
)

var (
	requestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"handler"})

	cacheHits = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cache_hits_total",
		Help: "Total number of Redis cache hits",
	})

	cacheMisses = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cache_misses_total",
		Help: "Total number of Redis cache misses",
	})
)

type WeatherHandler struct {
	repo  *repository.WeatherRepository
	cache *cache.RedisCache
}

func NewWeatherHandler(repo *repository.WeatherRepository, cache *cache.RedisCache) *WeatherHandler {
	return &WeatherHandler{repo: repo, cache: cache}
}

// GetCached implements the Cache-Aside pattern:
// Redis hit → return immediately; miss → fetch replica → populate cache → return.
func (h *WeatherHandler) GetCached(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		requestDuration.WithLabelValues("cached").Observe(time.Since(start).Seconds())
	}()

	city := chi.URLParam(r, "city")

	record, err := h.cache.Get(r.Context(), city)
	if err == nil {
		cacheHits.Inc()
		writeJSON(w, http.StatusOK, record)
		return
	}

	if !errors.Is(err, cache.ErrCacheMiss) {
		http.Error(w, "cache error", http.StatusInternalServerError)
		return
	}

	cacheMisses.Inc()

	record, err = h.repo.GetByCity(r.Context(), city)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	_ = h.cache.Set(r.Context(), record)

	writeJSON(w, http.StatusOK, record)
}

// GetNoCache bypasses Redis entirely and always queries a replica directly.
// Used as the baseline for latency comparison benchmarks.
func (h *WeatherHandler) GetNoCache(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		requestDuration.WithLabelValues("no_cache").Observe(time.Since(start).Seconds())
	}()

	city := chi.URLParam(r, "city")

	record, err := h.repo.GetByCity(r.Context(), city)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, record)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
