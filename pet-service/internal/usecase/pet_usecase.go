package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/zhandarbeks/petstore-final-project/pet-service/internal/domain" // Adjust import path
	"github.com/zhandarbeks/petstore-final-project/pet-service/internal/repository"
	// "go.mongodb.org/mongo-driver/bson/primitive" // If generating IDs here, but repo handles it
)

type petUsecase struct {
	petRepo  repository.PetRepository
	petCache repository.PetCache
	// userServiceClient some_interface.UserServiceClient // If needed to validate ListedByUserID against user service
}

// NewPetUsecase creates a new instance of petUsecase.
func NewPetUsecase(repo repository.PetRepository, cache repository.PetCache) PetUsecase {
	return &petUsecase{
		petRepo:  repo,
		petCache: cache,
	}
}

func (uc *petUsecase) CreatePet(ctx context.Context, reqData CreatePetRequestData) (*domain.Pet, error) {
	// Basic Validation
	if reqData.Name == "" || reqData.Species == "" {
		return nil, errors.New("pet name and species are required")
	}
	if reqData.Age < 0 {
		return nil, errors.New("pet age cannot be negative")
	}
	// Optional: Validate ListedByUserID if it's mandatory or by calling user service

	newPet := &domain.Pet{
		Name:           reqData.Name,
		Species:        reqData.Species,
		Breed:          reqData.Breed,
		Age:            reqData.Age,
		Description:    reqData.Description,
		ListedByUserID: reqData.ListedByUserID,
		ImageURLs:      reqData.ImageURLs,
		// AdoptionStatus will be defaulted by PrepareForCreate in the domain or repo
	}
	// newPet.PrepareForCreate() // This is called by the repository in our current setup

	createdPet, err := uc.petRepo.CreatePet(ctx, newPet)
	if err != nil {
		log.Printf("Pet Service | Error creating pet in repository: %v", err)
		return nil, fmt.Errorf("could not create pet: %w", err)
	}

	log.Printf("Pet Service | Pet created successfully: %s (ID: %s)", createdPet.Name, createdPet.ID)
	return createdPet, nil
}

func (uc *petUsecase) GetPetByID(ctx context.Context, id string) (*domain.Pet, error) {
	if id == "" {
		return nil, errors.New("pet ID is required")
	}

	// 1. Try cache
	cachedPet, err := uc.petCache.GetPet(ctx, id)
	if err == nil && cachedPet != nil {
		log.Printf("Pet Service | Pet %s found in cache", id)
		return cachedPet, nil
	}
	if err != nil && err.Error() != "pet not found in cache" {
		log.Printf("Pet Service | Error fetching pet %s from cache: %v", id, err)
	}

	// 2. Not in cache or cache error, get from repository
	log.Printf("Pet Service | Pet %s not in cache or cache error, fetching from repository", id)
	pet, err := uc.petRepo.GetPetByID(ctx, id)
	if err != nil {
		log.Printf("Pet Service | Error fetching pet %s from repository: %v", id, err)
		return nil, err // Could be "pet not found" or other DB error
	}

	// 3. Set in cache
	cacheErr := uc.petCache.SetPet(ctx, id, pet, 1*time.Hour) // Example 1-hour cache
	if cacheErr != nil {
		log.Printf("Pet Service | Warning: Failed to set pet %s in cache: %v", id, cacheErr)
	}

	return pet, nil
}

func (uc *petUsecase) UpdatePet(ctx context.Context, id string, reqData UpdatePetRequestData) (*domain.Pet, error) {
	if id == "" {
		return nil, errors.New("pet ID is required for update")
	}

	// Fetch existing pet
	pet, err := uc.petRepo.GetPetByID(ctx, id)
	if err != nil {
		log.Printf("Pet Service | Error fetching pet %s for update: %v", id, err)
		return nil, err // Could be "pet not found"
	}

	// Apply updates from reqData
	updated := false
	if reqData.Name != nil && *reqData.Name != "" && *reqData.Name != pet.Name {
		pet.Name = *reqData.Name
		updated = true
	}
	if reqData.Species != nil && *reqData.Species != "" && *reqData.Species != pet.Species {
		pet.Species = *reqData.Species
		updated = true
	}
	if reqData.Breed != nil && *reqData.Breed != pet.Breed { // Allow empty string to clear breed
		pet.Breed = *reqData.Breed
		updated = true
	}
	if reqData.Age != nil && *reqData.Age >= 0 && *reqData.Age != pet.Age {
		pet.Age = *reqData.Age
		updated = true
	}
	if reqData.Description != nil && *reqData.Description != pet.Description {
		pet.Description = *reqData.Description
		updated = true
	}
	if reqData.ImageURLs != nil { // Assuming full replacement of ImageURLs
		pet.ImageURLs = reqData.ImageURLs
		updated = true
	}

	if !updated {
		log.Printf("Pet Service | No changes detected for pet %s update.", id)
		return pet, nil // Return existing pet if no actual changes
	}

	// pet.PrepareForUpdate() // This is called by the repository in our current setup

	updatedPet, err := uc.petRepo.UpdatePet(ctx, pet)
	if err != nil {
		log.Printf("Pet Service | Error updating pet %s in repository: %v", id, err)
		return nil, fmt.Errorf("could not update pet: %w", err)
	}

	// Invalidate cache
	cacheErr := uc.petCache.DeletePet(ctx, id)
	if cacheErr != nil {
		log.Printf("Pet Service | Warning: Failed to delete pet %s from cache after update: %v", id, cacheErr)
	}

	log.Printf("Pet Service | Pet updated successfully: ID %s", id)
	return updatedPet, nil
}

