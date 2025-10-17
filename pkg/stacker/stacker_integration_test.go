package stacker

import (
	"testing"
	
	"github.com/majorfi/immich-stack/pkg/utils"
	"github.com/sirupsen/logrus"
)

func TestStackByWithTimeDeltas(t *testing.T) {
	// Create a test logger
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Only show errors in tests
	
	tests := []struct {
		name           string
		assets         []utils.TAsset
		criteria       string
		expectedStacks int
		description    string
	}{
		{
			name: "Issue #40 - Photos 383ms apart with 3000ms delta",
			assets: []utils.TAsset{
				{
					ID:               "1",
					OriginalFileName: "IMG_001.jpg",
					LocalDateTime:    "2025-10-10T13:14:05.945Z",
				},
				{
					ID:               "2",
					OriginalFileName: "IMG_001.dng",
					LocalDateTime:    "2025-10-10T13:14:06.328Z",
				},
			},
			criteria: `[{"key": "originalFileName", "split": {"delimiters": ["~", "."], "index": 0}}, {"key": "localDateTime", "delta": {"milliseconds": 3000}}]`,
			expectedStacks: 1,
			description:    "Photos with same filename prefix and 383ms apart should stack with 3000ms delta",
		},
		{
			name: "Burst sequence - all should group",
			assets: []utils.TAsset{
				{
					ID:               "1",
					OriginalFileName: "burst_001.jpg",
					LocalDateTime:    "2025-01-01T12:00:00.000Z",
				},
				{
					ID:               "2",
					OriginalFileName: "burst_001.jpg",
					LocalDateTime:    "2025-01-01T12:00:00.500Z",
				},
				{
					ID:               "3",
					OriginalFileName: "burst_001.jpg",
					LocalDateTime:    "2025-01-01T12:00:01.000Z",
				},
				{
					ID:               "4",
					OriginalFileName: "burst_001.jpg",
					LocalDateTime:    "2025-01-01T12:00:01.500Z",
				},
				{
					ID:               "5",
					OriginalFileName: "burst_001.jpg",
					LocalDateTime:    "2025-01-01T12:00:02.000Z",
				},
			},
			criteria: `[{"key": "originalFileName"}, {"key": "localDateTime", "delta": {"milliseconds": 1000}}]`,
			expectedStacks: 1,
			description:    "Burst photos with consecutive gaps ≤1000ms should form one stack",
		},
		{
			name: "Two distinct bursts with gap",
			assets: []utils.TAsset{
				{
					ID:               "1",
					OriginalFileName: "event.jpg",
					LocalDateTime:    "2025-01-01T12:00:00.000Z",
				},
				{
					ID:               "2", 
					OriginalFileName: "event.jpg",
					LocalDateTime:    "2025-01-01T12:00:00.500Z",
				},
				{
					ID:               "3",
					OriginalFileName: "event.jpg",
					LocalDateTime:    "2025-01-01T12:00:05.000Z", // 4.5 second gap
				},
				{
					ID:               "4",
					OriginalFileName: "event.jpg",
					LocalDateTime:    "2025-01-01T12:00:05.500Z",
				},
			},
			criteria: `[{"key": "originalFileName"}, {"key": "localDateTime", "delta": {"milliseconds": 1000}}]`,
			expectedStacks: 2,
			description:    "Photos with >1000ms gap should form separate stacks",
		},
		{
			name: "Edge case at bucket boundary",
			assets: []utils.TAsset{
				{
					ID:               "1",
					OriginalFileName: "edge.jpg",
					LocalDateTime:    "2025-01-01T00:00:02.999Z",
				},
				{
					ID:               "2",
					OriginalFileName: "edge.jpg", 
					LocalDateTime:    "2025-01-01T00:00:03.000Z", // 1ms apart at 3-second boundary
				},
			},
			criteria: `[{"key": "originalFileName"}, {"key": "localDateTime", "delta": {"milliseconds": 3000}}]`,
			expectedStacks: 1,
			description:    "Photos 1ms apart at bucket boundary should still stack",
		},
		{
			name: "Different filename prefixes",
			assets: []utils.TAsset{
				{
					ID:               "1",
					OriginalFileName: "IMG_001.jpg",
					LocalDateTime:    "2025-01-01T12:00:00.000Z",
				},
				{
					ID:               "2",
					OriginalFileName: "IMG_002.jpg",
					LocalDateTime:    "2025-01-01T12:00:00.100Z",
				},
				{
					ID:               "3",
					OriginalFileName: "IMG_001.dng",
					LocalDateTime:    "2025-01-01T12:00:00.200Z",
				},
			},
			criteria: `[{"key": "originalFileName", "split": {"delimiters": ["_", "."], "index": 1}}, {"key": "localDateTime", "delta": {"milliseconds": 1000}}]`,
			expectedStacks: 1, // IMG_001.jpg and IMG_001.dng should stack (same "001" after split)
			description:    "Files with same split prefix should stack",
		},
		{
			name: "No time delta - exact time matching only",
			assets: []utils.TAsset{
				{
					ID:               "1",
					OriginalFileName: "photo.jpg",
					LocalDateTime:    "2025-01-01T12:00:00.000Z",
				},
				{
					ID:               "2",
					OriginalFileName: "photo.jpg",
					LocalDateTime:    "2025-01-01T12:00:00.001Z", // 1ms apart
				},
			},
			criteria: `[{"key": "originalFileName"}, {"key": "localDateTime"}]`, // No delta specified
			expectedStacks: 0, // Without delta, times must match exactly
			description:    "Without delta, different times don't stack",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stacks, err := StackBy(tt.assets, tt.criteria, "", "", logger)
			if err != nil {
				t.Fatalf("StackBy failed: %v", err)
			}
			
			if len(stacks) != tt.expectedStacks {
				t.Errorf("%s\nExpected %d stacks, got %d stacks",
					tt.description, tt.expectedStacks, len(stacks))
				
				// Log the actual stacks for debugging
				for i, stack := range stacks {
					t.Logf("  Stack %d:", i+1)
					for _, asset := range stack {
						t.Logf("    - %s (%s) at %s", asset.ID, asset.OriginalFileName, asset.LocalDateTime)
					}
				}
			}
		})
	}
}

