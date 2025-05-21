package server

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/zhandarbeks/petstore-final-project/genprotos/pet" // Adjust import path to your generated pet protos
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection" // For Evans CLI, grpcurl, etc.
)

// GRPCServer holds the gRPC server instance and listener for the Pet Service.
type GRPCServer struct {
	server   *grpc.Server
	listener net.Listener
	Port     string
}

// NewGRPCServer creates and configures a new gRPC server instance for the Pet Service.
// It takes the port string (e.g., ":50052") and the PetServiceServer implementation.
func NewGRPCServer(port string, petService pb.PetServiceServer) (*GRPCServer, error) {
	if port == "" {
		return nil, fmt.Errorf("port cannot be empty for Pet Service gRPC server")
	}
	if petService == nil {
		return nil, fmt.Errorf("petService (handler) cannot be nil for Pet Service")
	}

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Printf("Pet Service | Failed to listen on port %s: %v", port, err)
		return nil, fmt.Errorf("failed to listen on port %s: %w", port, err)
	}

	// Create a new gRPC server
	s := grpc.NewServer(
	// Add interceptors here if needed in the future
	// grpc.UnaryInterceptor(yourUnaryInterceptor),
	)

	// Register your pet service implementation with the gRPC server.
	pb.RegisterPetServiceServer(s, petService)

	// Register reflection service on gRPC server.
	reflection.Register(s)

	// Register gRPC Health Checking Protocol service.
	healthService := health.NewServer()
	grpc_health_v1.RegisterHealthServer(s, healthService)
	// Set the overall server status to SERVING.
	healthService.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	// Optionally set status for the specific PetService
	// healthService.SetServingStatus("pet.PetService", grpc_health_v1.HealthCheckResponse_SERVING)

	log.Printf("Pet Service | gRPC server configured to listen on port %s", port)

	return &GRPCServer{
		server:   s,
		listener: lis,
		Port:     port,
	}, nil
}

// Start runs the gRPC server for the Pet Service.
// This function will block until the server is stopped.
func (gs *GRPCServer) Start() error {
	log.Printf("Pet Service | Starting gRPC server on %s...", gs.Port)
	if err := gs.server.Serve(gs.listener); err != nil {
		log.Printf("Pet Service | Failed to serve gRPC: %v", err)
		return fmt.Errorf("failed to serve gRPC: %w", err)
	}
	return nil
}

// Stop gracefully shuts down the gRPC server for the Pet Service.
func (gs *GRPCServer) Stop() {
	log.Println("Pet Service | Attempting to gracefully stop gRPC server...")
	gs.server.GracefulStop()
	log.Println("Pet Service | gRPC server stopped.")
}

// RunWithGracefulShutdown starts the Pet Service server and handles OS signals for graceful shutdown.
func (gs *GRPCServer) RunWithGracefulShutdown() {
	// Start the server in a new goroutine
	go func() {
		if err := gs.Start(); err != nil {
			log.Printf("Pet Service | Error starting gRPC server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Block until a signal is received
	sig := <-quit
	log.Printf("Pet Service | Received signal: %v. Shutting down gRPC server...", sig)

	gs.Stop()
}