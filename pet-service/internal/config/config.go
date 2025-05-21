package config

import (
	"log"
	"os"
	"strconv"
	// "time" // Not immediately needed for basic pet service config, can be added if token/expiry logic comes here.

	"github.com/joho/godotenv" // For loading .env files (optional)
)

// Config holds all configuration for the pet-service
type Config struct {
	ServerPort    string // Port for the gRPC server (e.g., ":50052")
	MongoURI      string // MongoDB connection URI for the pets database/collection
	RedisAddr     string // Redis server address for pet caching (e.g., "localhost:6379")
	RedisPassword string // Redis password (if any)
	RedisDB       int    // Redis database number for pet caching
	// Add other pet-service specific configurations here if needed
}

// Load loads configuration. It first attempts to load from a .env file (if present),
// then falls back to actual environment variables, and finally to defaults.
func Load() (*Config, error) {
	// Attempt to load .env file.
	// godotenv.Load() loads .env from the current working directory.
	// If running 'go run ./pet-service/cmd/main.go' from project root, it looks for '.env' in project root.
	err := godotenv.Load()
	if err != nil {
		log.Println("Pet Service | Info: No .env file found or error loading .env, relying on environment variables and defaults.")
	} else {
		log.Println("Pet Service | Info: Successfully loaded .env file.")
	}

	cfg := &Config{
		ServerPort:    getEnv("PET_SERVICE_PORT", ":50052"),
		MongoURI:      getEnv("MONGO_URI_PETS", "mongodb://localhost:27017/petdb_dev"), // Default for local, Docker will override
		RedisAddr:     getEnv("REDIS_ADDR_PETS", "localhost:6379"),                   // Default for local, Docker will override
		RedisPassword: getEnv("REDIS_PASSWORD_PETS", ""),                             // Default to no password
	}

	redisDBStr := getEnv("REDIS_DB_PETS", "1") // Using DB 1 for pets to separate from user cache (DB 0)
	redisDBVal, err := strconv.Atoi(redisDBStr)
	if err != nil {
		log.Printf("Pet Service | Warning: Invalid REDIS_DB_PETS value: '%s'. Using default 1. Error: %v", redisDBStr, err)
		cfg.RedisDB = 1
	} else {
		cfg.RedisDB = redisDBVal
	}

	// Critical validations
	if cfg.MongoURI == "" {
		log.Fatal("Pet Service | FATAL: MONGO_URI_PETS environment variable is required and was not found or set.")
	}
	if cfg.ServerPort == "" {
		log.Fatal("Pet Service | FATAL: PET_SERVICE_PORT environment variable is required and was not found or set.")
	}

	return cfg, nil
}

// Helper function to get an environment variable or return a default value.
// Logs if a fallback is used or if a variable is missing without a fallback.
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	if fallback != "" {
		log.Printf("Pet Service | Info: Environment variable %s not set, using default value: %s", key, fallback)
	} else {
		log.Printf("Pet Service | Warning: Environment variable %s not set and no default value provided for it.", key)
	}
	return fallback
}