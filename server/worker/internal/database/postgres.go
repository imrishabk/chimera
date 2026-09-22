package database

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPostgresConnectionPool(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, err
	}

	var (
		poolMaxConnection         = getEnvOrDefaultInt32("POOL_MAX_CONNECTION", 25)
		poolMinConnection         = getEnvOrDefaultInt32("POOL_MIN_CONNECTION", 5)
		poolMaxConnectionIdleTime = time.Duration(
			getEnvOrDefaultInt32("POOL_MAX_CONNECTION_IDLE_TIME", 5)) * time.Minute
		poolMaxConnectionLifeTimeJitter = time.Duration(
			getEnvOrDefaultInt32("POOL_MAX_CONNECTION_LIFETIME_JITTER", 3)) * time.Minute
		poolMaxConnectionLifetime = time.Duration(
			getEnvOrDefaultInt32("POOL_MAX_CONNECTION_LIFETIME", 10)) * time.Minute
		poolHealthCheckPeriod = time.Duration(
			getEnvOrDefaultInt32("POOL_HEALTH_CHECK_PERIOD", 1)) * time.Minute
	)

	config.MaxConns = poolMaxConnection
	config.MinConns = poolMinConnection
	config.MaxConnIdleTime = poolMaxConnectionIdleTime
	config.MaxConnLifetimeJitter = poolMaxConnectionLifeTimeJitter
	config.MaxConnLifetime = poolMaxConnectionLifetime
	config.HealthCheckPeriod = poolHealthCheckPeriod

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping the server")
	}
	return pool, nil
}

// Returns val of the key else return default (def) as fallback
func getEnvOrDefaultInt32(key string, def int32) int32 {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	parsed, err := strconv.ParseInt(val, 10, 32)
	if err != nil {
		return def
	}
	return int32(parsed)
}
