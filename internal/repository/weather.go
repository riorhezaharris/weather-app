package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/riorhezaharris/weather-app/internal/db"
	"github.com/riorhezaharris/weather-app/internal/model"
)

type WeatherRepository struct {
	replicas *db.ReplicaPool
}

func NewWeatherRepository(replicas *db.ReplicaPool) *WeatherRepository {
	return &WeatherRepository{replicas: replicas}
}

func (r *WeatherRepository) GetByCity(ctx context.Context, city string) (*model.WeatherRecord, error) {
	pool := r.replicas.Next()
	// pg_sleep(0.01) simulates the ~10ms round-trip latency of a remote managed
	// database (e.g. RDS, Cloud SQL). Without it, localhost Docker makes both
	// Redis and PostgreSQL equally fast, which hides the cache speedup entirely.
	// Remove this in a real deployment — the network round-trip provides the
	// latency naturally.
	row := pool.QueryRow(ctx, `
		SELECT city, temperature_celsius, humidity_percent, condition, wind_speed_kmh, last_updated
		FROM weather_records, (SELECT pg_sleep(0.01)) AS _delay
		WHERE city = $1
	`, city)

	var record model.WeatherRecord
	err := row.Scan(
		&record.City,
		&record.TemperatureCelsius,
		&record.HumidityPercent,
		&record.Condition,
		&record.WindSpeedKmh,
		&record.LastUpdated,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("city %q not found", city)
		}
		return nil, err
	}
	return &record, nil
}
