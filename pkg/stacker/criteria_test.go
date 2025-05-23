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
			name: "time difference within delta 1/2",
			assets: []utils.TAsset{
				{
					OriginalFileName: "20150628_0173.JPG",
					LocalDateTime:    "2015-06-28T12:55:31.000000000Z",
				},
				{
					OriginalFileName: "20150628_0173.CR2",
					LocalDateTime:    "2015-06-28T12:55:31.660000000Z",
				},
			},
			criteria: []utils.TCriteria{
				{
					Key: "originalFileName",
					Split: &utils.TSplit{
						Delimiters: []string{"~", "."},
						Index:      0,
					},
				},
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
			name: "time difference within delta 2/2",
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

				// Find the localDateTime criteria with delta
				var timeDelta *utils.TDelta
				for _, c := range tt.criteria {
					if c.Key == "localDateTime" && c.Delta != nil {
						timeDelta = c.Delta
						break
					}
				}

				firstTime, err := extractTimeWithDelta(firstAsset.LocalDateTime, timeDelta)
				require.NoError(t, err)

				for _, asset := range group[1:] {
					assetTime, err := extractTimeWithDelta(asset.LocalDateTime, timeDelta)
					require.NoError(t, err)
					assert.Equal(t, firstTime, assetTime, "All times in group should round to same value")
				}
			}
		})
	}
}

