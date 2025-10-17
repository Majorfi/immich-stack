package stacker

import (
	"testing"
	"time"

	"github.com/majorfi/immich-stack/pkg/utils"
)

func TestMergeTimeBasedGroups(t *testing.T) {
	// Test the merging of groups that should be together based on time proximity
	
	tests := []struct {
		name           string
		groups         map[string][]utils.TAsset
		criteria       []utils.TCriteria
		expectedGroups int
		description    string
	}{
		{
			name: "Merge groups within time delta",
			groups: map[string][]utils.TAsset{
				"prefix1|2025-10-10T13:14:03.000000000Z": {
					{ID: "1", OriginalFileName: "IMG_001.jpg", LocalDateTime: "2025-10-10T13:14:05.945Z"},
				},
				"prefix1|2025-10-10T13:14:06.000000000Z": {
					{ID: "2", OriginalFileName: "IMG_001.dng", LocalDateTime: "2025-10-10T13:14:06.328Z"},
				},
			},
			criteria: []utils.TCriteria{
				{Key: "originalFileName", Split: &utils.TSplit{Delimiters: []string{"."}, Index: 0}},
				{Key: "localDateTime", Delta: &utils.TDelta{Milliseconds: 3000}},
			},
			expectedGroups: 1, // Should merge into 1 group
			description:    "Groups with same prefix and times within delta should merge",
		},
		{
			name: "Keep separate groups with different prefixes",
			groups: map[string][]utils.TAsset{
				"prefix1|2025-10-10T13:14:03.000000000Z": {
					{ID: "1", OriginalFileName: "IMG_001.jpg", LocalDateTime: "2025-10-10T13:14:05.945Z"},
				},
				"prefix2|2025-10-10T13:14:03.000000000Z": {
					{ID: "2", OriginalFileName: "IMG_002.jpg", LocalDateTime: "2025-10-10T13:14:05.950Z"},
				},
			},
			criteria: []utils.TCriteria{
				{Key: "originalFileName", Split: &utils.TSplit{Delimiters: []string{"."}, Index: 0}},
				{Key: "localDateTime", Delta: &utils.TDelta{Milliseconds: 3000}},
			},
			expectedGroups: 2, // Should remain separate due to different prefixes
			description:    "Groups with different prefixes should not merge",
		},
		{
			name: "Merge multiple consecutive groups",
			groups: map[string][]utils.TAsset{
				"burst|2025-01-01T12:00:00.000000000Z": {
					{ID: "1", OriginalFileName: "burst.jpg", LocalDateTime: "2025-01-01T12:00:00.000Z"},
					{ID: "2", OriginalFileName: "burst.jpg", LocalDateTime: "2025-01-01T12:00:00.500Z"},
				},
				"burst|2025-01-01T12:00:01.000000000Z": {
					{ID: "3", OriginalFileName: "burst.jpg", LocalDateTime: "2025-01-01T12:00:01.000Z"},
					{ID: "4", OriginalFileName: "burst.jpg", LocalDateTime: "2025-01-01T12:00:01.500Z"},
				},
				"burst|2025-01-01T12:00:02.000000000Z": {
					{ID: "5", OriginalFileName: "burst.jpg", LocalDateTime: "2025-01-01T12:00:02.000Z"},
				},
			},
			criteria: []utils.TCriteria{
				{Key: "originalFileName"},
				{Key: "localDateTime", Delta: &utils.TDelta{Milliseconds: 1000}},
			},
			expectedGroups: 1, // All should merge into 1 group (consecutive photos within 1000ms)
			description:    "Multiple consecutive groups within delta should merge into one",
		},
		{
			name: "Keep groups with time gap larger than delta",
			groups: map[string][]utils.TAsset{
				"event|2025-01-01T12:00:00.000000000Z": {
					{ID: "1", OriginalFileName: "event.jpg", LocalDateTime: "2025-01-01T12:00:00.000Z"},
					{ID: "2", OriginalFileName: "event.jpg", LocalDateTime: "2025-01-01T12:00:00.500Z"},
				},
				"event|2025-01-01T12:00:06.000000000Z": {
					{ID: "3", OriginalFileName: "event.jpg", LocalDateTime: "2025-01-01T12:00:06.000Z"},
					{ID: "4", OriginalFileName: "event.jpg", LocalDateTime: "2025-01-01T12:00:06.500Z"},
				},
			},
			criteria: []utils.TCriteria{
				{Key: "originalFileName"},
				{Key: "localDateTime", Delta: &utils.TDelta{Milliseconds: 3000}},
			},
			expectedGroups: 2, // Should remain as 2 groups (6 second gap > 3 second delta)
			description:    "Groups with time gap larger than delta should remain separate",
		},
		{
			name: "No time criteria - no merging",
			groups: map[string][]utils.TAsset{
				"file1": {
					{ID: "1", OriginalFileName: "file1.jpg"},
				},
				"file2": {
					{ID: "2", OriginalFileName: "file2.jpg"},
				},
			},
			criteria: []utils.TCriteria{
				{Key: "originalFileName"},
			},
			expectedGroups: 2, // No time criteria, no merging
			description:    "Without time criteria, groups should not be merged",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mergedGroups, err := mergeTimeBasedGroups(tt.groups, tt.criteria)
			if err != nil {
				t.Fatalf("mergeTimeBasedGroups failed: %v", err)
			}
			
			// Count groups with more than 0 assets
			actualGroups := 0
			for _, group := range mergedGroups {
				if len(group) > 0 {
					actualGroups++
				}
			}
			
			if actualGroups != tt.expectedGroups {
				t.Errorf("%s\nExpected %d groups, got %d groups",
					tt.description, tt.expectedGroups, actualGroups)
				
				// Log the actual groups for debugging
				for key, assets := range mergedGroups {
					t.Logf("  Group %s:", key)
					for _, asset := range assets {
						t.Logf("    - %s (%s)", asset.OriginalFileName, asset.LocalDateTime)
					}
				}
			}
		})
	}
}

