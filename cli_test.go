package main

import "testing"

// TestRunPriorityOverride verifies that explicit CLI priority zero overrides the environment default.
func TestRunPriorityOverride(t *testing.T) {
	t.Setenv("PUSHOVER_PRIORITY", "-1")

	testCases := []struct {
		name string
		args []string
		want int
	}{
		{"long priority flag", []string{"pushover", "-priority", "0", "-m", "hi"}, 0},
		{"short priority flag", []string{"pushover", "-p", "0", "-m", "hi"}, 0},
		{"no priority flag", []string{"pushover", "-m", "hi"}, -1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &MockPushoverClient{}
			if err := Run(tc.args, mock); err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if mock.lastMessage == nil {
				t.Fatal("Run() did not send a message")
			}
			if mock.lastMessage.Priority != tc.want {
				t.Errorf("priority = %d, want %d", mock.lastMessage.Priority, tc.want)
			}
		})
	}
}

// TestParseArgsPrioritySet verifies that ParseArgs retains priority flag presence independently of its value.
func TestParseArgsPrioritySet(t *testing.T) {
	testCases := []struct {
		name            string
		args            []string
		wantPriority    int
		wantPrioritySet bool
	}{
		{"long priority flag", []string{"pushover", "-priority", "0", "-m", "hi"}, 0, true},
		{"short priority flag", []string{"pushover", "-p", "0", "-m", "hi"}, 0, true},
		{"no priority flag", []string{"pushover", "-m", "hi"}, 0, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args, err := ParseArgs(tc.args)
			if err != nil {
				t.Fatalf("ParseArgs() error = %v", err)
			}
			if args.Priority != tc.wantPriority {
				t.Errorf("Priority = %d, want %d", args.Priority, tc.wantPriority)
			}
			if args.PrioritySet != tc.wantPrioritySet {
				t.Errorf("PrioritySet = %t, want %t", args.PrioritySet, tc.wantPrioritySet)
			}
		})
	}
}
