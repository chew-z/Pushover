package main

import (
	"bytes"
	"context"
	"crypto/pbkdf2"
	"crypto/sha256"
	"encoding/hex"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestMCPCommandHashPasswordWithoutPushoverKeys(t *testing.T) {
	t.Setenv("APP_KEY", "")
	t.Setenv("RECIPIENT_KEY", "")
	var stdout, stderr bytes.Buffer
	if err := runMCPCommand([]string{"pushover", "mcp", "-hash-password"}, strings.NewReader("secret\n"), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "Password: ") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	parts := strings.Split(strings.TrimSpace(stdout.String()), ".")
	if len(parts) != 4 || parts[0] != "pbkdf2-sha256" {
		t.Fatalf("hash = %q", stdout.String())
	}
	iterations, err := strconv.Atoi(parts[1])
	if err != nil {
		t.Fatal(err)
	}
	salt, err := hex.DecodeString(parts[2])
	if err != nil {
		t.Fatal(err)
	}
	want, err := hex.DecodeString(parts[3])
	if err != nil {
		t.Fatal(err)
	}
	got, err := pbkdf2.Key(sha256.New, "secret", salt, iterations, len(want))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("hash does not verify")
	}
}

func TestMCPCommandGenerateTokenWithoutPushoverKeys(t *testing.T) {
	t.Setenv("APP_KEY", "")
	t.Setenv("RECIPIENT_KEY", "")
	t.Setenv("PUSHOVER_AUTH_SECRET_KEY", testSecret)
	var stdout, stderr bytes.Buffer
	if err := runMCPCommand([]string{"pushover", "mcp", "-generate-token", "-token-username", "alice"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/mcp", nil)
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(stdout.String()))
	claims, err := parseBearerToken(testSecret, req)
	if err != nil || claims.Username != "alice" {
		t.Fatalf("parseBearerToken() = %+v, %v", claims, err)
	}
}

func TestMCPCommandGenerateTokenRequiresSecret(t *testing.T) {
	t.Setenv("PUSHOVER_AUTH_SECRET_KEY", "")
	if err := runMCPCommand([]string{"pushover", "mcp", "-generate-token"}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
		t.Fatal("missing secret accepted")
	}
}

func TestMCPCommandHelpAndInvalidFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := runMCPCommand([]string{"pushover", "mcp", "-h"}, strings.NewReader(""), &stdout, &stderr); err != nil || !strings.Contains(stdout.String(), "Start MCP server") {
		t.Fatalf("help = %v, %q", err, stdout.String())
	}
	if err := runMCPCommand([]string{"pushover", "mcp", "-badflag"}, strings.NewReader(""), &stdout, &stderr); err == nil {
		t.Fatal("invalid flag accepted")
	}
}
