package actions

import (
	"net/http"
	"strings"
	"time"

	"github.com/akingundogdu/production-ready-go-backend-architecture/models"
	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/envy"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gofrs/uuid"
)

// JWT configuration
var jwtSecretKey = []byte(envy.Get("JWT_SECRET", "your-super-secret-jwt-key-change-in-production"))

// JWT Claims structure
type JWTClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// Auth request/response structures
type RegisterRequest struct {
	Name            string `json:"name" validate:"required,min=2,max=100"`
	Email           string `json:"email" validate:"required,email"`
	Password        string `json:"password" validate:"required,min=8,max=100"`
	PasswordConfirm string `json:"password_confirm" validate:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type AuthResponse struct {
	Token     string      `json:"token"`
	User      interface{} `json:"user"`
	ExpiresAt time.Time   `json:"expires_at"`
}

type ErrorResponse struct {
	Error   string            `json:"error"`
	Details map[string]string `json:"details,omitempty"`
}

// GenerateJWT creates a new JWT token for a user
func GenerateJWT(user *models.User) (string, time.Time, error) {
	expirationTime := time.Now().Add(24 * time.Hour) // Token expires in 24 hours
	
	now := time.Now()
	claims := &JWTClaims{
		UserID: user.ID.String(),
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "production-ready-go-backend",
			Subject:   user.ID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecretKey)
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expirationTime, nil
}

// ValidateJWT validates and parses a JWT token
func ValidateJWT(tokenString string) (*JWTClaims, error) {
	claims := &JWTClaims{}
	
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtSecretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, jwt.ErrTokenMalformed
	}

	return claims, nil
}

// RegisterHandler handles user registration
// POST /auth/register
func RegisterHandler(c buffalo.Context) error {
	var req RegisterRequest
	if err := c.Bind(&req); err != nil {
		return c.Render(http.StatusBadRequest, r.JSON(ErrorResponse{
			Error: "Invalid request format",
		}))
	}

	// Validate password confirmation
	if req.Password != req.PasswordConfirm {
		return c.Render(http.StatusBadRequest, r.JSON(ErrorResponse{
			Error: "Password confirmation does not match",
		}))
	}

	// Create new user
	user := &models.User{
		Name:            req.Name,
		Email:           req.Email,
		Password:        req.Password,
		PasswordConfirm: req.PasswordConfirm,
		Role:            models.RoleUser, // Default role
	}

	// Validate and create user
	verrs, err := models.DB.ValidateAndCreate(user)
	if err != nil {
		return c.Render(http.StatusInternalServerError, r.JSON(ErrorResponse{
			Error: "Failed to create user",
		}))
	}

	if verrs.HasAny() {
		details := make(map[string]string)
		for _, key := range verrs.Keys() {
			details[key] = strings.Join(verrs.Get(key), ", ")
		}
		return c.Render(http.StatusBadRequest, r.JSON(ErrorResponse{
			Error:   "Validation failed",
			Details: details,
		}))
	}

	// Generate JWT token
	tokenString, expiresAt, err := GenerateJWT(user)
	if err != nil {
		return c.Render(http.StatusInternalServerError, r.JSON(ErrorResponse{
			Error: "Failed to generate token",
		}))
	}

	// Return response
	response := AuthResponse{
		Token:     tokenString,
		User:      user,
		ExpiresAt: expiresAt,
	}

	return c.Render(http.StatusCreated, r.JSON(response))
}

// LoginHandler handles user login
// POST /auth/login
func LoginHandler(c buffalo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.Render(http.StatusBadRequest, r.JSON(ErrorResponse{
			Error: "Invalid request format",
		}))
	}

	// Find user by email
	user := &models.User{}
	err := models.DB.Where("email = ?", strings.ToLower(req.Email)).First(user)
	if err != nil {
		return c.Render(http.StatusUnauthorized, r.JSON(ErrorResponse{
			Error: "Invalid credentials",
		}))
	}

	// Validate password
	if !user.ValidatePassword(req.Password) {
		return c.Render(http.StatusUnauthorized, r.JSON(ErrorResponse{
			Error: "Invalid credentials",
		}))
	}

	// Generate JWT token
	tokenString, expiresAt, err := GenerateJWT(user)
	if err != nil {
		return c.Render(http.StatusInternalServerError, r.JSON(ErrorResponse{
			Error: "Failed to generate token",
		}))
	}

	// Return response
	response := AuthResponse{
		Token:     tokenString,
		User:      user,
		ExpiresAt: expiresAt,
	}

	return c.Render(http.StatusOK, r.JSON(response))
}

// MeHandler returns current user information
// GET /auth/me
func MeHandler(c buffalo.Context) error {
	user := c.Value("currentUser")
	if user == nil {
		return c.Render(http.StatusUnauthorized, r.JSON(ErrorResponse{
			Error: "Unauthorized",
		}))
	}

	return c.Render(http.StatusOK, r.JSON(user))
}

// AuthMiddleware validates JWT tokens and sets current user
func AuthMiddleware(next buffalo.Handler) buffalo.Handler {
	return func(c buffalo.Context) error {
		// Get token from Authorization header
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader == "" {
			return c.Render(http.StatusUnauthorized, r.JSON(ErrorResponse{
				Error: "Authorization header required",
			}))
		}

		// Check Bearer token format
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			return c.Render(http.StatusUnauthorized, r.JSON(ErrorResponse{
				Error: "Invalid authorization header format",
			}))
		}

		// Validate JWT token
		claims, err := ValidateJWT(tokenParts[1])
		if err != nil {
			return c.Render(http.StatusUnauthorized, r.JSON(ErrorResponse{
				Error: "Invalid token",
			}))
		}

		// Get user from database
		userID, err := uuid.FromString(claims.UserID)
		if err != nil {
			return c.Render(http.StatusUnauthorized, r.JSON(ErrorResponse{
				Error: "Invalid user ID in token",
			}))
		}

		user := &models.User{}
		err = models.DB.Find(user, userID)
		if err != nil {
			return c.Render(http.StatusUnauthorized, r.JSON(ErrorResponse{
				Error: "User not found",
			}))
		}

		// Set current user in context
		c.Set("currentUser", user)
		c.Set("currentUserID", userID)

		return next(c)
	}
}

// AdminMiddleware checks if user has admin role
func AdminMiddleware(next buffalo.Handler) buffalo.Handler {
	return func(c buffalo.Context) error {
		user := c.Value("currentUser")
		if user == nil {
			return c.Render(http.StatusUnauthorized, r.JSON(ErrorResponse{
				Error: "Unauthorized",
			}))
		}

		currentUser, ok := user.(*models.User)
		if !ok || !currentUser.IsAdmin() {
			return c.Render(http.StatusForbidden, r.JSON(ErrorResponse{
				Error: "Admin access required",
			}))
		}

		return next(c)
	}
}

// RefreshTokenHandler generates a new token for the current user
// POST /auth/refresh
func RefreshTokenHandler(c buffalo.Context) error {
	user := c.Value("currentUser")
	if user == nil {
		return c.Render(http.StatusUnauthorized, r.JSON(ErrorResponse{
			Error: "Unauthorized",
		}))
	}

	currentUser, ok := user.(*models.User)
	if !ok {
		return c.Render(http.StatusInternalServerError, r.JSON(ErrorResponse{
			Error: "Invalid user data",
		}))
	}

	// Generate new JWT token
	tokenString, expiresAt, err := GenerateJWT(currentUser)
	if err != nil {
		return c.Render(http.StatusInternalServerError, r.JSON(ErrorResponse{
			Error: "Failed to generate token",
		}))
	}

	// Return response
	response := AuthResponse{
		Token:     tokenString,
		User:      currentUser,
		ExpiresAt: expiresAt,
	}

	return c.Render(http.StatusOK, r.JSON(response))
} 