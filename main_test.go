package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gregdel/pushover"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// MockPushoverClient implements PushoverClient for testing
type MockPushoverClient struct {
	lastMessage   *pushover.Message
	lastRecipient *pushover.Recipient
	shouldError   bool
}

func (m *MockPushoverClient) SendMessage(message *pushover.Message, recipient *pushover.Recipient) (*pushover.Response, error) {
	m.lastMessage = message
	m.lastRecipient = recipient

	if m.shouldError {
		return nil, errors.New("mock error")
	}

	return &pushover.Response{Status: 1}, nil
}

func TestCreateMessage_UseDefaultTitle(t *testing.T) {
	config := Config{
		DefaultTitle: "Default Title",
		Priority:     int(pushover.PriorityNormal),
		Sound:        pushover.SoundVibrate,
		ExpireTime:   300,
	}

	cliArgs := &CLIArgs{
		Message: "Test message",
		Title:   "", // Empty title should use default
	}

	msg := CreateMessage("Test message", "", config, cliArgs)

	if msg.Title != "Default Title" {
		t.Errorf("Expected title 'Default Title', got '%s'", msg.Title)
	}
}

func TestCreateMessage_CLIOverridesConfig(t *testing.T) {
	config := Config{
		Priority:   int(pushover.PriorityLow),
		Sound:      pushover.SoundVibrate,
		ExpireTime: 180,
	}

	cliArgs := &CLIArgs{
		Message:    "Test message",
		Priority:   int(pushover.PriorityHigh),
		Sound:      pushover.SoundSiren,
		ExpireTime: 600,
	}

	msg := CreateMessage("Test message", "Test title", config, cliArgs)

	if msg.Priority != int(pushover.PriorityHigh) {
		t.Errorf("Expected priority %d, got %d", pushover.PriorityHigh, msg.Priority)
	}

	if msg.Sound != pushover.SoundSiren {
		t.Errorf("Expected sound %s, got %s", pushover.SoundSiren, msg.Sound)
	}

	expectedExpire := 600
	if int(msg.Expire.Seconds()) != expectedExpire {
		t.Errorf("Expected expire %d seconds, got %d", expectedExpire, int(msg.Expire.Seconds()))
	}
}

func TestRun_WithMockClient(t *testing.T) {
	mockClient := &MockPushoverClient{}

	// Simulate command line: pushover -m "test message" -t "test title"
	args := []string{"pushover", "-m", "test message", "-t", "test title"}

	err := Run(args, mockClient)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if mockClient.lastMessage == nil {
		t.Error("Expected message to be sent")
		return
	}

	if mockClient.lastMessage.Message != "test message" {
		t.Errorf("Expected message 'test message', got '%s'", mockClient.lastMessage.Message)
	}

	if mockClient.lastMessage.Title != "test title" {
		t.Errorf("Expected title 'test title', got '%s'", mockClient.lastMessage.Title)
	}
}

func TestParseArgs_ShowHelp(t *testing.T) {
	args := []string{"pushover", "-h"}

	cliArgs, err := ParseArgs(args)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !cliArgs.ShowHelp {
		t.Error("Expected ShowHelp to be true")
	}
}

func TestParseArgs_ShowVersion(t *testing.T) {
	args := []string{"pushover", "-version"}

	cliArgs, err := ParseArgs(args)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !cliArgs.ShowVersion {
		t.Error("Expected ShowVersion to be true")
	}
}

func TestParseArgs_PositionalArgs(t *testing.T) {
	args := []string{"pushover", "test message", "test title"}

	cliArgs, err := ParseArgs(args)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if cliArgs.Message != "test message" {
		t.Errorf("Expected message 'test message', got '%s'", cliArgs.Message)
	}

	if cliArgs.Title != "test title" {
		t.Errorf("Expected title 'test title', got '%s'", cliArgs.Title)
	}
}

func TestParseArgs_MissingMessage(t *testing.T) {
	args := []string{"pushover"}

	_, err := ParseArgs(args)
	if err == nil {
		t.Error("Expected error for missing message")
	}
}

func TestNewPushoverClient_MissingKeys(t *testing.T) {
	config := Config{
		AppKey:       "",
		RecipientKey: "test",
	}

	_, _, err := NewPushoverClient(config)
	if err == nil {
		t.Error("Expected error for missing APP_KEY")
	}

	config.AppKey = "test"
	config.RecipientKey = ""

	_, _, err = NewPushoverClient(config)
	if err == nil {
		t.Error("Expected error for missing RECIPIENT_KEY")
	}
}

// --- New Tests for MCP Server Functionality ---

// newTestMCPConfig provides a valid MCPConfig for testing purposes.
func newTestMCPConfig() *MCPConfig {
	return &MCPConfig{
		PushoverAppKey:          "test_app_key",
		PushoverRecipientKey:    "test_recipient_key",
		PushoverDefaultPriority: int(pushover.PriorityNormal),
		PushoverDefaultExpire:   180,
		PushoverDefaultRetry:    60,
		HTTPAddress:             ":8080",
		HTTPPath:                "/mcp",
		AuthEnabled:             true,
		AuthSecretKey:           "a-very-secret-key-for-testing-purpose",
	}
}

