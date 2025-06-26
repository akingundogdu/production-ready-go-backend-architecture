package models

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gobuffalo/validate/v3/validators"
	"github.com/gofrs/uuid"
	"golang.org/x/crypto/bcrypt"
)

// User roles constants
const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

// User is used by pop to map your users database table to your go code.
type User struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Name         string    `json:"name" db:"name"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"` // Never expose password hash in JSON
	Role         string    `json:"role" db:"role"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
	
	// Virtual fields (not stored in database)
	Password        string `json:"-" db:"-"` // For password input
	PasswordConfirm string `json:"-" db:"-"` // For password confirmation
}

// String is not required by pop and may be deleted
func (u User) String() string {
	// Create a safe copy without sensitive data
	safeCopy := struct {
		ID        uuid.UUID `json:"id"`
		Name      string    `json:"name"`
		Email     string    `json:"email"`
		Role      string    `json:"role"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		Role:      u.Role,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
	ju, _ := json.Marshal(safeCopy)
	return string(ju)
}

// Users is not required by pop and may be deleted
type Users []User

// String is not required by pop and may be deleted
func (u Users) String() string {
	ju, _ := json.Marshal(u)
	return string(ju)
}

// SetPassword hashes the password and sets the PasswordHash field
func (u *User) SetPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hash)
	return nil
}

// ValidatePassword checks if the provided password matches the stored hash
func (u *User) ValidatePassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// IsAdmin checks if the user has admin role
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// IsUser checks if the user has user role
func (u *User) IsUser() bool {
	return u.Role == RoleUser
}

// BeforeCreate sets default values before creating a user
func (u *User) BeforeCreate(tx *pop.Connection) error {
	// Hash password if provided
	if u.Password != "" {
		return u.SetPassword(u.Password)
	}
	
	return nil
}

// BeforeUpdate normalizes fields before updating
func (u *User) BeforeUpdate(tx *pop.Connection) error {
	// Hash password if provided
	if u.Password != "" {
		return u.SetPassword(u.Password)
	}
	
	return nil
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
func (u *User) Validate(tx *pop.Connection) (*validate.Errors, error) {
	// Normalize fields before validation
	u.Email = strings.ToLower(strings.TrimSpace(u.Email))
	u.Name = strings.TrimSpace(u.Name)
	
	// Set default role if not specified
	if u.Role == "" {
		u.Role = RoleUser
	}
	
	errors := validate.Validate(
		&validators.StringIsPresent{Field: u.Name, Name: "Name"},
		&validators.StringLengthInRange{Field: u.Name, Name: "Name", Min: 2, Max: 100},
		&validators.StringIsPresent{Field: u.Email, Name: "Email"},
		&validators.EmailIsPresent{Field: u.Email, Name: "Email"},
		// Role is not required as it will be set to default if empty
	)
	
	// Validate role is one of the allowed roles (empty is allowed, will be set to default)
	if u.Role != "" && u.Role != RoleUser && u.Role != RoleAdmin {
		errors.Add("role", "Role must be either 'user' or 'admin'")
	}
	
	// Validate email format with regex
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(u.Email) {
		errors.Add("email", "Email format is invalid")
	}
	
	return errors, nil
}

// ValidateCreate gets run every time you call "pop.ValidateAndCreate" method.
func (u *User) ValidateCreate(tx *pop.Connection) (*validate.Errors, error) {
	errors := validate.NewErrors()
	
	// Password is required for creation
	if u.Password == "" {
		errors.Add("password", "Password is required")
	} else {
		// Validate password strength
		if len(u.Password) < 8 {
			errors.Add("password", "Password must be at least 8 characters long")
		}
		if len(u.Password) > 100 {
			errors.Add("password", "Password must be less than 100 characters")
		}
		
		// Check password confirmation if provided
		if u.PasswordConfirm != "" && u.Password != u.PasswordConfirm {
			errors.Add("password_confirm", "Password confirmation does not match")
		}
	}
	
	// Check if email is already taken
	existingUser := &User{}
	err := tx.Where("email = ?", strings.ToLower(u.Email)).First(existingUser)
	if err == nil {
		errors.Add("email", "Email is already taken")
	}
	
	return errors, nil
}

// ValidateUpdate gets run every time you call "pop.ValidateAndUpdate" method.
func (u *User) ValidateUpdate(tx *pop.Connection) (*validate.Errors, error) {
	errors := validate.NewErrors()
	
	// Validate password only if provided for update
	if u.Password != "" {
		if len(u.Password) < 8 {
			errors.Add("password", "Password must be at least 8 characters long")
		}
		if len(u.Password) > 100 {
			errors.Add("password", "Password must be less than 100 characters")
		}
		
		// Check password confirmation if provided
		if u.PasswordConfirm != "" && u.Password != u.PasswordConfirm {
			errors.Add("password_confirm", "Password confirmation does not match")
		}
	}
	
	// Check if email is already taken by another user
	existingUser := &User{}
	err := tx.Where("email = ? AND id != ?", strings.ToLower(u.Email), u.ID).First(existingUser)
	if err == nil {
		errors.Add("email", "Email is already taken")
	}
	
	return errors, nil
}
