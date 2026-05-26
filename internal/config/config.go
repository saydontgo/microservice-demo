package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type Config struct {
	HTTPPort      string
	MySQLHost     string
	MySQLPort     string
	MySQLDatabase string
	MySQLUser     string
	MySQLPassword string
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int
	PasswordSalt  string
	TokenSecret   string
	TokenTTL      time.Duration
}

func Load() Config {
	return Config{
		HTTPPort:      getEnv("HTTP_PORT", "8000"),
		MySQLHost:     getEnv("MYSQL_HOST", "127.0.0.1"),
		MySQLPort:     getEnv("MYSQL_PORT", "3306"),
		MySQLDatabase: getEnv("MYSQL_DATABASE", "microservice_demo"),
		MySQLUser:     getEnv("MYSQL_USER", "demo"),
		MySQLPassword: getEnv("MYSQL_PASSWORD", "123456"),
		RedisHost:     getEnv("REDIS_HOST", "127.0.0.1"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       0,
		PasswordSalt:  getEnv("PASSWORD_SALT", "microservice-demo"),
		TokenSecret:   getEnv("TOKEN_SECRET", "microservice-demo-token"),
		TokenTTL:      time.Duration(getEnvInt("TOKEN_TTL_SECONDS", 7200)) * time.Second,
	}
}

func (c Config) MySQLDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&loc=Local", c.MySQLUser, c.MySQLPassword, c.MySQLHost, c.MySQLPort, c.MySQLDatabase)
}

func (c Config) RedisOptions() *redis.Options {
	return &redis.Options{
		Addr:     c.RedisHost + ":" + c.RedisPort,
		Password: c.RedisPassword,
		DB:       c.RedisDB,
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
