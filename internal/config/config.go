package config

import (
	"os"
	"strconv"
)

type Config struct {
	DB_USERNAME string
	DB_PASSWORD string
	DB_HOST     string
	DB_PORT     string
	DB_NAME     string
	DISABLE_TLS string

	REDIS_HOST     string
	REDIS_PORT     string
	REDIS_USERNAME string
	REDIS_PASSWORD string

	// ClickHouse configuration for traces
	CLICKHOUSE_HOST     string
	CLICKHOUSE_PORT     int
	CLICKHOUSE_DATABASE string
	CLICKHOUSE_USERNAME string
	CLICKHOUSE_PASSWORD string
	CLICKHOUSE_USE_TLS  bool

	// Otel
	OTEL_EXPORTER_OTLP_ENDPOINT string

	// Auth
	STATE_SECRET        string
	AUTH0_DOMAIN        string
	AUTH0_CLIENT_ID     string
	AUTH0_CLIENT_SECRET string
	AUTH0_CALLBACK_URL  string
}

func ReadConfig() *Config {
	// Default to HTTP port 8123 (more compatible than native port 9000)
	clickhousePort := 8123
	if portStr := os.Getenv("CLICKHOUSE_PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			clickhousePort = port
		}
	}

	return &Config{
		DB_USERNAME: os.Getenv("DB_USERNAME"),
		DB_PASSWORD: os.Getenv("DB_PASSWORD"),
		DB_HOST:     os.Getenv("DB_HOST"),
		DB_PORT:     os.Getenv("DB_PORT"),
		DB_NAME:     os.Getenv("DB_NAME"),
		DISABLE_TLS: os.Getenv("DISABLE_TLS"),

		REDIS_HOST:     os.Getenv("REDIS_HOST"),
		REDIS_PORT:     os.Getenv("REDIS_PORT"),
		REDIS_USERNAME: os.Getenv("REDIS_USERNAME"),
		REDIS_PASSWORD: os.Getenv("REDIS_PASSWORD"),

		CLICKHOUSE_HOST:     getEnvOrDefault("CLICKHOUSE_HOST", "localhost"),
		CLICKHOUSE_PORT:     clickhousePort,
		CLICKHOUSE_DATABASE: getEnvOrDefault("CLICKHOUSE_DATABASE", "otel"),
		CLICKHOUSE_USERNAME: getEnvOrDefault("CLICKHOUSE_USERNAME", "default"),
		CLICKHOUSE_PASSWORD: os.Getenv("CLICKHOUSE_PASSWORD"),
		CLICKHOUSE_USE_TLS:  os.Getenv("CLICKHOUSE_USE_TLS") == "true",

		OTEL_EXPORTER_OTLP_ENDPOINT: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),

		STATE_SECRET:        os.Getenv("STATE_SECRET"),
		AUTH0_DOMAIN:        os.Getenv("AUTH0_DOMAIN"),
		AUTH0_CLIENT_ID:     os.Getenv("AUTH0_CLIENT_ID"),
		AUTH0_CLIENT_SECRET: os.Getenv("AUTH0_CLIENT_SECRET"),
		AUTH0_CALLBACK_URL:  os.Getenv("AUTH0_CALLBACK_URL"),
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
