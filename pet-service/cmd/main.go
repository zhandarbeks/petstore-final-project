package main

import (
	"context"
	"log"
	"time"

	// Adjust these import paths to match your project's module path and structure
	"github.com/zhandarbeks/petstore-final-project/pet-service/internal/config"
	"github.com/zhandarbeks/petstore-final-project/pet-service/internal/handler"
	"github.com/zhandarbeks/petstore-final-project/pet-service/internal/repository"
	"github.com/zhandarbeks/petstore-final-project/pet-service/internal/server" // Using the server package we defined
	"github.com/zhandarbeks/petstore-final-project/pet-service/internal/usecase"
)

func main() {
	// 1. Load Pet Service Configuration
	cfg, err := config.Load() // This will be pet-service/internal/config.Load()
	if err != nil {
		log.Fatalf("Pet Service | FATAL: Error loading configuration: %v", err)
	}

	log.Println("Pet Service | Configuration loaded.")
	log.Printf("Pet Service | Server Port: %s", cfg.ServerPort)
	log.Printf("Pet Service | MongoDB URI: %s", cfg.MongoURI) // Be cautious logging full URIs with credentials in production
	log.Printf("Pet Service | Redis Address: %s, DB: %d", cfg.RedisAddr, cfg.RedisDB)

	// Create a main context that can be used to signal shutdown
	mainCtx, cancelMainCtx := context.WithCancel(context.Background())
	defer cancelMainCtx() // Ensure that the main context is canceled when main exits

	// Define a timeout for critical initializations
	initTimeout := 15 * time.Second

	// 2. Initialize Pet Database (MongoDB)
	mongoInitCtx, mongoCancel := context.WithTimeout(mainCtx, initTimeout)
	defer mongoCancel()
	// Using "petstore_pets" as DB name and "pets" as collection name, adjust if needed or move to config
	petMongoRepo, err := repository.NewMongoDBPetRepository(mongoInitCtx, cfg.MongoURI, "petstore_pets", "pets")
	if err != nil {
		log.Fatalf("Pet Service | FATAL: Failed to initialize MongoDB repository: %v", err)
	}
	log.Println("Pet Service | MongoDB repository initialized.")

	// Defer closing the MongoDB connection.
	if r, ok := petMongoRepo.(interface{ Close(ctx context.Context) error }); ok {
		defer func() {
			log.Println("Pet Service | Closing MongoDB connection...")
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			if err := r.Close(shutdownCtx); err != nil {
				log.Printf("Pet Service | Error closing MongoDB connection: %v", err)
			} else {
				log.Println("Pet Service | MongoDB connection closed.")
			}
		}()
	}

	// 3. Initialize Pet Cache (Redis)
	redisInitCtx, redisCancel := context.WithTimeout(mainCtx, initTimeout)
	defer redisCancel()
	// Using "petcache:" as key prefix, adjust if needed
	petRedisCache, err := repository.NewRedisPetCache(redisInitCtx, cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB, "petcache:")
	if err != nil {
		log.Fatalf("Pet Service | FATAL: Failed to initialize Redis cache: %v", err)
	}
	log.Println("Pet Service | Redis cache initialized.")

	if c, ok := petRedisCache.(interface{ Close() error }); ok {
		defer func() {
			log.Println("Pet Service | Closing Redis connection...")
			if err := c.Close(); err != nil { // Redis client Close usually doesn't take context
				log.Printf("Pet Service | Error closing Redis connection: %v", err)
			} else {
				log.Println("Pet Service | Redis connection closed.")
			}
		}()
	}

	// 4. Initialize Pet Usecase
	petUsecase := usecase.NewPetUsecase(petMongoRepo, petRedisCache)
	log.Println("Pet Service | Usecase layer initialized.")

	// 5. Initialize Pet gRPC Handler
	petGRPCHandler := handler.NewPetHandler(petUsecase)
	log.Println("Pet Service | gRPC handler initialized.")

	// 6. Initialize and Start Pet gRPC Server
	// This uses the server.NewGRPCServer from pet-service/internal/server/grpc_server.go
	grpcServer, err := server.NewGRPCServer(cfg.ServerPort, petGRPCHandler)
	if err != nil {
		log.Fatalf("Pet Service | FATAL: Failed to create gRPC server: %v", err)
	}

	log.Println("Pet Service | Starting up...")
	// RunWithGracefulShutdown will block until a shutdown signal is received.
	grpcServer.RunWithGracefulShutdown()

	log.Println("Pet Service | Shut down gracefully.")
}