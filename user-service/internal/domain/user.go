package domain

import (
	"time"

	"golang.org/x/crypto/bcrypt" // For password hashing
)

// User represents a user entity in the system.
// It aligns with the data stored in MongoDB and the gRPC messages.
type User struct {
	ID             string    `bson:"_id,omitempty" json:"id,omitempty"` // MongoDB primary key
	Username       string    `bson:"username" json:"username"`
	Email          string    `bson:"email" json:"email"`
	HashedPassword string    `bson:"hashed_password" json:"-"` // Avoid exposing this in JSON responses directly
	FullName       string    `bson:"full_name" json:"full_name"`
	CreatedAt      time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt      time.Time `bson:"updated_at" json:"updated_at"`
	// DeletedAt    *time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"` // For soft deletes, optional
}

// HashPassword generates a bcrypt hash of the password.
// It's a good practice to use a moderate cost for bcrypt.
func HashPassword(password string) (string, error) {
	// bcrypt.DefaultCost is 10, which is generally a good balance.
	// You can increase the cost for higher security, but it will be slower.
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPasswordHash compares a plain-text password with a stored bcrypt hash.
// Returns true if the password matches the hash, false otherwise.
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil // If err is nil, the password matches
}

// BeforeCreate (or similar, if using an ORM-like hook system, not directly applicable here but good concept)
// For manual setting, you'd call this from your usecase or repository layer before inserting.
func (u *User) PrepareForCreate() {
	u.CreatedAt = time.Now().UTC()
	u.UpdatedAt = time.Now().UTC()
	// Any other default setting or validation before creation
}

// BeforeUpdate (concept)
func (u *User) PrepareForUpdate() {
	u.UpdatedAt = time.Now().UTC()
	// Any other default setting or validation before update
}

// Validate (concept - actual validation might be more complex and use a library)
// func (u *User) Validate() error {
// 	if u.Email == "" {
// 		return errors.New("email is required")
// 	}
// 	if u.Username == "" {
// 		return errors.New("username is required")
// 	}
// 	// ... more validation rules
// 	return nil
// }

