package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/chew-z/Pushover/internal/oauth"
	"github.com/golang-jwt/jwt/v5"
)

// staticIssuer, staticAudience, authRealm, and maxMCPBodyBytes define the static authentication values and request limit.
const (
	staticIssuer    = "pushover-mcp"
	staticAudience  = "pushover-mcp-user"
	authRealm       = "pushover-mcp"
	maxMCPBodyBytes = 4 << 20
)

// claims contains the JWT claims accepted by the MCP HTTP transport.
type claims struct {
	Username string `json:"username"`
	Scope    string `json:"scope,omitempty"`
	jwt.RegisteredClaims
}

// contextKey identifies values stored in request contexts.
type contextKey string

// contextKeyHTTPMethod records whether a tool request originated over HTTP.
const contextKeyHTTPMethod contextKey = "http_method"

// SignToken signs a JWT for the supplied identity, audience, scope, and lifetime.
func SignToken(secret, issuer, audience, username, scope string, ttl time.Duration) (string, error) {
	if ttl <= 0 {
		return "", fmt.Errorf("token TTL must be greater than 0")
	}
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		Username: username,
		Scope:    scope,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: issuer, Audience: jwt.ClaimStrings{audience}, IssuedAt: jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	})
	value, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return value, nil
}

// GenerateStaticToken creates a legacy static bearer token with the project's fixed issuer and audience.
func GenerateStaticToken(secret, username string, ttl time.Duration) (string, error) {
	return SignToken(secret, staticIssuer, staticAudience, username, "", ttl)
}

// HS256Signer signs OAuth tokens with the configured HMAC secret and claims.
type HS256Signer struct{ Secret, Issuer, Audience string }

// _ verifies at compile time that HS256Signer implements oauth.Signer.
var _ oauth.Signer = HS256Signer{}

// Sign creates an OAuth token using the configured HS256 signer.
func (s HS256Signer) Sign(username, scope string, ttl time.Duration) (string, error) {
	return SignToken(s.Secret, s.Issuer, s.Audience, username, scope, ttl)
}

// extractTokenFromHeader returns the Bearer token from an Authorization header.
func extractTokenFromHeader(r *http.Request) string {
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}

// parseBearerToken validates an HTTP bearer token and returns its claims.
func parseBearerToken(secret string, r *http.Request) (*claims, error) {
	tokenString := extractTokenFromHeader(r)
	if tokenString == "" {
		return nil, fmt.Errorf("malformed bearer token")
	}
	parsedClaims := &claims{}
	_, err := jwt.ParseWithClaims(tokenString, parsedClaims, func(*jwt.Token) (any, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithLeeway(60*time.Second), jwt.WithExpirationRequired())
	if err != nil {
		return nil, fmt.Errorf("parse bearer token: %w", err)
	}
	if strings.TrimSpace(parsedClaims.Username) == "" {
		return nil, fmt.Errorf("token username is empty")
	}
	return parsedClaims, nil
}

// tokenPair describes an accepted issuer, audience, and optional scope.
type tokenPair struct{ issuer, audience, scope string }

// authGuard holds the secret and token profiles used to protect the MCP endpoint.
type authGuard struct {
	secret string
	pairs  []tokenPair
	oauth  *oauth.Config
}

// newAuthGuard builds the static pair (pushover-mcp/pushover-mcp-user, no scope), preserving
// legacy tokens with user_id/role, and adds the OAuth issuer/resource pair, which requires its scope.
func newAuthGuard(cfg *MCPConfig, oauthCfg *oauth.Config) *authGuard {
	guard := &authGuard{secret: cfg.AuthSecretKey, pairs: []tokenPair{{issuer: staticIssuer, audience: staticAudience}}, oauth: oauthCfg}
	if oauthCfg != nil {
		guard.pairs = append(guard.pairs, tokenPair{issuer: cfg.OAuthIssuer, audience: cfg.OAuthResource, scope: oauthCfg.Scope})
	}
	return guard
}

// protectMCP enforces auth at HTTP layer so clients receive a real 401 and WWW-Authenticate
// (claude.ai starts OAuth only on that); it caps bodies because mcp-go uses io.ReadAll, nil means open mode, and OPTIONS passes CORS preflight.
func (g *authGuard) protectMCP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxMCPBodyBytes)
		if g == nil || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		parsedClaims, err := parseBearerToken(g.secret, r)
		if err != nil {
			g.writeChallenge(w, "invalid_token")
			writeAuthError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		for _, pair := range g.pairs {
			if parsedClaims.Issuer != pair.issuer || !hasAudience(parsedClaims.Audience, pair.audience) {
				continue
			}
			if pair.scope != "" && parsedClaims.Scope != pair.scope {
				g.writeChallenge(w, "insufficient_scope")
				writeAuthError(w, http.StatusForbidden, "insufficient_scope")
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), actorKey{}, parsedClaims.Username)))
			return
		}
		g.writeChallenge(w, "invalid_token")
		writeAuthError(w, http.StatusUnauthorized, "unauthorized")
	})
}

// hasAudience reports whether the expected audience is present in a JWT claim set.
func hasAudience(audiences jwt.ClaimStrings, audience string) bool {
	for _, candidate := range audiences {
		if candidate == audience {
			return true
		}
	}
	return false
}

// writeChallenge writes a Bearer challenge, adding resource_metadata only when OAuth is enabled for RFC 9728 discovery.
func (g *authGuard) writeChallenge(w http.ResponseWriter, code string) {
	challenge := `Bearer realm="` + authRealm + `", error="` + code + `"`
	if g.oauth != nil {
		challenge += `, resource_metadata="` + oauth.IssuerOrigin(g.oauth.Issuer) + oauth.ProtectedResourceMetadataPath(g.oauth.Resource) + `", scope="` + g.oauth.Scope + `"`
	}
	w.Header().Set("WWW-Authenticate", challenge)
}

// writeAuthError writes a JSON authentication error with the supplied HTTP status.
func writeAuthError(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": code})
}

// actorKey identifies the authenticated actor in a request context.
type actorKey struct{}

// actorFrom returns the authenticated actor stored in a context.
func actorFrom(ctx context.Context) (string, bool) {
	actor, ok := ctx.Value(actorKey{}).(string)
	return actor, ok
}

// copyActor marks HTTP requests and copies protectMCP's actor so tools can distinguish HTTP
// from stdio via isHTTPRequest; it is installed in every mode because mcp-go passes the original request to WithHTTPContextFunc.
func copyActor(ctx context.Context, r *http.Request) context.Context {
	ctx = context.WithValue(ctx, contextKeyHTTPMethod, r.Method)
	if actor, ok := actorFrom(r.Context()); ok {
		ctx = context.WithValue(ctx, actorKey{}, actor)
	}
	return ctx
}

// isHTTPRequest reports whether a tool context came from an HTTP request rather than stdio.
func isHTTPRequest(ctx context.Context) bool {
	httpMethod, ok := ctx.Value(contextKeyHTTPMethod).(string)
	return ok && httpMethod != ""
}
