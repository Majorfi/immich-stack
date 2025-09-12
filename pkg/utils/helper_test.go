package utils

import (
	"reflect"
	"testing"
)

func TestAreArraysEqual(t *testing.T) {
	tests := []struct {
		name     string
		arr1     []string
		arr2     []string
		expected bool
	}{
		{
			name:     "identical arrays",
			arr1:     []string{"a", "b", "c"},
			arr2:     []string{"a", "b", "c"},
			expected: true,
		},
		{
			name:     "same elements different order",
			arr1:     []string{"c", "a", "b"},
			arr2:     []string{"a", "b", "c"},
			expected: true,
		},
		{
			name:     "different elements",
			arr1:     []string{"a", "b", "c"},
			arr2:     []string{"a", "b", "d"},
			expected: false,
		},
		{
			name:     "different lengths",
			arr1:     []string{"a", "b"},
			arr2:     []string{"a", "b", "c"},
			expected: false,
		},
		{
			name:     "both empty",
			arr1:     []string{},
			arr2:     []string{},
			expected: true,
		},
		{
			name:     "one empty one not",
			arr1:     []string{"a"},
			arr2:     []string{},
			expected: false,
		},
		{
			name:     "duplicate elements same frequency",
			arr1:     []string{"a", "b", "a"},
			arr2:     []string{"b", "a", "a"},
			expected: true,
		},
		{
			name:     "duplicate elements different frequency",
			arr1:     []string{"a", "b", "a"},
			arr2:     []string{"a", "b", "b"},
			expected: false,
		},
		{
			name:     "same elements but arr2 has extra",
			arr1:     []string{"a", "b"},
			arr2:     []string{"a", "b", "c"},
			expected: false,
		},
		{
			name:     "same elements but arr1 has extra",
			arr1:     []string{"a", "b", "c"},
			arr2:     []string{"a", "b"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AreArraysEqual(tt.arr1, tt.arr2)
			if result != tt.expected {
				t.Errorf("AreArraysEqual(%v, %v) = %v, expected %v", tt.arr1, tt.arr2, result, tt.expected)
			}
		})
	}
}

func TestRemoveEmptyStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "no empty strings",
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "some empty strings",
			input:    []string{"a", "", "b", "", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "all empty strings",
			input:    []string{"", "", ""},
			expected: []string{},
		},
		{
			name:     "empty array",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "single non-empty string",
			input:    []string{"a"},
			expected: []string{"a"},
		},
		{
			name:     "single empty string",
			input:    []string{""},
			expected: []string{},
		},
		{
			name:     "preserves order",
			input:    []string{"z", "", "a", "", "m"},
			expected: []string{"z", "a", "m"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RemoveEmptyStrings(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("RemoveEmptyStrings(%v) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		list     []string
		search   string
		expected bool
	}{
		{
			name:     "string found",
			list:     []string{"a", "b", "c"},
			search:   "b",
			expected: true,
		},
		{
			name:     "string not found",
			list:     []string{"a", "b", "c"},
			search:   "d",
			expected: false,
		},
		{
			name:     "empty list",
			list:     []string{},
			search:   "a",
			expected: false,
		},
		{
			name:     "search for empty string found",
			list:     []string{"a", "", "c"},
			search:   "",
			expected: true,
		},
		{
			name:     "search for empty string not found",
			list:     []string{"a", "b", "c"},
			search:   "",
			expected: false,
		},
		{
			name:     "duplicate entries",
			list:     []string{"a", "b", "b", "c"},
			search:   "b",
			expected: true,
		},
		{
			name:     "case sensitive",
			list:     []string{"A", "b", "c"},
			search:   "a",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Contains(tt.list, tt.search)
			if result != tt.expected {
				t.Errorf("Contains(%v, %q) = %v, expected %v", tt.list, tt.search, result, tt.expected)
			}
		})
	}
}

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