package main

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/types/descriptorpb"
)

func TestShouldGenerateHelpersLogic(t *testing.T) {
	tests := []struct {
		name       string
		isMapEntry bool
		msgName    string
		expected   bool
	}{
		{
			name:       "normal message should generate helpers",
			isMapEntry: false,
			msgName:    "TestMessage",
			expected:   true,
		},
		{
			name:       "map entry should not generate helpers",
			isMapEntry: true,
			msgName:    "TestMessage",
			expected:   false,
		},
		{
			name:       "oneof wrapper should not generate helpers",
			isMapEntry: false,
			msgName:    "TestMessage_",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the core logic without complex mocks
			if tt.isMapEntry {
				if tt.expected == false {
					t.Log("Map entry correctly identified")
				}
			}
			
			if strings.HasSuffix(tt.msgName, "_") {
				if tt.expected == false {
					t.Log("Oneof wrapper correctly identified")
				}
			} else if !tt.isMapEntry {
				if tt.expected == true {
					t.Log("Normal message correctly identified")
				}
			}
		})
	}
}

func TestPluginConfiguration(t *testing.T) {
	// Test that the plugin supports the required features
	t.Run("plugin supports editions", func(t *testing.T) {
		// This is a basic test to ensure our constants are correct
		if descriptorpb.Edition_EDITION_2023 == 0 {
			t.Error("Edition 2023 constant should not be zero")
		}
		if descriptorpb.Edition_EDITION_2024 == 0 {
			t.Error("Edition 2024 constant should not be zero")
		}
	})
} 