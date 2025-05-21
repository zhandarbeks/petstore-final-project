package main_test // Or a package name like 'notificationservicetest'

import (
	"context"
	"errors"
	"testing"
	"time"

	// Adjust these import paths to match your project's module path and structure
	"github.com/zhandarbeks/petstore-final-project/notification-service/internal/client"
	"github.com/zhandarbeks/petstore-final-project/notification-service/internal/consumer"
	"github.com/zhandarbeks/petstore-final-project/notification-service/internal/email"
	"github.com/zhandarbeks/petstore-final-project/notification-service/internal/service"

	pbPet "github.com/zhandarbeks/petstore-final-project/genprotos/pet"  // For Pet details mock
	pbUser "github.com/zhandarbeks/petstore-final-project/genprotos/user" // For User details mock

	// Optional: for assertions, e.g., "github.com/stretchr/testify/assert"
	// Optional: for mocking, e.g., "github.com/stretchr/testify/mock"
)

// --- Mock Implementations ---

// MockEmailSender is a mock for EmailSender
type MockEmailSender struct {
	SendEmailFunc func(to []string, subject, body string, isHTML bool) error
	// Track calls
	SendEmailCalled   bool
	LastTo            []string
	LastSubject       string
	LastBody          string
	LastIsHTML        bool
}

var _ email.EmailSender = (*MockEmailSender)(nil)

func (m *MockEmailSender) SendEmail(to []string, subject, body string, isHTML bool) error {
	m.SendEmailCalled = true
	m.LastTo = to
	m.LastSubject = subject
	m.LastBody = body
	m.LastIsHTML = isHTML
	if m.SendEmailFunc != nil {
		return m.SendEmailFunc(to, subject, body, isHTML)
	}
	return nil // Default to success
}

// MockUserServiceClient is a mock for UserServiceClient
type MockUserServiceClient struct {
	GetUserDetailsFunc func(ctx context.Context, userID string) (*pbUser.User, error)
	CloseFunc          func() error
}

var _ client.UserServiceClient = (*MockUserServiceClient)(nil)

func (m *MockUserServiceClient) GetUserDetails(ctx context.Context, userID string) (*pbUser.User, error) {
	if m.GetUserDetailsFunc != nil {
		return m.GetUserDetailsFunc(ctx, userID)
	}
	return nil, errors.New("GetUserDetailsFunc not implemented")
}
func (m *MockUserServiceClient) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

// MockPetServiceClient is a mock for PetServiceClient
type MockPetServiceClient struct {
	GetPetDetailsFunc func(ctx context.Context, petID string) (*pbPet.Pet, error)
	CloseFunc         func() error
}

var _ client.PetServiceClient = (*MockPetServiceClient)(nil)

func (m *MockPetServiceClient) GetPetDetails(ctx context.Context, petID string) (*pbPet.Pet, error) {
	if m.GetPetDetailsFunc != nil {
		return m.GetPetDetailsFunc(ctx, petID)
	}
	return nil, errors.New("GetPetDetailsFunc not implemented")
}
func (m *MockPetServiceClient) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

// --- Test Functions ---

func TestNotificationService_HandleAdoptionApplicationCreated_Success(t *testing.T) {
	mockEmailer := &MockEmailSender{}
	mockUserClient := &MockUserServiceClient{}
	mockPetClient := &MockPetServiceClient{}

	// Setup mock responses
	mockUserClient.GetUserDetailsFunc = func(ctx context.Context, userID string) (*pbUser.User, error) {
		if userID == "user123" {
			return &pbUser.User{Id: "user123", Email: "testuser@example.com", FullName: "Test User"}, nil
		}
		return nil, errors.New("user not found")
	}
	mockPetClient.GetPetDetailsFunc = func(ctx context.Context, petID string) (*pbPet.Pet, error) {
		if petID == "pet456" {
			return &pbPet.Pet{Id: "pet456", Name: "Buddy"}, nil
		}
		return nil, errors.New("pet not found")
	}
	mockEmailer.SendEmailFunc = func(to []string, subject, body string, isHTML bool) error {
		return nil // Simulate successful email send
	}

	notificationSvc := service.NewNotificationService(mockEmailer, mockUserClient, mockPetClient)

	event := consumer.AdoptionApplicationCreatedEvent{
		EventType:     "AdoptionApplicationCreated",
		ApplicationID: "app789",
		UserID:        "user123",
		PetID:         "pet456",
		Status:        "PENDING_REVIEW",
		AppliedAt:     time.Now(),
	}

	err := notificationSvc.HandleAdoptionApplicationCreated(context.Background(), event)

	if err != nil {
		t.Errorf("HandleAdoptionApplicationCreated() error = %v, wantErr %v", err, false)
	}
	if !mockEmailer.SendEmailCalled {
		t.Errorf("Expected SendEmail to be called, but it wasn't")
	}
	if len(mockEmailer.LastTo) != 1 || mockEmailer.LastTo[0] != "testuser@example.com" {
		t.Errorf("Email sent to wrong recipient. Got %v, want ['testuser@example.com']", mockEmailer.LastTo)
	}
	// Add more assertions for subject and body content if needed
}

