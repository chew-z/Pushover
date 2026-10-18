package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"

	"github.com/gregdel/pushover"
)

// allowedAttachmentTypes lists the image MIME types accepted as attachments.
var allowedAttachmentTypes = []string{
	"image/png",
	"image/jpeg",
	"image/gif",
	"image/webp",
}

// loadAttachment loads and validates an image attachment from a local path or base64 data.
func loadAttachment(ctx context.Context, path, b64, mimeType string) ([]byte, error) {
	if path == "" && b64 == "" {
		if mimeType != "" {
			return nil, errors.New("attachment_type requires attachment_path or attachment_base64")
		}
		return nil, nil
	}
	if path != "" && b64 != "" {
		return nil, errors.New("attachment_path and attachment_base64 are mutually exclusive")
	}
	if mimeType != "" && !slices.Contains(allowedAttachmentTypes, mimeType) {
		return nil, fmt.Errorf("attachment_type must be one of: %v", allowedAttachmentTypes)
	}

	maxBytes := pushover.MessageMaxAttachmentByte
	var data []byte
	if path != "" {
		if isHTTPRequest(ctx) {
			return nil, errors.New("attachment_path is not allowed over HTTP; use attachment_base64")
		}

		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("stat attachment path: %w", err)
		}
		if !info.Mode().IsRegular() {
			return nil, errors.New("attachment_path must be a regular file")
		}
		if info.Size() == 0 {
			return nil, errors.New("attachment file is empty")
		}
		if info.Size() > int64(maxBytes) {
			return nil, fmt.Errorf("attachment file size %d exceeds maximum of %d bytes", info.Size(), maxBytes)
		}

		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("open attachment path: %w", err)
		}
		defer func() { _ = f.Close() }()

		data, err = io.ReadAll(io.LimitReader(f, int64(maxBytes)+1))
		if err != nil {
			return nil, fmt.Errorf("read attachment path: %w", err)
		}
		if len(data) > maxBytes {
			return nil, fmt.Errorf("attachment file read exceeds maximum of %d bytes", maxBytes)
		}
		if len(data) == 0 {
			return nil, errors.New("attachment file is empty")
		}
	} else {
		if mimeType == "" {
			return nil, errors.New("attachment_type is required with attachment_base64")
		}
		if len(b64) > base64.StdEncoding.EncodedLen(maxBytes) {
			return nil, fmt.Errorf("attachment_base64 exceeds maximum encoded size of %d bytes", base64.StdEncoding.EncodedLen(maxBytes))
		}

		decoded, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return nil, fmt.Errorf("decode attachment_base64: %w", err)
		}
		data = decoded
		if len(data) == 0 {
			return nil, errors.New("attachment data is empty")
		}
		if len(data) > maxBytes {
			return nil, fmt.Errorf("decoded attachment_base64 exceeds maximum of %d bytes", maxBytes)
		}
	}

	detected := http.DetectContentType(data)
	if !slices.Contains(allowedAttachmentTypes, detected) {
		return nil, fmt.Errorf("attachment is not a supported image: got %s", detected)
	}
	if mimeType != "" && detected != mimeType {
		return nil, fmt.Errorf("attachment type mismatch: detected %s, expected %s", detected, mimeType)
	}

	return data, nil
}
