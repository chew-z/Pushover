package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	"github.com/chew-z/Pushover/internal/oauth"
	"github.com/mark3labs/mcp-go/server"
)

// HTTPServerManager manages the HTTP transport for the MCP server
type HTTPServerManager struct {
	httpServer *http.Server
	mcpServer  *server.StreamableHTTPServer
	// tools is the source of the tool list advertised by /capabilities.
	tools  *server.MCPServer
	config *MCPConfig
}

// NewHTTPServerManager creates a new HTTP server manager
func NewHTTPServerManager(config *MCPConfig) *HTTPServerManager {
	return &HTTPServerManager{config: config}
}

// buildHandler assembles the HTTP mux and MCP transport; it omits a "/" catch-all because
// mcp-go's ServeHTTP ignores the path, so the mux enforces HTTPPath and every other path returns 404.
func (hsm *HTTPServerManager) buildHandler(mcpServer *server.MCPServer) (http.Handler, error) {
	hsm.tools = mcpServer
	opts := []server.StreamableHTTPOption{server.WithStateLess(hsm.config.HTTPStateless), server.WithEndpointPath(hsm.config.HTTPPath), server.WithHTTPContextFunc(copyActor)}
	if hsm.config.HTTPHeartbeat > 0 {
		opts = append(opts, server.WithHeartbeatInterval(hsm.config.HTTPHeartbeat))
	}
	if hsm.config.HTTPDisableLocalhostProtection {
		// nginx forwards the real Host header, so DNS-rebinding protection would 403 every proxied request.
		opts = append(opts, server.WithDisableLocalhostProtection(true))
	}
	hsm.mcpServer = server.NewStreamableHTTPServer(mcpServer, opts...)
	mux := http.NewServeMux()
	var oauthCfg *oauth.Config
	if hsm.config.OAuthEnabled {
		cfg, err := hsm.config.OAuthConfig()
		if err != nil {
			return nil, fmt.Errorf("build OAuth configuration: %w", err)
		}
		oauthCfg = &cfg
	}
	var guard *authGuard
	if hsm.config.AuthEnabled {
		guard = newAuthGuard(hsm.config, oauthCfg)
	}
	mux.Handle(hsm.config.HTTPPath, guard.protectMCP(hsm.mcpServer))
	mux.HandleFunc("/health", hsm.handleHealth)
	mux.HandleFunc("/capabilities", hsm.handleCapabilities)
	if oauthCfg != nil {
		oauth.Mount(mux, *oauthCfg, oauth.NewStore(nil), slog.Default())
	}
	var handler http.Handler = mux
	if hsm.config.HTTPCORSEnabled {
		handler = hsm.corsMiddleware(handler)
	}
	return handler, nil
}

// Start starts the HTTP server with the given MCP server
func (hsm *HTTPServerManager) Start(mcpServer *server.MCPServer) error {
	handler, err := hsm.buildHandler(mcpServer)
	if err != nil {
		return fmt.Errorf("build HTTP handler: %w", err)
	}
	hsm.httpServer = &http.Server{Addr: hsm.config.HTTPAddress, Handler: handler, ReadTimeout: hsm.config.HTTPTimeout, WriteTimeout: hsm.config.HTTPTimeout, IdleTimeout: hsm.config.HTTPTimeout * 2}
	// Set up graceful shutdown
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stopChan)
	// Start server in a goroutine
	go func() {
		slog.Info("HTTP server listening", "address", hsm.config.HTTPAddress, "path", hsm.config.HTTPPath)
		if err := hsm.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server error", "error", err)
		}
	}()
	// Wait for shutdown signal
	<-stopChan
	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// Shutdown the server gracefully
	if err := hsm.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}
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
	_ = json.NewEncoder(w).Encode(map[string]any{"status": "healthy", "timestamp": time.Now().UTC().Format(time.RFC3339), "service": "pushover-mcp-server"})
}

// handleCapabilities handles capability discovery requests
func (hsm *HTTPServerManager) handleCapabilities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	capabilities := map[string]any{
		"name":        "pushover-mcp-server",
		"version":     version,
		"description": "MCP server for sending Pushover notifications",
		"transport":   "http",
		"tools":       capabilityTools(hsm.tools),
		"authentication": map[string]any{
			"enabled": hsm.config.AuthEnabled,
			"type":    "bearer",
			"oauth":   hsm.config.OAuthEnabled,
		},
		"features": []string{"notifications", "health_check", "capabilities_discovery"},
	}
	_ = json.NewEncoder(w).Encode(capabilities)
}

// capabilityTools returns deterministic capability data for the registered MCP tools.
func capabilityTools(s *server.MCPServer) []map[string]any {
	tools := make([]map[string]any, 0)
	if s == nil {
		return tools
	}

	registered := s.ListTools()
	names := make([]string, 0, len(registered))
	for name := range registered {
		names = append(names, name)
	}
	slices.Sort(names)

	for _, name := range names {
		serverTool := registered[name]
		if serverTool == nil {
			continue
		}
		tool := serverTool.Tool
		inputSchema := any(tool.InputSchema)
		if len(tool.RawInputSchema) > 0 {
			inputSchema = tool.RawInputSchema
		}
		tools = append(tools, map[string]any{
			"name":        tool.Name,
			"description": tool.Description,
			"inputSchema": inputSchema,
		})
	}

	return tools
}

// corsMiddleware adds CORS headers to HTTP responses
func (hsm *HTTPServerManager) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && hsm.isAllowedOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else if len(hsm.config.HTTPCORSOrigins) == 1 && hsm.config.HTTPCORSOrigins[0] == "*" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Mcp-Protocol-Version, Mcp-Session-Id, Last-Event-ID")
		w.Header().Set("Access-Control-Expose-Headers", "WWW-Authenticate, Mcp-Session-Id")
		w.Header().Set("Access-Control-Max-Age", "86400")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// isAllowedOrigin reports whether an origin is in the configured allowed list.
func (hsm *HTTPServerManager) isAllowedOrigin(origin string) bool {
	for _, allowedOrigin := range hsm.config.HTTPCORSOrigins {
		if allowedOrigin == "*" || allowedOrigin == origin {
			return true
		}
	}
	return false
}
