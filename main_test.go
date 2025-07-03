package main

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/gregdel/pushover"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockPushoverClient is a mock implementation of the PushoverClient interface
type MockPushoverClient struct {
	mock.Mock
}

// SendMessage mocks the SendMessage method
func (m *MockPushoverClient) SendMessage(message *pushover.Message, recipient *pushover.Recipient) (*pushover.Response, error) {
	args := m.Called(message, recipient)
	return args.Get(0).(*pushover.Response), args.Error(1)
}

func TestLoadConfig(t *testing.T) {
	// Save original env vars to restore later
	originalAppKey := os.Getenv("APP_KEY")
	originalRecipientKey := os.Getenv("RECIPIENT_KEY")
	originalDevice := os.Getenv("DEVICE_NAME")

	// Cleanup after test
	defer func() {
		os.Setenv("APP_KEY", originalAppKey)
		os.Setenv("RECIPIENT_KEY", originalRecipientKey)
		os.Setenv("DEVICE_NAME", originalDevice)
	}()

	// Test with environment variables set
	os.Setenv("APP_KEY", "test_app_key")
	os.Setenv("RECIPIENT_KEY", "test_recipient_key")
	os.Setenv("DEVICE_NAME", "test_device")

	config := LoadConfig()
	assert.Equal(t, "test_app_key", config.AppKey)
	assert.Equal(t, "test_recipient_key", config.RecipientKey)
	assert.Equal(t, "test_device", config.DeviceName)

	// Test with empty environment variables
	os.Setenv("APP_KEY", "")
	os.Setenv("RECIPIENT_KEY", "")
	os.Setenv("DEVICE_NAME", "")

	config = LoadConfig()
	assert.Equal(t, "", config.AppKey)
	assert.Equal(t, "", config.RecipientKey)
	assert.Equal(t, "", config.DeviceName)
}

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectedMsg   string
		expectedTitle string
		expectError   bool
	}{
		{
			name:          "Message without title",
			args:          []string{"push", "Test message"},
			expectedMsg:   "Test message",
			expectedTitle: "",
			expectError:   false,
		},
		{
			name:          "Message with title",
			args:          []string{"push", "Test message", "Test title"},
			expectedMsg:   "Test message",
			expectedTitle: "Test title",
			expectError:   false,
		},
		{
			name:        "No message provided",
			args:        []string{"push"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, title, err := ParseArgs(tt.args)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedMsg, msg)
				assert.Equal(t, tt.expectedTitle, title)
			}
		})
	}
}

func TestNewPushoverClient(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		expectNil bool
	}{
		{
			name: "With app key and recipient key",
			config: Config{
				AppKey:       "test_app_key",
				RecipientKey: "test_recipient_key",
				DeviceName:   "test_device",
			},
		},
		{
			name: "Without app key",
			config: Config{
				AppKey:       "",
				RecipientKey: "test_recipient_key",
				DeviceName:   "test_device",
			},
			expectNil: true,
		},
		{
			name: "Without recipient key",
			config: Config{
				AppKey:       "test_app_key",
				RecipientKey: "",
				DeviceName:   "test_device",
			},
			expectNil: true,
		},
		{
			name: "Without app key and recipient key",
			config: Config{
				AppKey:       "",
				RecipientKey: "",
				DeviceName:   "test_device",
			},
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, recipient := NewPushoverClient(tt.config)
			if tt.expectNil {
				assert.Nil(t, client)
				assert.Nil(t, recipient)
			} else {
				assert.NotNil(t, client)
				assert.NotNil(t, recipient)
			}
		})
	}
}

func TestCreateMessage(t *testing.T) {
	config := Config{
		DeviceName: "test_device",
	}

	message := CreateMessage("Test message", "Test title", config)

	assert.Equal(t, "Test message", message.Message)
	assert.Equal(t, "Test title", message.Title)
	assert.Equal(t, pushover.PriorityLow, message.Priority)
	assert.Equal(t, "test_device", message.DeviceName)
	assert.Equal(t, pushover.SoundVibrate, message.Sound)
	assert.True(t, time.Now().Unix()-message.Timestamp < 5) // Should be recent
	assert.Equal(t, time.Duration(180*time.Second), message.Expire)
}

func TestSendNotification(t *testing.T) {
	mockClient := new(MockPushoverClient)
	message := pushover.NewMessage("Test message")
	recipient := pushover.NewRecipient("test_recipient_key")

	// Success case
	mockClient.On("SendMessage", message, recipient).Return(&pushover.Response{}, nil).Once()
	err := SendNotification(mockClient, message, recipient)
	assert.NoError(t, err)

	// Error case
	mockErr := errors.New("invalid user key")
	mockClient.On("SendMessage", message, recipient).Return(&pushover.Response{}, mockErr).Once()
	err = SendNotification(mockClient, message, recipient)
	assert.Error(t, err)

	mockClient.AssertExpectations(t)
}

// Integration test that combines multiple components
func TestIntegrationFlow(t *testing.T) {
	// Setup
	mockClient := new(MockPushoverClient)
	config := Config{
		AppKey:       "test_app_key",
		RecipientKey: "test_recipient_key",
		DeviceName:   "test_device",
	}

	// Parse arguments
	msg, title, err := ParseArgs([]string{"push", "Test message", "Test title"})
	assert.NoError(t, err)

	// Create message
	message := CreateMessage(msg, title, config)

	// Setup mock client expectation
		_, recipient := NewPushoverClient(config)
	mockClient.On("SendMessage", mock.MatchedBy(func(m *pushover.Message) bool {
		return m.Message == "Test message" && m.Title == "Test title"
	}), recipient).Return(&pushover.Response{}, nil)

	// Send notification
	err = SendNotification(mockClient, message, recipient)
	assert.NoError(t, err)

	mockClient.AssertExpectations(t)
}
