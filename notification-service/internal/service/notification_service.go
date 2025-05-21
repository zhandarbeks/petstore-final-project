package service

import (
	"context"
	"fmt"
	"log"

	// Adjust import paths to match your project's module path and structure
	"github.com/zhandarbeks/petstore-final-project/notification-service/internal/client"    // For UserServiceClient, PetServiceClient
	"github.com/zhandarbeks/petstore-final-project/notification-service/internal/consumer"  // For event structs
	"github.com/zhandarbeks/petstore-final-project/notification-service/internal/email"     // For EmailSender
	pbPet "github.com/zhandarbeks/petstore-final-project/genprotos/pet"                   // For Pet details
	pbUser "github.com/zhandarbeks/petstore-final-project/genprotos/user"                  // For User details
)

// NotificationService handles the business logic for processing events and sending notifications.
// It implements the consumer.EventHandler interface.
type NotificationService struct {
	emailSender       email.EmailSender
	userServiceClient client.UserServiceClient
	petServiceClient  client.PetServiceClient
}

// NewNotificationService creates a new NotificationService.
func NewNotificationService(
	sender email.EmailSender,
	userClient client.UserServiceClient,
	petClient client.PetServiceClient,
) consumer.EventHandler { // Return the interface type
	if sender == nil || userClient == nil || petClient == nil {
		log.Fatal("Notification Service | FATAL: EmailSender, UserServiceClient, and PetServiceClient cannot be nil")
	}
	return &NotificationService{
		emailSender:       sender,
		userServiceClient: userClient,
		petServiceClient:  petClient,
	}
}

// HandleAdoptionApplicationCreated processes an event when a new adoption application is created.
func (s *NotificationService) HandleAdoptionApplicationCreated(ctx context.Context, event consumer.AdoptionApplicationCreatedEvent) error {
	log.Printf("Notification Service | Handling AdoptionApplicationCreated event for AppID: %s, UserID: %s, PetID: %s",
		event.ApplicationID, event.UserID, event.PetID)

	// 1. Fetch User Details (to get email and name)
	// The GetUserDetails method from client.UserServiceClient is expected to return *pbUser.User
	userDetails, err := s.userServiceClient.GetUserDetails(ctx, event.UserID)
	if err != nil {
		log.Printf("Notification Service | Error fetching user details for UserID %s: %v", event.UserID, err)
		return fmt.Errorf("failed to fetch user details for created application: %w", err)
	}
	// Accessing fields like userDetails.Email and userDetails.FullName uses the pbUser package.
	if userDetails == nil || userDetails.GetEmail() == "" { // Use GetEmail() getter
		log.Printf("Notification Service | User details or email not found for UserID %s", event.UserID)
		return fmt.Errorf("user email not found for UserID %s", event.UserID)
	}

	// 2. Fetch Pet Details (to get pet name)
	// The GetPetDetails method from client.PetServiceClient is expected to return *pbPet.Pet
	petDetails, err := s.petServiceClient.GetPetDetails(ctx, event.PetID)
	if err != nil {
		log.Printf("Notification Service | Error fetching pet details for PetID %s: %v", event.PetID, err)
		return fmt.Errorf("failed to fetch pet details for created application: %w", err)
	}
	// Accessing fields like petDetails.Name uses the pbPet package.
	if petDetails == nil || petDetails.GetName() == "" { // Use GetName() getter
		log.Printf("Notification Service | Pet details or name not found for PetID %s", event.PetID)
		return fmt.Errorf("pet details not found or name is empty for PetID %s", event.PetID)
	}

	// 3. Construct and Send Email
	recipientEmail := userDetails.GetEmail()
	subject := fmt.Sprintf("Adoption Application Received for %s (ID: %s)", petDetails.GetName(), event.ApplicationID)
	body := fmt.Sprintf(`
		<h1>Adoption Application Received!</h1>
		<p>Dear %s,</p>
		<p>Thank you for submitting your adoption application (ID: %s) for <strong>%s</strong> (Pet ID: %s).</p>
		<p>Your application is currently in status: <strong>%s</strong>.</p>
		<p>We will review your application and get back to you soon.</p>
		<p>Thank you,<br/>The PetStore Team</p>
	`, userDetails.GetFullName(), event.ApplicationID, petDetails.GetName(), event.PetID, event.Status)

	err = s.emailSender.SendEmail([]string{recipientEmail}, subject, body, true) // true for HTML email
	if err != nil {
		log.Printf("Notification Service | Error sending 'Application Created' email to %s for AppID %s: %v", recipientEmail, event.ApplicationID, err)
		return fmt.Errorf("failed to send application created email: %w", err)
	}

	log.Printf("Notification Service | 'Application Created' email sent successfully to %s for AppID %s.", recipientEmail, event.ApplicationID)
	return nil
}

