package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gregdel/pushover"
	"github.com/mark3labs/mcp-go/mcp"
)

// pngData returns PNG bytes padded with zeros to size.
func pngData(t *testing.T, size int) []byte {
	t.Helper()

	data := []byte{
		0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n',
		0, 0, 0, 13, 'I', 'H', 'D', 'R',
		0, 0, 0, 1, 0, 0, 0, 1, 8, 2, 0, 0, 0,
	}
	if size < len(data) {
		t.Fatalf("PNG size %d is smaller than header size %d", size, len(data))
	}
	return append(data, make([]byte, size-len(data))...)
}

// writeAttachmentFile writes data to a temporary attachment file and returns its path.
func writeAttachmentFile(t *testing.T, data []byte) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "attachment")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}

// requireAttachmentError checks that err contains want.
func requireAttachmentError(t *testing.T, err error, want string) {
	t.Helper()

	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("loadAttachment() error = %v, want text %q", err, want)
	}
}

// TestLoadAttachmentSparseOversizedFile verifies oversized files are rejected before their content is read.
func TestLoadAttachmentSparseOversizedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "oversized.png")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := os.Truncate(path, int64(pushover.MessageMaxAttachmentByte)+1); err != nil {
		t.Fatalf("Truncate() error = %v", err)
	}

	_, err = loadAttachment(context.Background(), path, "", "")
	requireAttachmentError(t, err, "attachment file size")
}

// TestLoadAttachmentBase64 verifies base64 attachment validation and decoding.
func TestLoadAttachmentBase64(t *testing.T) {
	testCases := []struct {
		name      string
		dataSize  int
		b64       string
		mimeType  string
		wantError string
		wantData  bool
	}{
		{name: "exactly at limit", dataSize: pushover.MessageMaxAttachmentByte, mimeType: "image/png", wantData: true},
		{name: "decoded bytes exceed limit", dataSize: pushover.MessageMaxAttachmentByte + 1, mimeType: "image/png", wantError: "decoded attachment_base64 exceeds"},
		{name: "encoded bytes exceed limit", b64: strings.Repeat("A", base64.StdEncoding.EncodedLen(pushover.MessageMaxAttachmentByte)+4), mimeType: "image/png", wantError: "maximum encoded size"},
		{name: "without attachment type", dataSize: 64, wantError: "required"},
		{name: "MIME mismatch", dataSize: 64, mimeType: "image/jpeg", wantError: "type mismatch"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			b64 := tc.b64
			var source []byte
			if tc.dataSize != 0 {
				source = pngData(t, tc.dataSize)
				b64 = base64.StdEncoding.EncodeToString(source)
			}

			data, err := loadAttachment(context.Background(), "", b64, tc.mimeType)
			if tc.wantError != "" {
				requireAttachmentError(t, err, tc.wantError)
				return
			}
			if err != nil {
				t.Fatalf("loadAttachment() error = %v", err)
			}
			if !tc.wantData || len(data) != pushover.MessageMaxAttachmentByte {
				t.Errorf("attachment length = %d, want %d", len(data), pushover.MessageMaxAttachmentByte)
			}
			if !bytes.Equal(data, source) {
				t.Error("attachment bytes differ from source")
			}
		})
	}
}

// TestLoadAttachmentPath verifies local attachment validation and transport restrictions.
func TestLoadAttachmentPath(t *testing.T) {
	testCases := []struct {
		name      string
		data      []byte
		mimeType  string
		ctx       context.Context
		wantError string
		wantData  bool
	}{
		{name: "non-image file", data: []byte("not an image"), ctx: context.Background(), wantError: "text/plain"},
		{name: "MIME mismatch", data: pngData(t, 64), mimeType: "image/jpeg", ctx: context.Background(), wantError: "type mismatch"},
		{name: "empty file", ctx: context.Background(), wantError: "empty"},
		{name: "HTTP transport refused", data: pngData(t, 64), ctx: context.WithValue(context.Background(), contextKeyHTTPMethod, http.MethodPost), wantError: "not allowed over HTTP"},
		{name: "stdio accepted", data: pngData(t, 64), ctx: context.Background(), wantData: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeAttachmentFile(t, tc.data)
			data, err := loadAttachment(tc.ctx, path, "", tc.mimeType)
			if tc.wantError != "" {
				requireAttachmentError(t, err, tc.wantError)
				return
			}
			if err != nil {
				t.Fatalf("loadAttachment() error = %v", err)
			}
			if !tc.wantData || !bytes.Equal(data, tc.data) {
				t.Error("attachment bytes differ from source")
			}
		})
	}
}

