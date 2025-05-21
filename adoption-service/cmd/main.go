package main

import (
	"context"
	"log"
	"time"

	// Adjust these import paths to match your project's module path and structure
	"github.com/zhandarbeks/petstore-final-project/adoption-service/internal/config"
	"github.com/zhandarbeks/petstore-final-project/adoption-service/internal/handler"
	"github.com/zhandarbeks/petstore-final-project/adoption-service/internal/publisher"
	"github.com/zhandarbeks/petstore-final-project/adoption-service/internal/repository"
	"github.com/zhandarbeks/petstore-final-project/adoption-service/internal/server" // Using the server package
	"github.com/zhandarbeks/petstore-final-project/adoption-service/internal/usecase"
)

func main() {
	// 1. Load Adoption Service Configuration
	cfg, err := config.Load() // This will be adoption-service/internal/config.Load()
	if err != nil {
		log.Fatalf("Adoption Service | FATAL: Error loading configuration: %v", err)
	}

	log.Println("Adoption Service | Configuration loaded.")
	log.Printf("Adoption Service | Server Port: %s", cfg.ServerPort)
	log.Printf("Adoption Service | MongoDB URI: %s", cfg.MongoURI)
	log.Printf("Adoption Service | Redis Address: %s, DB: %d", cfg.RedisAddr, cfg.RedisDB)
	log.Printf("Adoption Service | NATS URL: %s", cfg.NatsURL)

	// Create a main context that can be used to signal shutdown
	mainCtx, cancelMainCtx := context.WithCancel(context.Background())
	defer cancelMainCtx()

	initTimeout := 15 * time.Second // Timeout for critical initializations

	// 2. Initialize Adoption Database (MongoDB)
	mongoInitCtx, mongoCancel := context.WithTimeout(mainCtx, initTimeout)
	defer mongoCancel()
	// Using "petstore_adoptions" as DB name and "applications" as collection name
	adoptionMongoRepo, err := repository.NewMongoDBAdoptionRepository(mongoInitCtx, cfg.MongoURI, "petstore_adoptions", "applications")
	if err != nil {
		log.Fatalf("Adoption Service | FATAL: Failed to initialize MongoDB repository: %v", err)
	}
	log.Println("Adoption Service | MongoDB repository initialized.")

	if r, ok := adoptionMongoRepo.(interface{ Close(ctx context.Context) error }); ok {
		defer func() {
			log.Println("Adoption Service | Closing MongoDB connection...")
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			if err := r.Close(shutdownCtx); err != nil {
				log.Printf("Adoption Service | Error closing MongoDB connection: %v", err)
			} else {
				log.Println("Adoption Service | MongoDB connection closed.")
			}
		}()
	}

	// 3. Initialize Adoption Cache (Redis)
	redisInitCtx, redisCancel := context.WithTimeout(mainCtx, initTimeout)
	defer redisCancel()
	adoptionRedisCache, err := repository.NewRedisAdoptionCache(redisInitCtx, cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB, "adoptioncache:")
	if err != nil {
		log.Fatalf("Adoption Service | FATAL: Failed to initialize Redis cache: %v", err)
	}
	log.Println("Adoption Service | Redis cache initialized.")

	if c, ok := adoptionRedisCache.(interface{ Close() error }); ok {
		defer func() {
			log.Println("Adoption Service | Closing Redis connection...")
			if err := c.Close(); err != nil {
				log.Printf("Adoption Service | Error closing Redis connection: %v", err)
			} else {
				log.Println("Adoption Service | Redis connection closed.")
			}
		}()
	}

	// 4. Initialize NATS Publisher
	natsPublisher, err := publisher.NewNATSAdoptionPublisher(cfg.NatsURL)
	if err != nil {
		log.Fatalf("Adoption Service | FATAL: Failed to initialize NATS publisher: %v", err)
	}
	log.Println("Adoption Service | NATS publisher initialized.")
	defer natsPublisher.Close() // Ensure NATS connection is closed on shutdown

	// 5. Initialize Adoption Usecase
	// If your usecase needs clients to other services (e.g., pet-service), initialize them here and pass them in.
	adoptionUsecase := usecase.NewAdoptionUsecase(adoptionMongoRepo, adoptionRedisCache, natsPublisher)
	log.Println("Adoption Service | Usecase layer initialized.")

	// 6. Initialize Adoption gRPC Handler
	adoptionGRPCHandler := handler.NewAdoptionHandler(adoptionUsecase)
	log.Println("Adoption Service | gRPC handler initialized.")

	// 7. Initialize and Start Adoption gRPC Server
	// This uses the server.NewGRPCServer from adoption-service/internal/server/grpc_server.go (the selected code in Canvas)
	grpcServer, err := server.NewGRPCServer(cfg.ServerPort, adoptionGRPCHandler)
	if err != nil {
		log.Fatalf("Adoption Service | FATAL: Failed to create gRPC server: %v", err)
	}

	log.Println("Adoption Service | Starting up...")
	grpcServer.RunWithGracefulShutdown() // This will block until a shutdown signal is received

	log.Println("Adoption Service | Shut down gracefully.")
}