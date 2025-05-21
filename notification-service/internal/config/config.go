package config

import (
	"log"
	"os"
	"strconv" // For SMTP port

	"github.com/joho/godotenv" // For loading .env files (optional)
)

// Config holds all configuration for the notification-service
type Config struct {
	NatsURL             string // NATS server URL (e.g., "nats://localhost:4222")
	SMTPServer          string // SMTP server address (e.g., "smtp.example.com")
	SMTPPort            int    // SMTP server port (e.g., 587, 465)
	SMTPUsername        string // Username for SMTP authentication
	SMTPPassword        string // Password for SMTP authentication (use App Password for Gmail)
	SMTPSenderEmail     string // The "From" email address for notifications
	UserServiceGRPCURL  string // gRPC URL for the User Service (e.g., "user-service:50051")
	PetServiceGRPCURL   string // gRPC URL for the Pet Service (e.g., "pet-service:50052")
	// Optional: If this service also exposes its own gRPC server (e.g., for health checks)
	// ServerPort string
}

// Load loads configuration. It first attempts to load from a .env file (if present),
// then falls back to actual environment variables, and finally to defaults.
func Load() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		log.Println("Notification Service | Info: No .env file found or error loading .env, relying on environment variables and defaults.")
	} else {
		log.Println("Notification Service | Info: Successfully loaded .env file.")
	}

	cfg := &Config{
		NatsURL:             getEnv("NATS_URL", "nats://localhost:4222"),
		SMTPServer:          getEnv("SMTP_HOST", "smtp.example.com"), // Placeholder, MUST be configured
		SMTPUsername:        getEnv("SMTP_USERNAME", "user@example.com"),  // Placeholder
		SMTPPassword:        getEnv("SMTP_PASSWORD", "your_smtp_password"), // Placeholder, use App Password for Gmail
		SMTPSenderEmail:     getEnv("SENDER_EMAIL", "noreply@petstore.example"),
		UserServiceGRPCURL:  getEnv("USER_SERVICE_GRPC_URL", "localhost:50051"), // Default for local, Docker will override
		PetServiceGRPCURL:   getEnv("PET_SERVICE_GRPC_URL", "localhost:50052"),   // Default for local, Docker will override
		// ServerPort:       getEnv("NOTIFICATION_SERVICE_PORT", ":50054"), // If it has its own gRPC server
	}

	smtpPortStr := getEnv("SMTP_PORT", "587") // Common port for TLS
	smtpPortVal, err := strconv.Atoi(smtpPortStr)
	if err != nil {
		log.Printf("Notification Service | Warning: Invalid SMTP_PORT value: '%s'. Using default 587. Error: %v", smtpPortStr, err)
		cfg.SMTPPort = 587
	} else {
		cfg.SMTPPort = smtpPortVal
	}

	// Critical validations
	if cfg.NatsURL == "" {
		log.Fatal("Notification Service | FATAL: NATS_URL environment variable is required.")
	}
	if cfg.SMTPServer == "smtp.example.com" || cfg.SMTPServer == "" {
		log.Println("Notification Service | WARNING: SMTP_HOST is using a placeholder or is not set. Email sending will likely fail.")
	}
	if cfg.SMTPUsername == "user@example.com" || cfg.SMTPUsername == "" {
		log.Println("Notification Service | WARNING: SMTP_USERNAME is using a placeholder or is not set.")
	}
	if cfg.SMTPPassword == "your_smtp_password" || cfg.SMTPPassword == "" {
		log.Println("Notification Service | WARNING: SMTP_PASSWORD is using a placeholder or is not set.")
	}
	if cfg.UserServiceGRPCURL == "" {
		log.Fatal("Notification Service | FATAL: USER_SERVICE_GRPC_URL environment variable is required.")
	}
	if cfg.PetServiceGRPCURL == "" {
		log.Fatal("Notification Service | FATAL: PET_SERVICE_GRPC_URL environment variable is required.")
	}


	return cfg, nil
}

// Helper function to get an environment variable or return a default value.
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	if fallback != "" {
		log.Printf("Notification Service | Info: Environment variable %s not set, using default value: %s", key, fallback)
	} else {
		log.Printf("Notification Service | Warning: Environment variable %s not set and no default value provided.", key)
	}
	return fallback
}