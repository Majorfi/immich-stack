package stacker

import (
	"testing"

	"github.com/majorfi/immich-stack/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalkMatchingCriteria(t *testing.T) {
	tests := []struct {
		name           string
		asset          utils.TAsset
		expr           *utils.TCriteriaExpression
		expectedValues map[string]string
		expectError    bool
	}{
		{
			name:           "nil expression",
			asset:          utils.TAsset{ID: "1", OriginalFileName: "test.jpg"},
			expr:           nil,
			expectedValues: map[string]string{},
			expectError:    false,
		},
		{
			name:  "leaf node with matching criteria",
			asset: utils.TAsset{ID: "1", OriginalFileName: "IMG_001.jpg"},
			expr: &utils.TCriteriaExpression{
				Criteria: &utils.TCriteria{
					Key:   "originalFileName",
					Split: &utils.TSplit{Delimiters: []string{"."}, Index: 0},
				},
			},
			expectedValues: map[string]string{"originalFileName": "IMG_001"},
			expectError:    false,
		},
		{
			name:  "leaf node with non-matching regex criteria",
			asset: utils.TAsset{ID: "1", OriginalFileName: "random.jpg"},
			expr: &utils.TCriteriaExpression{
				Criteria: &utils.TCriteria{
					Key:   "originalFileName",
					Regex: &utils.TRegex{Key: `^IMG_\d+`, Index: 0},
				},
			},
			expectedValues: map[string]string{},
			expectError:    false,
		},
		{
			name:  "AND expression - all children match",
			asset: utils.TAsset{ID: "1", OriginalFileName: "IMG_001.jpg", LocalDateTime: "2024-01-15T10:00:00.000000000Z"},
			expr: &utils.TCriteriaExpression{
				Operator: stringPtr("AND"),
				Children: []utils.TCriteriaExpression{
					{
						Criteria: &utils.TCriteria{
							Key:   "originalFileName",
							Split: &utils.TSplit{Delimiters: []string{"."}, Index: 0},
						},
					},
					{
						Criteria: &utils.TCriteria{
							Key: "localDateTime",
						},
					},
				},
			},
			expectedValues: map[string]string{
				"originalFileName": "IMG_001",
				"localDateTime":    "2024-01-15T10:00:00.000000000Z",
			},
			expectError: false,
		},
		{
			name:  "OR expression - first child matches",
			asset: utils.TAsset{ID: "1", OriginalFileName: "IMG_001.jpg"},
			expr: &utils.TCriteriaExpression{
				Operator: stringPtr("OR"),
				Children: []utils.TCriteriaExpression{
					{
						Criteria: &utils.TCriteria{
							Key:   "originalFileName",
							Split: &utils.TSplit{Delimiters: []string{"."}, Index: 0},
						},
					},
					{
						Criteria: &utils.TCriteria{
							Key: "localDateTime",
						},
					},
				},
			},
			expectedValues: map[string]string{"originalFileName": "IMG_001"},
			expectError:    false,
		},
		{
			name:  "expression node without operator or children",
			asset: utils.TAsset{ID: "1", OriginalFileName: "test.jpg"},
			expr: &utils.TCriteriaExpression{
				Operator: nil,
				Children: nil,
			},
			expectedValues: map[string]string{},
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := make(map[string]string)
			err := walkMatchingCriteria(tt.asset, tt.expr, values)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedValues, values)
			}
		})
	}
}

func TestCollectMatchingCriteriaValues(t *testing.T) {
	tests := []struct {
		name           string
		asset          utils.TAsset
		expr           *utils.TCriteriaExpression
		criteria       []utils.TCriteria
		expectedValues map[string]string
		expectError    bool
	}{
		{
			name:           "nil expression returns empty map",
			asset:          utils.TAsset{ID: "1"},
			expr:           nil,
			criteria:       []utils.TCriteria{},
			expectedValues: map[string]string{},
			expectError:    false,
		},
		{
			name:  "simple criteria collects value",
			asset: utils.TAsset{ID: "1", OriginalFileName: "photo.jpg"},
			expr: &utils.TCriteriaExpression{
				Criteria: &utils.TCriteria{
					Key:   "originalFileName",
					Split: &utils.TSplit{Delimiters: []string{"."}, Index: 0},
				},
			},
			criteria: []utils.TCriteria{
				{Key: "originalFileName", Split: &utils.TSplit{Delimiters: []string{"."}, Index: 0}},
			},
			expectedValues: map[string]string{"originalFileName": "photo"},
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, err := collectMatchingCriteriaValues(tt.asset, tt.expr, tt.criteria)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedValues, values)
			}
		})
	}
}

