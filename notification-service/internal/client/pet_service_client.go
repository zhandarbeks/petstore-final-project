package client

import (
	"context"
	"fmt"
	"log"
	"time"

	pbPet "github.com/zhandarbeks/petstore-final-project/genprotos/pet" // Adjust import path to your generated pet protos
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure" // For connecting without TLS (dev environment)
)

// PetServiceClient defines the interface for interacting with the Pet gRPC service.
type PetServiceClient interface {
	GetPetDetails(ctx context.Context, petID string) (*pbPet.Pet, error)
	Close() error
}

// petServiceGRPCClient is the gRPC implementation of PetServiceClient.
type petServiceGRPCClient struct {
	conn   *grpc.ClientConn
	client pbPet.PetServiceClient
}

// NewPetServiceGRPCClient creates a new gRPC client for the Pet Service.
// It takes the target URL of the Pet Service (e.g., "pet-service:50052").
func NewPetServiceGRPCClient(ctx context.Context, targetURL string) (PetServiceClient, error) {
	if targetURL == "" {
		return nil, fmt.Errorf("pet service target URL cannot be empty")
	}

	log.Printf("Notification Service | Attempting to connect to Pet Service gRPC at %s", targetURL)

	conn, err := grpc.DialContext(
		ctx,
		targetURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()), // No TLS for now
		grpc.WithBlock(), // Block until connection is up or context times out
	)
	if err != nil {
		log.Printf("Notification Service | Failed to connect to Pet Service gRPC at %s: %v", targetURL, err)
		return nil, fmt.Errorf("did not connect to pet service: %w", err)
	}
	log.Printf("Notification Service | Successfully connected to Pet Service gRPC at %s", targetURL)

	client := pbPet.NewPetServiceClient(conn)

	return &petServiceGRPCClient{
		conn:   conn,
		client: client,
	}, nil
}

// GetPetDetails fetches pet details from the Pet Service.
func (c *petServiceGRPCClient) GetPetDetails(ctx context.Context, petID string) (*pbPet.Pet, error) {
	if petID == "" {
		return nil, fmt.Errorf("pet ID cannot be empty")
	}

	log.Printf("Notification Service | Calling Pet Service GetPet for PetID: %s", petID)

	req := &pbPet.GetPetRequest{PetId: petID}

	callCtx, cancel := context.WithTimeout(ctx, 5*time.Second) // Example 5-second timeout for the call
	defer cancel()

	res, err := c.client.GetPet(callCtx, req)
	if err != nil {
		log.Printf("Notification Service | Error calling Pet Service GetPet for PetID %s: %v", petID, err)
		return nil, fmt.Errorf("pet service GetPet call failed: %w", err)
	}

	log.Printf("Notification Service | Successfully fetched details for PetID: %s", petID)
	return res.GetPet(), nil
}

// Close closes the gRPC client connection to the Pet Service.
func (c *petServiceGRPCClient) Close() error {
	if c.conn != nil {
		log.Println("Notification Service | Closing Pet Service gRPC client connection...")
		return c.conn.Close()
	}
	return nil
}