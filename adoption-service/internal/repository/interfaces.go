package repository

import (
	"context"
	"time"

	"github.com/zhandarbeks/petstore-final-project/adoption-service/internal/domain" // Adjust import path
)

// AdoptionRepository defines the interface for database operations related to adoption applications.
type AdoptionRepository interface {
	CreateAdoptionApplication(ctx context.Context, app *domain.AdoptionApplication) (*domain.AdoptionApplication, error)
	GetAdoptionApplicationByID(ctx context.Context, id string) (*domain.AdoptionApplication, error)
	UpdateAdoptionApplicationStatus(ctx context.Context, id string, newStatus domain.ApplicationStatus, reviewNotes string) (*domain.AdoptionApplication, error)
	ListAdoptionApplicationsByUserID(ctx context.Context, userID string, page, limit int, statusFilter *domain.ApplicationStatus) ([]*domain.AdoptionApplication, int64, error)
	// ListAdoptionApplicationsByPetID(ctx context.Context, petID string, page, limit int) ([]*domain.AdoptionApplication, int64, error) // Optional
	// ListAllAdoptionApplications(ctx context.Context, page, limit int, statusFilter *domain.ApplicationStatus) ([]*domain.AdoptionApplication, int64, error) // Optional for admin
}

// AdoptionCache defines the interface for caching operations related to adoption applications.
type AdoptionCache interface {
	GetAdoptionApplication(ctx context.Context, id string) (*domain.AdoptionApplication, error)
	SetAdoptionApplication(ctx context.Context, id string, app *domain.AdoptionApplication, expiration time.Duration) error
	DeleteAdoptionApplication(ctx context.Context, id string) error
}