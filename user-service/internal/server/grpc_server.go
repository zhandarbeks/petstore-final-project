package server

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/zhandarbeks/petstore-final-project/genprotos/user" // Adjust import path to your generated protos
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection" // For Evans CLI, grpcurl, etc.
)

// GRPCServer holds the gRPC server instance and listener.
type GRPCServer struct {
	server   *grpc.Server
	listener net.Listener
	Port     string
}

// NewGRPCServer creates and configures a new gRPC server instance.
// It takes the port string (e.g., ":50051") and the UserServiceServer implementation.
func NewGRPCServer(port string, userService pb.UserServiceServer) (*GRPCServer, error) {
	if port == "" {
		return nil, fmt.Errorf("port cannot be empty")
	}
	if userService == nil {
		return nil, fmt.Errorf("userService (handler) cannot be nil")
	}

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Printf("Failed to listen on port %s: %v", port, err)
		return nil, fmt.Errorf("failed to listen on port %s: %w", port, err)
	}

	// Create a new gRPC server with options (e.g., interceptors if needed later)
	s := grpc.NewServer(
	// grpc.UnaryInterceptor(yourUnaryInterceptor), // Example for adding interceptors
	// grpc.StreamInterceptor(yourStreamInterceptor),
	)

	// Register your user service implementation with the gRPC server.
	pb.RegisterUserServiceServer(s, userService)

	// Register reflection service on gRPC server.
	// This allows tools like Evans CLI and grpcurl to introspect the server.
	reflection.Register(s)

	// Register gRPC Health Checking Protocol service.
	healthService := health.NewServer()
	grpc_health_v1.RegisterHealthServer(s, healthService)
	// Initially, set the overall server status to SERVING.
	// You can set individual services' statuses later if needed.
	healthService.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	// Example for a specific service: healthService.SetServingStatus("user.UserService", grpc_health_v1.HealthCheckResponse_SERVING)


	log.Printf("gRPC server configured to listen on port %s", port)

	return &GRPCServer{
		server:   s,
		listener: lis,
		Port:     port,
	}, nil
}

// Start runs the gRPC server.
// This function will block until the server is stopped.
func (gs *GRPCServer) Start() error {
	log.Printf("Starting gRPC server on %s...", gs.Port)
	if err := gs.server.Serve(gs.listener); err != nil {
		// Serve() always returns a non-nil error.
		// os.Exit or graceful shutdown handles server stopping.
		log.Printf("Failed to serve gRPC: %v", err)
		return fmt.Errorf("failed to serve gRPC: %w", err)
	}
	return nil // Should not be reached if Serve() blocks indefinitely
}

// Stop gracefully shuts down the gRPC server.
func (gs *GRPCServer) Stop() {
	log.Println("Attempting to gracefully stop gRPC server...")
	gs.server.GracefulStop()
	log.Println("gRPC server stopped.")
}

// RunWithGracefulShutdown starts the server and handles OS signals for graceful shutdown.
func (gs *GRPCServer) RunWithGracefulShutdown() {
	// Start the server in a new goroutine
	go func() {
		if err := gs.Start(); err != nil {
			// This error is expected when the server is stopped (e.g. by GracefulStop or listener closing)
			// So, we only log if it's not due to a graceful shutdown scenario.
			// However, Serve() blocks, so this log might only appear if Start() itself returns an immediate error.
			log.Printf("Error starting gRPC server: %v", err)
			// Consider a mechanism to signal the main goroutine about this fatal error.
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	// syscall.SIGINT: Terminal interrupt (Ctrl+C)
	// syscall.SIGTERM: Termination signal (often sent by orchestrators like Docker/Kubernetes)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Block until a signal is received
	sig := <-quit
	log.Printf("Received signal: %v. Shutting down gRPC server...", sig)

	gs.Stop()
}
