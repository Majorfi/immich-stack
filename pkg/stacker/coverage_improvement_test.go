package stacker

import (
	"testing"

	"github.com/majorfi/immich-stack/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/************************************************************************************************
** Tests to improve coverage for StackBy function (currently 46.2% coverage)
************************************************************************************************/
func TestStackBy_ErrorCases(t *testing.T) {
	logger := logrus.New()

	tests := []struct {
		name        string
		assets      []utils.TAsset
		criteria    string
		promoteList string
		expectError bool
		errorPart   string
	}{
		{
			name:        "invalid criteria JSON",
			assets:      []utils.TAsset{{ID: "1", OriginalFileName: "test.jpg"}},
			criteria:    `{"invalid":json}`,
			expectError: true,
			errorPart:   "failed to parse criteria",
		},
		{
			name:        "invalid regex in criteria",
			assets:      []utils.TAsset{{ID: "1", OriginalFileName: "test.jpg"}},
			criteria:    `[{"key":"originalFileName","regex":{"key":"[invalid"}}]`,
			expectError: true,
			errorPart:   "failed to compile regex",
		},
		{
			name:        "invalid advanced criteria regex",
			assets:      []utils.TAsset{{ID: "1", OriginalFileName: "test.jpg"}},
			criteria:    `{"mode":"advanced","groups":[{"criteria":[{"key":"originalFileName","regex":{"key":"[invalid"}}]}]}`,
			expectError: true,
			errorPart:   "failed to compile regex",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("CRITERIA", tt.criteria)

			stacks, err := StackBy(tt.assets, tt.criteria, "", "", logger)
			if tt.expectError {
				assert.Error(t, err)
				if err != nil && tt.errorPart != "" {
					assert.Contains(t, err.Error(), tt.errorPart)
				}
				assert.Empty(t, stacks)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStackBy_AdvancedMode(t *testing.T) {
	logger := logrus.New()

	assets := []utils.TAsset{
		{ID: "1", OriginalFileName: "IMG_001.jpg", LocalDateTime: "2023-01-01T12:00:00Z"},
		{ID: "2", OriginalFileName: "IMG_002.jpg", LocalDateTime: "2023-01-01T12:00:00Z"},
		{ID: "3", OriginalFileName: "IMG_003.jpg", LocalDateTime: "2023-01-01T13:00:00Z"},
	}

	// Advanced mode with OR groups
	criteria := `{
		"mode": "advanced",
		"groups": [
			{
				"operator": "OR",
				"criteria": [
					{"key": "originalFileName"},
					{"key": "localDateTime"}
				]
			}
		]
	}`

	t.Setenv("CRITERIA", criteria)
	stacks, err := StackBy(assets, "", "", "", logger)
	require.NoError(t, err)

	// Should create stacks based on OR logic
	assert.Greater(t, len(stacks), 0, "Should create at least one stack")
}

func TestStackBy_ExpressionMode(t *testing.T) {
	logger := logrus.New()

	assets := []utils.TAsset{
		{ID: "1", OriginalFileName: "IMG_001.jpg", IsArchived: true, LocalDateTime: "2023-01-01T12:00:00Z"},
		{ID: "2", OriginalFileName: "IMG_002.jpg", IsArchived: true, LocalDateTime: "2023-01-01T12:00:00Z"},
		{ID: "3", OriginalFileName: "IMG_003.jpg", IsArchived: false, LocalDateTime: "2023-01-01T13:00:00Z"},
	}

	// Expression mode with AND combination of isArchived and localDateTime for grouping
	criteria := `{
		"mode": "advanced",
		"expression": {
			"operator": "AND",
			"children": [
				{"criteria": {"key": "isArchived"}},
				{"criteria": {"key": "localDateTime"}}
			]
		}
	}`

	t.Setenv("CRITERIA", criteria)
	stacks, err := StackBy(assets, "", "", "", logger)
	require.NoError(t, err)

	// Should create stacks based on archived status and time grouping
	assert.Greater(t, len(stacks), 0, "Should create at least one stack")
}

func TestStackBy_EmptyAssets(t *testing.T) {
	logger := logrus.New()

	stacks, err := StackBy([]utils.TAsset{}, "", "", "", logger)
	require.NoError(t, err)
	assert.Empty(t, stacks, "Empty assets should return empty stacks")
}

/************************************************************************************************
** Tests to improve coverage for evaluateSingleCriteria function (currently 57.9% coverage)
************************************************************************************************/
func TestEvaluateSingleCriteria_EdgeCases(t *testing.T) {
	asset := utils.TAsset{
		ID:               "test-1",
		OriginalFileName: "IMG_001.jpg",
		IsArchived:       true,
		IsFavorite:       false,
		IsTrashed:        true,
		IsOffline:        false,
		HasMetadata:      true,
		Type:             "IMAGE",
		LocalDateTime:    "2023-01-01T12:00:00Z",
	}

	tests := []struct {
		name     string
		criteria utils.TCriteria
		expected bool
		wantErr  bool
	}{
		{
			name:     "unknown criteria key",
			criteria: utils.TCriteria{Key: "nonExistentKey"},
			expected: false,
			wantErr:  true,
		},
		{
			name:     "boolean field true matches",
			criteria: utils.TCriteria{Key: "isArchived"},
			expected: true,
			wantErr:  false,
		},
		{
			name:     "boolean field false doesn't match",
			criteria: utils.TCriteria{Key: "isFavorite"},
			expected: false,
			wantErr:  false,
		},
		{
			name:     "type field matches",
			criteria: utils.TCriteria{Key: "type"},
			expected: true,
			wantErr:  false,
		},
		{
			name:     "empty value doesn't match",
			criteria: utils.TCriteria{Key: "checksum"}, // Checksum is empty in test asset
			expected: false,
			wantErr:  false,
		},
		{
			name: "regex validation for generic field",
			criteria: utils.TCriteria{
				Key: "type",
				Regex: &utils.TRegex{
					Key:   "IMAGE",
					Index: 0,
				},
			},
			expected: true,
			wantErr:  false,
		},
		{
			name: "regex validation fails for generic field",
			criteria: utils.TCriteria{
				Key: "type",
				Regex: &utils.TRegex{
					Key:   "VIDEO",
					Index: 0,
				},
			},
			expected: false,
			wantErr:  false,
		},
		{
			name: "invalid regex pattern for generic field",
			criteria: utils.TCriteria{
				Key: "type",
				Regex: &utils.TRegex{
					Key:   "[invalid",
					Index: 0,
				},
			},
			expected: false,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluateSingleCriteria(tt.criteria, asset)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

/************************************************************************************************
** Tests to improve coverage for splitByDelimiters function (currently 76.9% coverage)
************************************************************************************************/
func TestSplitByDelimiters_EdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		delimiters []string
		index      int
		expected   string
		wantErr    bool
	}{
		{
			name:       "no delimiters with index 0 returns input",
			input:      "test_string",
			delimiters: []string{},
			index:      0,
			expected:   "test_string",
			wantErr:    false,
		},
		{
			name:       "no delimiters with index > 0 returns error",
			input:      "test_string",
			delimiters: []string{},
			index:      1,
			expected:   "",
			wantErr:    true,
		},
		{
			name:       "single delimiter splits correctly",
			input:      "part1_part2_part3",
			delimiters: []string{"_"},
			index:      1,
			expected:   "part2",
			wantErr:    false,
		},
		{
			name:       "multiple delimiters split sequentially",
			input:      "part1_part2.sub1.sub2",
			delimiters: []string{"_", "."},
			index:      3,
			expected:   "sub2",
			wantErr:    false,
		},
		{
			name:       "index out of range after splitting",
			input:      "part1_part2",
			delimiters: []string{"_"},
			index:      5,
			expected:   "",
			wantErr:    true,
		},
		{
			name:       "negative index returns error",
			input:      "part1_part2",
			delimiters: []string{"_"},
			index:      -1,
			expected:   "",
			wantErr:    true,
		},
		{
			name:       "delimiter not found in input",
			input:      "no_underscores_here",
			delimiters: []string{"-"},
			index:      0,
			expected:   "no_underscores_here",
			wantErr:    false,
		},
		{
			name:       "empty input string",
			input:      "",
			delimiters: []string{"_"},
			index:      0,
			expected:   "",
			wantErr:    false,
		},
		{
			name:       "empty input string with index > 0",
			input:      "",
			delimiters: []string{"_"},
			index:      1,
			expected:   "",
			wantErr:    true,
		},
		{
			name:       "complex multi-delimiter split",
			input:      "a_b.c-d",
			delimiters: []string{"_", ".", "-"},
			index:      2,
			expected:   "c",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := splitByDelimiters(tt.input, tt.delimiters, tt.index)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

/************************************************************************************************
** Additional coverage for EvaluateExpression function
************************************************************************************************/
func TestEvaluateExpression_ErrorCases(t *testing.T) {
	asset := utils.TAsset{
		ID:               "test-1",
		OriginalFileName: "IMG_001.jpg",
		IsArchived:       true,
	}

	tests := []struct {
		name     string
		expr     *utils.TCriteriaExpression
		expected bool
		wantErr  bool
	}{
		{
			name:     "nil expression returns error",
			expr:     nil,
			expected: false,
			wantErr:  true,
		},
		{
			name:     "expression with neither criteria nor operator",
			expr:     &utils.TCriteriaExpression{},
			expected: false,
			wantErr:  true,
		},
		{
			name: "AND operator with no children",
			expr: &utils.TCriteriaExpression{
				Operator: &[]string{"AND"}[0],
				Children: []utils.TCriteriaExpression{},
			},
			expected: false,
			wantErr:  true,
		},
		{
			name: "unknown operator",
			expr: &utils.TCriteriaExpression{
				Operator: &[]string{"UNKNOWN"}[0],
				Children: []utils.TCriteriaExpression{
					{Criteria: &utils.TCriteria{Key: "isArchived"}},
				},
			},
			expected: false,
			wantErr:  true,
		},
		{
			name: "NOT operator with multiple children",
			expr: &utils.TCriteriaExpression{
				Operator: &[]string{"NOT"}[0],
				Children: []utils.TCriteriaExpression{
					{Criteria: &utils.TCriteria{Key: "isArchived"}},
					{Criteria: &utils.TCriteria{Key: "isFavorite"}},
				},
			},
			expected: false,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EvaluateExpression(tt.expr, asset)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
