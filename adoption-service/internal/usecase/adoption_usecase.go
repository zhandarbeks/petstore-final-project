package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/zhandarbeks/petstore-final-project/adoption-service/internal/domain" // Adjust import path
	"github.com/zhandarbeks/petstore-final-project/adoption-service/internal/publisher"
	"github.com/zhandarbeks/petstore-final-project/adoption-service/internal/repository"
)

type adoptionUsecase struct {
	repo      repository.AdoptionRepository
	cache     repository.AdoptionCache
	publisher publisher.AdoptionEventPublisher
	// petServiceClient PetServiceInternalClient // Interface for internal PetService gRPC calls
}

// NewAdoptionUsecase creates a new instance of adoptionUsecase.
func NewAdoptionUsecase(
	repo repository.AdoptionRepository,
	cache repository.AdoptionCache,
	pub publisher.AdoptionEventPublisher,
	// petClient PetServiceInternalClient, // Inject if needed
) AdoptionUsecase {
	return &adoptionUsecase{
		repo:      repo,
		cache:     cache,
		publisher: pub,
		// petServiceClient: petClient,
	}
}

func (uc *adoptionUsecase) CreateAdoptionApplication(ctx context.Context, reqData CreateAdoptionApplicationRequestData) (*domain.AdoptionApplication, error) {
	if reqData.UserID == "" || reqData.PetID == "" {
		return nil, errors.New("user ID and pet ID are required")
	}

	// Optional: Check if pet is available for adoption by calling Pet Service
	// available, err := uc.petServiceClient.IsPetAvailableForAdoption(ctx, reqData.PetID)
	// if err != nil {
	// 	log.Printf("Adoption Service | Error checking pet availability for PetID %s: %v", reqData.PetID, err)
	// 	return nil, fmt.Errorf("failed to verify pet status: %w", err)
	// }
	// if !available {
	// 	return nil, errors.New("pet is not available for adoption")
	// }

	app := &domain.AdoptionApplication{
		UserID:           reqData.UserID,
		PetID:            reqData.PetID,
		ApplicationNotes: reqData.ApplicationNotes,
		// Status will be defaulted by PrepareForCreate
	}
	// app.PrepareForCreate() // Called by repository

	createdApp, err := uc.repo.CreateAdoptionApplication(ctx, app)
	if err != nil {
		log.Printf("Adoption Service | Error creating adoption application in repository: %v", err)
		return nil, fmt.Errorf("could not create adoption application: %w", err)
	}

	// Publish event to NATS
	if pubErr := uc.publisher.PublishAdoptionApplicationCreated(ctx, createdApp); pubErr != nil {
		// Log the error but don't fail the whole operation, as the application was created.
		// This depends on business requirements; sometimes event publishing failure is critical.
		log.Printf("Adoption Service | Warning: Failed to publish AdoptionApplicationCreated event for app ID %s: %v", createdApp.ID, pubErr)
	}

	log.Printf("Adoption Service | Adoption application created successfully: ID %s", createdApp.ID)
	return createdApp, nil
}

func (uc *adoptionUsecase) GetAdoptionApplicationByID(ctx context.Context, applicationID string) (*domain.AdoptionApplication, error) {
	if applicationID == "" {
		return nil, errors.New("application ID is required")
	}

	// 1. Try cache
	cachedApp, err := uc.cache.GetAdoptionApplication(ctx, applicationID)
	if err == nil && cachedApp != nil {
		log.Printf("Adoption Service | Application %s found in cache", applicationID)
		return cachedApp, nil
	}
	if err != nil && err.Error() != "adoption application not found in cache" {
		log.Printf("Adoption Service | Error fetching application %s from cache: %v", applicationID, err)
	}

	// 2. Not in cache or cache error, get from repository
	log.Printf("Adoption Service | Application %s not in cache, fetching from repository", applicationID)
	app, err := uc.repo.GetAdoptionApplicationByID(ctx, applicationID)
	if err != nil {
		log.Printf("Adoption Service | Error fetching application %s from repository: %v", applicationID, err)
		return nil, err // Could be "not found" or other DB error
	}

	// 3. Set in cache
	// Use a reasonable expiration, e.g., 1 hour
	cacheErr := uc.cache.SetAdoptionApplication(ctx, applicationID, app, 1*time.Hour)
	if cacheErr != nil {
		log.Printf("Adoption Service | Warning: Failed to set application %s in cache: %v", applicationID, cacheErr)
	}

	return app, nil
}

