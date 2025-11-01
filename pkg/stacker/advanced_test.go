package stacker

import (
	"strings"
	"testing"

	"github.com/majorfi/immich-stack/pkg/utils"
	"github.com/sirupsen/logrus"
)

func TestBuildExpressionGroupingKey(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	tests := []struct {
		name        string
		asset       utils.TAsset
		expression  *utils.TCriteriaExpression
		criteria    []utils.TCriteria
		expectedKey string
		expectError bool
	}{
		{
			name: "Simple AND expression with filename and time",
			asset: utils.TAsset{
				ID:               "asset1",
				OriginalFileName: "PXL_20230101_120000.jpg",
				LocalDateTime:    "2023-01-01T12:00:00Z",
			},
			expression: &utils.TCriteriaExpression{
				Operator: &[]string{"AND"}[0],
				Children: []utils.TCriteriaExpression{
					{
						Criteria: &utils.TCriteria{
							Key:   "originalFileName",
							Regex: &utils.TRegex{Key: "^PXL_", Index: 0},
						},
					},
					{
						Criteria: &utils.TCriteria{
							Key:   "localDateTime",
							Delta: &utils.TDelta{Milliseconds: 1000},
						},
					},
				},
			},
			criteria: []utils.TCriteria{
				{Key: "originalFileName", Regex: &utils.TRegex{Key: "^PXL_", Index: 0}},
				{Key: "localDateTime", Delta: &utils.TDelta{Milliseconds: 1000}},
			},
			expectedKey: "originalFileName=PXL_|localDateTime=2023-01-01T12:00:00.000000000Z",
		},
		{
			name: "OR expression - only first matching branch contributes to key",
			asset: utils.TAsset{
				ID:               "asset2",
				OriginalFileName: "IMG_001.jpg",
				LocalDateTime:    "2023-01-01T12:00:00Z",
			},
			expression: &utils.TCriteriaExpression{
				Operator: &[]string{"OR"}[0],
				Children: []utils.TCriteriaExpression{
					{
						Criteria: &utils.TCriteria{
							Key:   "originalFileName",
							Regex: &utils.TRegex{Key: "^IMG_", Index: 0},
						},
					},
					{
						Criteria: &utils.TCriteria{
							Key:   "originalFileName",
							Regex: &utils.TRegex{Key: "^DSC_", Index: 0},
						},
					},
				},
			},
			criteria: []utils.TCriteria{
				{Key: "originalFileName", Regex: &utils.TRegex{Key: "^IMG_", Index: 0}},
				{Key: "originalFileName", Regex: &utils.TRegex{Key: "^DSC_", Index: 0}},
			},
			expectedKey: "originalFileName=IMG_",
		},
		{
			name: "NOT expression - assets match but contribute no values",
			asset: utils.TAsset{
				ID:               "asset3",
				OriginalFileName: "normal_photo.jpg",
				LocalDateTime:    "2023-01-01T12:00:00Z",
				IsArchived:       false,
			},
			expression: &utils.TCriteriaExpression{
				Operator: &[]string{"AND"}[0],
				Children: []utils.TCriteriaExpression{
					{
						Criteria: &utils.TCriteria{
							Key:   "originalFileName",
							Regex: &utils.TRegex{Key: "normal", Index: 0},
						},
					},
					{
						Operator: &[]string{"NOT"}[0],
						Children: []utils.TCriteriaExpression{
							{
								Criteria: &utils.TCriteria{Key: "isArchived"},
							},
						},
					},
				},
			},
			criteria: []utils.TCriteria{
				{Key: "originalFileName", Regex: &utils.TRegex{Key: "normal", Index: 0}},
				{Key: "isArchived"},
			},
			expectedKey: "originalFileName=normal", // NOT contributes no values
		},
		{
			name: "Asset doesn't match expression",
			asset: utils.TAsset{
				ID:               "asset4",
				OriginalFileName: "DSC_001.jpg",
				LocalDateTime:    "2023-01-01T12:00:00Z",
			},
			expression: &utils.TCriteriaExpression{
				Criteria: &utils.TCriteria{
					Key:   "originalFileName",
					Regex: &utils.TRegex{Key: "^PXL_", Index: 0},
				},
			},
			criteria: []utils.TCriteria{
				{Key: "originalFileName", Regex: &utils.TRegex{Key: "^PXL_", Index: 0}},
			},
			expectedKey: "", // No match, empty key
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := buildExpressionGroupingKey(tt.asset, tt.expression, tt.criteria)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if key != tt.expectedKey {
				t.Errorf("Expected key %q, got %q", tt.expectedKey, key)
			}
		})
	}
}

