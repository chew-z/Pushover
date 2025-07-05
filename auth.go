package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// AuthMiddleware handles JWT-based authentication for HTTP requests
type AuthMiddleware struct {
	secretKey []byte
	enabled   bool
}

// Claims represents the JWT claims structure
type Claims struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
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

		// Check token expiration
		if time.Now().Unix() > claims.ExpiresAt {
			ctx = context.WithValue(ctx, contextKeyAuthError, "token expired")
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

	// Check for Bearer token format
	const bearerPrefix = "Bearer "
	if !strings.HasPrefix(authHeader, bearerPrefix) {
		return ""
	}

	return strings.TrimPrefix(authHeader, bearerPrefix)
}

// validateJWT validates a JWT token and returns the claims
func (am *AuthMiddleware) validateJWT(token string) (*Claims, error) {
	// Split token into parts
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	header, payload, signature := parts[0], parts[1], parts[2]

	// Validate signature
	expectedSignature := am.createSignature(header + "." + payload)
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return nil, fmt.Errorf("invalid signature")
	}

	// Decode and parse payload
	payloadBytes, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to decode payload: %w", err)
	}

	var claims Claims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %w", err)
	}

	return &claims, nil
}

// createSignature creates an HMAC signature for the given data
func (am *AuthMiddleware) createSignature(data string) string {
	h := hmac.New(sha256.New, am.secretKey)
	h.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

// GenerateJWT generates a JWT token for the given user information
func (am *AuthMiddleware) GenerateJWT(userID, username, role string, expirationHours int) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:    userID,
		Username:  username,
		Role:      role,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(time.Duration(expirationHours) * time.Hour).Unix(),
	}

	// Create header
	header := map[string]interface{}{
		"alg": "HS256",
		"typ": "JWT",
	}

	headerBytes, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("failed to marshal header: %w", err)
	}

	// Create payload
	payloadBytes, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("failed to marshal claims: %w", err)
	}

	// Encode header and payload
	encodedHeader := base64.RawURLEncoding.EncodeToString(headerBytes)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadBytes)

	// Create signature
	signature := am.createSignature(encodedHeader + "." + encodedPayload)

	// Combine all parts
	token := encodedHeader + "." + encodedPayload + "." + signature

	return token, nil
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
	fmt.Printf("Issued At: %s\n", time.Unix(claims.IssuedAt, 0).Format(time.RFC3339))
	fmt.Printf("Expires At: %s\n", time.Unix(claims.ExpiresAt, 0).Format(time.RFC3339))

	if time.Now().Unix() > claims.ExpiresAt {
		fmt.Printf("Status: EXPIRED\n")
	} else {
		fmt.Printf("Status: VALID\n")
		remaining := time.Until(time.Unix(claims.ExpiresAt, 0))
		fmt.Printf("Time Remaining: %s\n", remaining.String())
	}
}
