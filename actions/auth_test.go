package actions

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/akingundogdu/production-ready-go-backend-architecture/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/gofrs/uuid"
)

func (as *ActionSuite) Test_RegisterHandler_Success() {
	req := RegisterRequest{
		Name:            "John Doe",
		Email:           "john@example.com",
		Password:        "password123",
		PasswordConfirm: "password123",
	}

	res := as.JSON("/auth/register").Post(req)
	as.Equal(http.StatusCreated, res.Code)

	var response AuthResponse
	err := json.Unmarshal(res.Body.Bytes(), &response)
	as.NoError(err)

	// Check response fields
	as.NotEmpty(response.Token)
	as.NotEmpty(response.User)
	as.False(response.ExpiresAt.IsZero())

	// Verify user was created in database
	user := &models.User{}
	err = as.DB.Where("email = ?", "john@example.com").First(user)
	as.NoError(err)
	as.Equal("John Doe", user.Name)
	as.Equal("john@example.com", user.Email)
	as.Equal(models.RoleUser, user.Role)
}

func (as *ActionSuite) Test_RegisterHandler_Validation_Errors() {
	// Test missing required fields
	req := RegisterRequest{}
	res := as.JSON("/auth/register").Post(req)
	as.Equal(http.StatusBadRequest, res.Code)

	var response ErrorResponse
	err := json.Unmarshal(res.Body.Bytes(), &response)
	as.NoError(err)
	as.Equal("Validation failed", response.Error)
	as.NotEmpty(response.Details)
}

func (as *ActionSuite) Test_RegisterHandler_Password_Mismatch() {
	req := RegisterRequest{
		Name:            "John Doe",
		Email:           "john@example.com",
		Password:        "password123",
		PasswordConfirm: "password456",
	}

	res := as.JSON("/auth/register").Post(req)
	as.Equal(http.StatusBadRequest, res.Code)

	var response ErrorResponse
	err := json.Unmarshal(res.Body.Bytes(), &response)
	as.NoError(err)
	as.Equal("Password confirmation does not match", response.Error)
}

func (as *ActionSuite) Test_RegisterHandler_Duplicate_Email() {
	// Create first user
	user1 := &models.User{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "password123",
		Role:     models.RoleUser,
	}
	verrs, err := as.DB.ValidateAndCreate(user1)
	as.NoError(err)
	as.False(verrs.HasAny())

	// Try to register with same email
	req := RegisterRequest{
		Name:            "Jane Doe",
		Email:           "john@example.com",
		Password:        "password123",
		PasswordConfirm: "password123",
	}

	res := as.JSON("/auth/register").Post(req)
	as.Equal(http.StatusBadRequest, res.Code)

	var response ErrorResponse
	err = json.Unmarshal(res.Body.Bytes(), &response)
	as.NoError(err)
	as.Equal("Validation failed", response.Error)
	as.Contains(response.Details["email"], "already taken")
}

func (as *ActionSuite) Test_LoginHandler_Success() {
	// Create user
	user := &models.User{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "password123",
		Role:     models.RoleUser,
	}
	verrs, err := as.DB.ValidateAndCreate(user)
	as.NoError(err)
	as.False(verrs.HasAny())

	// Login
	req := LoginRequest{
		Email:    "john@example.com",
		Password: "password123",
	}

	res := as.JSON("/auth/login").Post(req)
	as.Equal(http.StatusOK, res.Code)

	var response AuthResponse
	err = json.Unmarshal(res.Body.Bytes(), &response)
	as.NoError(err)

	// Check response fields
	as.NotEmpty(response.Token)
	as.NotEmpty(response.User)
	as.False(response.ExpiresAt.IsZero())
}

func (as *ActionSuite) Test_LoginHandler_Invalid_Credentials() {
	// Create user
	user := &models.User{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "password123",
		Role:     models.RoleUser,
	}
	verrs, err := as.DB.ValidateAndCreate(user)
	as.NoError(err)
	as.False(verrs.HasAny())

	// Login with wrong password
	req := LoginRequest{
		Email:    "john@example.com",
		Password: "wrongpassword",
	}

	res := as.JSON("/auth/login").Post(req)
	as.Equal(http.StatusUnauthorized, res.Code)

	var response ErrorResponse
	err = json.Unmarshal(res.Body.Bytes(), &response)
	as.NoError(err)
	as.Equal("Invalid credentials", response.Error)
}

func (as *ActionSuite) Test_LoginHandler_Nonexistent_User() {
	req := LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "password123",
	}

	res := as.JSON("/auth/login").Post(req)
	as.Equal(http.StatusUnauthorized, res.Code)

	var response ErrorResponse
	err := json.Unmarshal(res.Body.Bytes(), &response)
	as.NoError(err)
	as.Equal("Invalid credentials", response.Error)
}

func (as *ActionSuite) Test_MeHandler_Success() {
	// Create user
	user := &models.User{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "password123",
		Role:     models.RoleUser,
	}
	verrs, err := as.DB.ValidateAndCreate(user)
	as.NoError(err)
	as.False(verrs.HasAny())

	// Generate token
	token, _, err := GenerateJWT(user)
	as.NoError(err)

	// Request user info
	req := as.JSON("/auth/me")
	req.Headers = map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", token),
	}
	res := req.Get()
	as.Equal(http.StatusOK, res.Code)

	var responseUser models.User
	err = json.Unmarshal(res.Body.Bytes(), &responseUser)
	as.NoError(err)
	as.Equal(user.ID, responseUser.ID)
	as.Equal(user.Email, responseUser.Email)
}

