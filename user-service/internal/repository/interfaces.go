package repository

import (
	"context"
	"time"

	"github.com/zhandarbeks/petstore-final-project/user-service/internal/domain" // Adjust import path as per your module
)

// UserRepository defines the interface for database operations related to users.
type UserRepository interface {
	CreateUser(ctx context.Context, user *domain.User) (*domain.User, error)
	GetUserByID(ctx context.Context, id string) (*domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	UpdateUser(ctx context.Context, user *domain.User) (*domain.User, error)
	DeleteUser(ctx context.Context, id string) error
	// ListUsers(ctx context.Context, page, limit int) ([]*domain.User, int64, error) // Example for listing users
}

// UserCache defines the interface for caching operations related to users.
// This can be a separate interface or its methods can be incorporated into
// a caching decorator around the UserRepository. For clarity, we'll define it separately.
type UserCache interface {
	GetUser(ctx context.Context, id string) (*domain.User, error)
	SetUser(ctx context.Context, id string, user *domain.User, expiration time.Duration) error
	DeleteUser(ctx context.Context, id string) error
}

// You might also define an interface that combines both direct DB access and caching logic,
// or use a decorator pattern where the caching repository wraps the database repository.
// For instance:
// type UserStorer interface {
//    UserRepository // Embeds all DB methods
//    UserCache      // Embeds all Cache methods (or cache methods are part of UserRepository if it's a caching repo)
// }
