package stacker

import (
	"testing"

	"github.com/majorfi/immich-stack/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func TestPrecompileExpressionRegexes(t *testing.T) {
	tests := []struct {
		name        string
		expr        *utils.TCriteriaExpression
		expectError bool
	}{
		{
			name:        "nil expression",
			expr:        nil,
			expectError: false,
		},
		{
			name: "valid regex in criteria",
			expr: &utils.TCriteriaExpression{
				Criteria: &utils.TCriteria{
					Key:   "originalFileName",
					Regex: &utils.TRegex{Key: `^IMG_\d+`, Index: 0},
				},
			},
			expectError: false,
		},
		{
			name: "invalid regex in criteria",
			expr: &utils.TCriteriaExpression{
				Criteria: &utils.TCriteria{
					Key:   "originalFileName",
					Regex: &utils.TRegex{Key: `[invalid`, Index: 0},
				},
			},
			expectError: true,
		},
		{
			name: "criteria without regex",
			expr: &utils.TCriteriaExpression{
				Criteria: &utils.TCriteria{
					Key:   "originalFileName",
					Split: &utils.TSplit{Delimiters: []string{"."}, Index: 0},
				},
			},
			expectError: false,
		},
		{
			name: "nested expression with valid regexes",
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
							Key:   "localDateTime",
							Regex: &utils.TRegex{Key: `^\d{4}-\d{2}-\d{2}`, Index: 0},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "nested expression with invalid regex in child",
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
							Key:   "localDateTime",
							Regex: &utils.TRegex{Key: `[broken`, Index: 0},
						},
					},
				},
			},
			expectError: true,
		},
		{
			name: "deeply nested expression",
			expr: &utils.TCriteriaExpression{
				Operator: stringPtr("AND"),
				Children: []utils.TCriteriaExpression{
					{
						Operator: stringPtr("OR"),
						Children: []utils.TCriteriaExpression{
							{
								Criteria: &utils.TCriteria{
									Key:   "originalFileName",
									Regex: &utils.TRegex{Key: `^PXL_`, Index: 0},
								},
							},
							{
								Criteria: &utils.TCriteria{
									Key:   "originalFileName",
									Regex: &utils.TRegex{Key: `^IMG_`, Index: 0},
								},
							},
						},
					},
					{
						Criteria: &utils.TCriteria{
							Key: "localDateTime",
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "empty regex key",
			expr: &utils.TCriteriaExpression{
				Criteria: &utils.TCriteria{
					Key:   "originalFileName",
					Regex: &utils.TRegex{Key: "", Index: 0},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := precompileExpressionRegexes(tt.expr)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPrecompileCriteriaRegex(t *testing.T) {
	tests := []struct {
		name        string
		criteria    utils.TCriteria
		expectError bool
	}{
		{
			name: "no regex",
			criteria: utils.TCriteria{
				Key:   "originalFileName",
				Split: &utils.TSplit{Delimiters: []string{"."}, Index: 0},
			},
			expectError: false,
		},
		{
			name: "valid regex",
			criteria: utils.TCriteria{
				Key:   "originalFileName",
				Regex: &utils.TRegex{Key: `^IMG_(\d+)`, Index: 1},
			},
			expectError: false,
		},
		{
			name: "invalid regex",
			criteria: utils.TCriteria{
				Key:   "originalFileName",
				Regex: &utils.TRegex{Key: `[unclosed`, Index: 0},
			},
			expectError: true,
		},
		{
			name: "nil regex",
			criteria: utils.TCriteria{
				Key:   "originalFileName",
				Regex: nil,
			},
			expectError: false,
		},
		{
			name: "complex valid regex",
			criteria: utils.TCriteria{
				Key:   "originalFileName",
				Regex: &utils.TRegex{Key: `^(PXL|IMG)_(\d{8})_(\d{9})\.`, Index: 2},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := precompileCriteriaRegex(tt.criteria)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to compile regex")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

