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

	LANGFUSE_USERNAME string
	LANGFUSE_PASSWORD string
	LANGFUSE_ENDPOINT string

	OPENAI_API_KEY string

	// ClickHouse configuration for traces
	CLICKHOUSE_HOST     string
	CLICKHOUSE_PORT     int
	CLICKHOUSE_DATABASE string
	CLICKHOUSE_USERNAME string
	CLICKHOUSE_PASSWORD string
	CLICKHOUSE_USE_TLS  bool

	// Otel
	OTEL_EXPORTER_OTLP_ENDPOINT string
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

		LANGFUSE_USERNAME: os.Getenv("LANGFUSE_USERNAME"),
		LANGFUSE_PASSWORD: os.Getenv("LANGFUSE_PASSWORD"),
		LANGFUSE_ENDPOINT: os.Getenv("LANGFUSE_ENDPOINT"),

		OPENAI_API_KEY: os.Getenv("OPENAI_API_KEY"),

		CLICKHOUSE_HOST:     getEnvOrDefault("CLICKHOUSE_HOST", "localhost"),
		CLICKHOUSE_PORT:     clickhousePort,
		CLICKHOUSE_DATABASE: getEnvOrDefault("CLICKHOUSE_DATABASE", "otel"),
		CLICKHOUSE_USERNAME: getEnvOrDefault("CLICKHOUSE_USERNAME", "default"),
		CLICKHOUSE_PASSWORD: os.Getenv("CLICKHOUSE_PASSWORD"),
		CLICKHOUSE_USE_TLS:  os.Getenv("CLICKHOUSE_USE_TLS") == "true",

		OTEL_EXPORTER_OTLP_ENDPOINT: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
