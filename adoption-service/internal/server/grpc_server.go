package server

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/zhandarbeks/petstore-final-project/genprotos/adoption" // Adjust import path
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// GRPCServer holds the gRPC server instance and listener for the Adoption Service.
type GRPCServer struct {
	server   *grpc.Server
	listener net.Listener
	Port     string
}

// NewGRPCServer creates and configures a new gRPC server instance for the Adoption Service.
// It takes the port string (e.g., ":50053") and the AdoptionServiceServer implementation.
func NewGRPCServer(port string, adoptionService pb.AdoptionServiceServer) (*GRPCServer, error) {
	if port == "" {
		return nil, fmt.Errorf("port cannot be empty for Adoption Service gRPC server")
	}
	if adoptionService == nil {
		return nil, fmt.Errorf("adoptionService (handler) cannot be nil for Adoption Service")
	}

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Printf("Adoption Service | Failed to listen on port %s: %v", port, err)
		return nil, fmt.Errorf("failed to listen on port %s: %w", port, err)
	}

	s := grpc.NewServer(
	// Add interceptors here if needed in the future
	)

	// Register your adoption service implementation.
	pb.RegisterAdoptionServiceServer(s, adoptionService)

	// Register reflection service.
	reflection.Register(s)

	// Register gRPC Health Checking Protocol service.
	healthService := health.NewServer()
	grpc_health_v1.RegisterHealthServer(s, healthService)
	healthService.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	// healthService.SetServingStatus("adoption.AdoptionService", grpc_health_v1.HealthCheckResponse_SERVING)


	log.Printf("Adoption Service | gRPC server configured to listen on port %s", port)

	return &GRPCServer{
		server:   s,
		listener: lis,
		Port:     port,
	}, nil
}

// Start runs the gRPC server for the Adoption Service.
func (gs *GRPCServer) Start() error {
	log.Printf("Adoption Service | Starting gRPC server on %s...", gs.Port)
	if err := gs.server.Serve(gs.listener); err != nil {
		log.Printf("Adoption Service | Failed to serve gRPC: %v", err)
		return fmt.Errorf("failed to serve gRPC: %w", err)
	}
	return nil
}

// Stop gracefully shuts down the gRPC server for the Adoption Service.
func (gs *GRPCServer) Stop() {
	log.Println("Adoption Service | Attempting to gracefully stop gRPC server...")
	gs.server.GracefulStop()
	log.Println("Adoption Service | gRPC server stopped.")
}

// RunWithGracefulShutdown starts the Adoption Service server and handles OS signals.
func (gs *GRPCServer) RunWithGracefulShutdown() {
	go func() {
		if err := gs.Start(); err != nil {
			log.Printf("Adoption Service | Error starting gRPC server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("Adoption Service | Received signal: %v. Shutting down gRPC server...", sig)

	gs.Stop()
}