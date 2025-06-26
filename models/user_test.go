package models

import (
	"strings"
	"testing"

	"github.com/gobuffalo/suite/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ModelSuite struct {
	*suite.Model
}

func TestModelSuite(t *testing.T) {
	model := suite.NewModel()

	ms := &ModelSuite{
		Model: model,
	}
	suite.Run(t, ms)
}

func (ms *ModelSuite) Test_User_Create() {
	count, err := ms.DB.Count("users")
	ms.NoError(err)
	ms.Equal(0, count)

	user := &User{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "password123",
		Role:     RoleUser,
	}

	verrs, err := ms.DB.ValidateAndCreate(user)
	ms.NoError(err)
	ms.False(verrs.HasAny())

	count, err = ms.DB.Count("users")
	ms.NoError(err)
	ms.Equal(1, count)

	// Check that password was hashed
	ms.NotEqual("password123", user.PasswordHash)
	ms.True(user.ValidatePassword("password123"))
}

func (ms *ModelSuite) Test_User_Create_Validation_Errors() {
	user := &User{}
	verrs, err := ms.DB.ValidateAndCreate(user)
	ms.NoError(err)
	ms.True(verrs.HasAny())

	// Check for required field errors
	ms.True(len(verrs.Get("name")) > 0)
	ms.True(len(verrs.Get("email")) > 0)
	ms.True(len(verrs.Get("password")) > 0)
}

func (ms *ModelSuite) Test_User_Create_Duplicate_Email() {
	// Create first user
	user1 := &User{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "password123",
		Role:     RoleUser,
	}
	verrs, err := ms.DB.ValidateAndCreate(user1)
	ms.NoError(err)
	ms.False(verrs.HasAny())

	// Try to create second user with same email
	user2 := &User{
		Name:     "Jane Doe",
		Email:    "john@example.com",
		Password: "password456",
		Role:     RoleUser,
	}
	verrs, err = ms.DB.ValidateAndCreate(user2)
	ms.NoError(err)
	ms.True(verrs.HasAny())
	ms.True(len(verrs.Get("email")) > 0)
}

func (ms *ModelSuite) Test_User_Email_Normalization() {
	user := &User{
		Name:     "John Doe",
		Email:    "  JOHN@EXAMPLE.COM  ",
		Password: "password123",
		Role:     RoleUser,
	}

	verrs, err := ms.DB.ValidateAndCreate(user)
	ms.NoError(err)
	
	// Check if there are any validation errors and print them for debugging
	if verrs.HasAny() {
		ms.T().Logf("Validation errors: %v", verrs.Error())
	}
	ms.False(verrs.HasAny())

	// Email should be normalized to lowercase and trimmed
	ms.Equal("john@example.com", user.Email)
}

func (ms *ModelSuite) Test_User_Name_Trimming() {
	user := &User{
		Name:     "  John Doe  ",
		Email:    "john@example.com",
		Password: "password123",
		Role:     RoleUser,
	}

	verrs, err := ms.DB.ValidateAndCreate(user)
	ms.NoError(err)
	ms.False(verrs.HasAny())

	// Name should be trimmed
	ms.Equal("John Doe", user.Name)
}

func (ms *ModelSuite) Test_User_Default_Role() {
	user := &User{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "password123",
		// No role specified
	}

	verrs, err := ms.DB.ValidateAndCreate(user)
	ms.NoError(err)
	
	// Check if there are any validation errors and print them for debugging
	if verrs.HasAny() {
		ms.T().Logf("Validation errors: %v", verrs.Error())
	}
	ms.False(verrs.HasAny())

	// Should default to user role
	ms.Equal(RoleUser, user.Role)
}

func (ms *ModelSuite) Test_User_Password_Validation() {
	// Test short password
	user := &User{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "short",
		Role:     RoleUser,
	}
	verrs, err := ms.DB.ValidateAndCreate(user)
	ms.NoError(err)
	ms.True(verrs.HasAny())
	ms.True(len(verrs.Get("password")) > 0)

	// Test long password
	user.Password = strings.Repeat("a", 101)
	verrs, err = ms.DB.ValidateAndCreate(user)
	ms.NoError(err)
	ms.True(verrs.HasAny())
	ms.True(len(verrs.Get("password")) > 0)
}