func TestStackByAdvanced(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	tests := []struct {
		name           string
		assets         []utils.TAsset
		config         CriteriaConfig
		expectedStacks int
		expectError    bool
	}{
		{
			name: "Expression creates multiple stacks by filename pattern",
			assets: []utils.TAsset{
				{ID: "1", OriginalFileName: "PXL_20230101_001.jpg", LocalDateTime: "2023-01-01T12:00:00Z"},
				{ID: "2", OriginalFileName: "PXL_20230101_002.jpg", LocalDateTime: "2023-01-01T12:00:00Z"}, // Same group
				{ID: "3", OriginalFileName: "IMG_20230201_001.jpg", LocalDateTime: "2023-02-01T12:00:00Z"},
				{ID: "4", OriginalFileName: "IMG_20230201_002.jpg", LocalDateTime: "2023-02-01T12:00:00Z"}, // Different group
				{ID: "5", OriginalFileName: "DSC_001.jpg", LocalDateTime: "2023-03-01T12:00:00Z"},          // No match
			},
			config: CriteriaConfig{
				Mode: "advanced",
				Expression: &utils.TCriteriaExpression{
					Operator: &[]string{"AND"}[0],
					Children: []utils.TCriteriaExpression{
						{
							Operator: &[]string{"OR"}[0],
							Children: []utils.TCriteriaExpression{
								{Criteria: &utils.TCriteria{Key: "originalFileName", Regex: &utils.TRegex{Key: "^PXL_", Index: 0}}},
								{Criteria: &utils.TCriteria{Key: "originalFileName", Regex: &utils.TRegex{Key: "^IMG_", Index: 0}}},
							},
						},
						{
							Criteria: &utils.TCriteria{Key: "localDateTime", Delta: &utils.TDelta{Milliseconds: 1000}},
						},
					},
				},
			},
			expectedStacks: 2, // PXL group and IMG group
		},
		{
			name: "Expression with time grouping creates stacks by time windows",
			assets: []utils.TAsset{
				{ID: "1", OriginalFileName: "photo1.jpg", LocalDateTime: "2023-01-01T12:00:00Z"},
				{ID: "2", OriginalFileName: "photo2.jpg", LocalDateTime: "2023-01-01T12:00:01Z"}, // Same time window
				{ID: "3", OriginalFileName: "photo3.jpg", LocalDateTime: "2023-01-01T12:05:00Z"}, // Different time window
				{ID: "4", OriginalFileName: "photo4.jpg", LocalDateTime: "2023-01-01T12:05:01Z"}, // Same window as photo3
			},
			config: CriteriaConfig{
				Mode: "advanced",
				Expression: &utils.TCriteriaExpression{
					Criteria: &utils.TCriteria{Key: "localDateTime", Delta: &utils.TDelta{Milliseconds: 2000}},
				},
			},
			expectedStacks: 2, // Two time windows
		},
		{
			name: "Single matching asset creates no stacks",
			assets: []utils.TAsset{
				{ID: "1", OriginalFileName: "PXL_001.jpg"},
				{ID: "2", OriginalFileName: "DSC_001.jpg"}, // Doesn't match expression
			},
			config: CriteriaConfig{
				Mode: "advanced",
				Expression: &utils.TCriteriaExpression{
					Criteria: &utils.TCriteria{Key: "originalFileName", Regex: &utils.TRegex{Key: "^PXL_", Index: 0}},
				},
			},
			expectedStacks: 0, // Only one asset matches, need 2+ for a stack
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stacks, err := stackByAdvanced(tt.assets, tt.config, "", "", logger)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if len(stacks) != tt.expectedStacks {
				t.Errorf("Expected %d stacks, got %d", tt.expectedStacks, len(stacks))
			}

			// Verify each stack has 2+ assets
			for i, stack := range stacks {
				if len(stack) < 2 {
					t.Errorf("Stack %d has only %d assets, expected 2+", i, len(stack))
				}
			}
		})
	}
}