func TestMCPConfig_Validation(t *testing.T) {
	testCases := []struct {
		name        string
		modifier    func(c *MCPConfig)
		expectError bool
		errContains string
	}{
		{"valid config", func(c *MCPConfig) {}, false, ""},
		{"missing app key", func(c *MCPConfig) { c.PushoverAppKey = "" }, true, "APP_KEY environment variable is required"},
		{"missing recipient key", func(c *MCPConfig) { c.PushoverRecipientKey = "" }, true, "RECIPIENT_KEY environment variable is required"},
		{"priority too low", func(c *MCPConfig) { c.PushoverDefaultPriority = -3 }, true, "PUSHOVER_PRIORITY must be between -2 and 2"},
		{"priority too high", func(c *MCPConfig) { c.PushoverDefaultPriority = 3 }, true, "PUSHOVER_PRIORITY must be between -2 and 2"},
		{"auth enabled but no secret", func(c *MCPConfig) { c.AuthSecretKey = "" }, true, "PUSHOVER_AUTH_SECRET_KEY is required"},
		{"emergency priority requires expire", func(c *MCPConfig) {
			c.PushoverDefaultPriority = int(pushover.PriorityEmergency)
			c.PushoverDefaultExpire = 0
		}, true, "PUSHOVER_EXPIRE must be > 0"},
		{"emergency priority requires retry", func(c *MCPConfig) {
			c.PushoverDefaultPriority = int(pushover.PriorityEmergency)
			c.PushoverDefaultRetry = 0
		}, true, "PUSHOVER_RETRY must be >= 30"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := newTestMCPConfig()
			tc.modifier(config)
			err := config.Validate()
			if tc.expectError {
				if err == nil {
					t.Fatalf("Expected an error, but got none")
				}
				if !strings.Contains(err.Error(), tc.errContains) {
					t.Errorf("Expected error to contain '%s', but got: %v", tc.errContains, err)
				}
			} else if err != nil {
				t.Fatalf("Expected no error, but got: %v", err)
			}
		})
	}
}

func TestAuthMiddleware_GenerateAndValidateJWT(t *testing.T) {
	secretKey := "test-secret"
	am := NewAuthMiddleware(secretKey, true)

	userID, username, role := "user123", "testuser", "user"
	token, err := am.GenerateJWT(userID, username, role, 1)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}

	claims, err := am.validateJWT(token)
	if err != nil {
		t.Fatalf("Failed to validate JWT: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("Expected UserID %s, got %s", userID, claims.UserID)
	}
	if claims.Username != username {
		t.Errorf("Expected Username %s, got %s", username, claims.Username)
	}
	if claims.Role != role {
		t.Errorf("Expected Role %s, got %s", role, claims.Role)
	}
	if claims.ExpiresAt.Unix() <= time.Now().Unix() {
		t.Error("Token expiration is not in the future")
	}
}

func TestAuthMiddleware_ValidateJWT_Errors(t *testing.T) {
	am := NewAuthMiddleware("secret1", true)
	token, _ := am.GenerateJWT("user", "test", "user", 1)

	amExpired := NewAuthMiddleware("secret-for-expired", true)
	expiredToken, _ := amExpired.GenerateJWT("user", "test", "user", -1) // Expired 1 hour ago

	testCases := []struct {
		name        string
		token       string
		middleware  *AuthMiddleware
		errContains error
	}{
		{"invalid signature", token, NewAuthMiddleware("secret2", true), jwt.ErrSignatureInvalid},
		{"expired token", expiredToken, amExpired, jwt.ErrTokenExpired},
		{"invalid format", "a.b.c", am, jwt.ErrTokenMalformed},
		{"malformed payload", "a.badpayload.c", am, jwt.ErrTokenMalformed},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.middleware.validateJWT(tc.token)
			if err == nil {
				t.Fatalf("Expected error, but got none")
			} else if !errors.Is(err, tc.errContains) {
				t.Errorf("Expected error %v, got %v", tc.errContains, err)
			}
		})
	}
}

