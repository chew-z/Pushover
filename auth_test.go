package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/chew-z/Pushover/internal/oauth"
	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "this-secret-is-longer-than-thirty-two-bytes"

func testMCPConfig() *MCPConfig {
	return &MCPConfig{HTTPPath: "/mcp", AuthEnabled: true, AuthSecretKey: testSecret, HTTPTimeout: time.Second,
		PushoverAppKey: "app", PushoverRecipientKey: "recipient", PushoverDefaultPriority: 0}
}

func testOAuthMCPConfig(t *testing.T) *MCPConfig {
	t.Helper()
	hash, err := oauth.HashPassword("pw")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	cfg := testMCPConfig()
	cfg.OAuthEnabled = true
	cfg.OAuthIssuer = "https://issuer.example"
	cfg.OAuthResource = "https://resource.example/mcp"
	cfg.OAuthLoginUser = "tester"
	cfg.OAuthLoginPasswordHash = hash
	cfg.OAuthAccessTTL = time.Hour
	cfg.OAuthRefreshTTL = 2 * time.Hour
	return cfg
}

func guardedRequest(t *testing.T, guard *authGuard, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/mcp", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	guard.protectMCP(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })).ServeHTTP(recorder, req)
	return recorder
}

func TestProtectMCPChallenges(t *testing.T) {
	t.Run("static", func(t *testing.T) {
		response := guardedRequest(t, newAuthGuard(testMCPConfig(), nil), "")
		if response.Code != http.StatusUnauthorized || !strings.Contains(response.Header().Get("WWW-Authenticate"), `realm="pushover-mcp"`) || strings.Contains(response.Header().Get("WWW-Authenticate"), "resource_metadata") {
			t.Fatalf("response = %d, %q", response.Code, response.Header().Get("WWW-Authenticate"))
		}
	})
	t.Run("oauth", func(t *testing.T) {
		cfg := testOAuthMCPConfig(t)
		oauthCfg, err := cfg.OAuthConfig()
		if err != nil {
			t.Fatal(err)
		}
		response := guardedRequest(t, newAuthGuard(cfg, &oauthCfg), "")
		challenge := response.Header().Get("WWW-Authenticate")
		if response.Code != http.StatusUnauthorized || !strings.Contains(challenge, "resource_metadata") || !strings.Contains(challenge, `scope="pushover"`) {
			t.Fatalf("response = %d, %q", response.Code, challenge)
		}
	})
}

func TestProtectMCPStaticAndLegacyTokens(t *testing.T) {
	token, err := GenerateStaticToken(testSecret, "admin", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	for _, cfg := range []*MCPConfig{testMCPConfig(), testOAuthMCPConfig(t)} {
		var oauthCfg *oauth.Config
		if cfg.OAuthEnabled {
			value, err := cfg.OAuthConfig()
			if err != nil {
				t.Fatal(err)
			}
			oauthCfg = &value
		}
		if response := guardedRequest(t, newAuthGuard(cfg, oauthCfg), token); response.Code != http.StatusNoContent {
			t.Fatalf("static response = %d", response.Code)
		}
	}
	legacy := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"username": "legacy", "user_id": "u1", "role": "admin", "iss": staticIssuer, "aud": staticAudience, "exp": time.Now().Add(time.Hour).Unix()})
	value, err := legacy.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatal(err)
	}
	if response := guardedRequest(t, newAuthGuard(testMCPConfig(), nil), value); response.Code != http.StatusNoContent {
		t.Fatalf("legacy response = %d", response.Code)
	}
}