func TestStackByLegacyGroups_UnionSemantics(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	tests := []struct {
		name           string
		assets         []utils.TAsset
		groups         []utils.TCriteriaGroup
		expectedStacks int
		expectError    bool
	}{
		{
			name: "OR group with union semantics connects assets sharing any key",
			assets: []utils.TAsset{
				// Group 1: Share folder /photos/2023/
				{ID: "1", OriginalFileName: "img1.jpg", OriginalPath: "/photos/2023/img1.jpg"},
				{ID: "2", OriginalFileName: "img2.jpg", OriginalPath: "/photos/2023/img2.jpg"},
				// Group 2: Share time but different folder - should connect to Group 1 via img3
				{ID: "3", OriginalFileName: "img3.jpg", OriginalPath: "/photos/2023/img3.jpg", LocalDateTime: "2023-01-01T12:00:00Z"},
				{ID: "4", OriginalFileName: "img4.jpg", OriginalPath: "/photos/2024/img4.jpg", LocalDateTime: "2023-01-01T12:00:00Z"},
				// Isolated asset
				{ID: "5", OriginalFileName: "img5.jpg", OriginalPath: "/other/img5.jpg", LocalDateTime: "2022-01-01T12:00:00Z"},
			},
			groups: []utils.TCriteriaGroup{
				{
					Operator: "OR",
					Criteria: []utils.TCriteria{
						{Key: "originalPath", Split: &utils.TSplit{Delimiters: []string{"/"}, Index: 2}}, // Extract folder
						{Key: "localDateTime", Delta: &utils.TDelta{Milliseconds: 1000}},                 // Time grouping
					},
				},
			},
			expectedStacks: 1, // All assets 1,2,3,4 should be connected via the OR union semantics
		},
		{
			name: "Multiple OR groups create separate connected components",
			assets: []utils.TAsset{
				{ID: "1", OriginalFileName: "PXL_001.jpg", OriginalPath: "/photos/pxl1.jpg"},
				{ID: "2", OriginalFileName: "PXL_002.jpg", OriginalPath: "/photos/pxl2.jpg"}, // Connected to 1
				{ID: "3", OriginalFileName: "IMG_001.jpg", OriginalPath: "/images/img1.jpg"},
				{ID: "4", OriginalFileName: "IMG_002.jpg", OriginalPath: "/images/img2.jpg"}, // Connected to 3
			},
			groups: []utils.TCriteriaGroup{
				{
					Operator: "OR",
					Criteria: []utils.TCriteria{
						{Key: "originalFileName", Regex: &utils.TRegex{Key: "^PXL_", Index: 0}},
						{Key: "originalPath", Split: &utils.TSplit{Delimiters: []string{"/"}, Index: 1}},
					},
				},
			},
			expectedStacks: 2, // PXL group (1,2) and IMG group (3,4) should be separate
		},
		{
			name: "AND group requires all criteria to match",
			assets: []utils.TAsset{
				{ID: "1", OriginalFileName: "photo1.jpg", OriginalPath: "/photos/2023/photo1.jpg"},
				{ID: "2", OriginalFileName: "photo2.jpg", OriginalPath: "/photos/2023/photo2.jpg"}, // Same folder and filename pattern
				{ID: "3", OriginalFileName: "image3.jpg", OriginalPath: "/photos/2023/image3.jpg"}, // Same folder, different pattern
				{ID: "4", OriginalFileName: "photo4.jpg", OriginalPath: "/other/2023/photo4.jpg"},  // Same pattern, different folder
			},
			groups: []utils.TCriteriaGroup{
				{
					Operator: "AND",
					Criteria: []utils.TCriteria{
						{Key: "originalFileName", Regex: &utils.TRegex{Key: "^photo", Index: 0}},
						{Key: "originalPath", Split: &utils.TSplit{Delimiters: []string{"/"}, Index: 2}}, // Extract folder
					},
				},
			},
			expectedStacks: 1, // Only assets 1 and 2 match both criteria
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := CriteriaConfig{
				Mode:   "advanced",
				Groups: tt.groups,
			}
			stacks, err := stackByLegacyGroups(tt.assets, config, "", "", logger)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if len(stacks) != tt.expectedStacks {
				t.Errorf("Expected %d stacks, got %d", tt.expectedStacks, len(stacks))
				for i, stack := range stacks {
					t.Logf("Stack %d has %d assets:", i, len(stack))
					for j, asset := range stack {
						t.Logf("  [%d] %s (%s)", j, asset.OriginalFileName, asset.ID)
					}
				}
			}

			// Verify each stack has 2+ assets
			for i, stack := range stacks {
				if len(stack) < 2 {
					t.Errorf("Stack %d has only %d assets, expected 2+", i, len(stack))
				}
			}
		})
	}
}

