package traces

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// ClickHouseConfig holds ClickHouse connection configuration
type ClickHouseConfig struct {
	Host     string
	Port     int
	Database string
	Username string
	Password string
	UseTLS   bool
}

// NewClickHouseConn creates a new ClickHouse connection using HTTP protocol
func NewClickHouseConn(cfg *ClickHouseConfig) (driver.Conn, error) {
	// Use HTTP protocol (port 8123) instead of native (port 9000) for better compatibility
	protocol := clickhouse.HTTP

	opts := &clickhouse.Options{
		Addr:     []string{fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)},
		Protocol: protocol,
		Auth: clickhouse.Auth{
			Database: cfg.Database,
			Username: cfg.Username,
			Password: cfg.Password,
		},
		Debug: false,
		Debugf: func(format string, v ...interface{}) {
			fmt.Printf(format+"\n", v...)
		},
		DialTimeout:     10 * time.Second,
		MaxOpenConns:    5,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
	}

	if cfg.UseTLS {
		opts.TLS = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	conn, err := clickhouse.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to clickhouse: %w", err)
	}

	// Use a timeout context for the ping
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := conn.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping clickhouse: %w", err)
	}

	return conn, nil
}
