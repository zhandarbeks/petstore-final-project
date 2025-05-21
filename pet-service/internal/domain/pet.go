package domain

import (
	"time"
	// "errors" // Uncomment if you add validation methods that return errors
)

// AdoptionStatus mirrors the enum defined in your pet.proto file.
// It's good practice to have a corresponding type in your domain.
type AdoptionStatus string

const (
	StatusUnspecified    AdoptionStatus = "UNSPECIFIED" // Should match proto default if any, or be a zero-value representation
	StatusAvailable      AdoptionStatus = "AVAILABLE"
	StatusPendingAdoption AdoptionStatus = "PENDING_ADOPTION"
	StatusAdopted        AdoptionStatus = "ADOPTED"
)

// Pet represents a pet entity in the system.
type Pet struct {
	ID               string         `bson:"_id,omitempty" json:"id,omitempty"` // MongoDB primary key
	Name             string         `bson:"name" json:"name"`
	Species          string         `bson:"species" json:"species"`   // e.g., Dog, Cat
	Breed            string         `bson:"breed" json:"breed"`     // e.g., Labrador, Siamese
	Age              int32          `bson:"age" json:"age"`         // Age in years or months, define unit consistently
	Description      string         `bson:"description" json:"description"`
	AdoptionStatus   AdoptionStatus `bson:"adoption_status" json:"adoption_status"`
	ListedByUserID   string         `bson:"listed_by_user_id,omitempty" json:"listed_by_user_id,omitempty"` // ID of the user who listed the pet
	AdoptedByUserID  string         `bson:"adopted_by_user_id,omitempty" json:"adopted_by_user_id,omitempty"` // ID of the user who adopted the pet
	ImageURLs        []string       `bson:"image_urls,omitempty" json:"image_urls,omitempty"`                 // List of URLs for pet images
	CreatedAt        time.Time      `bson:"created_at" json:"created_at"`
	UpdatedAt        time.Time      `bson:"updated_at" json:"updated_at"`
	// Additional fields like 'vaccination_status', 'gender', 'size', 'location' could be added.
}

// PrepareForCreate sets the CreatedAt and UpdatedAt timestamps for a new pet.
// It also defaults AdoptionStatus to AVAILABLE if not set.
func (p *Pet) PrepareForCreate() {
	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now
	if p.AdoptionStatus == "" || p.AdoptionStatus == StatusUnspecified {
		p.AdoptionStatus = StatusAvailable
	}
	if p.ImageURLs == nil { // Ensure ImageURLs is not nil to avoid issues with MongoDB if omitempty is used
		p.ImageURLs = []string{}
	}
}

// PrepareForUpdate sets the UpdatedAt timestamp for an existing pet.
func (p *Pet) PrepareForUpdate() {
	p.UpdatedAt = time.Now().UTC()
	if p.ImageURLs == nil {
		p.ImageURLs = []string{}
	}
}

// IsValidAdoptionStatus checks if the provided status is a valid AdoptionStatus.
func IsValidAdoptionStatus(status AdoptionStatus) bool {
	switch status {
	case StatusAvailable, StatusPendingAdoption, StatusAdopted, StatusUnspecified: // StatusUnspecified might be valid for filtering
		return true
	default:
		return false
	}
}

// Example validation (can be expanded or use a library)
// func (p *Pet) Validate() error {
// 	if p.Name == "" {
// 		return errors.New("pet name is required")
// 	}
// 	if p.Species == "" {
// 		return errors.New("pet species is required")
// 	}
// 	if p.Age < 0 {
// 		return errors.New("pet age cannot be negative")
// 	}
// 	if !IsValidAdoptionStatus(p.AdoptionStatus) && p.AdoptionStatus != "" { // Allow empty if it defaults
// 		return errors.New("invalid adoption status")
// 	}
// 	return nil
// }
