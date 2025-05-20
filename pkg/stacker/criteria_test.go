package stacker

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/majorfi/immich-stack/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/************************************************************************************************
** Test cases for criteria matching
************************************************************************************************/

func TestApplyCriteria(t *testing.T) {
	tests := []struct {
		name     string
		fileList []string
		want     int // number of groups
	}{
		{
			name: "same file different folder",
			fileList: []string{
				"IMG_2482.jpg",
				"IMG_2482.jpg",
			},
			want: 1,
		},
		{
			name: "same base different extension",
			fileList: []string{
				"IMG_2482.jpg",
				"IMG_2482.cr2",
			},
			want: 1, // Should group by base name, regardless of extension
		},
		{
			name: "different files",
			fileList: []string{
				"IMG_2482.jpg",
				"IMG_2483.cr2",
			},
			want: 0, // Different base names, so no group (per implementation)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			assets := make([]utils.TAsset, len(tt.fileList))
			for i, f := range tt.fileList {
				assets[i] = assetFactory(f, time.Now())
			}

			// Act
			groups, err := StackBy(assets, "", "", "", logrus.New())

			// Assert
			require.NoError(t, err)
			assert.Equal(t, tt.want, len(groups))
		})
	}
}

/************************************************************************************************
** Test extractOriginalFileName extension removal edge cases
************************************************************************************************/
func TestExtractOriginalFileNameExtensionRemoval(t *testing.T) {
	type testCase struct {
		filename string
		expected string
	}
	tests := []testCase{
		{"1234.jpg", "1234"},
		{"1234.edit.jpg", "1234.edit"},
	}

	criteria := utils.TCriteria{
		Key:   "originalFileName",
		Split: nil, // Only test extension removal
	}

	for _, tc := range tests {
		t.Run(tc.filename, func(t *testing.T) {
			asset := utils.TAsset{OriginalFileName: tc.filename}
			result, err := extractOriginalFileName(asset, criteria)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

/************************************************************************************************
** Test extractOriginalFileName with multi-delimiter split (Delimiters: ["~", "."]).
************************************************************************************************/
func TestExtractOriginalFileNameMultiDelimiter(t *testing.T) {
	type testCase struct {
		filename string
		expected string
	}
	tests := []testCase{
		{"PXL_20250503_152823814.jpg", "PXL_20250503_152823814"},
		{"PXL_20250503_152823814~2.jpg", "PXL_20250503_152823814"},
		{"PXL_20250503_152823814~3.jpg", "PXL_20250503_152823814"},
		{"PXL_20250503_152823814.edit.jpg", "PXL_20250503_152823814"},
	}

	criteria := utils.TCriteria{
		Key: "originalFileName",
		Split: &utils.TSplit{
			Delimiters: []string{"~", "."},
			Index:      0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.filename, func(t *testing.T) {
			asset := utils.TAsset{OriginalFileName: tc.filename}
			result, err := extractOriginalFileName(asset, criteria)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

/************************************************************************************************
** Test sortStack with 'biggestNumber' in promote list prioritizes the largest numeric suffix.
************************************************************************************************/
func TestSortStackBiggestNumber(t *testing.T) {
	assets := []utils.TAsset{
		{OriginalFileName: "PXL_20250503_152823814.jpg"},
		{OriginalFileName: "PXL_20250503_152823814~2.jpg"},
		{OriginalFileName: "PXL_20250503_152823814~3.jpg"},
		{OriginalFileName: "PXL_20250503_152823814.7.jpg"},
		{OriginalFileName: "PXL_20250503_152823814.edit99.jpg"},
	}
	// Promote list includes 'biggestNumber' only
	promote := "edit,biggestNumber"
	result := sortStack(assets, promote, "", []string{"~", "."})
	// The first asset should be the one with the largest number (edit99)
	assert.Equal(t, "PXL_20250503_152823814.edit99.jpg", result[0].OriginalFileName)
	assert.Equal(t, "PXL_20250503_152823814.7.jpg", result[1].OriginalFileName)
	assert.Equal(t, "PXL_20250503_152823814~3.jpg", result[2].OriginalFileName)
	assert.Equal(t, "PXL_20250503_152823814~2.jpg", result[3].OriginalFileName)
	assert.Equal(t, "PXL_20250503_152823814.jpg", result[4].OriginalFileName)
}

/************************************************************************************************
** Test cases for time delta functionality
************************************************************************************************/
func TestExtractTimeWithDelta(t *testing.T) {
	tests := []struct {
		name     string
		timeStr  string
		delta    *utils.TDelta
		expected string
		wantErr  bool
	}{
		{
			name:     "no delta returns original time in UTC",
			timeStr:  "2023-08-24T17:00:15.915Z",
			delta:    nil,
			expected: "2023-08-24T17:00:15.915000000Z",
			wantErr:  false,
		},
		{
			name:     "zero delta returns original time in UTC",
			timeStr:  "2023-08-24T17:00:15.915Z",
			delta:    &utils.TDelta{Milliseconds: 0},
			expected: "2023-08-24T17:00:15.915000000Z",
			wantErr:  false,
		},
		{
			name:     "1000ms delta rounds down",
			timeStr:  "2023-08-24T17:00:15.915Z",
			delta:    &utils.TDelta{Milliseconds: 1000},
			expected: "2023-08-24T17:00:15.000000000Z",
			wantErr:  false,
		},
		{
			name:     "500ms delta rounds to nearest interval",
			timeStr:  "2023-08-24T17:00:15.750Z",
			delta:    &utils.TDelta{Milliseconds: 500},
			expected: "2023-08-24T17:00:15.500000000Z",
			wantErr:  false,
		},
		{
			name:     "empty time string returns empty",
			timeStr:  "",
			delta:    &utils.TDelta{Milliseconds: 1000},
			expected: "",
			wantErr:  false,
		},
		{
			name:     "invalid time format returns error",
			timeStr:  "invalid-time",
			delta:    &utils.TDelta{Milliseconds: 1000},
			expected: "",
			wantErr:  true,
		},
		{
			name:     "handles non-UTC input",
			timeStr:  "2023-08-24T19:00:15.915+02:00",
			delta:    nil,
			expected: "2023-08-24T17:00:15.915000000Z",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractTimeWithDelta(tt.timeStr, tt.delta)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result, "Time format mismatch")
			}
		})
	}
}

/************************************************************************************************
** Test cases for time-based criteria matching with delta
************************************************************************************************/
func TestApplyCriteriaWithTimeDelta(t *testing.T) {
	tests := []struct {
		name     string
		assets   []utils.TAsset
		criteria []utils.TCriteria
		want     int // number of groups
	}{
		{
			name: "exact time match",
			assets: []utils.TAsset{
				{
					OriginalFileName: "IMG_001.jpg",
					LocalDateTime:    "2023-08-24T17:00:15.000Z",
				},
				{
					OriginalFileName: "IMG_002.jpg",
					LocalDateTime:    "2023-08-24T17:00:15.000Z",
				},
			},
			criteria: []utils.TCriteria{
				{
					Key: "localDateTime",
				},
			},
			want: 1, // Should group together
		},
		{
			name: "time difference within delta",
			assets: []utils.TAsset{
				{
					OriginalFileName: "IMG_001.jpg",
					LocalDateTime:    "2023-08-24T17:00:15.915Z",
				},
				{
					OriginalFileName: "IMG_002.jpg",
					LocalDateTime:    "2023-08-24T17:00:15.810Z",
				},
			},
			criteria: []utils.TCriteria{
				{
					Key: "localDateTime",
					Delta: &utils.TDelta{
						Milliseconds: 1000,
					},
				},
			},
			want: 1, // Should group together with 1s delta
		},
		{
			name: "time difference outside delta",
			assets: []utils.TAsset{
				{
					OriginalFileName: "IMG_001.jpg",
					LocalDateTime:    "2023-08-24T17:00:15.915Z",
				},
				{
					OriginalFileName: "IMG_002.jpg",
					LocalDateTime:    "2023-08-24T17:00:16.810Z",
				},
			},
			criteria: []utils.TCriteria{
				{
					Key: "localDateTime",
					Delta: &utils.TDelta{
						Milliseconds: 500,
					},
				},
			},
			want: 0, // Should not group together with 500ms delta
		},
		{
			name: "multiple time fields with delta",
			assets: []utils.TAsset{
				{
					OriginalFileName: "IMG_001.jpg",
					LocalDateTime:    "2023-08-24T17:00:15.915Z",
					FileCreatedAt:    "2023-08-24T17:00:15.900Z",
				},
				{
					OriginalFileName: "IMG_002.jpg",
					LocalDateTime:    "2023-08-24T17:00:15.810Z",
					FileCreatedAt:    "2023-08-24T17:00:15.800Z",
				},
			},
			criteria: []utils.TCriteria{
				{
					Key: "localDateTime",
					Delta: &utils.TDelta{
						Milliseconds: 1000,
					},
				},
				{
					Key: "fileCreatedAt",
					Delta: &utils.TDelta{
						Milliseconds: 1000,
					},
				},
			},
			want: 1, // Should group together with 1s delta on both fields
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up test criteria in environment
			t.Setenv("CRITERIA", mustMarshalJSON(t, tt.criteria))

			groups, err := StackBy(tt.assets, "", "", "", logrus.New())
			require.NoError(t, err)
			assert.Equal(t, tt.want, len(groups), "Expected %d groups but got %d", tt.want, len(groups))

			if tt.want > 0 && len(groups) > 0 {
				// Verify all assets in the group have the same rounded time
				group := groups[0]
				firstAsset := group[0]
				firstTime, err := extractTimeWithDelta(firstAsset.LocalDateTime, tt.criteria[0].Delta)
				require.NoError(t, err)

				for _, asset := range group[1:] {
					assetTime, err := extractTimeWithDelta(asset.LocalDateTime, tt.criteria[0].Delta)
					require.NoError(t, err)
					assert.Equal(t, firstTime, assetTime, "All times in group should round to same value")
				}
			}
		})
	}
}

// Helper function to marshal criteria to JSON for environment variable
func mustMarshalJSON(t *testing.T, v interface{}) string {
	data, err := json.Marshal(v)
	require.NoError(t, err)
	return string(data)
}
