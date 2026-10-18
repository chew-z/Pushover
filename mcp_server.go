package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"

	"github.com/gregdel/pushover"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// setupMCPServer creates and configures the MCP server with tools
func setupMCPServer(config *MCPConfig) *server.MCPServer {
	// Create MCP server
	mcpServer := server.NewMCPServer(
		"pushover-mcp-server",
		version,
		server.WithInputSchemaValidation(),
	)

	// Add the send_notification tool
	mcpServer.AddTool(
		mcp.NewTool("send_notification",
			mcp.WithDescription("Send a notification via Pushover"),
			mcp.WithString("message",
				mcp.Description("The notification message to send (required)"),
				mcp.Required(),
				mcp.MaxLength(1024),
			),
			mcp.WithString("title",
				mcp.Description("Optional title for the notification"),
			),
			mcp.WithString("attachment_path",
				mcp.Description("Local path to an image file; refused over HTTP transport, stdio only; mutually exclusive with attachment_base64"),
			),
			mcp.WithString("attachment_base64",
				mcp.Description("Standard base64 image data (max 2.5 MiB decoded); requires attachment_type; mutually exclusive with attachment_path"),
				mcp.MaxLength(base64.StdEncoding.EncodedLen(pushover.MessageMaxAttachmentByte)),
			),
			mcp.WithString("attachment_type",
				mcp.Description("Expected image MIME type; required with attachment_base64 and optional with attachment_path, where it must match the detected type"),
				mcp.Enum(allowedAttachmentTypes...),
			),
			mcp.WithString("url",
				mcp.Description("Supplementary URL shown with the notification (max 512 characters)"),
				mcp.MaxLength(512),
			),
			mcp.WithString("url_title",
				mcp.Description("Title for the supplementary URL (max 100 characters; requires url)"),
				mcp.MaxLength(100),
			),
			mcp.WithBoolean("html",
				mcp.Description("Render the message as HTML (mutually exclusive with monospace)"),
			),
			mcp.WithBoolean("monospace",
				mcp.Description("Render the message in a monospace font (mutually exclusive with html)"),
			),
			mcp.WithInteger("ttl",
				mcp.Description("Seconds until the notification is deleted from devices (min 1; ignored for priority 2 emergency messages)"),
				mcp.Min(1),
			),
			mcp.WithInteger("timestamp",
				mcp.Description("Unix timestamp in seconds to display instead of the time Pushover received the message (min 1)"),
				mcp.Min(1),
			),
			mcp.WithInteger("priority",
				mcp.Description("Priority level (-2=lowest, -1=low, 0=normal, 1=high, 2=emergency). Priority 2 requires both retry and expire."),
				mcp.Min(-2),
				mcp.Max(2),
			),
			mcp.WithString("device",
				mcp.Description("Target device name"),
			),
			mcp.WithString("sound",
				mcp.Description("Notification sound name"),
			),
			mcp.WithInteger("expire",
				mcp.Description("Seconds Pushover keeps retrying an emergency (priority 2) notification (1–10800). Priority 2 requires both retry and expire."),
				mcp.Min(1),
				mcp.Max(maxExpireSeconds),
			),
			mcp.WithInteger("retry",
				mcp.Description("Seconds between retries for an emergency (priority 2) notification (30–86400). Priority 2 requires both retry and expire."),
				mcp.Min(30),
				mcp.Max(86400),
			),
		),
		wrapWithLogging(handleSendNotification, "send_notification", config),
	)
	mcpServer.AddTool(
		mcp.NewTool("cancel_emergency",
			mcp.WithDescription("Cancel retries of an emergency (priority 2) notification by its receipt"),
			mcp.WithString("receipt",
				mcp.Description("Receipt returned when the emergency notification was sent (30 alphanumeric characters)"),
				mcp.Required(),
				mcp.Pattern(receiptPatternString),
				mcp.MinLength(30),
				mcp.MaxLength(30),
			),
		),
		wrapWithLogging(handleCancelEmergency, "cancel_emergency", config),
	)

	return mcpServer
}

// runMCPServerWithTransport starts the MCP server with the specified transport mode
func runMCPServerWithTransport(transport string, authEnabled bool) error {
	setupServerLogging(transport)

	// Load MCP configuration
	config, err := NewMCPConfig(authEnabled)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create MCP server
	mcpServer := setupMCPServer(config)

	switch transport {
	case "stdio":
		slog.Info("starting MCP server")
		return server.ServeStdio(mcpServer)

	case "http":
		slog.Info("starting MCP server", "address", config.HTTPAddress, "path", config.HTTPPath)
		httpManager := NewHTTPServerManager(config)
		return httpManager.Start(mcpServer)

	default:
		return fmt.Errorf("unsupported transport: %s (supported: stdio, http)", transport)
	}
}