func TestBuildConnectedComponents(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	tests := []struct {
		name               string
		assets             []utils.TAsset
		assetKeys          map[string][]string
		expectedComponents int
	}{
		{
			name: "Simple chain: A-B-C connected via shared keys",
			assets: []utils.TAsset{
				{ID: "A", OriginalFileName: "a.jpg"},
				{ID: "B", OriginalFileName: "b.jpg"},
				{ID: "C", OriginalFileName: "c.jpg"},
			},
			assetKeys: map[string][]string{
				"A": {"key1", "key2"},
				"B": {"key2", "key3"}, // Shares key2 with A, key3 with C
				"C": {"key3", "key4"},
			},
			expectedComponents: 1, // All connected through B
		},
		{
			name: "Two separate components",
			assets: []utils.TAsset{
				{ID: "A", OriginalFileName: "a.jpg"},
				{ID: "B", OriginalFileName: "b.jpg"},
				{ID: "C", OriginalFileName: "c.jpg"},
				{ID: "D", OriginalFileName: "d.jpg"},
			},
			assetKeys: map[string][]string{
				"A": {"key1"},
				"B": {"key1"}, // A-B connected
				"C": {"key2"},
				"D": {"key2"}, // C-D connected
			},
			expectedComponents: 2, // Two separate components
		},
		{
			name: "Star pattern: one asset connects to many",
			assets: []utils.TAsset{
				{ID: "center", OriginalFileName: "center.jpg"},
				{ID: "leaf1", OriginalFileName: "leaf1.jpg"},
				{ID: "leaf2", OriginalFileName: "leaf2.jpg"},
				{ID: "leaf3", OriginalFileName: "leaf3.jpg"},
			},
			assetKeys: map[string][]string{
				"center": {"key1", "key2", "key3"},
				"leaf1":  {"key1"},
				"leaf2":  {"key2"},
				"leaf3":  {"key3"},
			},
			expectedComponents: 1, // All connected through center
		},
		{
			name: "No shared keys - each asset is its own component",
			assets: []utils.TAsset{
				{ID: "A", OriginalFileName: "a.jpg"},
				{ID: "B", OriginalFileName: "b.jpg"},
			},
			assetKeys: map[string][]string{
				"A": {"key1"},
				"B": {"key2"}, // No shared keys
			},
			expectedComponents: 0, // No components with 2+ assets
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			components := buildConnectedComponents(tt.assets, tt.assetKeys, logger)

			// Filter out components with fewer than 2 assets (like the actual function does)
			validComponents := 0
			for _, comp := range components {
				if len(comp) >= 2 {
					validComponents++
				}
			}

			if validComponents != tt.expectedComponents {
				t.Errorf("Expected %d valid components, got %d", tt.expectedComponents, validComponents)
				for i, comp := range components {
					if len(comp) >= 2 {
						t.Logf("Component %d has %d assets:", i, len(comp))
						for j, asset := range comp {
							t.Logf("  [%d] %s", j, asset.ID)
						}
					}
				}
			}
		})
	}
}