func TestHandleSendNotification(t *testing.T) {
	testCases := []struct {
		name        string
		message     string
		title       string
		priority    any
		device      string
		sound       string
		expire      any
		wantErr     bool
		errContains string
	}{
		{
			name:        "successful send attempt",
			message:     "test message",
			title:       "test title",
			wantErr:     true, // Expects error because pushover client will fail with dummy keys
			errContains: "Failed to send notification",
		},
		{
			name:        "missing message",
			message:     "", // Empty message should trigger required parameter error
			title:       "test",
			wantErr:     true,
			errContains: "Message parameter is required",
		},
		{
			name:        "emergency priority with expire",
			message:     "emergency",
			priority:    int(pushover.PriorityEmergency),
			expire:      60,
			wantErr:     true, // Still fails on send
			errContains: "Failed to send notification",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Build arguments map
			args := make(map[string]any)
			if tc.message != "" {
				args["message"] = tc.message
			}
			if tc.title != "" {
				args["title"] = tc.title
			}
			if tc.priority != nil {
				args["priority"] = tc.priority
			}
			if tc.device != "" {
				args["device"] = tc.device
			}
			if tc.sound != "" {
				args["sound"] = tc.sound
			}
			if tc.expire != nil {
				args["expire"] = tc.expire
			}

			// Create MCP request with proper structure
			req := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: args,
				},
			}
			config := newTestMCPConfig()
			result, err := handleSendNotification(context.Background(), req, config)

			if err != nil {
				t.Fatalf("Handler returned an unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("Result is nil")
			}

			resultText := fmt.Sprintf("%v", result)

			if tc.wantErr {
				if !strings.Contains(resultText, tc.errContains) {
					t.Errorf("Expected error to contain '%s', but got: %s", tc.errContains, resultText)
				}
			} else {
				if strings.Contains(resultText, "error") || strings.Contains(resultText, "Error") {
					t.Fatalf("Expected success result, but got error: %s", resultText)
				}
			}
		})
	}
}

func TestHttpServerEndpoints(t *testing.T) {
	config := newTestMCPConfig()
	hsm := NewHTTPServerManager(config)

	// We don't start the server, just test handlers directly
	// Create an MCP server instance to handle capability requests
	mcpServer := setupMCPServer(config)
	hsm.mcpServer = server.NewStreamableHTTPServer(mcpServer)

	t.Run("Health Endpoint", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rr := httptest.NewRecorder()
		hsm.handleHealth(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var resp map[string]interface{}
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("Could not decode response: %v", err)
		}

		if resp["status"] != "healthy" {
			t.Errorf("Expected status 'healthy', got '%s'", resp["status"])
		}
	})

	t.Run("Capabilities Endpoint", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/capabilities", nil)
		rr := httptest.NewRecorder()
		hsm.handleCapabilities(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var resp map[string]interface{}
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("Could not decode response: %v", err)
		}

		if resp["name"] != "pushover-mcp-server" {
			t.Errorf("Expected name 'pushover-mcp-server', got '%s'", resp["name"])
		}
		if !resp["authentication"].(map[string]interface{})["enabled"].(bool) {
			t.Error("Expected authentication to be enabled in capabilities")
		}
	})

	t.Run("Generate Token Endpoint", func(t *testing.T) {
		body := `{"user_id": "test_user", "username": "tester", "role": "admin", "expires_in": 1}`
		req := httptest.NewRequest(http.MethodPost, "/generate-token", strings.NewReader(body))
		rr := httptest.NewRecorder()

		hsm.handleGenerateToken(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var resp map[string]interface{}
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("Could not decode response: %v", err)
		}

		if token, ok := resp["token"].(string); !ok || token == "" {
			t.Error("Expected a non-empty token in response")
		}

		if resp["username"] != "tester" {
			t.Errorf("Expected username 'tester', got '%s'", resp["username"])
		}
	})

	t.Run("Generate Token Endpoint - Bad Method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/generate-token", nil)
		rr := httptest.NewRecorder()
		hsm.handleGenerateToken(rr, req)

		if status := rr.Code; status != http.StatusMethodNotAllowed {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusMethodNotAllowed)
		}
	})
}

func TestMCPSchemaValidation(t *testing.T) {
	config := newTestMCPConfig()
	mcpServer := setupMCPServer(config)

	c, err := client.NewInProcessClient(mcpServer)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	ctx := context.Background()
	if err := c.Start(ctx); err != nil {
		t.Fatalf("failed to start client: %v", err)
	}
	defer func() { _ = c.Close() }()
	if _, err := c.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo:      mcp.Implementation{Name: "test-client", Version: "0.0.1"},
		},
	}); err != nil {
		t.Fatalf("failed to initialize client: %v", err)
	}

	cases := []struct {
		name string
		args map[string]any
	}{
		{
			name: "priority as string rejected",
			args: map[string]any{"message": "test", "priority": "5"},
		},
		{
			name: "priority out of range",
			args: map[string]any{"message": "test", "priority": 5},
		},
		{
			name: "message too long",
			args: map[string]any{"message": strings.Repeat("a", 1025)},
		},
		{
			name: "expire out of range",
			args: map[string]any{"message": "test", "expire": 99999},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := c.CallTool(ctx, mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name:      "send_notification",
					Arguments: tc.args,
				},
			})
			if err != nil {
				t.Fatalf("unexpected Go error: %v", err)
			}
			if !result.IsError {
				t.Errorf("expected schema validation error (IsError=true), got success")
			}
		})
	}
}

