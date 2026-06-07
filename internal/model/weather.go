package model

import "time"

type WeatherRecord struct {
	City               string    `json:"city"`
	TemperatureCelsius float64   `json:"temperature_celsius"`
	HumidityPercent    int       `json:"humidity_percent"`
	Condition          string    `json:"condition"`
	WindSpeedKmh       float64   `json:"wind_speed_kmh"`
	LastUpdated        time.Time `json:"last_updated"`
}
