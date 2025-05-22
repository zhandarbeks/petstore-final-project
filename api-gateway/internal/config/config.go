package config

import (
	"log"
	"os"
	// "strconv" // Not immediately needed for this basic config
	// "time"    // Not immediately needed for this basic config

	"github.com/joho/godotenv" // For loading .env files (optional)
)

// Config holds all configuration for the api-gateway service
type Config struct {
	ServerPort             string // Port for the Gin HTTP server (e.g., ":8080")
	UserServiceGRPCURL   string // Target URL for the User gRPC Service
	PetServiceGRPCURL    string // Target URL for the Pet gRPC Service
	AdoptionServiceGRPCURL string // Target URL for the Adoption gRPC Service
	JWTSecretKey         string // Secret key for validating JWT tokens (if gateway handles this)
	GinMode              string // Gin's run mode (e.g., "debug", "release", "test")
}

// Load loads configuration. It first attempts to load from a .env file (if present),
// then falls back to actual environment variables, and finally to defaults.
func Load() (*Config, error) {
	err := godotenv.Load() // Tries to load .env from the current working directory
	if err != nil {
		log.Println("API Gateway | Info: No .env file found or error loading .env, relying on environment variables and defaults.")
	} else {
		log.Println("API Gateway | Info: Successfully loaded .env file.")
	}

	cfg := &Config{
		ServerPort:             getEnv("API_GATEWAY_PORT", ":8080"),
		UserServiceGRPCURL:   getEnv("USER_SERVICE_GRPC_URL", "localhost:50051"),   // Default for local, Docker will override
		PetServiceGRPCURL:    getEnv("PET_SERVICE_GRPC_URL", "localhost:50052"),     // Default for local, Docker will override
		AdoptionServiceGRPCURL: getEnv("ADOPTION_SERVICE_GRPC_URL", "localhost:50053"), // Default for local, Docker will override
		JWTSecretKey:         getEnv("JWT_SECRET_KEY", "your_default_strong_jwt_secret_key_for_gateway"), // Should match user-service if gateway validates
		GinMode:              getEnv("GIN_MODE", "debug"),                               // Default to debug mode
	}

	// Critical validations
	if cfg.ServerPort == "" {
		log.Fatal("API Gateway | FATAL: API_GATEWAY_PORT environment variable is required.")
	}
	if cfg.UserServiceGRPCURL == "" {
		log.Fatal("API Gateway | FATAL: USER_SERVICE_GRPC_URL environment variable is required.")
	}
	if cfg.PetServiceGRPCURL == "" {
		log.Fatal("API Gateway | FATAL: PET_SERVICE_GRPC_URL environment variable is required.")
	}
	if cfg.AdoptionServiceGRPCURL == "" {
		log.Fatal("API Gateway | FATAL: ADOPTION_SERVICE_GRPC_URL environment variable is required.")
	}
	if cfg.JWTSecretKey == "your_default_strong_jwt_secret_key_for_gateway" || len(cfg.JWTSecretKey) < 32 {
		log.Println("API Gateway | WARNING: JWT_SECRET_KEY is using a default or is too short. Ensure it matches the signing key if validating tokens.")
	}

	return cfg, nil
}

// Helper function to get an environment variable or return a default value.
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	if fallback != "" {
		log.Printf("API Gateway | Info: Environment variable %s not set, using default value: %s", key, fallback)
	} else {
		log.Printf("API Gateway | Warning: Environment variable %s not set and no default value provided.", key)
	}
	return fallback
}