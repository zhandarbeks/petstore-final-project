package config

import (
	"log"
	"os"
	"strconv"
	// "time" // Not immediately needed for basic adoption service config

	"github.com/joho/godotenv" // For loading .env files (optional)
)

// Config holds all configuration for the adoption-service
type Config struct {
	ServerPort    string // Port for the gRPC server (e.g., ":50053")
	MongoURI      string // MongoDB connection URI for adoption applications
	RedisAddr     string // Redis server address for adoption caching
	RedisPassword string // Redis password (if any)
	RedisDB       int    // Redis database number for adoption caching
	NatsURL       string // NATS server URL (e.g., "nats://localhost:4222")

	// Optional: If adoption service needs to directly call other services
	// UserServiceClientURL string // e.g., "user-service:50051"
	// PetServiceClientURL  string // e.g., "pet-service:50052"
}

// Load loads configuration. It first attempts to load from a .env file (if present),
// then falls back to actual environment variables, and finally to defaults.
func Load() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		log.Println("Adoption Service | Info: No .env file found or error loading .env, relying on environment variables and defaults.")
	} else {
		log.Println("Adoption Service | Info: Successfully loaded .env file.")
	}

	cfg := &Config{
		ServerPort:    getEnv("ADOPTION_SERVICE_PORT", ":50053"),
		MongoURI:      getEnv("MONGO_URI_ADOPTIONS", "mongodb://localhost:27017/adoptiondb_dev"), // Default for local
		RedisAddr:     getEnv("REDIS_ADDR_ADOPTIONS", "localhost:6379"),                       // Default for local
		RedisPassword: getEnv("REDIS_PASSWORD_ADOPTIONS", ""),                                   // Default to no password
		NatsURL:       getEnv("NATS_URL", "nats://localhost:4222"),                             // Default for local NATS
		// UserServiceClientURL: getEnv("USER_SERVICE_GRPC_URL", "user-service:50051"), // Example
		// PetServiceClientURL:  getEnv("PET_SERVICE_GRPC_URL", "pet-service:50052"),   // Example
	}

	redisDBStr := getEnv("REDIS_DB_ADOPTIONS", "2") // Using DB 2 for adoptions to separate
	redisDBVal, err := strconv.Atoi(redisDBStr)
	if err != nil {
		log.Printf("Adoption Service | Warning: Invalid REDIS_DB_ADOPTIONS value: '%s'. Using default 2. Error: %v", redisDBStr, err)
		cfg.RedisDB = 2
	} else {
		cfg.RedisDB = redisDBVal
	}

	// Critical validations
	if cfg.MongoURI == "" {
		log.Fatal("Adoption Service | FATAL: MONGO_URI_ADOPTIONS environment variable is required.")
	}
	if cfg.ServerPort == "" {
		log.Fatal("Adoption Service | FATAL: ADOPTION_SERVICE_PORT environment variable is required.")
	}
	if cfg.NatsURL == "" {
		log.Fatal("Adoption Service | FATAL: NATS_URL environment variable is required.")
	}

	return cfg, nil
}

// Helper function to get an environment variable or return a default value.
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	if fallback != "" {
		log.Printf("Adoption Service | Info: Environment variable %s not set, using default value: %s", key, fallback)
	} else {
		log.Printf("Adoption Service | Warning: Environment variable %s not set and no default value provided.", key)
	}
	return fallback
}