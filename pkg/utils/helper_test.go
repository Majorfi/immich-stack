package utils

import "testing"

func TestBoolToString(t *testing.T) {
	tests := []struct {
		name     string
		input    bool
		expected string
	}{
		{
			name:     "true converts to 'true'",
			input:    true,
			expected: "true",
		},
		{
			name:     "false converts to 'false'",
			input:    false,
			expected: "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BoolToString(tt.input)
			if result != tt.expected {
				t.Errorf("BoolToString(%v) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}