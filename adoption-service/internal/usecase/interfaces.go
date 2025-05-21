package usecase

import (
	"context"

	"github.com/zhandarbeks/petstore-final-project/adoption-service/internal/domain" // Adjust import path
)

// CreateAdoptionApplicationRequestData holds data for creating an application.
type CreateAdoptionApplicationRequestData struct {
	UserID           string
	PetID            string
	ApplicationNotes string
}

// UpdateAdoptionApplicationStatusRequestData holds data for updating an application's status.
type UpdateAdoptionApplicationStatusRequestData struct {
	NewStatus   domain.ApplicationStatus
	ReviewNotes string
}

// AdoptionUsecase defines the interface for adoption application business logic.
type AdoptionUsecase interface {
	CreateAdoptionApplication(ctx context.Context, reqData CreateAdoptionApplicationRequestData) (*domain.AdoptionApplication, error)
	GetAdoptionApplicationByID(ctx context.Context, applicationID string) (*domain.AdoptionApplication, error)
	UpdateAdoptionApplicationStatus(ctx context.Context, applicationID string, reqData UpdateAdoptionApplicationStatusRequestData) (*domain.AdoptionApplication, error)
	ListUserAdoptionApplications(ctx context.Context, userID string, page, limit int, statusFilter *domain.ApplicationStatus) ([]*domain.AdoptionApplication, int64, error)
}