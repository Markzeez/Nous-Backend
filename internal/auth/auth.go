package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const UserContextKey contextKey = "user"

// Claims represents the JWT claims for our application
// These are our custom claims, not Supabase claims
type Claims struct {
	UserID_ string `json:"user_id"`
	Email   string `json:"email"`
	Role    string `json:"role"`
	jwt.RegisteredClaims
}

// UserID returns the user ID from claims
func (c *Claims) UserID() string {
	return c.UserID_
}

// AppRole returns the user's app-level role
func (c *Claims) AppRole() string {
	return c.Role
}

// GenerateToken creates a new JWT token for a user
func GenerateToken(userID, email, role, secret string, expiryHours int) (string, error) {
	claims := &Claims{
		UserID_: userID,
		Email:   email,
		Role:    role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiryHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   userID,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ValidateToken verifies a JWT token using the provided secret
func ValidateToken(tokenStr, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrTokenSignatureInvalid
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}
	if claims.UserID_ == "" {
		return nil, errors.New("token missing user_id")
	}
	return claims, nil
}

// Middleware verifies the JWT sent by the frontend and attaches
// the parsed claims to the request context.
func Middleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				ErrorJSON(w, http.StatusUnauthorized, "Unauthorized")
				return
			}
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := ValidateToken(tokenStr, secret)
			if err != nil {
				ErrorJSON(w, http.StatusUnauthorized, "Invalid token")
				return
			}
			ctx := context.WithValue(r.Context(), UserContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUser extracts authenticated claims from the request context.
func GetUser(r *http.Request) *Claims {
	claims, _ := r.Context().Value(UserContextKey).(*Claims)
	return claims
}

// RequireRole is optional middleware for endpoints that need a specific
// app-level role (e.g., "ADMIN", "DOCTOR").
func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetUser(r)
			if claims == nil || claims.AppRole() != role {
				ErrorJSON(w, http.StatusForbidden, "Forbidden")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyRole allows access if user has any of the specified roles
func RequireAnyRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetUser(r)
			if claims == nil {
				ErrorJSON(w, http.StatusUnauthorized, "Unauthorized")
				return
			}
			for _, role := range roles {
				if claims.AppRole() == role {
					next.ServeHTTP(w, r)
					return
				}
			}
			ErrorJSON(w, http.StatusForbidden, "Forbidden")
		})
	}
}

// JSON helpers

func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func ErrorJSON(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, map[string]string{"error": msg})
}