func TestStackByRAWJPEGPairs(t *testing.T) {
	// Test specifically for RAW+JPEG pairs from the same camera burst
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	assets := []utils.TAsset{
		{
			ID:               "1",
			OriginalFileName: "DSC_0001.NEF", // Nikon RAW
			LocalDateTime:    "2025-01-01T12:00:00.000Z",
		},
		{
			ID:               "2",
			OriginalFileName: "DSC_0001.JPG", // Matching JPEG
			LocalDateTime:    "2025-01-01T12:00:00.100Z", // 100ms later
		},
		{
			ID:               "3",
			OriginalFileName: "DSC_0002.NEF",
			LocalDateTime:    "2025-01-01T12:00:02.000Z", // Different shot
		},
		{
			ID:               "4",
			OriginalFileName: "DSC_0002.JPG",
			LocalDateTime:    "2025-01-01T12:00:02.100Z",
		},
	}
	
	// Using typical RAW+JPEG criteria
	criteria := `[{"key": "originalFileName", "split": {"delimiters": ["."], "index": 0}}, {"key": "localDateTime", "delta": {"milliseconds": 5000}}]`
	
	stacks, err := StackBy(assets, criteria, "", "", logger)
	if err != nil {
		t.Fatalf("StackBy failed: %v", err)
	}
	
	expectedStacks := 2 // DSC_0001 pair and DSC_0002 pair
	if len(stacks) != expectedStacks {
		t.Errorf("Expected %d stacks for RAW+JPEG pairs, got %d stacks", expectedStacks, len(stacks))
		for i, stack := range stacks {
			t.Logf("  Stack %d:", i+1)
			for _, asset := range stack {
				t.Logf("    - %s at %s", asset.OriginalFileName, asset.LocalDateTime)
			}
		}
	}
	
	// Verify each stack has both RAW and JPEG
	for i, stack := range stacks {
		if len(stack) != 2 {
			t.Errorf("Stack %d should have 2 files (RAW+JPEG), got %d", i+1, len(stack))
		}
	}
}