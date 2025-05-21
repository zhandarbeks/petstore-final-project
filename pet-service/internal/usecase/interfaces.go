package usecase

import (
	"context"

	"github.com/zhandarbeks/petstore-final-project/pet-service/internal/domain" // Adjust import path
)

// CreatePetRequestData holds the data needed to create a new pet.
// Using a struct for request data can be cleaner than many individual parameters.
type CreatePetRequestData struct {
	Name           string
	Species        string
	Breed          string
	Age            int32
	Description    string
	ListedByUserID string // ID of the user listing the pet
	ImageURLs      []string
}

// UpdatePetRequestData holds the data for updating an existing pet.
// Using pointers for fields that are optional to update.
type UpdatePetRequestData struct {
	Name           *string
	Species        *string
	Breed          *string
	Age            *int32
	Description    *string
	ImageURLs      []string // For ImageURLs, decide if it's a full replacement or partial update
	// AdoptionStatus is handled by a separate method for clarity and control
}

// PetUsecase defines the interface for pet-related business logic.
type PetUsecase interface {
	CreatePet(ctx context.Context, reqData CreatePetRequestData) (*domain.Pet, error)
	GetPetByID(ctx context.Context, id string) (*domain.Pet, error)
	UpdatePet(ctx context.Context, id string, reqData UpdatePetRequestData) (*domain.Pet, error)
	DeletePet(ctx context.Context, id string) error
	ListPets(ctx context.Context, page, limit int, filters map[string]interface{}) ([]*domain.Pet, int64, error)
	UpdatePetAdoptionStatus(ctx context.Context, id string, newStatus domain.AdoptionStatus, adopterUserID *string) (*domain.Pet, error)
}