func TestNotificationService_HandleAdoptionApplicationStatusUpdated_Approved(t *testing.T) {
	mockEmailer := &MockEmailSender{}
	mockUserClient := &MockUserServiceClient{}
	mockPetClient := &MockPetServiceClient{}

	mockUserClient.GetUserDetailsFunc = func(ctx context.Context, userID string) (*pbUser.User, error) {
		return &pbUser.User{Id: "user123", Email: "testuser@example.com", FullName: "Test User"}, nil
	}
	mockPetClient.GetPetDetailsFunc = func(ctx context.Context, petID string) (*pbPet.Pet, error) {
		return &pbPet.Pet{Id: "pet456", Name: "Buddy"}, nil
	}

	notificationSvc := service.NewNotificationService(mockEmailer, mockUserClient, mockPetClient)

	event := consumer.AdoptionApplicationStatusUpdatedEvent{
		EventType:     "AdoptionApplicationStatusUpdated",
		ApplicationID: "app789",
		UserID:        "user123",
		PetID:         "pet456",
		NewStatus:     "APPROVED", // Corresponds to domain.StatusAppApproved
		UpdatedAt:     time.Now(),
		ReviewNotes:   "Looks good!",
	}

	err := notificationSvc.HandleAdoptionApplicationStatusUpdated(context.Background(), event)

	if err != nil {
		t.Errorf("HandleAdoptionApplicationStatusUpdated() error = %v, wantErr %v", err, false)
	}
	if !mockEmailer.SendEmailCalled {
		t.Errorf("Expected SendEmail to be called for status update, but it wasn't")
	}
	// Add more specific assertions for email content based on "APPROVED" status
}

func TestNotificationService_HandleAdoptionApplicationCreated_UserFetchFail(t *testing.T) {
	mockEmailer := &MockEmailSender{}
	mockUserClient := &MockUserServiceClient{}
	mockPetClient := &MockPetServiceClient{}

	// Simulate UserServiceClient failing
	mockUserClient.GetUserDetailsFunc = func(ctx context.Context, userID string) (*pbUser.User, error) {
		return nil, errors.New("simulated user service error")
	}
	// Pet client and emailer should not be called if user fetch fails early
	mockPetClient.GetPetDetailsFunc = func(ctx context.Context, petID string) (*pbPet.Pet, error) {
		t.Error("PetServiceClient.GetPetDetails should not have been called")
		return nil, nil
	}
	mockEmailer.SendEmailFunc = func(to []string, subject, body string, isHTML bool) error {
		t.Error("EmailSender.SendEmail should not have been called")
		return nil
	}


	notificationSvc := service.NewNotificationService(mockEmailer, mockUserClient, mockPetClient)
	event := consumer.AdoptionApplicationCreatedEvent{UserID: "user123", PetID: "pet456"}

	err := notificationSvc.HandleAdoptionApplicationCreated(context.Background(), event)

	if err == nil {
		t.Errorf("Expected an error when user fetch fails, but got nil")
	}
	// Check for specific error message if desired
	// assert.ErrorContains(t, err, "failed to fetch user details")
}


// TODO: Add more test cases:
// - HandleAdoptionApplicationStatusUpdated for REJECTED status
// - HandleAdoptionApplicationCreated when PetServiceClient fails
// - HandleAdoptionApplicationCreated when EmailSender fails
// - HandleAdoptionApplicationStatusUpdated when UserServiceClient fails
// - HandleAdoptionApplicationStatusUpdated when PetServiceClient fails
// - HandleAdoptionApplicationStatusUpdated when EmailSender fails