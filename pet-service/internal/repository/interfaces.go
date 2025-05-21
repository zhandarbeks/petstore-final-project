package repository

import (
	"context"
	"time"

	"github.com/zhandarbeks/petstore-final-project/pet-service/internal/domain" // Adjust import path
)

// PetRepository defines the interface for database operations related to pets.
type PetRepository interface {
	CreatePet(ctx context.Context, pet *domain.Pet) (*domain.Pet, error)
	GetPetByID(ctx context.Context, id string) (*domain.Pet, error)
	UpdatePet(ctx context.Context, pet *domain.Pet) (*domain.Pet, error)
	DeletePet(ctx context.Context, id string) error
	ListPets(ctx context.Context, page, limit int, filters map[string]interface{}) ([]*domain.Pet, int64, error) // For listing with filters & pagination
	UpdatePetAdoptionStatus(ctx context.Context, id string, newStatus domain.AdoptionStatus, adopterUserID *string) (*domain.Pet, error)
}

// PetCache defines the interface for caching operations related to pets.
type PetCache interface {
	GetPet(ctx context.Context, id string) (*domain.Pet, error)
	SetPet(ctx context.Context, id string, pet *domain.Pet, expiration time.Duration) error
	DeletePet(ctx context.Context, id string) error
	// Consider methods for caching lists of pets if that's a frequent operation
	// SetListedPets(ctx context.Context, cacheKey string, pets []*domain.Pet, expiration time.Duration) error
	// GetListedPets(ctx context.Context, cacheKey string) ([]*domain.Pet, error)
	// DeleteListedPets(ctx context.Context, cacheKey string) error
}