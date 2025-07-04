package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/server"
)

// HTTPServerManager manages the HTTP transport for the MCP server
type HTTPServerManager struct {
	httpServer *http.Server
	mcpServer  *server.StreamableHTTPServer
	config     *MCPConfig
}

// NewHTTPServerManager creates a new HTTP server manager
func NewHTTPServerManager(mcpServer *server.MCPServer, config *MCPConfig) *HTTPServerManager {
	return &HTTPServerManager{
		config: config,
	}
}

// Start starts the HTTP server with the given MCP server
func (hsm *HTTPServerManager) Start(mcpServer *server.MCPServer) error {
	// Create HTTP server options
	var opts []server.StreamableHTTPOption

	// Configure heartbeat interval
	if hsm.config.HTTPHeartbeat > 0 {
		opts = append(opts, server.WithHeartbeatInterval(hsm.config.HTTPHeartbeat))
	}

	// Configure stateless mode
	opts = append(opts, server.WithStateLess(hsm.config.HTTPStateless))

	// Configure endpoint path
	opts = append(opts, server.WithEndpointPath(hsm.config.HTTPPath))

	// Configure authentication middleware
	if hsm.config.AuthEnabled {
		authMiddleware := NewAuthMiddleware(hsm.config.AuthSecretKey, hsm.config.AuthEnabled)
		opts = append(opts, server.WithHTTPContextFunc(authMiddleware.HTTPContextFunc()))
	}

	// Create the streamable HTTP server
	hsm.mcpServer = server.NewStreamableHTTPServer(mcpServer, opts...)

	// Create custom HTTP server with additional endpoints
	mux := http.NewServeMux()

	// Mount the MCP server at the root path
	mux.Handle("/", hsm.mcpServer)

	// Add health endpoint
	mux.HandleFunc("/health", hsm.handleHealth)

	// Add capabilities endpoint
	mux.HandleFunc("/capabilities", hsm.handleCapabilities)

	// Add token generation endpoint (if auth is enabled)
	if hsm.config.AuthEnabled {
		mux.HandleFunc("/generate-token", hsm.handleGenerateToken)
	}

	// Wrap with CORS middleware if enabled
	var handler http.Handler = mux
	if hsm.config.HTTPCORSEnabled {
		handler = hsm.corsMiddleware(handler)
	}

	// Create HTTP server
	hsm.httpServer = &http.Server{
		Addr:         hsm.config.HTTPAddress,
		Handler:      handler,
		ReadTimeout:  hsm.config.HTTPTimeout,
		WriteTimeout: hsm.config.HTTPTimeout,
		IdleTimeout:  hsm.config.HTTPTimeout * 2,
	}

	// Set up graceful shutdown
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		log.Printf("MCP Pushover server starting on %s%s", hsm.config.HTTPAddress, hsm.config.HTTPPath)
		if err := hsm.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-stopChan
	log.Println("Shutting down HTTP server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown the server gracefully
	if err := hsm.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	log.Println("HTTP server stopped")
	return nil
}

// Shutdown gracefully shuts down the HTTP server
func (hsm *HTTPServerManager) Shutdown(ctx context.Context) error {
	if hsm.httpServer != nil {
		return hsm.httpServer.Shutdown(ctx)
	}
	return nil
}

// handleHealth handles health check requests
func (hsm *HTTPServerManager) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"service":   "pushover-mcp-server",
	}

	json.NewEncoder(w).Encode(response)
}

// handleCapabilities handles capability discovery requests
func (hsm *HTTPServerManager) handleCapabilities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	capabilities := map[string]interface{}{
		"name":        "pushover-mcp-server",
		"version":     version,
		"description": "MCP server for sending Pushover notifications",
		"transport":   "http",
		"tools": []map[string]interface{}{
			{
				"name":        "send_notification",
				"description": "Send a notification via Pushover",
				"parameters": map[string]interface{}{
					"message":  "required - The notification message to send",
					"title":    "optional - Title for the notification",
					"priority": "optional - Priority level (-2 to 2)",
					"device":   "optional - Target device name",
					"sound":    "optional - Notification sound",
					"expire":   "optional - Expiration time for emergency messages",
				},
			},
		},
		"authentication": map[string]interface{}{
			"enabled": hsm.config.AuthEnabled,
			"type":    "JWT",
		},
		"features": []string{
			"notifications",
			"health_check",
			"capabilities_discovery",
		},
	}

	json.NewEncoder(w).Encode(capabilities)
}

// handleGenerateToken handles token generation requests (for testing/development)
func (hsm *HTTPServerManager) handleGenerateToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// This is a simple implementation for testing purposes
	// In production, you'd want proper user authentication and validation
	var request struct {
		UserID    string `json:"user_id"`
		Username  string `json:"username"`
		Role      string `json:"role"`
		ExpiresIn int    `json:"expires_in"` // hours
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Set defaults
	if request.UserID == "" {
		request.UserID = "default_user"
	}
	if request.Username == "" {
		request.Username = "pushover_user"
	}
	if request.Role == "" {
		request.Role = "user"
	}
	if request.ExpiresIn <= 0 {
		request.ExpiresIn = 24 // 24 hours default
	}

	// Generate token
	authMiddleware := NewAuthMiddleware(hsm.config.AuthSecretKey, true)
	token, err := authMiddleware.GenerateJWT(request.UserID, request.Username, request.Role, request.ExpiresIn)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"token":      token,
		"user_id":    request.UserID,
		"username":   request.Username,
		"role":       request.Role,
		"expires_in": request.ExpiresIn,
		"expires_at": time.Now().Add(time.Duration(request.ExpiresIn) * time.Hour).UTC().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}

// corsMiddleware adds CORS headers to HTTP responses
func (hsm *HTTPServerManager) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		origin := r.Header.Get("Origin")
		if origin != "" && hsm.isAllowedOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else if len(hsm.config.HTTPCORSOrigins) == 1 && hsm.config.HTTPCORSOrigins[0] == "*" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// isAllowedOrigin checks if the origin is in the allowed list
func (hsm *HTTPServerManager) isAllowedOrigin(origin string) bool {
	for _, allowedOrigin := range hsm.config.HTTPCORSOrigins {
		if allowedOrigin == "*" || allowedOrigin == origin {
			return true
		}
	}
	return false
}