func TestEvaluateExpressionAdditional(t *testing.T) {
	tests := []struct {
		name        string
		asset       utils.TAsset
		expr        *utils.TCriteriaExpression
		expected    bool
		expectError bool
	}{
		{
			name:        "nil expression returns error",
			asset:       utils.TAsset{ID: "1"},
			expr:        nil,
			expected:    false,
			expectError: true,
		},
		{
			name:  "leaf node with matching criteria",
			asset: utils.TAsset{ID: "1", OriginalFileName: "IMG_001.jpg"},
			expr: &utils.TCriteriaExpression{
				Criteria: &utils.TCriteria{
					Key:   "originalFileName",
					Regex: &utils.TRegex{Key: `^IMG_\d+`, Index: 0},
				},
			},
			expected:    true,
			expectError: false,
		},
		{
			name:  "leaf node with non-matching criteria",
			asset: utils.TAsset{ID: "1", OriginalFileName: "photo.jpg"},
			expr: &utils.TCriteriaExpression{
				Criteria: &utils.TCriteria{
					Key:   "originalFileName",
					Regex: &utils.TRegex{Key: `^IMG_\d+`, Index: 0},
				},
			},
			expected:    false,
			expectError: false,
		},
		{
			name:  "AND expression - all true",
			asset: utils.TAsset{ID: "1", OriginalFileName: "IMG_001.jpg", LocalDateTime: "2024-01-01T12:00:00Z"},
			expr: &utils.TCriteriaExpression{
				Operator: stringPtr("AND"),
				Children: []utils.TCriteriaExpression{
					{
						Criteria: &utils.TCriteria{
							Key:   "originalFileName",
							Regex: &utils.TRegex{Key: `^IMG_\d+`, Index: 0},
						},
					},
					{
						Criteria: &utils.TCriteria{
							Key: "localDateTime",
						},
					},
				},
			},
			expected:    true,
			expectError: false,
		},
		{
			name:  "AND expression - one false",
			asset: utils.TAsset{ID: "1", OriginalFileName: "photo.jpg", LocalDateTime: "2024-01-01T12:00:00Z"},
			expr: &utils.TCriteriaExpression{
				Operator: stringPtr("AND"),
				Children: []utils.TCriteriaExpression{
					{
						Criteria: &utils.TCriteria{
							Key:   "originalFileName",
							Regex: &utils.TRegex{Key: `^IMG_\d+`, Index: 0},
						},
					},
					{
						Criteria: &utils.TCriteria{
							Key: "localDateTime",
						},
					},
				},
			},
			expected:    false,
			expectError: false,
		},
		{
			name:  "OR expression - first true",
			asset: utils.TAsset{ID: "1", OriginalFileName: "IMG_001.jpg"},
			expr: &utils.TCriteriaExpression{
				Operator: stringPtr("OR"),
				Children: []utils.TCriteriaExpression{
					{
						Criteria: &utils.TCriteria{
							Key:   "originalFileName",
							Regex: &utils.TRegex{Key: `^IMG_\d+`, Index: 0},
						},
					},
					{
						Criteria: &utils.TCriteria{
							Key:   "originalFileName",
							Regex: &utils.TRegex{Key: `^PHOTO_`, Index: 0},
						},
					},
				},
			},
			expected:    true,
			expectError: false,
		},
		{
			name:  "OR expression - all false",
			asset: utils.TAsset{ID: "1", OriginalFileName: "random.jpg"},
			expr: &utils.TCriteriaExpression{
				Operator: stringPtr("OR"),
				Children: []utils.TCriteriaExpression{
					{
						Criteria: &utils.TCriteria{
							Key:   "originalFileName",
							Regex: &utils.TRegex{Key: `^IMG_\d+`, Index: 0},
						},
					},
					{
						Criteria: &utils.TCriteria{
							Key:   "originalFileName",
							Regex: &utils.TRegex{Key: `^PHOTO_`, Index: 0},
						},
					},
				},
			},
			expected:    false,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EvaluateExpression(tt.expr, tt.asset)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestFlattenCriteriaFromExpression(t *testing.T) {
	tests := []struct {
		name          string
		expr          *utils.TCriteriaExpression
		expectedCount int
	}{
		{
			name:          "nil expression",
			expr:          nil,
			expectedCount: 0,
		},
		{
			name: "single criteria",
			expr: &utils.TCriteriaExpression{
				Criteria: &utils.TCriteria{Key: "originalFileName"},
			},
			expectedCount: 1,
		},
		{
			name: "AND with two criteria",
			expr: &utils.TCriteriaExpression{
				Operator: stringPtr("AND"),
				Children: []utils.TCriteriaExpression{
					{Criteria: &utils.TCriteria{Key: "originalFileName"}},
					{Criteria: &utils.TCriteria{Key: "localDateTime"}},
				},
			},
			expectedCount: 2,
		},
		{
			name: "nested expression",
			expr: &utils.TCriteriaExpression{
				Operator: stringPtr("OR"),
				Children: []utils.TCriteriaExpression{
					{
						Operator: stringPtr("AND"),
						Children: []utils.TCriteriaExpression{
							{Criteria: &utils.TCriteria{Key: "originalFileName"}},
							{Criteria: &utils.TCriteria{Key: "localDateTime"}},
						},
					},
					{Criteria: &utils.TCriteria{Key: "fileCreatedAt"}},
				},
			},
			expectedCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := flattenCriteriaFromExpression(tt.expr)
			assert.Len(t, result, tt.expectedCount)
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
