package client

import (
	"context"
	"fmt"
	"log"
	"time"

	pbUser "github.com/zhandarbeks/petstore-final-project/genprotos/user" // Adjust import path
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// UserServiceClient defines the interface for the User Service client.
type UserServiceClient interface {
	RegisterUser(ctx context.Context, req *pbUser.RegisterUserRequest) (*pbUser.UserResponse, error)
	LoginUser(ctx context.Context, req *pbUser.LoginUserRequest) (*pbUser.LoginUserResponse, error)
	GetUser(ctx context.Context, req *pbUser.GetUserRequest) (*pbUser.UserResponse, error)
	UpdateUserProfile(ctx context.Context, req *pbUser.UpdateUserProfileRequest) (*pbUser.UserResponse, error)
	DeleteUser(ctx context.Context, req *pbUser.DeleteUserRequest) (*pbUser.EmptyResponse, error)
	Close() error
}

type userServiceGRPCClient struct {
	conn   *grpc.ClientConn
	client pbUser.UserServiceClient
}

// NewUserServiceGRPCClient creates a new gRPC client for the User Service.
func NewUserServiceGRPCClient(ctx context.Context, targetURL string) (UserServiceClient, error) {
	if targetURL == "" {
		return nil, fmt.Errorf("user service target URL cannot be empty for API Gateway client")
	}
	log.Printf("API Gateway | Attempting to connect to User Service gRPC at %s", targetURL)

	conn, err := grpc.DialContext(
		ctx,
		targetURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(5*time.Second), // Connection timeout
	)
	if err != nil {
		log.Printf("API Gateway | Failed to connect to User Service gRPC at %s: %v", targetURL, err)
		return nil, fmt.Errorf("did not connect to user service: %w", err)
	}
	log.Printf("API Gateway | Successfully connected to User Service gRPC at %s", targetURL)
	return &userServiceGRPCClient{
		conn:   conn,
		client: pbUser.NewUserServiceClient(conn),
	}, nil
}

func (c *userServiceGRPCClient) RegisterUser(ctx context.Context, req *pbUser.RegisterUserRequest) (*pbUser.UserResponse, error) {
	log.Printf("API Gateway | Calling User Service RegisterUser for email: %s", req.GetEmail())
	// Consider adding a timeout specific to this call if not inherited or too long from parent context
	// callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	// defer cancel()
	return c.client.RegisterUser(ctx, req)
}

func (c *userServiceGRPCClient) LoginUser(ctx context.Context, req *pbUser.LoginUserRequest) (*pbUser.LoginUserResponse, error) {
	log.Printf("API Gateway | Calling User Service LoginUser for email: %s", req.GetEmail())
	return c.client.LoginUser(ctx, req)
}

func (c *userServiceGRPCClient) GetUser(ctx context.Context, req *pbUser.GetUserRequest) (*pbUser.UserResponse, error) {
	log.Printf("API Gateway | Calling User Service GetUser for ID: %s", req.GetUserId())
	return c.client.GetUser(ctx, req)
}

func (c *userServiceGRPCClient) UpdateUserProfile(ctx context.Context, req *pbUser.UpdateUserProfileRequest) (*pbUser.UserResponse, error) {
	log.Printf("API Gateway | Calling User Service UpdateUserProfile for ID: %s", req.GetUserId())
	return c.client.UpdateUserProfile(ctx, req)
}

func (c *userServiceGRPCClient) DeleteUser(ctx context.Context, req *pbUser.DeleteUserRequest) (*pbUser.EmptyResponse, error) {
	log.Printf("API Gateway | Calling User Service DeleteUser for ID: %s", req.GetUserId())
	return c.client.DeleteUser(ctx, req)
}

func (c *userServiceGRPCClient) Close() error {
	if c.conn != nil {
		log.Println("API Gateway | Closing User Service gRPC client connection...")
		return c.conn.Close()
	}
	return nil
}