func TestNewMCPConfigFromEnv(t *testing.T) {
	t.Setenv("PUSHOVER_HTTP_ADDRESS", ":9090")
	t.Setenv("PUSHOVER_AUTH_ENABLED", "true")
	t.Setenv("PUSHOVER_AUTH_SECRET_KEY", "env-secret")
	t.Setenv("APP_KEY", "env-app-key")
	t.Setenv("RECIPIENT_KEY", "env-recipient-key")

	// Test that flag=false but env=true results in true
	config, err := NewMCPConfig(false)
	if err != nil {
		t.Fatalf("NewMCPConfig failed: %v", err)
	}

	if config.HTTPAddress != ":9090" {
		t.Errorf("Expected HTTPAddress :9090, got %s", config.HTTPAddress)
	}
	if !config.AuthEnabled {
		t.Error("Expected AuthEnabled to be true from environment")
	}
	if config.AuthSecretKey != "env-secret" {
		t.Errorf("Expected AuthSecretKey from env, but it was not set")
	}

	// Test that flag=true overrides env=false (or unset)
	os.Unsetenv("PUSHOVER_AUTH_ENABLED")
	config, err = NewMCPConfig(true)
	if err != nil {
		t.Fatalf("NewMCPConfig failed: %v", err)
	}
	if !config.AuthEnabled {
		t.Error("Expected AuthEnabled to be true from flag")
	}
}

// --- Auth Context Helper Tests ---

func TestContextHelpers(t *testing.T) {
	ctx := context.WithValue(context.Background(), contextKeyUserID, "uid-1")
	ctx = context.WithValue(ctx, contextKeyUsername, "alice")
	ctx = context.WithValue(ctx, contextKeyRole, "admin")
	ctx = context.WithValue(ctx, contextKeyAuthError, "some error")
	ctx = context.WithValue(ctx, contextKeyAuthenticated, true)
	ctx = context.WithValue(ctx, contextKeyHTTPMethod, "POST")

	if got := getUserID(ctx); got != "uid-1" {
		t.Errorf("getUserID = %q, want %q", got, "uid-1")
	}
	if got := getUsername(ctx); got != "alice" {
		t.Errorf("getUsername = %q, want %q", got, "alice")
	}
	if got := getRole(ctx); got != "admin" {
		t.Errorf("getRole = %q, want %q", got, "admin")
	}
	if got := getAuthError(ctx); got != "some error" {
		t.Errorf("getAuthError = %q, want %q", got, "some error")
	}
	if !isAuthenticated(ctx) {
		t.Error("isAuthenticated = false, want true")
	}
	if !isHTTPRequest(ctx) {
		t.Error("isHTTPRequest = false, want true")
	}
}

func TestContextHelpers_EmptyContext(t *testing.T) {
	ctx := context.Background()

	if got := getUserID(ctx); got != "" {
		t.Errorf("getUserID = %q, want empty", got)
	}
	if got := getUsername(ctx); got != "" {
		t.Errorf("getUsername = %q, want empty", got)
	}
	if got := getRole(ctx); got != "" {
		t.Errorf("getRole = %q, want empty", got)
	}
	if got := getAuthError(ctx); got != "" {
		t.Errorf("getAuthError = %q, want empty", got)
	}
	if isAuthenticated(ctx) {
		t.Error("isAuthenticated = true, want false")
	}
	if isHTTPRequest(ctx) {
		t.Error("isHTTPRequest = true, want false")
	}
}

// --- HTTPContextFunc Tests ---

func TestHTTPContextFunc_AuthDisabled(t *testing.T) {
	am := NewAuthMiddleware("secret", false)
	fn := am.HTTPContextFunc()

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	ctx := fn(context.Background(), req)

	if !isAuthenticated(ctx) {
		t.Error("Expected authenticated when auth is disabled")
	}
	if !isHTTPRequest(ctx) {
		t.Error("Expected isHTTPRequest to be true")
	}
}

func TestHTTPContextFunc_MissingToken(t *testing.T) {
	am := NewAuthMiddleware("secret", true)
	fn := am.HTTPContextFunc()

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	ctx := fn(context.Background(), req)

	if isAuthenticated(ctx) {
		t.Error("Expected not authenticated with missing token")
	}
	if got := getAuthError(ctx); got == "" {
		t.Error("Expected auth error to be set")
	}
}

func TestHTTPContextFunc_InvalidToken(t *testing.T) {
	am := NewAuthMiddleware("secret", true)
	fn := am.HTTPContextFunc()

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	ctx := fn(context.Background(), req)

	if isAuthenticated(ctx) {
		t.Error("Expected not authenticated with invalid token")
	}
	if got := getAuthError(ctx); !strings.Contains(got, "invalid token") {
		t.Errorf("Expected auth error about invalid token, got: %q", got)
	}
}

