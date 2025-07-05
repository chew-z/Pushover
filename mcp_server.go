package main

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// setupMCPServer creates and configures the MCP server with tools
func setupMCPServer(config *MCPConfig) *server.MCPServer {
	// Create MCP server
	mcpServer := server.NewMCPServer(
		"pushover-mcp-server",
		version,
	)

	// Add the send_notification tool
	mcpServer.AddTool(
		mcp.NewTool("send_notification",
			mcp.WithDescription("Send a notification via Pushover"),
			mcp.WithString("message",
				mcp.Description("The notification message to send (required)"),
				mcp.Required(),
			),
			mcp.WithString("title",
				mcp.Description("Optional title for the notification"),
			),
			mcp.WithString("priority",
				mcp.Description("Priority level (-2=lowest, -1=low, 0=normal, 1=high, 2=emergency)"),
			),
			mcp.WithString("device",
				mcp.Description("Target device name"),
			),
			mcp.WithString("sound",
				mcp.Description("Notification sound name"),
			),
			mcp.WithString("expire",
				mcp.Description("Expiration time in seconds for emergency messages"),
			),
		),
		wrapWithAuth(handleSendNotification, "send_notification", config),
	)

	return mcpServer
}

// runMCPServerWithTransport starts the MCP server with the specified transport mode
func runMCPServerWithTransport(transport string, authEnabled bool) error {
	// Load MCP configuration
	config, err := NewMCPConfig(authEnabled)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create MCP server
	mcpServer := setupMCPServer(config)

	switch transport {
	case "stdio":
		log.Println("Starting MCP server with STDIO transport...")
		return server.ServeStdio(mcpServer)

	case "http":
		log.Printf("Starting MCP server with HTTP transport on %s%s...", config.HTTPAddress, config.HTTPPath)
		httpManager := NewHTTPServerManager(config)
		return httpManager.Start(mcpServer)

	default:
		return fmt.Errorf("unsupported transport: %s (supported: stdio, http)", transport)
	}
}

// wrapWithAuth wraps tool handlers with authentication and logging
func wrapWithAuth(
	handler func(context.Context, mcp.CallToolRequest, *MCPConfig) (*mcp.CallToolResult, error),
	toolName string,
	config *MCPConfig,
) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		log.Printf("Calling tool '%s'...", toolName)

		// Check authentication for HTTP requests
		if isHTTPRequest(ctx) && config.AuthEnabled {
			if !isAuthenticated(ctx) {
				authError := getAuthError(ctx)
				return mcp.NewToolResultError(fmt.Sprintf("Authentication required: %s", authError)), nil
			}
			log.Printf("Tool '%s' called by user: %s", toolName, getUsername(ctx))
		}

		// Call the actual handler
		result, err := handler(ctx, req, config)

		// Log result
		if err != nil {
			log.Printf("Tool '%s' failed: %v", toolName, err)
		} else {
			log.Printf("Tool '%s' completed successfully", toolName)
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

	// Validate message length
	if len(message) > 1024 {
		return mcp.NewToolResultError("Message too long (max 1024 characters)"), nil
	}

	// Extract optional parameters
	title := request.GetString("title", "")
	priorityStr := request.GetString("priority", "")
	device := request.GetString("device", "")
	sound := request.GetString("sound", "")
	expireStr := request.GetString("expire", "")

	// Parse priority
	priority := config.PushoverDefaultPriority
	if priorityStr != "" {
		if p, err := strconv.Atoi(priorityStr); err == nil {
			if p < -2 || p > 2 {
				return mcp.NewToolResultError("Priority must be between -2 and 2"), nil
			}
			priority = p
		} else {
			return mcp.NewToolResultError("Invalid priority value"), nil
		}
	}

	// Parse expire time
	expireTime := config.PushoverDefaultExpire
	if expireStr != "" {
		if e, err := strconv.Atoi(expireStr); err == nil {
			expireTime = e
		} else {
			return mcp.NewToolResultError("Invalid expire value"), nil
		}
	}

	// Create legacy config for compatibility
	legacyConfig := config.ToLegacyConfig()

	// Create Pushover client
	client, recipient, err := NewPushoverClient(legacyConfig)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create Pushover client: %v", err)), nil
	}

	// Create CLI args structure for message creation
	cliArgs := &CLIArgs{
		Message:    message,
		Title:      title,
		Priority:   priority,
		Sound:      sound,
		ExpireTime: expireTime,
		DeviceName: device,
	}

	// Create and send message
	pushMessage := CreateMessage(message, title, legacyConfig, cliArgs)

	response, err := client.SendMessage(pushMessage, recipient)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to send notification: %v", err)), nil
	}

	// Prepare response message
	responseText := "Notification sent successfully"
	if response != nil && response.Receipt != "" {
		responseText += fmt.Sprintf(". Receipt: %s", response.Receipt)
	}

	// Add user info if authenticated
	if isHTTPRequest(ctx) && isAuthenticated(ctx) {
		username := getUsername(ctx)
		if username != "" {
			responseText += fmt.Sprintf(" (sent by %s)", username)
		}
	}

	return mcp.NewToolResultText(responseText), nil
}