func (uc *adoptionUsecase) UpdateAdoptionApplicationStatus(ctx context.Context, applicationID string, reqData UpdateAdoptionApplicationStatusRequestData) (*domain.AdoptionApplication, error) {
	if applicationID == "" {
		return nil, errors.New("application ID is required for status update")
	}
	if !domain.IsValidApplicationStatus(reqData.NewStatus) {
		return nil, errors.New("invalid new application status")
	}

	// Fetch the application to ensure it exists before updating
	// (though the repository update method might also do this check)
	// existingApp, err := uc.repo.GetAdoptionApplicationByID(ctx, applicationID)
	// if err != nil {
	// 	log.Printf("Adoption Service | Error fetching application %s for status update: %v", applicationID, err)
	// 	return nil, err // Could be "not found"
	// }
	// Perform any state transition validation if needed (e.g., cannot move from REJECTED to APPROVED directly)

	// The repository's UpdateAdoptionApplicationStatus should handle updating UpdatedAt.
	// This is where a transaction might be implemented if, for example, updating the status
	// also required writing to an audit log collection within the same database transaction.
	// For now, the repo method updates a single document, which is atomic.
	updatedApp, err := uc.repo.UpdateAdoptionApplicationStatus(ctx, applicationID, reqData.NewStatus, reqData.ReviewNotes)
	if err != nil {
		log.Printf("Adoption Service | Error updating application status for ID %s in repository: %v", applicationID, err)
		return nil, fmt.Errorf("could not update application status: %w", err)
	}

	// Invalidate/update cache
	cacheErr := uc.cache.DeleteAdoptionApplication(ctx, applicationID) // Simple invalidation
	if cacheErr != nil {
		log.Printf("Adoption Service | Warning: Failed to delete application %s from cache after status update: %v", applicationID, cacheErr)
	}

	// Publish event to NATS
	if pubErr := uc.publisher.PublishAdoptionApplicationStatusUpdated(ctx, updatedApp); pubErr != nil {
		log.Printf("Adoption Service | Warning: Failed to publish AdoptionApplicationStatusUpdated event for app ID %s: %v", updatedApp.ID, pubErr)
	}

	// If status is APPROVED, consider interaction with Pet Service to update pet's status.
	// This could be done here via a gRPC call to Pet Service, or Pet Service could subscribe to NATS events.
	// For simplicity and to avoid distributed transactions in this call, Pet Service could subscribe to "adoption.application.status.updated"
	// and if new_status is APPROVED, it then updates its own pet record.
	// if updatedApp.Status == domain.StatusAppApproved {
	// 	 petUpdateErr := uc.petServiceClient.MarkPetAsPendingOrAdopted(ctx, updatedApp.PetID, updatedApp.UserID)
	// 	 if petUpdateErr != nil {
	// 	 	log.Printf("Adoption Service | Warning: Failed to update pet status for PetID %s after application approval: %v", updatedApp.PetID, petUpdateErr)
	// 	 	// Decide how to handle this - is it critical? Should the adoption status be rolled back?
	// 	 }
	// }

	log.Printf("Adoption Service | Application status updated successfully for ID: %s to %s", updatedApp.ID, updatedApp.Status)
	return updatedApp, nil
}

func (uc *adoptionUsecase) ListUserAdoptionApplications(ctx context.Context, userID string, page, limit int, statusFilter *domain.ApplicationStatus) ([]*domain.AdoptionApplication, int64, error) {
	if userID == "" {
		return nil, 0, errors.New("user ID is required")
	}

	// Caching for lists can be complex due to pagination and filters, so often skipped or done with care.
	// For now, fetch directly from repository.
	apps, totalCount, err := uc.repo.ListAdoptionApplicationsByUserID(ctx, userID, page, limit, statusFilter)
	if err != nil {
		log.Printf("Adoption Service | Error listing adoption applications for UserID %s: %v", userID, err)
		return nil, 0, fmt.Errorf("could not list user adoption applications: %w", err)
	}
	return apps, totalCount, nil
}