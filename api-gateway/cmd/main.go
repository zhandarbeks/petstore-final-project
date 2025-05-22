package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zhandarbeks/petstore-final-project/api-gateway/internal/client"
	"github.com/zhandarbeks/petstore-final-project/api-gateway/internal/config"
	"github.com/zhandarbeks/petstore-final-project/api-gateway/internal/handler"
	"github.com/zhandarbeks/petstore-final-project/api-gateway/internal/router"
)

func main() {
	// 1. Load API Gateway Configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("API Gateway | FATAL: Error loading configuration: %v", err)
	}

	log.Println("API Gateway | Configuration loaded.")
	log.Printf("API Gateway | Server Port: %s", cfg.ServerPort)
	log.Printf("API Gateway | User Service URL: %s", cfg.UserServiceGRPCURL)
	log.Printf("API Gateway | Pet Service URL: %s", cfg.PetServiceGRPCURL)
	log.Printf("API Gateway | Adoption Service URL: %s", cfg.AdoptionServiceGRPCURL)
	log.Printf("API Gateway | Gin Mode: %s", cfg.GinMode)

	// Set Gin mode
	gin.SetMode(cfg.GinMode)

	// Create a main context that can be used to signal shutdown
	mainCtx, cancelMainCtx := context.WithCancel(context.Background())
	defer cancelMainCtx()

	initTimeout := 10 * time.Second // Timeout for client initializations

	// 2. Initialize gRPC Clients
	userClientInitCtx, userClientCancel := context.WithTimeout(mainCtx, initTimeout)
	defer userClientCancel()
	userServiceClient, err := client.NewUserServiceGRPCClient(userClientInitCtx, cfg.UserServiceGRPCURL)
	if err != nil {
		log.Fatalf("API Gateway | FATAL: Failed to initialize User Service gRPC client: %v", err)
	}
	log.Println("API Gateway | User Service gRPC client initialized.")
	defer func() {
		log.Println("API Gateway | Closing User Service gRPC client connection...")
		if err := userServiceClient.Close(); err != nil {
			log.Printf("API Gateway | Error closing User Service gRPC client: %v", err)
		}
	}()

	petClientInitCtx, petClientCancel := context.WithTimeout(mainCtx, initTimeout)
	defer petClientCancel()
	petServiceClient, err := client.NewPetServiceGRPCClient(petClientInitCtx, cfg.PetServiceGRPCURL)
	if err != nil {
		log.Fatalf("API Gateway | FATAL: Failed to initialize Pet Service gRPC client: %v", err)
	}
	log.Println("API Gateway | Pet Service gRPC client initialized.")
	defer func() {
		log.Println("API Gateway | Closing Pet Service gRPC client connection...")
		if err := petServiceClient.Close(); err != nil {
			log.Printf("API Gateway | Error closing Pet Service gRPC client: %v", err)
		}
	}()

	adoptionClientInitCtx, adoptionClientCancel := context.WithTimeout(mainCtx, initTimeout)
	defer adoptionClientCancel()
	adoptionServiceClient, err := client.NewAdoptionServiceGRPCClient(adoptionClientInitCtx, cfg.AdoptionServiceGRPCURL)
	if err != nil {
		log.Fatalf("API Gateway | FATAL: Failed to initialize Adoption Service gRPC client: %v", err)
	}
	log.Println("API Gateway | Adoption Service gRPC client initialized.")
	defer func() {
		log.Println("API Gateway | Closing Adoption Service gRPC client connection...")
		if err := adoptionServiceClient.Close(); err != nil {
			log.Printf("API Gateway | Error closing Adoption Service gRPC client: %v", err)
		}
	}()

	// 3. Initialize HTTP Handlers (injecting gRPC clients)
	userHandler := handler.NewUserHandler(userServiceClient)
	petHandler := handler.NewPetHandler(petServiceClient)
	adoptionHandler := handler.NewAdoptionHandler(adoptionServiceClient)
	log.Println("API Gateway | HTTP handlers initialized.")

	// 4. Initialize Gin Router (injecting handlers)
	// Since authMiddleware is not implemented yet, we pass nil or don't include it in New's signature.
	// For now, assuming router.New doesn't strictly require authMiddleware if it's not used.
	// If router.New expects it, we'd pass a dummy or nil.
	// Based on the router.go in Canvas (ID: api_gateway_router_go), it doesn't require it.
	r := router.New(userHandler, petHandler, adoptionHandler)
	log.Println("API Gateway | Gin router initialized.")

	// 5. Start HTTP Server
	srv := &http.Server{
		Addr:    cfg.ServerPort,
		Handler: r,
	}

	// Goroutine for graceful shutdown
	go func() {
		log.Printf("API Gateway | Starting HTTP server on port %s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("API Gateway | ListenAndServe error: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("API Gateway | Received signal: %v. Shutting down HTTP server...", sig)

	// Create a context with timeout for the shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second) // 10-second timeout for shutdown
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("API Gateway | Server forced to shutdown: %v", err)
	}

	// Cancel the main context to signal gRPC client connection closures if they are long-lived
	cancelMainCtx()

	log.Println("API Gateway | Server exiting.")
}