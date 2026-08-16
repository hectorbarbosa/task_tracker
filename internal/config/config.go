package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration.
type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	JWT       JWTConfig
	RateLimit RateLimitConfig
}

type ServerConfig struct {
	Port string
	Env  string // "development" | "production"
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

// DSN returns a MySQL DSN for go-sql-driver/mysql.
func (d DatabaseConfig) DSN() string {
	return d.User + ":" + d.Password + "@tcp(" + d.Host + ":" + d.Port + ")/" + d.Name + "?parseTime=true&charset=utf8mb4&timeout=10s&readTimeout=10s&writeTimeout=10s"
}

type RedisConfig struct {
	Host string
	Port string
}

// Addr returns a "host:port" address for go-redis.
func (r RedisConfig) Addr() string {
	return r.Host + ":" + r.Port
}

type JWTConfig struct {
	Secret     string
	Expiration int // hours
}

// RateLimitConfig controls the Redis-backed fixed-window rate limiter.
type RateLimitConfig struct {
	Enabled bool
	Limit   int           // max requests per window
	Window  time.Duration // window duration
}

// Load reads configuration from environment variables with sensible defaults
// for local development (matching docker-compose.dev.yml + .env).
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port: getEnv("APP_PORT", "8080"),
			Env:  getEnv("APP_ENV", "development"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "127.0.0.1"),
			Port:     getEnv("DB_PORT", "3306"),
			User:     getEnv("DB_USER", "task_user"),
			Password: getEnv("DB_PASS", "task_pass"),
			Name:     getEnv("DB_NAME", "task_tracker"),
		},
		Redis: RedisConfig{
			Host: getEnv("REDIS_HOST", "127.0.0.1"),
			Port: getEnv("REDIS_PORT", "6379"),
		},
		JWT: JWTConfig{
			Secret:     getEnv("JWT_SECRET", "dev-secret-change-me"),
			Expiration: getEnvInt("JWT_EXPIRATION_HOURS", 24),
		},
		RateLimit: RateLimitConfig{
			Enabled: getEnvBool("RATE_LIMIT_ENABLED", true),
			Limit:   getEnvInt("RATE_LIMIT_REQUESTS", 60),
			Window:  time.Duration(getEnvInt("RATE_LIMIT_WINDOW_SECONDS", 60)) * time.Second,
		},
	}
}

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

func getEnvBool(key string, def bool) bool {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}
