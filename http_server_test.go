package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func TestBuildHandlerRoutes(t *testing.T) {
	cfg := testMCPConfig()
	cfg.AuthEnabled = false
	cfg.HTTPCORSEnabled = false
	handler, err := NewHTTPServerManager(cfg).buildHandler(setupMCPServer(cfg))
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"/other", "/authorize", "/token", "/register", "/.well-known/oauth-protected-resource"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequestWithContext(context.Background(), http.MethodGet, target, nil))
		if response.Code != http.StatusNotFound {
			t.Errorf("%s: status = %d", target, response.Code)
		}
	}
	for _, target := range []string{"/health", "/capabilities"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequestWithContext(context.Background(), http.MethodGet, target, nil))
		if response.Code != http.StatusOK {
			t.Errorf("%s: status = %d", target, response.Code)
		}
	}
}

func TestBuildHandlerOAuthDiscovery(t *testing.T) {
	cfg := testOAuthMCPConfig(t)
	cfg.HTTPCORSEnabled = false
	cfg.OAuthIssuer = "http://host/pushover"
	cfg.OAuthResource = "http://host/pushover/mcp"
	handler, err := NewHTTPServerManager(cfg).buildHandler(setupMCPServer(cfg))
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"/.well-known/oauth-protected-resource/pushover/mcp", "/.well-known/oauth-authorization-server/pushover", "/.well-known/oauth-protected-resource", "/.well-known/oauth-authorization-server"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequestWithContext(context.Background(), http.MethodGet, target, nil))
		if response.Code != http.StatusOK {
			t.Errorf("%s: status = %d, body = %s", target, response.Code, response.Body.String())
		}
	}
}

func TestCORSMiddlewareExposesMCPHeaders(t *testing.T) {
	cfg := testMCPConfig()
	cfg.HTTPCORSOrigins = []string{"https://example.test"}
	handler := NewHTTPServerManager(cfg).corsMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	req := httptest.NewRequestWithContext(context.Background(), http.MethodOptions, "/mcp", nil)
	req.Header.Set("Origin", "https://example.test")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != http.StatusOK || response.Header().Get("Access-Control-Allow-Methods") != "GET, POST, DELETE, OPTIONS" || response.Header().Get("Access-Control-Allow-Headers") != "Content-Type, Authorization, Mcp-Protocol-Version, Mcp-Session-Id, Last-Event-ID" || response.Header().Get("Access-Control-Expose-Headers") != "WWW-Authenticate, Mcp-Session-Id" {
		t.Fatalf("headers = %#v", response.Header())
	}
}

func TestCorsMiddleware(t *testing.T) {
	config := newTestMCPConfig()
	config.HTTPCORSEnabled = true
	config.HTTPCORSOrigins = []string{"https://example.com", "https://app.io"}
	hsm := NewHTTPServerManager(config)
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	handler := hsm.corsMiddleware(inner)

	t.Run("allowed origin", func(t *testing.T) {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		req.Header.Set("Origin", "https://example.com")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
			t.Errorf("CORS origin = %q, want %q", got, "https://example.com")
		}
	})
	t.Run("disallowed origin", func(t *testing.T) {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
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
		handler := NewHTTPServerManager(config).corsMiddleware(inner)
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		req.Header.Set("Origin", "https://anywhere.com")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://anywhere.com" {
			t.Errorf("CORS origin = %q, want reflected origin", got)
		}
	})
	t.Run("preflight request", func(t *testing.T) {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodOptions, "/", nil)
		req.Header.Set("Origin", "https://example.com")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("Preflight status = %d, want %d", rr.Code, http.StatusOK)
		}
		if got := rr.Header().Get("Access-Control-Allow-Methods"); got != "GET, POST, DELETE, OPTIONS" {
			t.Errorf("methods = %q", got)
		}
		if got := rr.Header().Get("Access-Control-Allow-Headers"); got != "Content-Type, Authorization, Mcp-Protocol-Version, Mcp-Session-Id, Last-Event-ID" {
			t.Errorf("headers = %q", got)
		}
		if got := rr.Header().Get("Access-Control-Expose-Headers"); got != "WWW-Authenticate, Mcp-Session-Id" {
			t.Errorf("expose headers = %q", got)
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

func TestHTTPContextFuncOpenMode(t *testing.T) {
	cfg := testMCPConfig()
	cfg.AuthEnabled, cfg.HTTPCORSEnabled = false, false
	mcpServer := server.NewMCPServer("test", "test")
	seenHTTP := false
	mcpServer.AddTool(mcp.NewTool("http_context"), func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if isHTTPRequest(ctx) {
			seenHTTP = true
			return mcp.NewToolResultText("true"), nil
		}
		return mcp.NewToolResultText("false"), nil
	})
	handler, err := NewHTTPServerManager(cfg).buildHandler(mcpServer)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)
	mcpClient, err := client.NewStreamableHttpClient(ts.URL + cfg.HTTPPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mcpClient.Close() })
	if err := mcpClient.Start(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := mcpClient.Initialize(t.Context(), mcp.InitializeRequest{Params: mcp.InitializeParams{ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION, ClientInfo: mcp.Implementation{Name: "test", Version: "0"}}}); err != nil {
		t.Fatal(err)
	}
	result, err := mcpClient.CallTool(t.Context(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "http_context", Arguments: map[string]any{}}})
	if err != nil || result.IsError || !seenHTTP {
		t.Fatalf("tool = %+v, HTTP marker = %t, err = %v", result, seenHTTP, err)
	}
}