func (uc *petUsecase) DeletePet(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("pet ID is required for deletion")
	}

	// Optional: Check if pet exists before trying to delete, to return a more specific "not found"
	// _, err := uc.petRepo.GetPetByID(ctx, id)
	// if err != nil {
	// 	return err // Could be "pet not found"
	// }

	err := uc.petRepo.DeletePet(ctx, id)
	if err != nil {
		log.Printf("Pet Service | Error deleting pet %s from repository: %v", id, err)
		return fmt.Errorf("could not delete pet: %w", err)
	}

	// Invalidate cache
	cacheErr := uc.petCache.DeletePet(ctx, id)
	if cacheErr != nil {
		log.Printf("Pet Service | Warning: Failed to delete pet %s from cache after DB deletion: %v", id, cacheErr)
	}

	log.Printf("Pet Service | Pet deleted successfully: ID %s", id)
	return nil
}

func (uc *petUsecase) ListPets(ctx context.Context, page, limit int, filters map[string]interface{}) ([]*domain.Pet, int64, error) {
	// TODO: Implement caching for list operations if beneficial.
	// This can be complex due to varying filters and pagination.
	// For now, directly fetch from repository.

	// Sanitize/validate filters if necessary
	// For example, if filtering by domain.AdoptionStatus, ensure the value is valid
	if statusStr, ok := filters["adoption_status"].(string); ok {
		if statusStr != "" && !domain.IsValidAdoptionStatus(domain.AdoptionStatus(statusStr)) {
			return nil, 0, errors.New("invalid adoption_status filter value")
		}
		if statusStr == "" { // If empty string filter, remove it or handle as "all statuses"
			delete(filters, "adoption_status")
		}
	}


	pets, totalCount, err := uc.petRepo.ListPets(ctx, page, limit, filters)
	if err != nil {
		log.Printf("Pet Service | Error listing pets from repository: %v", err)
		return nil, 0, fmt.Errorf("could not list pets: %w", err)
	}
	return pets, totalCount, nil
}

func (uc *petUsecase) UpdatePetAdoptionStatus(ctx context.Context, id string, newStatus domain.AdoptionStatus, adopterUserID *string) (*domain.Pet, error) {
	if id == "" {
		return nil, errors.New("pet ID is required for status update")
	}
	if !domain.IsValidAdoptionStatus(newStatus) {
		return nil, errors.New("invalid new adoption status")
	}
	if newStatus == domain.StatusAdopted && (adopterUserID == nil || *adopterUserID == "") {
		return nil, errors.New("adopter user ID is required when setting status to ADOPTED")
	}

	// Optional: Fetch pet first to ensure it exists or to perform other checks
	// _, err := uc.petRepo.GetPetByID(ctx, id)
	// if err != nil {
	// 	return nil, err // Could be "pet not found"
	// }

	updatedPet, err := uc.petRepo.UpdatePetAdoptionStatus(ctx, id, newStatus, adopterUserID)
	if err != nil {
		log.Printf("Pet Service | Error updating pet adoption status for ID %s: %v", id, err)
		return nil, fmt.Errorf("could not update pet adoption status: %w", err)
	}

	// Invalidate cache for this pet
	cacheErr := uc.petCache.DeletePet(ctx, id)
	if cacheErr != nil {
		log.Printf("Pet Service | Warning: Failed to delete pet %s from cache after status update: %v", id, cacheErr)
	}

	log.Printf("Pet Service | Pet adoption status updated successfully for ID: %s to %s", id, newStatus)
	return updatedPet, nil
}