func (ms *ModelSuite) Test_User_Password_Confirmation() {
	user := &User{
		Name:            "John Doe",
		Email:           "john@example.com",
		Password:        "password123",
		PasswordConfirm: "password456",
		Role:            RoleUser,
	}

	verrs, err := ms.DB.ValidateAndCreate(user)
	ms.NoError(err)
	ms.True(verrs.HasAny())
	ms.True(len(verrs.Get("password_confirm")) > 0)
}

func (ms *ModelSuite) Test_User_Email_Validation() {
	user := &User{
		Name:     "John Doe",
		Email:    "invalid-email",
		Password: "password123",
		Role:     RoleUser,
	}

	verrs, err := ms.DB.ValidateAndCreate(user)
	ms.NoError(err)
	ms.True(verrs.HasAny())
	ms.True(len(verrs.Get("email")) > 0)
}

func (ms *ModelSuite) Test_User_Role_Validation() {
	user := &User{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "password123",
		Role:     "invalid_role",
	}

	verrs, err := ms.DB.ValidateAndCreate(user)
	ms.NoError(err)
	ms.True(verrs.HasAny())
	ms.True(len(verrs.Get("role")) > 0)
}

func (ms *ModelSuite) Test_User_Update() {
	user := &User{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "password123",
		Role:     RoleUser,
	}

	verrs, err := ms.DB.ValidateAndCreate(user)
	ms.NoError(err)
	ms.False(verrs.HasAny())

	// Update user
	user.Name = "John Smith"
	user.Email = "johnsmith@example.com"
	verrs, err = ms.DB.ValidateAndUpdate(user)
	ms.NoError(err)
	ms.False(verrs.HasAny())

	ms.Equal("John Smith", user.Name)
	ms.Equal("johnsmith@example.com", user.Email)
}

func (ms *ModelSuite) Test_User_Update_Password() {
	user := &User{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "password123",
		Role:     RoleUser,
	}

	verrs, err := ms.DB.ValidateAndCreate(user)
	ms.NoError(err)
	ms.False(verrs.HasAny())

	oldPasswordHash := user.PasswordHash

	// Update password
	user.Password = "newpassword123"
	verrs, err = ms.DB.ValidateAndUpdate(user)
	ms.NoError(err)
	ms.False(verrs.HasAny())

	// Password hash should be different
	ms.NotEqual(oldPasswordHash, user.PasswordHash)
	ms.True(user.ValidatePassword("newpassword123"))
	ms.False(user.ValidatePassword("password123"))
}

// Unit tests (non-database tests)
func TestUser_SetPassword(t *testing.T) {
	user := &User{}
	err := user.SetPassword("testpassword")
	require.NoError(t, err)
	assert.NotEmpty(t, user.PasswordHash)
	assert.NotEqual(t, "testpassword", user.PasswordHash)
}

func TestUser_ValidatePassword(t *testing.T) {
	user := &User{}
	err := user.SetPassword("testpassword")
	require.NoError(t, err)

	// Correct password
	assert.True(t, user.ValidatePassword("testpassword"))

	// Wrong password
	assert.False(t, user.ValidatePassword("wrongpassword"))
}

func TestUser_IsAdmin(t *testing.T) {
	user := &User{Role: RoleAdmin}
	assert.True(t, user.IsAdmin())

	user.Role = RoleUser
	assert.False(t, user.IsAdmin())
}

func TestUser_IsUser(t *testing.T) {
	user := &User{Role: RoleUser}
	assert.True(t, user.IsUser())

	user.Role = RoleAdmin
	assert.False(t, user.IsUser())
}

func TestUser_String(t *testing.T) {
	user := &User{
		Name:         "John Doe",
		Email:        "john@example.com",
		PasswordHash: "secret_hash",
		Role:         RoleUser,
	}

	jsonStr := user.String()
	
	// Should contain user data
	assert.Contains(t, jsonStr, "John Doe")
	assert.Contains(t, jsonStr, "john@example.com")
	assert.Contains(t, jsonStr, RoleUser)
	
	// Should NOT contain password hash
	assert.NotContains(t, jsonStr, "secret_hash")
}

func TestUsers_String(t *testing.T) {
	users := Users{
		{Name: "John Doe", Email: "john@example.com"},
		{Name: "Jane Doe", Email: "jane@example.com"},
	}

	jsonStr := users.String()
	assert.Contains(t, jsonStr, "John Doe")
	assert.Contains(t, jsonStr, "Jane Doe")
}
