package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// AuthMiddleware handles JWT-based authentication for HTTP requests
type AuthMiddleware struct {
	secretKey []byte
	enabled   bool
}

// Claims represents the JWT claims structure
type Claims struct {
	jwt.RegisteredClaims
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

// Context keys for storing authentication information
const (
	contextKeyUserID        contextKey = "user_id"
	contextKeyUsername      contextKey = "username"
	contextKeyRole          contextKey = "role"
	contextKeyAuthError     contextKey = "auth_error"
	contextKeyHTTPMethod    contextKey = "http_method"
	contextKeyAuthenticated contextKey = "authenticated"
)

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(secretKey string, enabled bool) *AuthMiddleware {
	return &AuthMiddleware{
		secretKey: []byte(secretKey),
		enabled:   enabled,
	}
}

// HTTPContextFunc returns a function that adds authentication context to HTTP requests
func (am *AuthMiddleware) HTTPContextFunc() func(context.Context, *http.Request) context.Context {
	return func(ctx context.Context, r *http.Request) context.Context {
		// Add HTTP method to context for tool handlers to distinguish HTTP vs STDIO
		ctx = context.WithValue(ctx, contextKeyHTTPMethod, r.Method)

		// If authentication is disabled, mark as authenticated and return
		if !am.enabled {
			ctx = context.WithValue(ctx, contextKeyAuthenticated, true)
			return ctx
		}

		// Extract and validate JWT token
		token := extractTokenFromHeader(r)
		if token == "" {
			ctx = context.WithValue(ctx, contextKeyAuthError, "missing or invalid authorization header")
			ctx = context.WithValue(ctx, contextKeyAuthenticated, false)
			return ctx
		}

		// Validate JWT token
		claims, err := am.validateJWT(token)
		if err != nil {
			ctx = context.WithValue(ctx, contextKeyAuthError, fmt.Sprintf("invalid token: %v", err))
			ctx = context.WithValue(ctx, contextKeyAuthenticated, false)
			return ctx
		}

		// Add user information to context
		ctx = context.WithValue(ctx, contextKeyUserID, claims.UserID)
		ctx = context.WithValue(ctx, contextKeyUsername, claims.Username)
		ctx = context.WithValue(ctx, contextKeyRole, claims.Role)
		ctx = context.WithValue(ctx, contextKeyAuthenticated, true)

		return ctx
	}
}

// extractTokenFromHeader extracts the Bearer token from the Authorization header
func extractTokenFromHeader(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.Fields(authHeader)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}

// validateJWT validates a JWT token and returns the claims
func (am *AuthMiddleware) validateJWT(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return am.secretKey, nil
	},
		jwt.WithIssuer("pushover-mcp"),
		jwt.WithAudience("pushover-mcp-user"),
		jwt.WithLeeway(60*time.Second),
	)

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// GenerateJWT generates a JWT token for the given user information
func (am *AuthMiddleware) GenerateJWT(userID, username, role string, expirationHours int) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "pushover-mcp",
			Audience:  jwt.ClaimStrings{"pushover-mcp-user"},
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(expirationHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
		UserID:   userID,
		Username: username,
		Role:     role,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(am.secretKey)
}

// Helper functions for extracting authentication information from context

// getUserID retrieves the user ID from the context
//
//nolint:unused // Part of complete auth API, may be needed for future features
func getUserID(ctx context.Context) string {
	if userID, ok := ctx.Value(contextKeyUserID).(string); ok {
		return userID
	}
	return ""
}

// getUsername retrieves the username from the context
func getUsername(ctx context.Context) string {
	if username, ok := ctx.Value(contextKeyUsername).(string); ok {
		return username
	}
	return ""
}

// getRole retrieves the user role from the context
//
//nolint:unused // Part of complete auth API, may be needed for future features
func getRole(ctx context.Context) string {
	if role, ok := ctx.Value(contextKeyRole).(string); ok {
		return role
	}
	return ""
}

// getAuthError retrieves any authentication error from the context
func getAuthError(ctx context.Context) string {
	if authError, ok := ctx.Value(contextKeyAuthError).(string); ok {
		return authError
	}
	return ""
}

// isAuthenticated checks if the request is authenticated
func isAuthenticated(ctx context.Context) bool {
	if authenticated, ok := ctx.Value(contextKeyAuthenticated).(bool); ok {
		return authenticated
	}
	return false
}

// isHTTPRequest checks if the request came via HTTP (vs STDIO)
func isHTTPRequest(ctx context.Context) bool {
	if httpMethod, ok := ctx.Value(contextKeyHTTPMethod).(string); ok {
		return httpMethod != ""
	}
	return false
}

// GenerateTokenCommand is a utility function to generate a token from command line
// This can be used for testing or initial token generation
func GenerateTokenCommand(secretKey, userID, username, role string, expirationHours int) {
	auth := NewAuthMiddleware(secretKey, true)
	token, err := auth.GenerateJWT(userID, username, role, expirationHours)
	if err != nil {
		fmt.Printf("Error generating token: %v\n", err)
		return
	}

	fmt.Printf("Generated JWT token:\n%s\n", token)
	fmt.Printf("\nTo use this token, include it in the Authorization header:\n")
	fmt.Printf("Authorization: Bearer %s\n", token)
	fmt.Printf("\nToken details:\n")
	fmt.Printf("User ID: %s\n", userID)
	fmt.Printf("Username: %s\n", username)
	fmt.Printf("Role: %s\n", role)
	fmt.Printf("Expires in: %d hours\n", expirationHours)
}

// TokenInfo displays information about a JWT token
func (am *AuthMiddleware) TokenInfo(token string) {
	claims, err := am.validateJWT(token)
	if err != nil {
		fmt.Printf("Invalid token: %v\n", err)
		return
	}

	fmt.Printf("Token Information:\n")
	fmt.Printf("User ID: %s\n", claims.UserID)
	fmt.Printf("Username: %s\n", claims.Username)
	fmt.Printf("Role: %s\n", claims.Role)
	fmt.Printf("Issued At: %s\n", claims.IssuedAt.Time.Format(time.RFC3339))
	fmt.Printf("Expires At: %s\n", claims.ExpiresAt.Time.Format(time.RFC3339))

	if time.Now().After(claims.ExpiresAt.Time) {
		fmt.Printf("Status: EXPIRED\n")
	} else {
		fmt.Printf("Status: VALID\n")
		remaining := time.Until(claims.ExpiresAt.Time)
		fmt.Printf("Time Remaining: %s\n", remaining.String())
	}
}