// wrapWithLogging wraps tool handlers with request logging.
func wrapWithLogging(
	handler func(context.Context, mcp.CallToolRequest, *MCPConfig) (*mcp.CallToolResult, error),
	toolName string,
	config *MCPConfig,
) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.Debug("tool call started", "tool", toolName)

		if actor, ok := actorFrom(ctx); ok {
			slog.Info("tool call actor", "tool", toolName, "actor", actor)
		}

		// Call the actual handler
		result, err := handler(ctx, req, config)

		// Log result
		if err != nil {
			slog.Error("tool call failed", "tool", toolName, "error", err)
		} else {
			slog.Info("tool call completed", "tool", toolName)
		}

		return result, err
	}
}

// handleSendNotification handles the send_notification tool
func handleSendNotification(ctx context.Context, request mcp.CallToolRequest, config *MCPConfig) (*mcp.CallToolResult, error) {
	// Extract required message parameter
	message, err := request.RequireString("message")
	if err != nil {
		return mcp.NewToolResultError("Message parameter is required"), nil
	}

	// Extract optional parameters
	arguments := request.GetArguments()
	title := request.GetString("title", "")
	url := request.GetString("url", "")
	urlTitle := request.GetString("url_title", "")
	html := request.GetBool("html", false)
	monospace := request.GetBool("monospace", false)
	priority := 0
	prioritySet := false
	if _, ok := arguments["priority"]; ok {
		priority = request.GetInt("priority", 0)
		prioritySet = true
	}
	device := request.GetString("device", "")
	sound := request.GetString("sound", "")
	attachmentPath := request.GetString("attachment_path", "")
	attachmentBase64 := request.GetString("attachment_base64", "")
	attachmentType := request.GetString("attachment_type", "")
	expireTime := request.GetInt("expire", config.PushoverDefaultExpire)
	retryTime := request.GetInt("retry", config.PushoverDefaultRetry)

	// Create legacy config for compatibility
	legacyConfig := config.ToLegacyConfig()

	// Create CLI args structure for message creation
	cliArgs := &CLIArgs{
		Message:     message,
		Title:       title,
		URL:         url,
		URLTitle:    urlTitle,
		HTML:        html,
		Monospace:   monospace,
		Priority:    priority,
		PrioritySet: prioritySet,
		Sound:       sound,
		ExpireTime:  expireTime,
		RetryTime:   retryTime,
		DeviceName:  device,
	}
	if _, ok := arguments["ttl"]; ok {
		cliArgs.TTL = request.GetInt("ttl", 0)
	}
	if _, ok := arguments["timestamp"]; ok {
		cliArgs.Timestamp = int64(request.GetInt("timestamp", 0))
	}
	attachment, err := loadAttachment(ctx, attachmentPath, attachmentBase64, attachmentType)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid attachment: %v", err)), nil
	}
	cliArgs.Attachment = attachment

	// Create Pushover client
	client, recipient, err := newPushoverClient(legacyConfig)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create Pushover client: %v", err)), nil
	}

	// Create and send message
	pushMessage, err := CreateMessage(message, title, legacyConfig, cliArgs)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid notification: %v", err)), nil
	}

	response, err := client.SendMessage(pushMessage, recipient)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to send notification: %v", err)), nil
	}

	actor, _ := actorFrom(ctx)
	return mcp.NewToolResultText(formatSendResult(response, actor)), nil
}

// handleCancelEmergency handles the cancel_emergency tool.
func handleCancelEmergency(ctx context.Context, request mcp.CallToolRequest, config *MCPConfig) (*mcp.CallToolResult, error) {
	receipt, err := request.RequireString("receipt")
	if err != nil {
		return mcp.NewToolResultError("Receipt parameter is required"), nil
	}
	if err := validateReceipt(receipt); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid receipt: %v", err)), nil
	}

	client, _, err := newPushoverClient(config.ToLegacyConfig())
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create Pushover client: %v", err)), nil
	}
	if _, err := client.CancelEmergencyNotification(receipt); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to cancel emergency notification: %v", err)), nil
	}

	actor, _ := actorFrom(ctx)
	return mcp.NewToolResultText(formatCancelResult(receipt, actor)), nil
}

// formatCancelResult formats a successful emergency notification cancellation result.
func formatCancelResult(receipt, actor string) string {
	result := "Emergency notification " + receipt + " canceled"
	if actor != "" {
		result += " (sent by " + actor + ")"
	}
	return result
}
