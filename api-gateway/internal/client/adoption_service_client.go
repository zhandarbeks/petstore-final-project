package client

import (
	"context"
	"fmt"
	"log"
	"time"

	pbAdoption "github.com/zhandarbeks/petstore-final-project/genprotos/adoption" // Adjust import path
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// AdoptionServiceClient defines the interface for the Adoption Service client.
type AdoptionServiceClient interface {
	CreateAdoptionApplication(ctx context.Context, req *pbAdoption.CreateAdoptionApplicationRequest) (*pbAdoption.AdoptionApplicationResponse, error)
	GetAdoptionApplication(ctx context.Context, req *pbAdoption.GetAdoptionApplicationRequest) (*pbAdoption.AdoptionApplicationResponse, error)
	UpdateAdoptionApplicationStatus(ctx context.Context, req *pbAdoption.UpdateAdoptionApplicationStatusRequest) (*pbAdoption.AdoptionApplicationResponse, error)
	ListUserAdoptionApplications(ctx context.Context, req *pbAdoption.ListUserAdoptionApplicationsRequest) (*pbAdoption.ListAdoptionApplicationsResponse, error)
	Close() error
}

type adoptionServiceGRPCClient struct {
	conn   *grpc.ClientConn
	client pbAdoption.AdoptionServiceClient
}

// NewAdoptionServiceGRPCClient creates a new gRPC client for the Adoption Service.
func NewAdoptionServiceGRPCClient(ctx context.Context, targetURL string) (AdoptionServiceClient, error) {
	if targetURL == "" {
		return nil, fmt.Errorf("adoption service target URL cannot be empty for API Gateway client")
	}
	log.Printf("API Gateway | Attempting to connect to Adoption Service gRPC at %s", targetURL)

	conn, err := grpc.DialContext(
		ctx,
		targetURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(5*time.Second), // Connection timeout
	)
	if err != nil {
		log.Printf("API Gateway | Failed to connect to Adoption Service gRPC at %s: %v", targetURL, err)
		return nil, fmt.Errorf("did not connect to adoption service: %w", err)
	}
	log.Printf("API Gateway | Successfully connected to Adoption Service gRPC at %s", targetURL)
	return &adoptionServiceGRPCClient{
		conn:   conn,
		client: pbAdoption.NewAdoptionServiceClient(conn),
	}, nil
}

func (c *adoptionServiceGRPCClient) CreateAdoptionApplication(ctx context.Context, req *pbAdoption.CreateAdoptionApplicationRequest) (*pbAdoption.AdoptionApplicationResponse, error) {
	log.Printf("API Gateway | Calling Adoption Service CreateAdoptionApplication for UserID: %s, PetID: %s", req.GetUserId(), req.GetPetId())
	return c.client.CreateAdoptionApplication(ctx, req)
}

func (c *adoptionServiceGRPCClient) GetAdoptionApplication(ctx context.Context, req *pbAdoption.GetAdoptionApplicationRequest) (*pbAdoption.AdoptionApplicationResponse, error) {
	log.Printf("API Gateway | Calling Adoption Service GetAdoptionApplication for ID: %s", req.GetApplicationId())
	return c.client.GetAdoptionApplication(ctx, req)
}

func (c *adoptionServiceGRPCClient) UpdateAdoptionApplicationStatus(ctx context.Context, req *pbAdoption.UpdateAdoptionApplicationStatusRequest) (*pbAdoption.AdoptionApplicationResponse, error) {
	log.Printf("API Gateway | Calling Adoption Service UpdateAdoptionApplicationStatus for ID: %s, NewStatus: %s", req.GetApplicationId(), req.GetNewStatus().String())
	return c.client.UpdateAdoptionApplicationStatus(ctx, req)
}

func (c *adoptionServiceGRPCClient) ListUserAdoptionApplications(ctx context.Context, req *pbAdoption.ListUserAdoptionApplicationsRequest) (*pbAdoption.ListAdoptionApplicationsResponse, error) {
	log.Printf("API Gateway | Calling Adoption Service ListUserAdoptionApplications for UserID: %s", req.GetUserId())
	return c.client.ListUserAdoptionApplications(ctx, req)
}

func (c *adoptionServiceGRPCClient) Close() error {
	if c.conn != nil {
		log.Println("API Gateway | Closing Adoption Service gRPC client connection...")
		return c.conn.Close()
	}
	return nil
}