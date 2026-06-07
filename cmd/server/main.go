package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/riorhezaharris/weather-app/internal/cache"
	"github.com/riorhezaharris/weather-app/internal/db"
	"github.com/riorhezaharris/weather-app/internal/handler"
	"github.com/riorhezaharris/weather-app/internal/repository"
)

func main() {
	ctx := context.Background()

	port := getEnv("PORT", "8080")
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	replica1URL := mustEnv("DB_REPLICA_1_URL")
	replica2URL := mustEnv("DB_REPLICA_2_URL")
	cacheTTL := time.Duration(getEnvInt("CACHE_TTL_MINUTES", 15)) * time.Minute

	replicaPool, err := db.NewReplicaPool(ctx, replica1URL, replica2URL)
	if err != nil {
		log.Fatalf("connecting to replicas: %v", err)
	}
	defer replicaPool.Close()

	redisCache := cache.NewRedisCache(redisAddr, cacheTTL)
	defer redisCache.Close()

	repo := repository.NewWeatherRepository(replicaPool)
	weatherHandler := handler.NewWeatherHandler(repo, redisCache)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", handler.Health)
	r.Handle("/metrics", promhttp.Handler())
	r.Get("/v1/weather/{city}", weatherHandler.GetCached)
	r.Get("/v1/weather/{city}/no-cache", weatherHandler.GetNoCache)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("server listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s is not set", key)
	}
	return v
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