func (as *ActionSuite) Test_MeHandler_Invalid_Token() {
	req := as.JSON("/auth/me")
	req.Headers = map[string]string{
		"Authorization": "Bearer invalid-token",
	}
	res := req.Get()
	as.Equal(http.StatusUnauthorized, res.Code)

	var response ErrorResponse
	err := json.Unmarshal(res.Body.Bytes(), &response)
	as.NoError(err)
	as.Equal("Invalid token", response.Error)
}

func (as *ActionSuite) Test_MeHandler_No_Authorization_Header() {
	res := as.JSON("/auth/me").Get()
	as.Equal(http.StatusUnauthorized, res.Code)

	var response ErrorResponse
	err := json.Unmarshal(res.Body.Bytes(), &response)
	as.NoError(err)
	as.Equal("Authorization header required", response.Error)
}

func (as *ActionSuite) Test_RefreshTokenHandler_Success() {
	// Create user
	user := &models.User{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "password123",
		Role:     models.RoleUser,
	}
	verrs, err := as.DB.ValidateAndCreate(user)
	as.NoError(err)
	as.False(verrs.HasAny())

	// Generate token
	token, _, err := GenerateJWT(user)
	as.NoError(err)

	// Wait a moment to ensure different timestamps
	time.Sleep(time.Second * 1)

	// Refresh token
	req := as.JSON("/auth/refresh")
	req.Headers = map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", token),
	}
	res := req.Post(nil)
	as.Equal(http.StatusOK, res.Code)

	var response AuthResponse
	err = json.Unmarshal(res.Body.Bytes(), &response)
	as.NoError(err)

	// Check response fields
	as.NotEmpty(response.Token)
	as.NotEqual(token, response.Token) // New token should be different
	as.NotEmpty(response.User)
	as.False(response.ExpiresAt.IsZero())
}

func (as *ActionSuite) Test_AuthMiddleware_Invalid_Authorization_Format() {
	req := as.JSON("/auth/me")
	req.Headers = map[string]string{
		"Authorization": "InvalidFormat token",
	}
	res := req.Get()
	as.Equal(http.StatusUnauthorized, res.Code)

	var response ErrorResponse
	err := json.Unmarshal(res.Body.Bytes(), &response)
	as.NoError(err)
	as.Equal("Invalid authorization header format", response.Error)
}

func (as *ActionSuite) Test_AdminMiddleware_Success() {
	// Create admin user
	user := &models.User{
		Name:     "Admin User",
		Email:    "admin@example.com",
		Password: "password123",
		Role:     models.RoleAdmin,
	}
	verrs, err := as.DB.ValidateAndCreate(user)
	as.NoError(err)
	as.False(verrs.HasAny())

	// Generate token
	token, _, err := GenerateJWT(user)
	as.NoError(err)

	// Since we don't have admin routes yet, we need to create a test route
	// This would be tested when we add actual admin endpoints
	as.NotEmpty(token) // Just verify token was generated
}

func (as *ActionSuite) Test_AdminMiddleware_Forbidden() {
	// Create regular user
	user := &models.User{
		Name:     "Regular User",
		Email:    "user@example.com",
		Password: "password123",
		Role:     models.RoleUser,
	}
	verrs, err := as.DB.ValidateAndCreate(user)
	as.NoError(err)
	as.False(verrs.HasAny())

	// Generate token
	token, _, err := GenerateJWT(user)
	as.NoError(err)

	// Try to access admin route (would be tested when we add actual admin endpoints)
	// This is a placeholder for when we implement admin endpoints
	as.NotEmpty(token) // Just verify token was generated
}

// Unit tests for JWT functions
func TestGenerateJWT(t *testing.T) {
	user := &models.User{
		Name:  "John Doe",
		Email: "john@example.com",
		Role:  models.RoleUser,
	}
	user.ID = uuid.Must(uuid.NewV4()) // Generate a UUID

	token, expiresAt, err := GenerateJWT(user)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.True(t, expiresAt.After(time.Now()))
}

func TestValidateJWT(t *testing.T) {
	user := &models.User{
		Name:  "John Doe",
		Email: "john@example.com",
		Role:  models.RoleUser,
	}
	user.ID = uuid.Must(uuid.NewV4()) // Generate a UUID

	// Generate token
	token, _, err := GenerateJWT(user)
	require.NoError(t, err)

	// Validate token
	claims, err := ValidateJWT(token)
	require.NoError(t, err)
	assert.Equal(t, user.ID.String(), claims.UserID)
	assert.Equal(t, user.Email, claims.Email)
	assert.Equal(t, user.Role, claims.Role)
}

func TestValidateJWT_Invalid_Token(t *testing.T) {
	_, err := ValidateJWT("invalid-token")
	assert.Error(t, err)
}

func TestValidateJWT_Empty_Token(t *testing.T) {
	_, err := ValidateJWT("")
	assert.Error(t, err)
}

// Helper function to create authenticated request
func (as *ActionSuite) createAuthenticatedUser(role string) (*models.User, string) {
	user := &models.User{
		Name:     "Test User",
		Email:    "test@example.com",
		Password: "password123",
		Role:     role,
	}
	verrs, err := as.DB.ValidateAndCreate(user)
	as.NoError(err)
	as.False(verrs.HasAny())

	token, _, err := GenerateJWT(user)
	as.NoError(err)

	return user, token
} 