package client

import (
	"context"
	"fmt"
	"log"
	"time"

	pbUser "github.com/zhandarbeks/petstore-final-project/genprotos/user" // Adjust import path to your generated user protos
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure" // For connecting without TLS (dev environment)
)

// UserServiceClient defines the interface for interacting with the User gRPC service.
// This helps in mocking the client for testing purposes.
type UserServiceClient interface {
	GetUserDetails(ctx context.Context, userID string) (*pbUser.User, error)
	Close() error
}

// userServiceGRPCClient is the gRPC implementation of UserServiceClient.
type userServiceGRPCClient struct {
	conn   *grpc.ClientConn
	client pbUser.UserServiceClient
}

// NewUserServiceGRPCClient creates a new gRPC client for the User Service.
// It takes the target URL of the User Service (e.g., "user-service:50051").
func NewUserServiceGRPCClient(ctx context.Context, targetURL string) (UserServiceClient, error) {
	if targetURL == "" {
		return nil, fmt.Errorf("user service target URL cannot be empty")
	}

	log.Printf("Notification Service | Attempting to connect to User Service gRPC at %s", targetURL)

	// Create a gRPC client connection.
	// For local/dev, using insecure credentials. In production, use TLS.
	// Adding DialOptions for non-blocking connect with timeout.
	conn, err := grpc.DialContext(
		ctx,
		targetURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()), // No TLS for now
		grpc.WithBlock(), // Block until connection is up or context times out
	)
	if err != nil {
		log.Printf("Notification Service | Failed to connect to User Service gRPC at %s: %v", targetURL, err)
		return nil, fmt.Errorf("did not connect to user service: %w", err)
	}
	log.Printf("Notification Service | Successfully connected to User Service gRPC at %s", targetURL)

	client := pbUser.NewUserServiceClient(conn)

	return &userServiceGRPCClient{
		conn:   conn,
		client: client,
	}, nil
}

// GetUserDetails fetches user details from the User Service.
func (c *userServiceGRPCClient) GetUserDetails(ctx context.Context, userID string) (*pbUser.User, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	log.Printf("Notification Service | Calling User Service GetUser for UserID: %s", userID)

	req := &pbUser.GetUserRequest{UserId: userID}

	// Add a timeout to the context for this specific call, if not already present or too long.
	callCtx, cancel := context.WithTimeout(ctx, 5*time.Second) // Example 5-second timeout for the call
	defer cancel()

	res, err := c.client.GetUser(callCtx, req)
	if err != nil {
		log.Printf("Notification Service | Error calling User Service GetUser for UserID %s: %v", userID, err)
		return nil, fmt.Errorf("user service GetUser call failed: %w", err)
	}

	log.Printf("Notification Service | Successfully fetched details for UserID: %s", userID)
	return res.GetUser(), nil
}

// Close closes the gRPC client connection to the User Service.
func (c *userServiceGRPCClient) Close() error {
	if c.conn != nil {
		log.Println("Notification Service | Closing User Service gRPC client connection...")
		return c.conn.Close()
	}
	return nil
}