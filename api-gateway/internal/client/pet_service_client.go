package client

import (
	"context"
	"fmt"
	"log"
	"time"

	pbPet "github.com/zhandarbeks/petstore-final-project/genprotos/pet" // Adjust import path
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// PetServiceClient defines the interface for the Pet Service client.
type PetServiceClient interface {
	CreatePet(ctx context.Context, req *pbPet.CreatePetRequest) (*pbPet.PetResponse, error)
	GetPet(ctx context.Context, req *pbPet.GetPetRequest) (*pbPet.PetResponse, error)
	UpdatePet(ctx context.Context, req *pbPet.UpdatePetRequest) (*pbPet.PetResponse, error)
	DeletePet(ctx context.Context, req *pbPet.DeletePetRequest) (*pbPet.EmptyResponse, error)
	ListPets(ctx context.Context, req *pbPet.ListPetsRequest) (*pbPet.ListPetsResponse, error)
	UpdatePetAdoptionStatus(ctx context.Context, req *pbPet.UpdatePetAdoptionStatusRequest) (*pbPet.PetResponse, error)
	Close() error
}

type petServiceGRPCClient struct {
	conn   *grpc.ClientConn
	client pbPet.PetServiceClient
}

// NewPetServiceGRPCClient creates a new gRPC client for the Pet Service.
func NewPetServiceGRPCClient(ctx context.Context, targetURL string) (PetServiceClient, error) {
	if targetURL == "" {
		return nil, fmt.Errorf("pet service target URL cannot be empty for API Gateway client")
	}
	log.Printf("API Gateway | Attempting to connect to Pet Service gRPC at %s", targetURL)

	conn, err := grpc.DialContext(
		ctx,
		targetURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(5*time.Second), // Connection timeout
	)
	if err != nil {
		log.Printf("API Gateway | Failed to connect to Pet Service gRPC at %s: %v", targetURL, err)
		return nil, fmt.Errorf("did not connect to pet service: %w", err)
	}
	log.Printf("API Gateway | Successfully connected to Pet Service gRPC at %s", targetURL)
	return &petServiceGRPCClient{
		conn:   conn,
		client: pbPet.NewPetServiceClient(conn),
	}, nil
}

func (c *petServiceGRPCClient) CreatePet(ctx context.Context, req *pbPet.CreatePetRequest) (*pbPet.PetResponse, error) {
	log.Printf("API Gateway | Calling Pet Service CreatePet for name: %s", req.GetName())
	return c.client.CreatePet(ctx, req)
}

func (c *petServiceGRPCClient) GetPet(ctx context.Context, req *pbPet.GetPetRequest) (*pbPet.PetResponse, error) {
	log.Printf("API Gateway | Calling Pet Service GetPet for ID: %s", req.GetPetId())
	return c.client.GetPet(ctx, req)
}

func (c *petServiceGRPCClient) UpdatePet(ctx context.Context, req *pbPet.UpdatePetRequest) (*pbPet.PetResponse, error) {
	log.Printf("API Gateway | Calling Pet Service UpdatePet for ID: %s", req.GetPetId())
	return c.client.UpdatePet(ctx, req)
}

func (c *petServiceGRPCClient) DeletePet(ctx context.Context, req *pbPet.DeletePetRequest) (*pbPet.EmptyResponse, error) {
	log.Printf("API Gateway | Calling Pet Service DeletePet for ID: %s", req.GetPetId())
	return c.client.DeletePet(ctx, req)
}

func (c *petServiceGRPCClient) ListPets(ctx context.Context, req *pbPet.ListPetsRequest) (*pbPet.ListPetsResponse, error) {
	log.Printf("API Gateway | Calling Pet Service ListPets. Page: %d, Limit: %d", req.GetPage(), req.GetLimit())
	return c.client.ListPets(ctx, req)
}

func (c *petServiceGRPCClient) UpdatePetAdoptionStatus(ctx context.Context, req *pbPet.UpdatePetAdoptionStatusRequest) (*pbPet.PetResponse, error) {
	log.Printf("API Gateway | Calling Pet Service UpdatePetAdoptionStatus for ID: %s, NewStatus: %s", req.GetPetId(), req.GetNewStatus().String())
	return c.client.UpdatePetAdoptionStatus(ctx, req)
}

func (c *petServiceGRPCClient) Close() error {
	if c.conn != nil {
		log.Println("API Gateway | Closing Pet Service gRPC client connection...")
		return c.conn.Close()
	}
	return nil
}