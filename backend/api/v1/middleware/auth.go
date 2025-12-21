package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/GHutch55/fragments/backend/api/v1/database"
	"github.com/GHutch55/fragments/backend/api/v1/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type contextKey string

const UserContextKey contextKey = "user"

// AuthMiddleware handles JWT authentication
type AuthMiddleware struct {
	DB        *pgxpool.Pool
	JWTSecret string
}

// Claims represents JWT token claims
type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// ErrorResponse represents a JSON error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// NewAuthMiddleware creates a new AuthMiddleware instance
func NewAuthMiddleware(pool *pgxpool.Pool, jwtSecret string) *AuthMiddleware {
	return &AuthMiddleware{
		DB:        pool,
		JWTSecret: jwtSecret,
	}
}

// GenerateToken creates a new JWT token for the given user
func (am *AuthMiddleware) GenerateToken(user *models.User) (string, error) {
	if user == nil {
		return "", errors.New("user cannot be nil")
	}

	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "fragments-api",
			Subject:   fmt.Sprintf("%d", user.ID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(am.JWTSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signedToken, nil
}

// ValidateToken validates and parses a JWT token string
func (am *AuthMiddleware) ValidateToken(tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, errors.New("token string cannot be empty")
	}

	claims := &Claims{}

	// Add leeway for clock skew (5 minutes)
	parser := jwt.NewParser(jwt.WithLeeway(5 * time.Minute))

	token, err := parser.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(am.JWTSecret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("token is not valid")
	}

	if claims.UserID <= 0 {
		return nil, errors.New("invalid user ID in token")
	}

	return claims, nil
}

// RequireAuth middleware that validates JWT tokens and loads user context
func (am *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Extract Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			am.sendError(w, "Missing authorization header", http.StatusUnauthorized)
			return
		}

		// Parse Bearer token
		bearerToken := strings.Fields(authHeader)
		if len(bearerToken) != 2 || !strings.EqualFold(bearerToken[0], "Bearer") {
			am.sendError(w, "Invalid authorization header format. Expected 'Bearer <token>'", http.StatusUnauthorized)
			return
		}

		// Validate token
		claims, err := am.ValidateToken(bearerToken[1])
		if err != nil {
			am.sendError(w, fmt.Sprintf("Invalid token: %s", err.Error()), http.StatusUnauthorized)
			return
		}

		// Load user from database with context
		var user models.User
		err = database.GetUser(r.Context(), am.DB, claims.UserID, &user)
		if err != nil {
			// Check for specific database errors
			if errors.Is(err, database.ErrNoUserError) || strings.Contains(err.Error(), "not found") {
				am.sendError(w, "User not found", http.StatusUnauthorized)
				return
			}
			if errors.Is(err, database.ErrDatabaseError) {
				am.sendError(w, "Unable to verify user", http.StatusInternalServerError)
				return
			}
			am.sendError(w, "Authentication failed", http.StatusUnauthorized)
			return
		}

		// Verify token claims match database user
		if user.Username != claims.Username {
			am.sendError(w, "Token claims do not match user data", http.StatusUnauthorized)
			return
		}

		// Add user to request context
		ctx := context.WithValue(r.Context(), UserContextKey, &user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalAuth middleware that loads user context if a valid token is provided
// but doesn't require authentication
func (am *AuthMiddleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		// If no auth header, continue without user context
		if authHeader == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Try to parse and validate token
		bearerToken := strings.Fields(authHeader)
		if len(bearerToken) == 2 && strings.EqualFold(bearerToken[0], "Bearer") {
			if claims, err := am.ValidateToken(bearerToken[1]); err == nil {
				var user models.User
				if err := database.GetUser(r.Context(), am.DB, claims.UserID, &user); err == nil {
					if user.Username == claims.Username {
						ctx := context.WithValue(r.Context(), UserContextKey, &user)
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					}
				}
			}
		}

		// If token parsing/validation fails, continue without user context
		next.ServeHTTP(w, r)
	})
}

// GetUserFromContext retrieves the authenticated user from the request context
func GetUserFromContext(ctx context.Context) (*models.User, bool) {
	user, ok := ctx.Value(UserContextKey).(*models.User)
	return user, ok
}

// GetUserIDFromContext is a helper to quickly get just the user ID
func GetUserIDFromContext(ctx context.Context) (int64, bool) {
	if user, ok := GetUserFromContext(ctx); ok {
		return user.ID, true
	}
	return 0, false
}

// sendError sends a JSON error response
func (am *AuthMiddleware) sendError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	response := ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		// In rare cases, fall back to plain text
		http.Error(w, fmt.Sprintf(`{"error": "Internal Server Error", "message": "%v"}`, err), http.StatusInternalServerError)
	}
}
