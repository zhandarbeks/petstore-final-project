package main

import (
	"context"
	"log"
	"time"

	"github.com/zhandarbeks/petstore-final-project/user-service/internal/config"
	"github.com/zhandarbeks/petstore-final-project/user-service/internal/handler"
	"github.com/zhandarbeks/petstore-final-project/user-service/internal/repository"
	"github.com/zhandarbeks/petstore-final-project/user-service/internal/server"
	"github.com/zhandarbeks/petstore-final-project/user-service/internal/usecase"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("FATAL: Error loading user-service configuration: %v", err)
	}

	log.Println("User Service | Configuration loaded.")
	log.Printf("User Service | Server Port: %s", cfg.ServerPort)
	log.Printf("User Service | MongoDB URI: %s", cfg.MongoURI)
	log.Printf("User Service | Redis Address: %s, DB: %d", cfg.RedisAddr, cfg.RedisDB)
	log.Printf("User Service | Token Expiry: %v", cfg.TokenExpiry)

	mainCtx, cancelMainCtx := context.WithCancel(context.Background())
	defer cancelMainCtx()

	initTimeout := 15 * time.Second

	mongoInitCtx, mongoCancel := context.WithTimeout(mainCtx, initTimeout)
	defer mongoCancel()
	userMongoRepo, err := repository.NewMongoDBUserRepository(mongoInitCtx, cfg.MongoURI, "petstore_users", "users") // DB name "petstore_users", collection "users"
	if err != nil {
		log.Fatalf("FATAL: Failed to initialize MongoDB repository: %v", err)
	}
	log.Println("User Service | MongoDB repository initialized.")

	if r, ok := userMongoRepo.(interface{ Close(ctx context.Context) error }); ok {
		defer func() {
			log.Println("User Service | Closing MongoDB connection...")
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			if err := r.Close(shutdownCtx); err != nil {
				log.Printf("Error closing MongoDB connection: %v", err)
			} else {
				log.Println("User Service | MongoDB connection closed.")
			}
		}()
	}

	//redis
	redisInitCtx, redisCancel := context.WithTimeout(mainCtx, initTimeout)
	defer redisCancel()
	userRedisCache, err := repository.NewRedisUserCache(redisInitCtx, cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB, "usercache:") // "usercache:" as key prefix
	if err != nil {
		log.Fatalf("FATAL: Failed to initialize Redis cache: %v", err)
	}
	log.Println("User Service | Redis cache initialized.")

	if c, ok := userRedisCache.(interface{ Close() error }); ok {
		defer func() {
			log.Println("User Service | Closing Redis connection...")
			if err := c.Close(); err != nil {
				log.Printf("Error closing Redis connection: %v", err)
			} else {
				log.Println("User Service | Redis connection closed.")
			}
		}()
	}

	userUsecase := usecase.NewUserUsecase(userMongoRepo, userRedisCache, cfg.JWTSecretKey, cfg.TokenExpiry)
	log.Println("User Service | Usecase layer initialized.")

	userGRPCHandler := handler.NewUserHandler(userUsecase)
	log.Println("User Service | gRPC handler initialized.")

	grpcServer, err := server.NewGRPCServer(cfg.ServerPort, userGRPCHandler)
	if err != nil {
		log.Fatalf("FATAL: Failed to create gRPC server: %v", err)
	}

	log.Println("User Service | Starting up...")
	grpcServer.RunWithGracefulShutdown()

	log.Println("User Service | Shut down gracefully.")
}
