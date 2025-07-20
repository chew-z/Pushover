package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gregdel/pushover"
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
	if claims.ExpiresAt <= time.Now().Unix() {
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
		errContains string
	}{
		{"invalid signature", token, NewAuthMiddleware("secret2", true), "invalid signature"},
		{"expired token", expiredToken, amExpired, ""}, // validateJWT doesn't check expiration itself
		{"invalid format", "a.b", am, "invalid token format"},
		{"malformed payload", "a.badpayload.c", am, "invalid signature"}, // This actually fails on signature validation first
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			claims, err := tc.middleware.validateJWT(tc.token)
			if err == nil {
				// Specific case for expired token, which is valid structurally
				if tc.name == "expired token" {
					if time.Now().Unix() < claims.ExpiresAt {
						t.Error("Expected token to be expired, but it is not")
					}
				} else {
					t.Fatalf("Expected error, but got none")
				}
			} else if !strings.Contains(err.Error(), tc.errContains) {
				t.Errorf("Expected error to contain '%s', got '%v'", tc.errContains, err)
			}
		})
	}
}

func TestHandleSendNotification(t *testing.T) {
	testCases := []struct {
		name        string
		message     string
		title       string
		priority    string
		device      string
		sound       string
		expire      string
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
			name:        "message too long",
			message:     strings.Repeat("a", 1025),
			wantErr:     true,
			errContains: "Message too long",
		},
		{
			name:        "invalid priority string",
			message:     "test",
			priority:    "high",
			wantErr:     true,
			errContains: "Invalid priority value",
		},
		{
			name:        "priority out of range",
			message:     "test",
			priority:    "5",
			wantErr:     true,
			errContains: "Priority must be between -2 and 2",
		},
		{
			name:        "emergency priority with expire",
			message:     "emergency",
			priority:    strconv.Itoa(int(pushover.PriorityEmergency)),
			expire:      "60",
			wantErr:     true, // Still fails on send
			errContains: "Failed to send notification",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Build arguments map
			args := make(map[string]interface{})
			if tc.message != "" {
				args["message"] = tc.message
			}
			if tc.title != "" {
				args["title"] = tc.title
			}
			if tc.priority != "" {
				args["priority"] = tc.priority
			}
			if tc.device != "" {
				args["device"] = tc.device
			}
			if tc.sound != "" {
				args["sound"] = tc.sound
			}
			if tc.expire != "" {
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

			// Check if result indicates error (by checking the result text for error patterns)
			resultText := ""
			if result != nil {
				// Since we can't access fields directly, we'll examine the result type
				// Error results from mcp.NewToolResultError should be distinguishable
				resultText = fmt.Sprintf("%v", result)
			}

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
