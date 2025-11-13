package models

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ConnectionPoolConfig holds connection pool configuration
type ConnectionPoolConfig struct {
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// GetConnectionPoolConfig reads connection pool settings from environment variables
// with sensible defaults if not provided
func GetConnectionPoolConfig() ConnectionPoolConfig {
	config := ConnectionPoolConfig{
		MaxIdleConns:    10,
		MaxOpenConns:    25,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}

	// Read from environment variables
	if val := os.Getenv("DB_MAX_IDLE_CONNS"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			config.MaxIdleConns = n
		}
	}

	if val := os.Getenv("DB_MAX_OPEN_CONNS"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			config.MaxOpenConns = n
		}
	}

	if val := os.Getenv("DB_CONN_MAX_LIFETIME_MINUTES"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			config.ConnMaxLifetime = time.Duration(n) * time.Minute
		}
	}

	if val := os.Getenv("DB_CONN_MAX_IDLE_TIME_MINUTES"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			config.ConnMaxIdleTime = time.Duration(n) * time.Minute
		}
	}

	return config
}

// InitDatabase initializes the database connection with connection pooling
func InitDatabase(dsn string) (*gorm.DB, error) {
	if dsn == "" {
		return nil, nil // Database is optional
	}

	// Configure GORM logger
	config := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // Set to logger.Info for SQL logging
	}

	// Open database connection
	db, err := gorm.Open(postgres.Open(dsn), config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying sql.DB to configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// Configure connection pool
	poolConfig := GetConnectionPoolConfig()
	sqlDB.SetMaxIdleConns(poolConfig.MaxIdleConns)
	sqlDB.SetMaxOpenConns(poolConfig.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(poolConfig.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(poolConfig.ConnMaxIdleTime)

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	runMigrations(db)

	return db, nil
}

func runMigrations(db *gorm.DB) {
	db.AutoMigrate(&TechnicalSignal{})
	db.AutoMigrate(&DeepSearchRequest{})
}