func TestHTTPContextFunc_ValidToken(t *testing.T) {
	am := NewAuthMiddleware("secret", true)
	token, _ := am.GenerateJWT("u1", "bob", "admin", 1)
	fn := am.HTTPContextFunc()

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	ctx := fn(context.Background(), req)

	if !isAuthenticated(ctx) {
		t.Error("Expected authenticated with valid token")
	}
	if got := getUserID(ctx); got != "u1" {
		t.Errorf("getUserID = %q, want %q", got, "u1")
	}
	if got := getUsername(ctx); got != "bob" {
		t.Errorf("getUsername = %q, want %q", got, "bob")
	}
	if got := getRole(ctx); got != "admin" {
		t.Errorf("getRole = %q, want %q", got, "admin")
	}
}

// --- extractTokenFromHeader Tests ---

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
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			if got := extractTokenFromHeader(req); got != tc.want {
				t.Errorf("extractTokenFromHeader = %q, want %q", got, tc.want)
			}
		})
	}
}

// --- CORS Middleware Tests ---

func TestCorsMiddleware(t *testing.T) {
	config := newTestMCPConfig()
	config.HTTPCORSEnabled = true
	config.HTTPCORSOrigins = []string{"https://example.com", "https://app.io"}
	hsm := NewHTTPServerManager(config)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := hsm.corsMiddleware(inner)

	t.Run("allowed origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Origin", "https://example.com")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
			t.Errorf("CORS origin = %q, want %q", got, "https://example.com")
		}
	})

	t.Run("disallowed origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Origin", "https://evil.com")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("CORS origin = %q, want empty", got)
		}
	})

	t.Run("wildcard origin", func(t *testing.T) {
		config := newTestMCPConfig()
		config.HTTPCORSOrigins = []string{"*"}
		hsm := NewHTTPServerManager(config)
		handler := hsm.corsMiddleware(inner)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Origin", "https://anywhere.com")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://anywhere.com" {
			t.Errorf("CORS origin = %q, want reflected origin", got)
		}
	})

	t.Run("preflight request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/", nil)
		req.Header.Set("Origin", "https://example.com")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Preflight status = %d, want %d", rr.Code, http.StatusOK)
		}
		if got := rr.Header().Get("Access-Control-Allow-Methods"); got == "" {
			t.Error("Expected Access-Control-Allow-Methods header")
		}
	})
}

func TestIsAllowedOrigin(t *testing.T) {
	config := newTestMCPConfig()
	config.HTTPCORSOrigins = []string{"https://a.com", "https://b.com"}
	hsm := NewHTTPServerManager(config)

	if !hsm.isAllowedOrigin("https://a.com") {
		t.Error("Expected https://a.com to be allowed")
	}
	if hsm.isAllowedOrigin("https://c.com") {
		t.Error("Expected https://c.com to be disallowed")
	}

	config.HTTPCORSOrigins = []string{"*"}
	if !hsm.isAllowedOrigin("https://anything.com") {
		t.Error("Expected wildcard to allow any origin")
	}
}

// --- wrapWithAuth Tests ---

func TestWrapWithAuth_HTTPUnauthenticated(t *testing.T) {
	config := newTestMCPConfig()
	config.AuthEnabled = true

	handler := func(ctx context.Context, req mcp.CallToolRequest, cfg *MCPConfig) (*mcp.CallToolResult, error) {
		t.Fatal("Handler should not be called when unauthenticated")
		return nil, nil
	}

	wrapped := wrapWithAuth(handler, "test_tool", config)

	ctx := context.WithValue(context.Background(), contextKeyHTTPMethod, "POST")
	ctx = context.WithValue(ctx, contextKeyAuthenticated, false)
	ctx = context.WithValue(ctx, contextKeyAuthError, "missing token")

	result, err := wrapped(ctx, mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("Expected error result for unauthenticated request")
	}
}

