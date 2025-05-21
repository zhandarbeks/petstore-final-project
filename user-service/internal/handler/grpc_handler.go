package handler

import (
	"context"
	"errors"
	"fmt" // Added import for fmt
	"log"
	"time" // Added import for time

	"github.com/zhandarbeks/petstore-final-project/user-service/internal/domain"   // Adjust import path
	"github.com/zhandarbeks/petstore-final-project/user-service/internal/usecase" // Adjust import path
	pb "github.com/zhandarbeks/petstore-final-project/genprotos/user"            // Adjust import path to your generated protos

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	// "google.golang.org/protobuf/types/known/timestamppb" // If using timestamppb in your domain/proto
)

// UserHandler implements the gRPC service for user operations.
type UserHandler struct {
	pb.UnimplementedUserServiceServer        // Embed for forward compatibility
	usecase                       usecase.UserUsecase
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(uc usecase.UserUsecase) *UserHandler {
	if uc == nil {
		log.Fatal("UserUsecase cannot be nil in NewUserHandler")
	}
	return &UserHandler{usecase: uc}
}

// Helper function to convert domain.User to pb.User
func domainUserToPbUser(du *domain.User) *pb.User {
	if du == nil {
		return nil
	}
	return &pb.User{
		Id:        du.ID,
		Username:  du.Username,
		Email:     du.Email,
		FullName:  du.FullName,
		CreatedAt: du.CreatedAt.Format(time.RFC3339), // Or use timestamppb.New(du.CreatedAt)
		UpdatedAt: du.UpdatedAt.Format(time.RFC3339), // Or use timestamppb.New(du.UpdatedAt)
	}
}

// RegisterUser handles the gRPC request for user registration.
func (h *UserHandler) RegisterUser(ctx context.Context, req *pb.RegisterUserRequest) (*pb.UserResponse, error) {
	log.Printf("gRPC RegisterUser request received for email: %s", req.GetEmail())

	if req.GetUsername() == "" || req.GetEmail() == "" || req.GetPassword() == "" || req.GetFullName() == "" {
		log.Println("RegisterUser: Missing required fields")
		return nil, status.Errorf(codes.InvalidArgument, "Username, email, password, and full name are required")
	}

	// The token variable is not used in this response, so use blank identifier _
	createdUser, _, err := h.usecase.RegisterUser(ctx, req.GetUsername(), req.GetEmail(), req.GetPassword(), req.GetFullName())
	if err != nil {
		log.Printf("Error during RegisterUser usecase call for email %s: %v", req.GetEmail(), err)
		// Map domain-specific errors to gRPC status codes
		if errors.Is(err, errors.New("user with this email already exists")) { // Example of specific error check
			return nil, status.Errorf(codes.AlreadyExists, err.Error())
		}
		if errors.Is(err, errors.New("username, email, password, and full name are required")) {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		// Check for the specific error from usecase regarding token generation failure
		// The error message from usecase is "user registered, but token generation failed: <original_token_error>"
		// We need to check if the original_token_error is part of the error string.
		// A more robust way would be to define custom error types.
		// For now, we'll assume the usecase returns the user even if token generation fails,
		// and the error indicates this specific scenario.
		if createdUser != nil && err != nil && (err.Error() == fmt.Sprintf("user registered, but token generation failed: %v", errors.Unwrap(err)) || errors.Is(err, errors.New("user registered, but token generation failed"))) {
			log.Printf("RegisterUser: User %s created, but token generation failed: %v", createdUser.Email, err)
			// Return the user as the gRPC response for RegisterUser only contains User.
			// The client should be aware that login is needed to get a token.
			return &pb.UserResponse{User: domainUserToPbUser(createdUser)}, nil
		}
		return nil, status.Errorf(codes.Internal, "Failed to register user: %v", err)
	}

	log.Printf("User registered successfully via gRPC: %s", createdUser.Email)
	return &pb.UserResponse{User: domainUserToPbUser(createdUser)}, nil
}

// LoginUser handles the gRPC request for user login.
func (h *UserHandler) LoginUser(ctx context.Context, req *pb.LoginUserRequest) (*pb.LoginUserResponse, error) {
	log.Printf("gRPC LoginUser request received for email: %s", req.GetEmail())

	if req.GetEmail() == "" || req.GetPassword() == "" {
		log.Println("LoginUser: Missing email or password")
		return nil, status.Errorf(codes.InvalidArgument, "Email and password are required")
	}

	user, token, err := h.usecase.LoginUser(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		log.Printf("Error during LoginUser usecase call for email %s: %v", req.GetEmail(), err)
		if err.Error() == "invalid email or password" {
			return nil, status.Errorf(codes.Unauthenticated, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "Login failed: %v", err)
	}

	log.Printf("User logged in successfully via gRPC: %s", user.Email)
	return &pb.LoginUserResponse{
		User:        domainUserToPbUser(user),
		AccessToken: token,
	}, nil
}

// GetUser handles the gRPC request to retrieve a user by ID.
func (h *UserHandler) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.UserResponse, error) {
	log.Printf("gRPC GetUser request received for ID: %s", req.GetUserId())

	if req.GetUserId() == "" {
		log.Println("GetUser: User ID is required")
		return nil, status.Errorf(codes.InvalidArgument, "User ID is required")
	}

	user, err := h.usecase.GetUserByID(ctx, req.GetUserId())
	if err != nil {
		log.Printf("Error during GetUserByID usecase call for ID %s: %v", req.GetUserId(), err)
		if err.Error() == "user not found" { // Assuming usecase returns this specific error string
			return nil, status.Errorf(codes.NotFound, "User not found")
		}
		if err.Error() == "invalid user ID format" {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid user ID format")
		}
		return nil, status.Errorf(codes.Internal, "Failed to get user: %v", err)
	}

	log.Printf("User retrieved successfully via gRPC: %s", user.Email)
	return &pb.UserResponse{User: domainUserToPbUser(user)}, nil
}

// UpdateUserProfile handles the gRPC request to update a user's profile.
func (h *UserHandler) UpdateUserProfile(ctx context.Context, req *pb.UpdateUserProfileRequest) (*pb.UserResponse, error) {
	log.Printf("gRPC UpdateUserProfile request received for ID: %s", req.GetUserId())

	if req.GetUserId() == "" {
		log.Println("UpdateUserProfile: User ID is required")
		return nil, status.Errorf(codes.InvalidArgument, "User ID is required")
	}

	var usernamePtr *string
	// For proto3, if a string field is not set, its default value is an empty string.
	// If your .proto uses `optional string username = 2;`, then you'd have a `req.HasUsername()` method.
	// Assuming non-optional string fields for now, and an empty string means "do not update".
	// The usecase expects nil if the field is not to be updated.
	if req.GetUsername() != "" { // This logic means empty string cannot be set for username.
		val := req.GetUsername()
		usernamePtr = &val
	}

	var fullNamePtr *string
	if req.GetFullName() != "" { // Same logic for full name.
		val := req.GetFullName()
		fullNamePtr = &val
	}

	if usernamePtr == nil && fullNamePtr == nil {
		log.Println("UpdateUserProfile: No fields provided for update")
		return nil, status.Errorf(codes.InvalidArgument, "At least one field (username or full name) must be provided for update")
	}


	updatedUser, err := h.usecase.UpdateUserProfile(ctx, req.GetUserId(), usernamePtr, fullNamePtr)
	if err != nil {
		log.Printf("Error during UpdateUserProfile usecase call for ID %s: %v", req.GetUserId(), err)
		if err.Error() == "user not found for update" { // Match error from usecase
			return nil, status.Errorf(codes.NotFound, "User not found for update")
		}
		if err.Error() == "invalid user ID format for update" { // Match error from usecase
			return nil, status.Errorf(codes.InvalidArgument, "Invalid user ID format")
		}
		return nil, status.Errorf(codes.Internal, "Failed to update user profile: %v", err)
	}

	log.Printf("User profile updated successfully via gRPC for ID: %s", updatedUser.ID)
	return &pb.UserResponse{User: domainUserToPbUser(updatedUser)}, nil
}

// DeleteUser handles the gRPC request to delete a user.
func (h *UserHandler) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.EmptyResponse, error) {
	log.Printf("gRPC DeleteUser request received for ID: %s", req.GetUserId())

	if req.GetUserId() == "" {
		log.Println("DeleteUser: User ID is required")
		return nil, status.Errorf(codes.InvalidArgument, "User ID is required")
	}

	err := h.usecase.DeleteUser(ctx, req.GetUserId())
	if err != nil {
		log.Printf("Error during DeleteUser usecase call for ID %s: %v", req.GetUserId(), err)
		if err.Error() == "user not found for deletion" { // Match error from usecase
			return nil, status.Errorf(codes.NotFound, "User not found for deletion")
		}
		if err.Error() == "invalid user ID format for delete" { // Match error from usecase
			return nil, status.Errorf(codes.InvalidArgument, "Invalid user ID format")
		}
		return nil, status.Errorf(codes.Internal, "Failed to delete user: %v", err)
	}

	log.Printf("User deleted successfully via gRPC: ID %s", req.GetUserId())
	return &pb.EmptyResponse{}, nil
}
