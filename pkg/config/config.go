package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct{
	App				AppConfig
	Database		DatabaseConfig
	Redis			RedisConfig
	RabbitMQ		RabbitMQConfig
	MinIO			MinIOConfig
	JWT				JWTConfig
	AI				AIConfig
}

type AppConfig struct{
	Env				string
	Port			string
}

type DatabaseConfig struct{
	Host			string
	Port			string
	User			string
	Password 		string
	Name			string
}

type RedisConfig struct{
	Host			string
	Port			string
	Password		string
}

type RabbitMQConfig struct{
	URL				string
}

type MinIOConfig struct{
	Endpoint		string
	AccessKey		string
	SecretKey		string
	UseSSL			bool
	Bucket			string
}

type JWTConfig struct{
	Secret			string
	AccessExpiry	time.Duration
	RefreshExpiry 	time.Duration
}

type AIConfig struct{
	ServiceURL	string
}


func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	accessExpiry, err := time.ParseDuration(getEnv("JWT_ACCESS_EXPIRY", "15m"))
	if err != nil {
		accessExpiry = 15 * time.Minute
	}

	refreshExpiry, err := time.ParseDuration(getEnv("JWT_REFRESH_EXPIRY", "168h"))
	if err != nil {
		refreshExpiry = 168 * time.Hour
	}
	
	return &Config{
		App: AppConfig{
			Port: getEnv("APP_PORT", "8080"),
			Env:  getEnv("APP_ENV", "development"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "neknowbot"),
			Password: getEnv("DB_PASSWORD", ""),
			Name:     getEnv("DB_NAME", "neknow_bot_db"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
		},
		RabbitMQ: RabbitMQConfig{
			URL: getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		},
		MinIO: MinIOConfig{
			Endpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
			AccessKey: getEnv("MINIO_ACCESS_KEY", "minioadmin"),
			SecretKey: getEnv("MINIO_SECRET_KEY", ""),
			UseSSL:    getEnvBool("MINIO_USE_SSL", false),
			Bucket:    getEnv("MINIO_BUCKET", "neknowbot-files"),
		},
		JWT: JWTConfig{
			Secret:        getEnv("JWT_SECRET", ""),
			AccessExpiry:  accessExpiry,
			RefreshExpiry: refreshExpiry,
		},
		AI: AIConfig{
			ServiceURL: getEnv("AI_SERVICE_URL", "http://localhost:8000"),
		},
	}
}

func getEnv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	result, err := strconv.ParseBool(val)
	if err != nil {
		return fallback
	}
	return result
}