func TestProtectMCPOAuthScopesAndAlgorithms(t *testing.T) {
	cfg := testOAuthMCPConfig(t)
	oauthCfg, err := cfg.OAuthConfig()
	if err != nil {
		t.Fatal(err)
	}
	guard := newAuthGuard(cfg, &oauthCfg)
	oauthToken, err := SignToken(testSecret, cfg.OAuthIssuer, cfg.OAuthResource, "tester", oauthCfg.Scope, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if response := guardedRequest(t, guard, oauthToken); response.Code != http.StatusNoContent {
		t.Fatalf("oauth response = %d", response.Code)
	}
	if response := guardedRequest(t, newAuthGuard(testMCPConfig(), nil), oauthToken); response.Code != http.StatusUnauthorized {
		t.Fatalf("OAuth off response = %d", response.Code)
	}
	wrongScope, err := SignToken(testSecret, cfg.OAuthIssuer, cfg.OAuthResource, "tester", "other", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if response := guardedRequest(t, guard, wrongScope); response.Code != http.StatusForbidden || !strings.Contains(response.Header().Get("WWW-Authenticate"), "insufficient_scope") {
		t.Fatalf("wrong scope = %d, %q", response.Code, response.Header().Get("WWW-Authenticate"))
	}
	for _, method := range []jwt.SigningMethod{jwt.SigningMethodHS384, jwt.SigningMethodNone} {
		token := jwt.NewWithClaims(method, claims{Username: "admin", RegisteredClaims: jwt.RegisteredClaims{Issuer: staticIssuer, Audience: []string{staticAudience}, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}})
		var value string
		if method == jwt.SigningMethodNone {
			value, err = token.SignedString(jwt.UnsafeAllowNoneSignatureType)
		} else {
			value, err = token.SignedString([]byte(testSecret))
		}
		if err != nil {
			t.Fatal(err)
		}
		if response := guardedRequest(t, newAuthGuard(testMCPConfig(), nil), value); response.Code != http.StatusUnauthorized {
			t.Fatalf("%s response = %d", method.Alg(), response.Code)
		}
	}
	expired, err := SignToken(testSecret, staticIssuer, staticAudience, "admin", "", -time.Hour)
	if err == nil || expired != "" {
		t.Fatal("SignToken accepted an expired TTL")
	}
	old := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{Username: "admin", RegisteredClaims: jwt.RegisteredClaims{Issuer: staticIssuer, Audience: []string{staticAudience}, ExpiresAt: jwt.NewNumericDate(time.Now().Add(-2 * time.Minute))}})
	oldValue, err := old.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatal(err)
	}
	if response := guardedRequest(t, newAuthGuard(testMCPConfig(), nil), oldValue); response.Code != http.StatusUnauthorized {
		t.Fatalf("expired response = %d", response.Code)
	}
}

func TestProtectMCPContextAndOpenMode(t *testing.T) {
	guard := newAuthGuard(testMCPConfig(), nil)
	token, err := GenerateStaticToken(testSecret, "alice", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	seenActor := ""
	guard.protectMCP(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) { seenActor, _ = actorFrom(request.Context()) })).ServeHTTP(httptest.NewRecorder(), req)
	if seenActor != "alice" {
		t.Fatalf("actor = %q", seenActor)
	}
	options := httptest.NewRequestWithContext(context.Background(), http.MethodOptions, "/mcp", nil)
	if response := httptest.NewRecorder(); func() *httptest.ResponseRecorder {
		guard.protectMCP(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })).ServeHTTP(response, options)
		return response
	}().Code != http.StatusNoContent {
		t.Fatal("OPTIONS was blocked")
	}
	if response := guardedRequest(t, nil, ""); response.Code != http.StatusNoContent {
		t.Fatalf("open response = %d", response.Code)
	}
	ctx := copyActor(context.Background(), req.WithContext(context.WithValue(req.Context(), actorKey{}, "alice")))
	if actor, ok := actorFrom(ctx); !ok || actor != "alice" || !isHTTPRequest(ctx) {
		t.Fatalf("copyActor() = actor %q, ok %t, HTTP %t", actor, ok, isHTTPRequest(ctx))
	}
}

func TestExtractTokenFromHeader(t *testing.T) {
	cases := []struct {
		name   string
		header string
		want   string
	}{
		{"valid bearer", "Bearer abc123", "abc123"},
		{"case insensitive", "bearer abc123", "abc123"},
		{"empty header", "", ""},
		{"no bearer prefix", "abc123", ""},
		{"too many parts", "Bearer abc def", ""},
		{"basic auth", "Basic dXNlcjpwYXNz", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			if got := extractTokenFromHeader(req); got != tc.want {
				t.Errorf("extractTokenFromHeader = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestHS256SignerRoundTrip(t *testing.T) {
	signer := HS256Signer{Secret: testSecret, Issuer: staticIssuer, Audience: staticAudience}
	token, err := signer.Sign("alice", "", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/mcp", nil)
	req.Header.Set("Authorization", "bEaReR "+token)
	parsed, err := parseBearerToken(testSecret, req)
	if err != nil || parsed.Username != "alice" {
		t.Fatalf("parseBearerToken() = %+v, %v", parsed, err)
	}
}