func TestWrapWithAuth_HTTPAuthenticated(t *testing.T) {
	config := newTestMCPConfig()
	config.AuthEnabled = true

	called := false
	handler := func(ctx context.Context, req mcp.CallToolRequest, cfg *MCPConfig) (*mcp.CallToolResult, error) {
		called = true
		return mcp.NewToolResultText("ok"), nil
	}

	wrapped := wrapWithAuth(handler, "test_tool", config)

	ctx := context.WithValue(context.Background(), contextKeyHTTPMethod, "POST")
	ctx = context.WithValue(ctx, contextKeyAuthenticated, true)
	ctx = context.WithValue(ctx, contextKeyUsername, "alice")

	result, err := wrapped(ctx, mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result.IsError {
		t.Error("Expected success for authenticated request")
	}
	if !called {
		t.Error("Handler was not called")
	}
}

func TestWrapWithAuth_STDIOBypass(t *testing.T) {
	config := newTestMCPConfig()
	config.AuthEnabled = true

	called := false
	handler := func(ctx context.Context, req mcp.CallToolRequest, cfg *MCPConfig) (*mcp.CallToolResult, error) {
		called = true
		return mcp.NewToolResultText("ok"), nil
	}

	wrapped := wrapWithAuth(handler, "test_tool", config)

	// STDIO context has no HTTP method set
	result, err := wrapped(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result.IsError {
		t.Error("Expected success for STDIO request (auth bypass)")
	}
	if !called {
		t.Error("Handler was not called")
	}
}

// --- handleSendNotification with authenticated context ---

func TestHandleSendNotification_AuthenticatedContext(t *testing.T) {
	config := newTestMCPConfig()

	ctx := context.WithValue(context.Background(), contextKeyHTTPMethod, "POST")
	ctx = context.WithValue(ctx, contextKeyAuthenticated, true)
	ctx = context.WithValue(ctx, contextKeyUsername, "alice")

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{"message": "hello"},
		},
	}

	result, err := handleSendNotification(ctx, req, config)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	// Will fail to send (dummy keys) but exercises the authenticated path
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

// --- Run error paths ---

func TestRun_SendFailure(t *testing.T) {
	mockClient := &MockPushoverClient{shouldError: true}
	args := []string{"pushover", "-m", "test"}

	err := Run(args, mockClient)
	if err == nil {
		t.Fatal("Expected error when send fails")
	}
	if !strings.Contains(err.Error(), "failed to send message") {
		t.Errorf("Expected send failure error, got: %v", err)
	}
}

func TestRun_ArgParseError(t *testing.T) {
	err := Run([]string{"pushover"}, nil)
	if err == nil {
		t.Fatal("Expected error for missing message")
	}
	if !strings.Contains(err.Error(), "argument parsing failed") {
		t.Errorf("Expected parse error, got: %v", err)
	}
}

func TestRun_HelpFlag(t *testing.T) {
	err := Run([]string{"pushover", "-h"}, nil)
	if err != nil {
		t.Errorf("Expected no error for help flag, got: %v", err)
	}
}

func TestRun_InvalidFlag(t *testing.T) {
	err := Run([]string{"pushover", "-bogus-flag", "test"}, nil)
	if err == nil {
		t.Fatal("Expected error for unknown flag")
	}
	if !strings.Contains(err.Error(), "argument parsing failed") {
		t.Errorf("Expected parse error, got: %v", err)
	}
}

func TestRun_ClientCreationFailure(t *testing.T) {
	t.Setenv("APP_KEY", "")
	t.Setenv("RECIPIENT_KEY", "")

	err := Run([]string{"pushover", "-m", "test"}, nil)
	if err == nil {
		t.Fatal("Expected error when client creation fails")
	}
	if !strings.Contains(err.Error(), "failed to create Pushover client") {
		t.Errorf("Expected client creation error, got: %v", err)
	}
}

// --- SendNotification ---

func TestSendNotification_Error(t *testing.T) {
	mockClient := &MockPushoverClient{shouldError: true}
	msg := pushover.NewMessageWithTitle("test", "")
	recipient := pushover.NewRecipient("dummy")

	err := SendNotification(mockClient, msg, recipient)
	if err == nil {
		t.Error("Expected error from SendNotification")
	}
}

func TestSendNotification_Success(t *testing.T) {
	mockClient := &MockPushoverClient{}
	msg := pushover.NewMessageWithTitle("test", "")
	recipient := pushover.NewRecipient("dummy")

	err := SendNotification(mockClient, msg, recipient)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

// --- CreateMessage additional coverage ---

func TestCreateMessage_DeviceAndRetry(t *testing.T) {
	config := Config{
		DeviceName: "default-device",
		RetryTime:  60,
	}
	cliArgs := &CLIArgs{
		Message:    "test",
		DeviceName: "my-phone",
		RetryTime:  120,
	}

	msg := CreateMessage("test", "", config, cliArgs)

	if msg.DeviceName != "my-phone" {
		t.Errorf("DeviceName = %q, want %q", msg.DeviceName, "my-phone")
	}
	if msg.Retry != 120*time.Second {
		t.Errorf("Retry = %v, want %v", msg.Retry, 120*time.Second)
	}
}

func TestCreateMessage_ConfigFallbacks(t *testing.T) {
	config := Config{
		DeviceName: "config-device",
		RetryTime:  45,
		Priority:   1,
	}
	cliArgs := &CLIArgs{Message: "test"}

	msg := CreateMessage("test", "", config, cliArgs)

	if msg.DeviceName != "config-device" {
		t.Errorf("DeviceName = %q, want %q", msg.DeviceName, "config-device")
	}
	if msg.Retry != 45*time.Second {
		t.Errorf("Retry = %v, want %v", msg.Retry, 45*time.Second)
	}
	if msg.Priority != 1 {
		t.Errorf("Priority = %d, want 1", msg.Priority)
	}
}

// --- HTTPServerManager.Shutdown ---

func TestShutdown_NilServer(t *testing.T) {
	hsm := &HTTPServerManager{}
	err := hsm.Shutdown(context.Background())
	if err != nil {
		t.Errorf("Expected nil error for nil server, got: %v", err)
	}
}

// --- Health/Capabilities wrong method ---

func TestHealthEndpoint_WrongMethod(t *testing.T) {
	config := newTestMCPConfig()
	hsm := NewHTTPServerManager(config)

	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	rr := httptest.NewRecorder()
	hsm.handleHealth(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}

func TestCapabilitiesEndpoint_WrongMethod(t *testing.T) {
	config := newTestMCPConfig()
	hsm := NewHTTPServerManager(config)

	req := httptest.NewRequest(http.MethodDelete, "/capabilities", nil)
	rr := httptest.NewRecorder()
	hsm.handleCapabilities(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}

// --- GenerateTokenCommand / TokenInfo (output capture) ---

func TestGenerateTokenCommand(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	GenerateTokenCommand("secret", "u1", "alice", "admin", 24)

	_ = w.Close()
	os.Stdout = old

	out, _ := io.ReadAll(r)
	output := string(out)

	if !strings.Contains(output, "Generated JWT token:") {
		t.Error("Expected token generation output")
	}
	if !strings.Contains(output, "User ID: u1") {
		t.Error("Expected user ID in output")
	}
}

func TestTokenInfo_Valid(t *testing.T) {
	am := NewAuthMiddleware("secret", true)
	token, _ := am.GenerateJWT("u1", "alice", "admin", 1)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	am.TokenInfo(token)

	_ = w.Close()
	os.Stdout = old

	out, _ := io.ReadAll(r)
	output := string(out)

	if !strings.Contains(output, "User ID: u1") {
		t.Error("Expected user ID in token info")
	}
	if !strings.Contains(output, "Status: VALID") {
		t.Error("Expected VALID status")
	}
}

func TestTokenInfo_Invalid(t *testing.T) {
	am := NewAuthMiddleware("secret", true)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	am.TokenInfo("garbage.token.here")

	_ = w.Close()
	os.Stdout = old

	out, _ := io.ReadAll(r)
	output := string(out)

	if !strings.Contains(output, "Invalid token:") {
		t.Error("Expected invalid token message")
	}
}

// --- parseEnvDuration Tests ---

func TestParseEnvDuration(t *testing.T) {
	t.Run("valid duration", func(t *testing.T) {
		t.Setenv("TEST_DUR", "5m")
		got := parseEnvDuration("TEST_DUR", 30*time.Second)
		if got != 5*time.Minute {
			t.Errorf("got %v, want 5m", got)
		}
	})

	t.Run("invalid duration falls back to default", func(t *testing.T) {
		t.Setenv("TEST_DUR", "notaduration")
		got := parseEnvDuration("TEST_DUR", 10*time.Second)
		if got != 10*time.Second {
			t.Errorf("got %v, want default 10s", got)
		}
	})

	t.Run("empty env falls back to default", func(t *testing.T) {
		got := parseEnvDuration("TEST_DUR_UNSET", 15*time.Second)
		if got != 15*time.Second {
			t.Errorf("got %v, want default 15s", got)
		}
	})
}

// --- parseEnvStringSlice Tests ---

func TestParseEnvStringSlice(t *testing.T) {
	t.Run("comma separated", func(t *testing.T) {
		t.Setenv("TEST_SLICE", "a, b ,c")
		got := parseEnvStringSlice("TEST_SLICE", nil)
		want := []string{"a", "b", "c"}
		if len(got) != len(want) {
			t.Fatalf("got %v, want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("got[%d] = %q, want %q", i, got[i], want[i])
			}
		}
	})

	t.Run("empty items filtered", func(t *testing.T) {
		t.Setenv("TEST_SLICE", " , , ")
		got := parseEnvStringSlice("TEST_SLICE", []string{"default"})
		if len(got) != 1 || got[0] != "default" {
			t.Errorf("got %v, want [default]", got)
		}
	})

	t.Run("unset env returns default", func(t *testing.T) {
		got := parseEnvStringSlice("TEST_SLICE_UNSET", []string{"x"})
		if len(got) != 1 || got[0] != "x" {
			t.Errorf("got %v, want [x]", got)
		}
	})
}

// --- hasSubcommand Tests ---

func TestHasSubcommand(t *testing.T) {
	if sub, ok := hasSubcommand([]string{"pushover", "mcp"}); !ok || sub != "mcp" {
		t.Errorf("got (%q, %v), want (mcp, true)", sub, ok)
	}
	if _, ok := hasSubcommand([]string{"pushover", "unknown"}); ok {
		t.Error("Expected false for unknown subcommand")
	}
	if _, ok := hasSubcommand([]string{"pushover"}); ok {
		t.Error("Expected false for no subcommand")
	}
}

// --- parseMCPArgs Tests ---

func TestParseMCPArgs_Help(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := parseMCPArgs([]string{"pushover", "mcp", "-h"})

	_ = w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("Expected no error for help, got: %v", err)
	}
	out, _ := io.ReadAll(r)
	if !strings.Contains(string(out), "Start MCP server") {
		t.Error("Expected MCP help output")
	}
}

func TestParseMCPArgs_GenerateToken(t *testing.T) {
	t.Setenv("APP_KEY", "test-key")
	t.Setenv("RECIPIENT_KEY", "test-recipient")
	t.Setenv("PUSHOVER_AUTH_SECRET_KEY", "gen-secret")

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := parseMCPArgs([]string{"pushover", "mcp", "-generate-token", "-token-user-id", "u9"})

	_ = w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	out, _ := io.ReadAll(r)
	if !strings.Contains(string(out), "User ID: u9") {
		t.Error("Expected token output with user ID")
	}
}

func TestParseMCPArgs_InvalidFlag(t *testing.T) {
	err := parseMCPArgs([]string{"pushover", "mcp", "-badflag"})
	if err == nil {
		t.Error("Expected error for invalid flag")
	}
}

// --- generateTokenFromArgs Tests ---

func TestGenerateTokenFromArgs_MissingSecret(t *testing.T) {
	t.Setenv("APP_KEY", "k")
	t.Setenv("RECIPIENT_KEY", "r")
	os.Unsetenv("PUSHOVER_AUTH_SECRET_KEY")

	err := generateTokenFromArgs("u", "user", "admin", 1)
	if err == nil {
		t.Fatal("Expected error for missing secret")
	}
	if !strings.Contains(err.Error(), "PUSHOVER_AUTH_SECRET_KEY") {
		t.Errorf("Expected secret key error, got: %v", err)
	}
}

func TestGenerateTokenFromArgs_Success(t *testing.T) {
	t.Setenv("APP_KEY", "k")
	t.Setenv("RECIPIENT_KEY", "r")
	t.Setenv("PUSHOVER_AUTH_SECRET_KEY", "s")

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := generateTokenFromArgs("u1", "alice", "admin", 24)

	_ = w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	out, _ := io.ReadAll(r)
	if !strings.Contains(string(out), "Generated JWT token:") {
		t.Error("Expected token generation output")
	}
}

// --- handleGenerateToken additional paths ---

func TestHandleGenerateToken_InvalidJSON(t *testing.T) {
	config := newTestMCPConfig()
	hsm := NewHTTPServerManager(config)

	req := httptest.NewRequest(http.MethodPost, "/generate-token", strings.NewReader("{invalid"))
	rr := httptest.NewRecorder()
	hsm.handleGenerateToken(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestHandleGenerateToken_Defaults(t *testing.T) {
	config := newTestMCPConfig()
	hsm := NewHTTPServerManager(config)

	req := httptest.NewRequest(http.MethodPost, "/generate-token", strings.NewReader("{}"))
	rr := httptest.NewRecorder()
	hsm.handleGenerateToken(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Status = %d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if resp["user_id"] != "default_user" {
		t.Errorf("user_id = %v, want default_user", resp["user_id"])
	}
	if resp["username"] != "pushover_user" {
		t.Errorf("username = %v, want pushover_user", resp["username"])
	}
	if resp["role"] != "user" {
		t.Errorf("role = %v, want user", resp["role"])
	}
}

// --- Structured Logging Tests ---

func TestParseLogLevel(t *testing.T) {
	cases := []struct {
		input string
		want  slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"error", slog.LevelError},
		{" error ", slog.LevelError},
		{"", slog.LevelInfo},
		{"bogus", slog.LevelInfo},
	}

	for _, tc := range cases {
		if got := parseLogLevel(tc.input); got != tc.want {
			t.Errorf("parseLogLevel(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestSetupServerLogging(t *testing.T) {
	defer slog.SetDefault(slog.Default())

	t.Setenv("PUSHOVER_LOG_LEVEL", "warn")

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	setupServerLogging("http")
	slog.Info("should be filtered")
	slog.Warn("visible message", "key", "value")

	_ = w.Close()
	os.Stderr = oldStderr

	out, _ := io.ReadAll(r)
	output := string(out)

	if strings.Contains(output, "should be filtered") {
		t.Error("Info message should be filtered at warn level")
	}
	if !strings.Contains(output, "visible message") {
		t.Fatal("Warn message missing from output")
	}

	var entry map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &entry); err != nil {
		t.Fatalf("Log output is not valid JSON: %v\noutput: %s", err, output)
	}
	if entry["service"] != "pushover-mcp" {
		t.Errorf("service = %v, want pushover-mcp", entry["service"])
	}
	if entry["transport"] != "http" {
		t.Errorf("transport = %v, want http", entry["transport"])
	}
	if entry["key"] != "value" {
		t.Errorf("key = %v, want value", entry["key"])
	}
}
