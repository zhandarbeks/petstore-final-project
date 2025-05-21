package main

import (
	"context"
	"log"
	"time"

	// Adjust these import paths to match your project's module path and structure
	"github.com/zhandarbeks/petstore-final-project/user-service/internal/config"
	"github.com/zhandarbeks/petstore-final-project/user-service/internal/handler"
	"github.com/zhandarbeks/petstore-final-project/user-service/internal/repository"
	"github.com/zhandarbeks/petstore-final-project/user-service/internal/server"
	"github.com/zhandarbeks/petstore-final-project/user-service/internal/usecase"
)

func main() {
	// 1. Load Configuration
	// The config.Load() function will attempt to load from .env (if using godotenv)
	// and then fall back to environment variables and defaults.
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("FATAL: Error loading user-service configuration: %v", err)
	}

	log.Println("User Service | Configuration loaded.")
	log.Printf("User Service | Server Port: %s", cfg.ServerPort)
	log.Printf("User Service | MongoDB URI: %s", cfg.MongoURI) // Be cautious logging full URIs with credentials in production
	log.Printf("User Service | Redis Address: %s, DB: %d", cfg.RedisAddr, cfg.RedisDB)
	log.Printf("User Service | Token Expiry: %v", cfg.TokenExpiry)
	// log.Printf("User Service | JWT Secret Loaded: %t", cfg.JWTSecretKey != "") // Avoid logging the secret itself

	// Create a main context that can be used to signal shutdown to long-running initializations
	// or background tasks if needed. For now, primarily for initialization timeouts.
	mainCtx, cancelMainCtx := context.WithCancel(context.Background())
	defer cancelMainCtx() // Ensure that the main context is canceled when main exits

	// Define a timeout for critical initializations (DB, Cache)
	initTimeout := 15 * time.Second // Adjust as needed

	// 2. Initialize Database (MongoDB)
	mongoInitCtx, mongoCancel := context.WithTimeout(mainCtx, initTimeout)
	defer mongoCancel()
	userMongoRepo, err := repository.NewMongoDBUserRepository(mongoInitCtx, cfg.MongoURI, "petstore_users", "users") // DB name "petstore_users", collection "users"
	if err != nil {
		log.Fatalf("FATAL: Failed to initialize MongoDB repository: %v", err)
	}
	log.Println("User Service | MongoDB repository initialized.")

	// Defer closing the MongoDB connection.
	// This requires NewMongoDBUserRepository to return a type that has a Close method,
	// or we cast to the concrete type if we know it.
	// Assuming mongoUserRepository has a Close method:
	if r, ok := userMongoRepo.(interface{ Close(ctx context.Context) error }); ok {
		defer func() {
			log.Println("User Service | Closing MongoDB connection...")
			// Use a new context for cleanup if mainCtx might already be canceled by a signal
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			if err := r.Close(shutdownCtx); err != nil {
				log.Printf("Error closing MongoDB connection: %v", err)
			} else {
				log.Println("User Service | MongoDB connection closed.")
			}
		}()
	}

	// 3. Initialize Cache (Redis)
	redisInitCtx, redisCancel := context.WithTimeout(mainCtx, initTimeout)
	defer redisCancel()
	userRedisCache, err := repository.NewRedisUserCache(redisInitCtx, cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB, "usercache:") // "usercache:" as key prefix
	if err != nil {
		log.Fatalf("FATAL: Failed to initialize Redis cache: %v", err)
	}
	log.Println("User Service | Redis cache initialized.")

	// Defer closing the Redis connection.
	// Assuming redisUserCache has a Close method:
	if c, ok := userRedisCache.(interface{ Close() error }); ok {
		defer func() {
			log.Println("User Service | Closing Redis connection...")
			if err := c.Close(); err != nil { // Redis client Close usually doesn't take context
				log.Printf("Error closing Redis connection: %v", err)
			} else {
				log.Println("User Service | Redis connection closed.")
			}
		}()
	}

	// 4. Initialize Usecase
	userUsecase := usecase.NewUserUsecase(userMongoRepo, userRedisCache, cfg.JWTSecretKey, cfg.TokenExpiry)
	log.Println("User Service | Usecase layer initialized.")

	// 5. Initialize gRPC Handler
	userGRPCHandler := handler.NewUserHandler(userUsecase)
	log.Println("User Service | gRPC handler initialized.")

	// 6. Initialize and Start gRPC Server
	grpcServer, err := server.NewGRPCServer(cfg.ServerPort, userGRPCHandler)
	if err != nil {
		log.Fatalf("FATAL: Failed to create gRPC server: %v", err)
	}

	log.Println("User Service | Starting up...")
	// RunWithGracefulShutdown will block until a shutdown signal is received.
	// It handles starting the server and then waiting for SIGINT/SIGTERM.
	grpcServer.RunWithGracefulShutdown()

	log.Println("User Service | Shut down gracefully.")
}
