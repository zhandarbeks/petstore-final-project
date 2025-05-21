package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv" // For loading .env files (optional)
)

// Config holds all configuration for the user-service
type Config struct {
	ServerPort    string        // Port for the gRPC server (e.g., ":50051")
	MongoURI      string        // MongoDB connection URI
	RedisAddr     string        // Redis server address (e.g., "localhost:6379")
	RedisPassword string        // Redis password (if any, leave empty if none)
	RedisDB       int           // Redis database number
	JWTSecretKey  string        // Secret key for signing JWT tokens
	TokenExpiry   time.Duration // Duration for token expiry
}

// Load loads configuration. It first attempts to load from a .env file (if present),
// then falls back to actual environment variables, and finally to defaults.
func Load() (*Config, error) {
	// Attempt to load .env file.
	// This is useful for local development. In production or Docker environments,
	// environment variables should be set directly by the system or orchestrator.
	// We don't fail if it's not found, just log it.
	// godotenv.Load() loads .env from the current working directory.
	// If running 'go run ./user-service/cmd/main.go' from project root, it looks for '.env' in project root.
	// If 'cd user-service && go run cmd/main.go', it looks for '.env' in 'user-service/'.
	// For consistency with Docker Compose, a .env at the project root is common.
	err := godotenv.Load()
	if err != nil {
		log.Println("Info: No .env file found or error loading .env, relying on environment variables and defaults.")
	} else {
		log.Println("Info: Successfully loaded .env file.")
	}

	cfg := &Config{
		ServerPort:    getEnv("USER_SERVICE_PORT", ":50051"),
		MongoURI:      getEnv("MONGO_URI", "mongodb://localhost:27017/userdb_dev"), // Default for local, Docker will override
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),                 // Default for local, Docker will override
		RedisPassword: getEnv("REDIS_PASSWORD", ""),                           // Default to no password
		JWTSecretKey:  getEnv("JWT_SECRET_KEY", "your-very-secret-and-long-key-!@#$%^&*()_dev"), // !! CHANGE THIS !!
	}

	redisDBStr := getEnv("REDIS_DB", "0")
	redisDBVal, err := strconv.Atoi(redisDBStr)
	if err != nil {
		log.Printf("Warning: Invalid REDIS_DB value: '%s'. Using default 0. Error: %v", redisDBStr, err)
		cfg.RedisDB = 0
	} else {
		cfg.RedisDB = redisDBVal
	}

	tokenExpiryStr := getEnv("TOKEN_EXPIRY_MINUTES", "60") // Default to 60 minutes
	tokenExpiryMinutes, err := strconv.Atoi(tokenExpiryStr)
	if err != nil {
		log.Printf("Warning: Invalid TOKEN_EXPIRY_MINUTES value: '%s'. Using default 60 minutes. Error: %v", tokenExpiryStr, err)
		cfg.TokenExpiry = 60 * time.Minute
	} else {
		cfg.TokenExpiry = time.Duration(tokenExpiryMinutes) * time.Minute
	}

	// Critical validations
	if cfg.MongoURI == "" {
		log.Fatal("FATAL: MONGO_URI environment variable is required and was not found or set.")
	}
	if cfg.ServerPort == "" {
		log.Fatal("FATAL: USER_SERVICE_PORT environment variable is required and was not found or set.")
	}
	if cfg.JWTSecretKey == "your-very-secret-and-long-key-!@#$%^&*()_dev" || len(cfg.JWTSecretKey) < 32 {
		log.Println("WARNING: JWT_SECRET_KEY is using a default or is too short. Please set a strong, unique key via environment variable for production.")
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
		log.Printf("Info: Environment variable %s not set, using default value: %s", key, fallback)
	} else {
		log.Printf("Warning: Environment variable %s not set and no default value provided for it.", key)
	}
	return fallback
}