func TestPerformSlidingWindowGrouping(t *testing.T) {
	tests := []struct {
		name        string
		assets      []AssetWithTime
		deltaMs     int
		expectedGroups int
	}{
		{
			name: "Single asset",
			assets: []AssetWithTime{
				{Asset: utils.TAsset{ID: "1"}, ParsedTime: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)},
			},
			deltaMs: 1000,
			expectedGroups: 1,
		},
		{
			name: "Two assets within delta",
			assets: []AssetWithTime{
				{Asset: utils.TAsset{ID: "1"}, ParsedTime: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)},
				{Asset: utils.TAsset{ID: "2"}, ParsedTime: time.Date(2025, 1, 1, 12, 0, 0, 500000000, time.UTC)},
			},
			deltaMs: 1000,
			expectedGroups: 1,
		},
		{
			name: "Two assets outside delta",
			assets: []AssetWithTime{
				{Asset: utils.TAsset{ID: "1"}, ParsedTime: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)},
				{Asset: utils.TAsset{ID: "2"}, ParsedTime: time.Date(2025, 1, 1, 12, 0, 2, 0, time.UTC)},
			},
			deltaMs: 1000,
			expectedGroups: 2,
		},
		{
			name: "Chain of assets within delta",
			assets: []AssetWithTime{
				{Asset: utils.TAsset{ID: "1"}, ParsedTime: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)},
				{Asset: utils.TAsset{ID: "2"}, ParsedTime: time.Date(2025, 1, 1, 12, 0, 0, 900000000, time.UTC)},
				{Asset: utils.TAsset{ID: "3"}, ParsedTime: time.Date(2025, 1, 1, 12, 0, 1, 800000000, time.UTC)},
				{Asset: utils.TAsset{ID: "4"}, ParsedTime: time.Date(2025, 1, 1, 12, 0, 2, 700000000, time.UTC)},
			},
			deltaMs: 1000,
			expectedGroups: 1, // Each consecutive pair is within 1000ms
		},
		{
			name: "Two separate bursts",
			assets: []AssetWithTime{
				{Asset: utils.TAsset{ID: "1"}, ParsedTime: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)},
				{Asset: utils.TAsset{ID: "2"}, ParsedTime: time.Date(2025, 1, 1, 12, 0, 0, 500000000, time.UTC)},
				{Asset: utils.TAsset{ID: "3"}, ParsedTime: time.Date(2025, 1, 1, 12, 0, 5, 0, time.UTC)},
				{Asset: utils.TAsset{ID: "4"}, ParsedTime: time.Date(2025, 1, 1, 12, 0, 5, 500000000, time.UTC)},
			},
			deltaMs: 1000,
			expectedGroups: 2,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groups := performSlidingWindowGrouping(tt.assets, tt.deltaMs)
			
			if len(groups) != tt.expectedGroups {
				t.Errorf("Expected %d groups, got %d groups", tt.expectedGroups, len(groups))
				for i, group := range groups {
					t.Logf("  Group %d:", i)
					for _, awt := range group {
						t.Logf("    - Asset %s at %s", awt.Asset.ID, awt.ParsedTime.Format(time.RFC3339Nano))
					}
				}
			}
		})
	}
}