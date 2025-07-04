package main

import (
	"errors"
	"testing"

	"github.com/gregdel/pushover"
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
