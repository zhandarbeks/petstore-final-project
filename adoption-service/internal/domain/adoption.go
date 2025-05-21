package domain

import (
	"time"
	// "errors" // Uncomment if you add validation methods
)

// ApplicationStatus mirrors the enum defined in your adoption.proto file.
type ApplicationStatus string

const (
	StatusAppUnspecified    ApplicationStatus = "UNSPECIFIED"
	StatusAppPendingReview  ApplicationStatus = "PENDING_REVIEW"
	StatusAppApproved       ApplicationStatus = "APPROVED"
	StatusAppRejected       ApplicationStatus = "REJECTED"
	StatusAppCancelledByUser ApplicationStatus = "CANCELLED_BY_USER"
	// StatusAppCompleted might be another status if 'APPROVED' doesn't mean finalized.
	// Your proto has CANCELLED_BY_USER = 5, so I've included it.
)

// AdoptionApplication represents an adoption application entity in the system.
type AdoptionApplication struct {
	ID                 string            `bson:"_id,omitempty" json:"id,omitempty"` // MongoDB primary key
	UserID             string            `bson:"user_id" json:"user_id"`            // ID of the user applying
	PetID              string            `bson:"pet_id" json:"pet_id"`              // ID of the pet being applied for
	Status             ApplicationStatus `bson:"status" json:"status"`
	ApplicationNotes   string            `bson:"application_notes,omitempty" json:"application_notes,omitempty"` // Notes from the applicant
	ReviewNotes        string            `bson:"review_notes,omitempty" json:"review_notes,omitempty"`          // Notes from the admin/reviewer
	CreatedAt          time.Time         `bson:"created_at" json:"created_at"`
	UpdatedAt          time.Time         `bson:"updated_at" json:"updated_at"`
}

// PrepareForCreate sets the CreatedAt, UpdatedAt timestamps and default status for a new application.
func (app *AdoptionApplication) PrepareForCreate() {
	now := time.Now().UTC()
	app.CreatedAt = now
	app.UpdatedAt = now
	if app.Status == "" || app.Status == StatusAppUnspecified {
		app.Status = StatusAppPendingReview // Default status for a new application
	}
}

// PrepareForUpdate sets the UpdatedAt timestamp for an existing application.
func (app *AdoptionApplication) PrepareForUpdate() {
	app.UpdatedAt = time.Now().UTC()
}

// IsValidApplicationStatus checks if the provided status is a valid ApplicationStatus.
func IsValidApplicationStatus(status ApplicationStatus) bool {
	switch status {
	case StatusAppPendingReview, StatusAppApproved, StatusAppRejected, StatusAppCancelledByUser, StatusAppUnspecified:
		return true
	default:
		return false
	}
}

// Example validation (can be expanded or use a library)
// func (app *AdoptionApplication) Validate() error {
// 	if app.UserID == "" {
// 		return errors.New("user ID is required for application")
// 	}
// 	if app.PetID == "" {
// 		return errors.New("pet ID is required for application")
// 	}
// 	if !IsValidApplicationStatus(app.Status) && app.Status != "" {
// 		return errors.New("invalid application status")
// 	}
// 	return nil
// }