// TestLoadAttachmentInputValidation verifies mutually exclusive and incomplete attachment inputs.
func TestLoadAttachmentInputValidation(t *testing.T) {
	testCases := []struct {
		name      string
		path      string
		b64       string
		mimeType  string
		wantError string
	}{
		{name: "path and base64 together", path: "image.png", b64: base64.StdEncoding.EncodeToString(pngData(t, 64)), mimeType: "image/png", wantError: "mutually exclusive"},
		{name: "attachment type alone", mimeType: "image/png", wantError: "requires"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := loadAttachment(context.Background(), tc.path, tc.b64, tc.mimeType)
			requireAttachmentError(t, err, tc.wantError)
		})
	}

	data, err := loadAttachment(context.Background(), "", "", "")
	if err != nil {
		t.Fatalf("loadAttachment() error = %v", err)
	}
	if data != nil {
		t.Errorf("attachment = %v, want nil", data)
	}
}

// TestAttachmentBytesReachPushover verifies attachment bytes are sent unchanged in the multipart request.
func TestAttachmentBytesReachPushover(t *testing.T) {
	source := pngData(t, 64)
	path := writeAttachmentFile(t, source)
	data, err := loadAttachment(context.Background(), path, "", "image/png")
	if err != nil {
		t.Fatalf("loadAttachment() error = %v", err)
	}

	var recorded []byte
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(int64(pushover.MessageMaxAttachmentByte) + 1024); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		file, _, err := r.FormFile("attachment")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer func() { _ = file.Close() }()
		recorded, err = io.ReadAll(file)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Limit-App-Limit", "7500")
		w.Header().Set("X-Limit-App-Remaining", "6000")
		w.Header().Set("X-Limit-App-Reset", "1393653600")
		_, _ = w.Write([]byte(`{"status":1,"request":"x"}`))
	}))
	t.Cleanup(testServer.Close)

	originalEndpoint := pushover.APIEndpoint
	pushover.APIEndpoint = testServer.URL
	t.Cleanup(func() {
		pushover.APIEndpoint = originalEndpoint
	})

	message, err := CreateMessage("test", "", Config{}, &CLIArgs{Attachment: data})
	if err != nil {
		t.Fatalf("CreateMessage() error = %v", err)
	}
	app := pushover.New(strings.Repeat("a", 30))
	if _, err := app.SendMessage(message, pushover.NewRecipient(strings.Repeat("b", 30))); err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}
	if !bytes.Equal(recorded, source) {
		t.Error("wire attachment bytes differ from source")
	}
}

// TestHandleSendNotificationAttachment verifies MCP attachment handling for base64 and HTTP paths.
func TestHandleSendNotificationAttachment(t *testing.T) {
	t.Run("base64 attachment", func(t *testing.T) {
		mock := &MockPushoverClient{}
		swapNewPushoverClient(t, mock)
		request := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
			"message":           "test message",
			"attachment_base64": base64.StdEncoding.EncodeToString(pngData(t, 64)),
			"attachment_type":   "image/png",
		}}}

		result, err := handleSendNotification(context.Background(), request, newTestMCPConfig())
		if err != nil {
			t.Fatalf("handleSendNotification() error = %v", err)
		}
		if result.IsError {
			t.Fatalf("handleSendNotification() result = %v, want success", result)
		}
		if mock.lastMessage == nil {
			t.Fatal("mock did not receive a message")
		}
	})

	t.Run("HTTP path attachment refused", func(t *testing.T) {
		mock := &MockPushoverClient{}
		swapNewPushoverClient(t, mock)
		path := writeAttachmentFile(t, pngData(t, 64))
		request := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
			"message":         "test message",
			"attachment_path": path,
		}}}
		ctx := context.WithValue(context.Background(), contextKeyHTTPMethod, http.MethodPost)

		result, err := handleSendNotification(ctx, request, newTestMCPConfig())
		if err != nil {
			t.Fatalf("handleSendNotification() error = %v", err)
		}
		if !result.IsError {
			t.Fatalf("handleSendNotification() result = %v, want tool error", result)
		}
		if mock.lastMessage != nil {
			t.Fatalf("mock received a message: %#v", mock.lastMessage)
		}
	})
}