// HandleAdoptionApplicationStatusUpdated processes an event when an adoption application's status changes.
func (s *NotificationService) HandleAdoptionApplicationStatusUpdated(ctx context.Context, event consumer.AdoptionApplicationStatusUpdatedEvent) error {
	log.Printf("Notification Service | Handling AdoptionApplicationStatusUpdated event for AppID: %s, NewStatus: %s",
		event.ApplicationID, event.NewStatus)

	// 1. Fetch User Details
	userDetails, err := s.userServiceClient.GetUserDetails(ctx, event.UserID)
	if err != nil {
		log.Printf("Notification Service | Error fetching user details for UserID %s: %v", event.UserID, err)
		return fmt.Errorf("failed to fetch user details for status update: %w", err)
	}
	if userDetails == nil || userDetails.GetEmail() == "" { // Use GetEmail()
		log.Printf("Notification Service | User details or email not found for UserID %s", event.UserID)
		return fmt.Errorf("user email not found for UserID %s", event.UserID)
	}

	// 2. Fetch Pet Details
	petDetails, err := s.petServiceClient.GetPetDetails(ctx, event.PetID)
	if err != nil {
		log.Printf("Notification Service | Error fetching pet details for PetID %s: %v", event.PetID, err)
		return fmt.Errorf("failed to fetch pet details for status update: %w", err)
	}
	if petDetails == nil || petDetails.GetName() == "" { // Use GetName()
		log.Printf("Notification Service | Pet details or name not found for PetID %s", event.PetID)
		return fmt.Errorf("pet details not found or name is empty for PetID %s", event.PetID)
	}

	// 3. Construct and Send Email
	recipientEmail := userDetails.GetEmail()
	subject := fmt.Sprintf("Update on Your Adoption Application for %s (ID: %s)", petDetails.GetName(), event.ApplicationID)
	body := fmt.Sprintf(`
		<h1>Adoption Application Status Update!</h1>
		<p>Dear %s,</p>
		<p>There's an update on your adoption application (ID: %s) for <strong>%s</strong> (Pet ID: %s).</p>
		<p>Your application status is now: <strong>%s</strong>.</p>
	`, userDetails.GetFullName(), event.ApplicationID, petDetails.GetName(), event.PetID, event.NewStatus)

	if event.ReviewNotes != "" {
		body += fmt.Sprintf("<p>Reviewer's Notes: %s</p>", event.ReviewNotes)
	}

	if event.NewStatus == "APPROVED" { // Assuming "APPROVED" is the string representation from domain/consumer event
		body += "<p>Congratulations! Your application has been approved. We will contact you shortly with the next steps.</p>"
	} else if event.NewStatus == "REJECTED" {
		body += "<p>We regret to inform you that your application was not approved at this time. Thank you for your interest.</p>"
	}

	body += "<p>Thank you,<br/>The PetStore Team</p>"

	err = s.emailSender.SendEmail([]string{recipientEmail}, subject, body, true) // true for HTML email
	if err != nil {
		log.Printf("Notification Service | Error sending 'Status Updated' email to %s for AppID %s: %v", recipientEmail, event.ApplicationID, err)
		return fmt.Errorf("failed to send status update email: %w", err)
	}

	log.Printf("Notification Service | 'Status Updated' email sent successfully to %s for AppID %s.", recipientEmail, event.ApplicationID)
	return nil
}

// Ensure NotificationService implements consumer.EventHandler at compile time
var _ consumer.EventHandler = (*NotificationService)(nil)