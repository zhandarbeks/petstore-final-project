package usecase

import (
	"context"

	"github.com/zhandarbeks/petstore-final-project/user-service/internal/domain" // Adjust import path as per your module
	// You might need to define request/response structs specific to usecases if they differ significantly from domain/gRPC,
	// but often gRPC request/response types (or domain types) can be used directly or with minimal mapping.
)

// UserUsecase defines the interface for user-related business logic.
type UserUsecase interface {
	RegisterUser(ctx context.Context, username, email, password, fullName string) (*domain.User, string, error) // Returns User, AccessToken, Error
	LoginUser(ctx context.Context, email, password string) (*domain.User, string, error)    // Returns User, AccessToken, Error
	GetUserByID(ctx context.Context, id string) (*domain.User, error)
	UpdateUserProfile(ctx context.Context, id string, username, fullName *string) (*domain.User, error) // Pointers allow partial updates
	DeleteUser(ctx context.Context, id string) error
}
