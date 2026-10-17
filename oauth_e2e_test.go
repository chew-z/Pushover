package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/chew-z/Pushover/internal/oauth"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
)

const oauthVerifier = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"

func oauthRequest(t *testing.T, method, target string, body io.Reader) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), method, target, body)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	return req
}

func oauthDo(t *testing.T, client *http.Client, req *http.Request) (int, http.Header, string) {
	t.Helper()
	response, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", req.Method, req.URL, err)
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, response.Header, string(body)
}

func oauthHiddenField(t *testing.T, body, name string) string {
	t.Helper()
	marker := `name="` + name + `" value="`
	start := strings.Index(body, marker)
	if start < 0 {
		t.Fatalf("missing %s in %q", name, body)
	}
	value := body[start+len(marker):]
	end := strings.IndexByte(value, '"')
	if end < 0 {
		t.Fatalf("unterminated %s", name)
	}
	return value[:end]
}

func oauthServer(t *testing.T) (*httptest.Server, *MCPConfig) {
	t.Helper()
	var handler http.Handler
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { handler.ServeHTTP(w, r) }))
	cfg := testOAuthMCPConfig(t)
	cfg.HTTPCORSEnabled = false
	cfg.OAuthIssuer = server.URL
	cfg.OAuthResource = server.URL + "/mcp"
	var err error
	handler, err = NewHTTPServerManager(cfg).buildHandler(setupMCPServer(cfg))
	if err != nil {
		server.Close()
		t.Fatal(err)
	}
	t.Cleanup(server.Close)
	return server, cfg
}

func oauthRegister(t *testing.T, server *httptest.Server) string {
	t.Helper()
	body := `{"client_name":"test client","redirect_uris":["https://example.test/callback"],"token_endpoint_auth_method":"none"}`
	req := oauthRequest(t, http.MethodPost, server.URL+"/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	status, _, response := oauthDo(t, http.DefaultClient, req)
	if status != http.StatusCreated {
		t.Fatalf("register = %d: %s", status, response)
	}
	var registered struct {
		ClientID string `json:"client_id"`
	}
	if err := json.Unmarshal([]byte(response), &registered); err != nil || registered.ClientID == "" {
		t.Fatalf("registration = %s: %v", response, err)
	}
	return registered.ClientID
}

func obtainOAuthTokens(t *testing.T, server *httptest.Server, cfg *MCPConfig, clientID string) oauth.TokenResponse {
	t.Helper()
	sum := sha256.Sum256([]byte(oauthVerifier))
	query := url.Values{"response_type": {"code"}, "client_id": {clientID}, "redirect_uri": {"https://example.test/callback"}, "code_challenge": {base64.RawURLEncoding.EncodeToString(sum[:])}, "code_challenge_method": {"S256"}, "state": {"state"}}
	httpClient := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	status, _, body := oauthDo(t, httpClient, oauthRequest(t, http.MethodGet, server.URL+"/authorize?"+query.Encode(), nil))
	if status != http.StatusOK {
		t.Fatalf("authorize = %d: %s", status, body)
	}
	form := url.Values{"pending_id": {oauthHiddenField(t, body, "pending_id")}, "csrf_token": {oauthHiddenField(t, body, "csrf_token")}, "username": {cfg.OAuthLoginUser}, "password": {"pw"}}
	login := oauthRequest(t, http.MethodPost, server.URL+"/login", strings.NewReader(form.Encode()))
	login.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	status, header, body := oauthDo(t, httpClient, login)
	if status != http.StatusFound {
		t.Fatalf("login = %d: %s", status, body)
	}
	redirect, err := url.Parse(header.Get("Location"))
	if err != nil || redirect.Query().Get("code") == "" {
		t.Fatalf("redirect = %q, %v", header.Get("Location"), err)
	}
	form = url.Values{"grant_type": {"authorization_code"}, "code": {redirect.Query().Get("code")}, "redirect_uri": {"https://example.test/callback"}, "client_id": {clientID}, "code_verifier": {oauthVerifier}}
	token := oauthRequest(t, http.MethodPost, server.URL+"/token", strings.NewReader(form.Encode()))
	token.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	status, _, body = oauthDo(t, httpClient, token)
	if status != http.StatusOK {
		t.Fatalf("token = %d: %s", status, body)
	}
	var tokens oauth.TokenResponse
	if err := json.Unmarshal([]byte(body), &tokens); err != nil || tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatalf("token response = %s: %v", body, err)
	}
	return tokens
}

func TestEndToEndOAuthThenTools(t *testing.T) {
	server, cfg := oauthServer(t)
	clientID := oauthRegister(t, server)
	tokens := obtainOAuthTokens(t, server, cfg, clientID)
	listToolsWithBearer(t, server.URL+"/mcp", tokens.AccessToken)
	refresh := url.Values{"grant_type": {"refresh_token"}, "refresh_token": {tokens.RefreshToken}, "client_id": {clientID}}
	req := oauthRequest(t, http.MethodPost, server.URL+"/token", strings.NewReader(refresh.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	status, _, body := oauthDo(t, http.DefaultClient, req)
	if status != http.StatusOK {
		t.Fatalf("refresh = %d: %s", status, body)
	}
	var renewed oauth.TokenResponse
	if err := json.Unmarshal([]byte(body), &renewed); err != nil || renewed.AccessToken == "" {
		t.Fatalf("refresh response = %s: %v", body, err)
	}
	listToolsWithBearer(t, server.URL+"/mcp", renewed.AccessToken)
	staticToken, err := GenerateStaticToken(testSecret, "admin", cfg.OAuthAccessTTL)
	if err != nil {
		t.Fatal(err)
	}
	listToolsWithBearer(t, server.URL+"/mcp", staticToken)
	status, header, _ := oauthDo(t, http.DefaultClient, oauthRequest(t, http.MethodPost, server.URL+"/mcp", nil))
	if status != http.StatusUnauthorized || !strings.Contains(header.Get("WWW-Authenticate"), "resource_metadata") {
		t.Fatalf("missing token = %d, %q", status, header.Get("WWW-Authenticate"))
	}
}

func listToolsWithBearer(t *testing.T, endpoint, bearer string) {
	t.Helper()
	mcpClient, err := client.NewStreamableHttpClient(endpoint, transport.WithHTTPHeaders(map[string]string{"Authorization": "Bearer " + bearer}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mcpClient.Close() })
	if err := mcpClient.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := mcpClient.Initialize(context.Background(), mcpgo.InitializeRequest{Params: mcpgo.InitializeParams{ProtocolVersion: mcpgo.LATEST_PROTOCOL_VERSION, ClientInfo: mcpgo.Implementation{Name: "test", Version: "0"}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := mcpClient.ListTools(context.Background(), mcpgo.ListToolsRequest{}); err != nil {
		t.Fatal(err)
	}
}