func TestBuildConnectedComponents_DuplicateKeyDeduplication(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Test that assets sharing multiple grouping keys don't create duplicate neighbors
	// This verifies the neighbor deduplication optimization works correctly
	assets := []utils.TAsset{
		{ID: "1", OriginalFileName: "photo1.jpg", OriginalPath: "/2023/album1/"},
		{ID: "2", OriginalFileName: "photo2.jpg", OriginalPath: "/2023/album1/"},
	}

	// Asset grouping data that creates multiple connections between the same assets
	assetGroupingData := map[string][]string{
		"1": {"key1", "key2", "key3"}, // Asset 1 has 3 keys
		"2": {"key1", "key2", "key3"}, // Asset 2 has the same 3 keys
	}

	components := buildConnectedComponents(assets, assetGroupingData, logger)

	// Both assets should be in the same component despite multiple shared keys
	if len(components) != 1 {
		t.Errorf("Expected 1 connected component, got %d", len(components))
		return
	}

	if len(components[0]) != 2 {
		t.Errorf("Expected component to contain 2 assets, got %d", len(components[0]))
		return
	}

	// Verify both assets are present
	assetIDs := make(map[string]bool)
	for _, asset := range components[0] {
		assetIDs[asset.ID] = true
	}

	if !assetIDs["1"] {
		t.Error("Component should contain asset 1")
	}
	if !assetIDs["2"] {
		t.Error("Component should contain asset 2")
	}
}

func TestAdvancedCriteriaIntegration(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Test complete workflow with complex expressions
	assets := []utils.TAsset{
		{ID: "1", OriginalFileName: "PXL_20230101_120000.jpg", LocalDateTime: "2023-01-01T12:00:00Z"},
		{ID: "2", OriginalFileName: "PXL_20230101_120001.jpg", LocalDateTime: "2023-01-01T12:00:01Z"}, // Same group
		{ID: "3", OriginalFileName: "IMG_20230101_120000.jpg", LocalDateTime: "2023-01-01T12:00:00Z"},
		{ID: "4", OriginalFileName: "IMG_20230101_120001.jpg", LocalDateTime: "2023-01-01T12:00:01Z"}, // Different group
		{ID: "5", OriginalFileName: "DSC_001.jpg", LocalDateTime: "2023-01-01T12:00:00Z"},             // No match (different pattern)
	}

	config := CriteriaConfig{
		Mode: "advanced",
		Expression: &utils.TCriteriaExpression{
			Operator: &[]string{"AND"}[0],
			Children: []utils.TCriteriaExpression{
				{
					Operator: &[]string{"OR"}[0],
					Children: []utils.TCriteriaExpression{
						{Criteria: &utils.TCriteria{Key: "originalFileName", Regex: &utils.TRegex{Key: "^PXL_", Index: 0}}},
						{Criteria: &utils.TCriteria{Key: "originalFileName", Regex: &utils.TRegex{Key: "^IMG_", Index: 0}}},
					},
				},
				{
					Criteria: &utils.TCriteria{Key: "localDateTime", Delta: &utils.TDelta{Milliseconds: 2000}},
				},
			},
		},
	}

	stacks, err := stackByAdvanced(assets, config, "", "", logger)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should create 2 stacks: PXL group (assets 1,2) and IMG group (assets 3,4)
	if len(stacks) != 2 {
		t.Errorf("Expected 2 stacks, got %d", len(stacks))
		return
	}

	// Verify each stack has 2 assets
	for i, stack := range stacks {
		if len(stack) != 2 {
			t.Errorf("Stack %d has %d assets, expected 2", i, len(stack))
		}
	}

	// Verify assets are grouped correctly (PXL together, IMG together)
	pxlCount := 0
	imgCount := 0
	for _, stack := range stacks {
		pxlInStack := 0
		imgInStack := 0
		for _, asset := range stack {
			if strings.HasPrefix(asset.OriginalFileName, "PXL_") {
				pxlInStack++
			} else if strings.HasPrefix(asset.OriginalFileName, "IMG_") {
				imgInStack++
			}
		}
		if pxlInStack > 0 && imgInStack > 0 {
			t.Errorf("Stack contains mixed PXL and IMG assets, expected homogeneous grouping")
		}
		if pxlInStack > 0 {
			pxlCount++
		}
		if imgInStack > 0 {
			imgCount++
		}
	}

	if pxlCount != 1 || imgCount != 1 {
		t.Errorf("Expected 1 PXL stack and 1 IMG stack, got %d PXL and %d IMG", pxlCount, imgCount)
	}
}