/************************************************************************************************
** Test extractOriginalPath with various path formats and split configurations
************************************************************************************************/
func TestExtractOriginalPath(t *testing.T) {
	type testCase struct {
		name     string
		path     string
		criteria utils.TCriteria
		expected string
		wantErr  bool
	}

	tests := []testCase{
		{
			name: "simple path without split",
			path: "photos/2023/vacation/IMG_001.jpg",
			criteria: utils.TCriteria{
				Key: "originalPath",
			},
			expected: "photos/2023/vacation/IMG_001.jpg",
			wantErr:  false,
		},
		{
			name: "windows path without split",
			path: "photos\\2023\\vacation\\IMG_001.jpg",
			criteria: utils.TCriteria{
				Key: "originalPath",
			},
			expected: "photos/2023/vacation/IMG_001.jpg",
			wantErr:  false,
		},
		{
			name: "split by forward slash",
			path: "photos/2023/vacation/IMG_001.jpg",
			criteria: utils.TCriteria{
				Key: "originalPath",
				Split: &utils.TSplit{
					Delimiters: []string{"/"},
					Index:      1,
				},
			},
			expected: "2023",
			wantErr:  false,
		},
		{
			name: "split by multiple delimiters",
			path: "photos/2023-vacation/IMG_001.jpg",
			criteria: utils.TCriteria{
				Key: "originalPath",
				Split: &utils.TSplit{
					Delimiters: []string{"/", "-"},
					Index:      2,
				},
			},
			expected: "vacation",
			wantErr:  false,
		},
		{
			name: "split index out of range",
			path: "photos/2023/vacation/IMG_001.jpg",
			criteria: utils.TCriteria{
				Key: "originalPath",
				Split: &utils.TSplit{
					Delimiters: []string{"/"},
					Index:      10,
				},
			},
			expected: "",
			wantErr:  true,
		},
		{
			name: "empty path",
			path: "",
			criteria: utils.TCriteria{
				Key: "originalPath",
			},
			expected: "",
			wantErr:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			asset := utils.TAsset{OriginalPath: tc.path}
			result, err := extractOriginalPath(asset, tc.criteria)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

/************************************************************************************************
** Test applyCriteria with originalPath in criteria
************************************************************************************************/
func TestApplyCriteriaWithOriginalPath(t *testing.T) {
	tests := []struct {
		name     string
		assets   []utils.TAsset
		criteria []utils.TCriteria
		want     int // number of groups
	}{
		{
			name: "group by path directory",
			assets: []utils.TAsset{
				{
					OriginalFileName: "IMG_001.jpg",
					OriginalPath:     "photos/2023/vacation/IMG_001.jpg",
				},
				{
					OriginalFileName: "IMG_002.jpg",
					OriginalPath:     "photos/2023/vacation/IMG_002.jpg",
				},
			},
			criteria: []utils.TCriteria{
				{
					Key: "originalPath",
					Split: &utils.TSplit{
						Delimiters: []string{"/"},
						Index:      2,
					},
				},
			},
			want: 1, // Should group together as they're in the same directory
		},
		{
			name: "different directories",
			assets: []utils.TAsset{
				{
					OriginalFileName: "IMG_001.jpg",
					OriginalPath:     "photos/2023/vacation/IMG_001.jpg",
				},
				{
					OriginalFileName: "IMG_002.jpg",
					OriginalPath:     "photos/2023/work/IMG_002.jpg",
				},
			},
			criteria: []utils.TCriteria{
				{
					Key: "originalPath",
					Split: &utils.TSplit{
						Delimiters: []string{"/"},
						Index:      2,
					},
				},
			},
			want: 0, // Should not group together as they're in different directories
		},
		{
			name: "same name in same directory regardless of date",
			assets: []utils.TAsset{
				{
					OriginalFileName: "IMG_001.jpg",
					OriginalPath:     "photos/2023/vacation/IMG_001.jpg",
					LocalDateTime:    "2023-01-01T12:00:00.000Z",
				},
				{
					OriginalFileName: "IMG_001.jpg",
					OriginalPath:     "photos/2023/vacation/IMG_001.jpg",
					LocalDateTime:    "2023-01-02T15:30:00.000Z",
				},
				{
					OriginalFileName: "IMG_001.jpg",
					OriginalPath:     "photos/2023/vacation/IMG_001.jpg",
					LocalDateTime:    "2023-01-03T09:45:00.000Z",
				},
			},
			criteria: []utils.TCriteria{
				{
					Key: "originalFileName",
				},
				{
					Key: "originalPath",
				},
			},
			want: 1, // Should group together as they have same name and path, regardless of date
		},
		{
			name: "same name in different directories",
			assets: []utils.TAsset{
				{
					OriginalFileName: "IMG_001.jpg",
					OriginalPath:     "photos/2023/vacation/IMG_001.jpg",
					LocalDateTime:    "2023-01-01T12:00:00.000Z",
				},
				{
					OriginalFileName: "IMG_001.jpg",
					OriginalPath:     "photos/2023/work/IMG_001.jpg",
					LocalDateTime:    "2023-01-01T12:00:00.000Z",
				},
				{
					OriginalFileName: "IMG_001.jpg",
					OriginalPath:     "photos/2023/family/IMG_001.jpg",
					LocalDateTime:    "2023-01-01T12:00:00.000Z",
				},
			},
			criteria: []utils.TCriteria{
				{
					Key: "originalFileName",
					Split: &utils.TSplit{
						Delimiters: []string{"~", "."},
						Index:      0,
					},
				},
				{
					Key: "originalPath",
					Split: &utils.TSplit{
						Delimiters: []string{"/"},
						Index:      2,
					},
				},
			},
			want: 0, // Should create no stacks since files are in different directories
		},
		{
			name: "different names in same directory",
			assets: []utils.TAsset{
				{
					OriginalFileName: "IMG_001.jpg",
					OriginalPath:     "photos/2023/vacation/IMG_001.jpg",
					LocalDateTime:    "2023-01-01T12:00:00.000Z",
				},
				{
					OriginalFileName: "IMG_002.jpg",
					OriginalPath:     "photos/2023/vacation/IMG_002.jpg",
					LocalDateTime:    "2023-01-01T12:00:00.000Z",
				},
				{
					OriginalFileName: "IMG_003.jpg",
					OriginalPath:     "photos/2023/vacation/IMG_003.jpg",
					LocalDateTime:    "2023-01-01T12:00:00.000Z",
				},
			},
			criteria: []utils.TCriteria{
				{
					Key: "originalFileName",
				},
				{
					Key: "originalPath",
				},
			},
			want: 0, // Should create no stacks since files have different names
		},
		{
			name: "mixed scenarios",
			assets: []utils.TAsset{
				// Group 1: Same name, same directory
				{
					OriginalFileName: "IMG_001.jpg",
					OriginalPath:     "photos/2023/vacation/IMG_001.jpg",
					LocalDateTime:    "2023-01-01T12:00:00.000Z",
				},
				{
					OriginalFileName: "IMG_001.jpg",
					OriginalPath:     "photos/2023/vacation/IMG_001.jpg",
					LocalDateTime:    "2023-01-02T12:00:00.000Z",
				},
				// Group 2: Same name, different directory
				{
					OriginalFileName: "IMG_002.jpg",
					OriginalPath:     "photos/2023/work/IMG_002.jpg",
					LocalDateTime:    "2023-01-01T12:00:00.000Z",
				},
				{
					OriginalFileName: "IMG_002.jpg",
					OriginalPath:     "photos/2023/family/IMG_002.jpg",
					LocalDateTime:    "2023-01-01T12:00:00.000Z",
				},
				// Group 3: Different name, same directory
				{
					OriginalFileName: "IMG_003.jpg",
					OriginalPath:     "photos/2023/vacation/IMG_003.jpg",
					LocalDateTime:    "2023-01-01T12:00:00.000Z",
				},
				{
					OriginalFileName: "IMG_004.jpg",
					OriginalPath:     "photos/2023/vacation/IMG_004.jpg",
					LocalDateTime:    "2023-01-01T12:00:00.000Z",
				},
			},
			criteria: []utils.TCriteria{
				{
					Key: "originalFileName",
				},
				{
					Key: "originalPath",
				},
			},
			want: 1, // Should create 1 stack:
			// Only IMG_001.jpg files in vacation directory will be grouped (2 files)
			// All other files have different names or paths, so no stacks formed
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
				// For the "same name in same directory" test, verify all assets are in the group
				if tt.name == "same name in same directory regardless of date" {
					assert.Equal(t, len(tt.assets), len(groups[0]),
						"Expected all %d assets to be in the same group", len(tt.assets))

					// Verify all assets have the same name and path
					firstAsset := groups[0][0]
					for i, asset := range groups[0][1:] {
						assert.Equal(t, firstAsset.OriginalFileName, asset.OriginalFileName,
							"Asset %d should have same name as first asset", i+1)
						assert.Equal(t, firstAsset.OriginalPath, asset.OriginalPath,
							"Asset %d should have same path as first asset", i+1)
					}
				}

				// For the "mixed scenarios" test, verify the correct grouping
				if tt.name == "mixed scenarios" {
					// Only one group should exist: IMG_001.jpg files in vacation directory
					assert.Equal(t, 1, len(groups), "Should have exactly 1 group")
					assert.Equal(t, 2, len(groups[0]), "Group should have 2 files")

					// Verify both assets have the same name and path
					for _, asset := range groups[0] {
						assert.Equal(t, "IMG_001.jpg", asset.OriginalFileName)
						assert.Equal(t, "photos/2023/vacation/IMG_001.jpg", asset.OriginalPath)
					}
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
