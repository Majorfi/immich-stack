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
			result, _, err := extractOriginalFileName(asset, criteria)
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
			result, _, err := extractOriginalFileName(asset, criteria)
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
	result := sortStack(assets, promote, "", []string{"~", "."}, utils.DefaultCriteria, &safePromoteData{data: make(map[string]map[string]string)}, make(map[int]map[string]int))
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
			result, _, err := extractOriginalPath(asset, tc.criteria)
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

/************************************************************************************************
** Test extractOriginalFileName with regex functionality
************************************************************************************************/
func TestExtractOriginalFileNameRegex(t *testing.T) {
	type testCase struct {
		name     string
		filename string
		criteria utils.TCriteria
		expected string
		wantErr  bool
	}

	tests := []testCase{
		{
			name:     "simple regex capture group 0 (full match)",
			filename: "PXL_20230503_152823814.jpg",
			criteria: utils.TCriteria{
				Key: "originalFileName",
				Regex: &utils.TRegex{
					Key:   `PXL_(\d{8})_(\d{9})\.jpg`,
					Index: 0, // Full match
				},
			},
			expected: "PXL_20230503_152823814.jpg",
			wantErr:  false,
		},
		{
			name:     "regex capture group 1 (date)",
			filename: "PXL_20230503_152823814.jpg",
			criteria: utils.TCriteria{
				Key: "originalFileName",
				Regex: &utils.TRegex{
					Key:   `PXL_(\d{8})_(\d{9})\.jpg`,
					Index: 1, // First capture group (date)
				},
			},
			expected: "20230503",
			wantErr:  false,
		},
		{
			name:     "regex capture group 2 (time)",
			filename: "PXL_20230503_152823814.jpg",
			criteria: utils.TCriteria{
				Key: "originalFileName",
				Regex: &utils.TRegex{
					Key:   `PXL_(\d{8})_(\d{9})\.jpg`,
					Index: 2, // Second capture group (time)
				},
			},
			expected: "152823814",
			wantErr:  false,
		},
		{
			name:     "regex with named groups",
			filename: "IMG_20230503_152823.jpg",
			criteria: utils.TCriteria{
				Key: "originalFileName",
				Regex: &utils.TRegex{
					Key:   `IMG_(?P<date>\d{8})_(?P<time>\d{6})\.jpg`,
					Index: 1, // First capture group (date)
				},
			},
			expected: "20230503",
			wantErr:  false,
		},
		{
			name:     "regex no match returns empty",
			filename: "different_format.jpg",
			criteria: utils.TCriteria{
				Key: "originalFileName",
				Regex: &utils.TRegex{
					Key:   `PXL_(\d{8})_(\d{9})\.jpg`,
					Index: 1,
				},
			},
			expected: "",
			wantErr:  false,
		},
		{
			name:     "regex index out of range",
			filename: "PXL_20230503_152823814.jpg",
			criteria: utils.TCriteria{
				Key: "originalFileName",
				Regex: &utils.TRegex{
					Key:   `PXL_(\d{8})_(\d{9})\.jpg`,
					Index: 5, // Only 2 capture groups available (plus full match)
				},
			},
			expected: "",
			wantErr:  true,
		},
		{
			name:     "invalid regex pattern",
			filename: "test.jpg",
			criteria: utils.TCriteria{
				Key: "originalFileName",
				Regex: &utils.TRegex{
					Key:   `[invalid regex`,
					Index: 0,
				},
			},
			expected: "",
			wantErr:  true,
		},
		{
			name:     "complex regex with alternation",
			filename: "DSC01234.jpg",
			criteria: utils.TCriteria{
				Key: "originalFileName",
				Regex: &utils.TRegex{
					Key:   `(IMG|PXL|DSC)(\d+)\.jpg`,
					Index: 2, // Number part
				},
			},
			expected: "01234",
			wantErr:  false,
		},
		{
			name:     "regex with special characters",
			filename: "photo-2023.05.03-15:28:23.jpg",
			criteria: utils.TCriteria{
				Key: "originalFileName",
				Regex: &utils.TRegex{
					Key:   `photo-(\d{4})\.(\d{2})\.(\d{2})-(\d{2}):(\d{2}):(\d{2})\.jpg`,
					Index: 1, // Year
				},
			},
			expected: "2023",
			wantErr:  false,
		},
		{
			name:     "match specific extension",
			filename: "test.edit.jpg",
			criteria: utils.TCriteria{
				Key: "originalFileName",
				Regex: &utils.TRegex{
					Key:   `(.+)\.edit\.jpg$`,
					Index: 1,
				},
			},
			expected: "test",
			wantErr:  false,
		},
		{
			name:     "match different extensions",
			filename: "IMG_1234.JPG",
			criteria: utils.TCriteria{
				Key: "originalFileName",
				Regex: &utils.TRegex{
					Key:   `IMG_(\d+)\.(jpg|JPG)$`,
					Index: 1,
				},
			},
			expected: "1234",
			wantErr:  false,
		},
		{
			name:     "case insensitive extension match",
			filename: "IMG_1234.JPG",
			criteria: utils.TCriteria{
				Key: "originalFileName",
				Regex: &utils.TRegex{
					Key:   `(?i)IMG_(\d+)\.jpg$`,
					Index: 1,
				},
			},
			expected: "1234",
			wantErr:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			asset := utils.TAsset{OriginalFileName: tc.filename}
			result, _, err := extractOriginalFileName(asset, tc.criteria)
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
** Test extractOriginalPath with regex functionality
************************************************************************************************/
func TestExtractOriginalPathRegex(t *testing.T) {
	type testCase struct {
		name     string
		path     string
		criteria utils.TCriteria
		expected string
		wantErr  bool
	}

	tests := []testCase{
		{
			name: "regex extract year from path",
			path: "photos/2023/vacation/IMG_001.jpg",
			criteria: utils.TCriteria{
				Key: "originalPath",
				Regex: &utils.TRegex{
					Key:   `photos/(\d{4})/([^/]+)/`,
					Index: 1, // Year
				},
			},
			expected: "2023",
			wantErr:  false,
		},
		{
			name: "regex extract folder name",
			path: "photos/2023/vacation/IMG_001.jpg",
			criteria: utils.TCriteria{
				Key: "originalPath",
				Regex: &utils.TRegex{
					Key:   `photos/(\d{4})/([^/]+)/`,
					Index: 2, // Folder name
				},
			},
			expected: "vacation",
			wantErr:  false,
		},
		{
			name: "windows path normalized before regex",
			path: "photos\\2023\\vacation\\IMG_001.jpg",
			criteria: utils.TCriteria{
				Key: "originalPath",
				Regex: &utils.TRegex{
					Key:   `photos/(\d{4})/([^/]+)/`,
					Index: 1,
				},
			},
			expected: "2023",
			wantErr:  false,
		},
		{
			name: "complex path structure with regex",
			path: "camera_uploads/2023-05-03/DCIM/Camera/IMG_20230503_152823.jpg",
			criteria: utils.TCriteria{
				Key: "originalPath",
				Regex: &utils.TRegex{
					Key:   `camera_uploads/(\d{4}-\d{2}-\d{2})/DCIM/([^/]+)/`,
					Index: 1, // Date part
				},
			},
			expected: "2023-05-03",
			wantErr:  false,
		},
		{
			name: "regex no match returns empty",
			path: "photos/random/path/IMG_001.jpg",
			criteria: utils.TCriteria{
				Key: "originalPath",
				Regex: &utils.TRegex{
					Key:   `archive/(\d{4})/`,
					Index: 1,
				},
			},
			expected: "",
			wantErr:  false,
		},
		{
			name: "regex index out of range",
			path: "photos/2023/vacation/IMG_001.jpg",
			criteria: utils.TCriteria{
				Key: "originalPath",
				Regex: &utils.TRegex{
					Key:   `photos/(\d{4})/`,
					Index: 3, // Only 1 capture group available
				},
			},
			expected: "",
			wantErr:  true,
		},
		{
			name: "regex extract filename from path",
			path: "photos/2023/vacation/IMG_001.jpg",
			criteria: utils.TCriteria{
				Key: "originalPath",
				Regex: &utils.TRegex{
					Key:   `([^/]+)\.jpg$`,
					Index: 1, // Filename without extension
				},
			},
			expected: "IMG_001",
			wantErr:  false,
		},
		{
			name: "regex full path match",
			path: "photos/2023/vacation/IMG_001.jpg",
			criteria: utils.TCriteria{
				Key: "originalPath",
				Regex: &utils.TRegex{
					Key:   `photos/\d{4}/vacation/.*`,
					Index: 0, // Full match
				},
			},
			expected: "photos/2023/vacation/IMG_001.jpg",
			wantErr:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			asset := utils.TAsset{OriginalPath: tc.path}
			result, _, err := extractOriginalPath(asset, tc.criteria)
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
** Test applyCriteria with regex-based grouping
************************************************************************************************/
func TestApplyCriteriaWithRegex(t *testing.T) {
	tests := []struct {
		name     string
		assets   []utils.TAsset
		criteria []utils.TCriteria
		want     int // number of groups
	}{
		{
			name: "group by date extracted with regex",
			assets: []utils.TAsset{
				{
					OriginalFileName: "PXL_20230503_152823814.jpg",
					OriginalPath:     "photos/2023/vacation/PXL_20230503_152823814.jpg",
				},
				{
					OriginalFileName: "PXL_20230503_152830456.jpg",
					OriginalPath:     "photos/2023/vacation/PXL_20230503_152830456.jpg",
				},
				{
					OriginalFileName: "PXL_20230504_091234567.jpg",
					OriginalPath:     "photos/2023/vacation/PXL_20230504_091234567.jpg",
				},
				{
					OriginalFileName: "PXL_20230504_091240789.jpg",
					OriginalPath:     "photos/2023/vacation/PXL_20230504_091240789.jpg",
				},
			},
			criteria: []utils.TCriteria{
				{
					Key: "originalFileName",
					Regex: &utils.TRegex{
						Key:   `PXL_(\d{8})_\d{9}`,
						Index: 1, // Extract date
					},
				},
			},
			want: 2, // Two groups: 20230503 (2 files) and 20230504 (2 files)
		},
		{
			name: "group by path year with regex",
			assets: []utils.TAsset{
				{
					OriginalFileName: "IMG_001.jpg",
					OriginalPath:     "photos/2023/vacation/IMG_001.jpg",
				},
				{
					OriginalFileName: "IMG_002.jpg",
					OriginalPath:     "photos/2023/work/IMG_002.jpg",
				},
				{
					OriginalFileName: "IMG_003.jpg",
					OriginalPath:     "photos/2024/family/IMG_003.jpg",
				},
				{
					OriginalFileName: "IMG_004.jpg",
					OriginalPath:     "photos/2024/vacation/IMG_004.jpg",
				},
			},
			criteria: []utils.TCriteria{
				{
					Key: "originalPath",
					Regex: &utils.TRegex{
						Key:   `photos/(\d{4})/`,
						Index: 1, // Extract year
					},
				},
			},
			want: 2, // Two groups: 2023 (2 files) and 2024 (2 files)
		},
		{
			name: "complex grouping with filename regex and path regex",
			assets: []utils.TAsset{
				{
					OriginalFileName: "PXL_20230503_152823814.jpg",
					OriginalPath:     "photos/2023/vacation/PXL_20230503_152823814.jpg",
				},
				{
					OriginalFileName: "PXL_20230503_152830456.jpg",
					OriginalPath:     "photos/2023/vacation/PXL_20230503_152830456.jpg",
				},
				{
					OriginalFileName: "PXL_20230503_091234567.jpg",
					OriginalPath:     "photos/2023/work/PXL_20230503_091234567.jpg",
				},
				{
					OriginalFileName: "PXL_20230503_091240789.jpg",
					OriginalPath:     "photos/2023/work/PXL_20230503_091240789.jpg",
				},
			},
			criteria: []utils.TCriteria{
				{
					Key: "originalFileName",
					Regex: &utils.TRegex{
						Key:   `PXL_(\d{8})_\d{9}`,
						Index: 1, // Extract date
					},
				},
				{
					Key: "originalPath",
					Regex: &utils.TRegex{
						Key:   `photos/\d{4}/([^/]+)/`,
						Index: 1, // Extract folder name (vacation/work)
					},
				},
			},
			want: 2, // Two groups: (20230503, vacation) and (20230503, work)
		},
		{
			name: "regex no match results in no grouping",
			assets: []utils.TAsset{
				{
					OriginalFileName: "random_file_001.jpg",
					OriginalPath:     "photos/random/random_file_001.jpg",
				},
				{
					OriginalFileName: "random_file_002.jpg",
					OriginalPath:     "photos/random/random_file_002.jpg",
				},
			},
			criteria: []utils.TCriteria{
				{
					Key: "originalFileName",
					Regex: &utils.TRegex{
						Key:   `PXL_(\d{8})_\d{9}`,
						Index: 1, // No match for this pattern
					},
				},
			},
			want: 0, // No groups since regex doesn't match
		},
		{
			name: "mixed regex success and failure",
			assets: []utils.TAsset{
				{
					OriginalFileName: "PXL_20230503_152823814.jpg",
					OriginalPath:     "photos/2023/vacation/PXL_20230503_152823814.jpg",
				},
				{
					OriginalFileName: "PXL_20230503_152830456.jpg",
					OriginalPath:     "photos/2023/vacation/PXL_20230503_152830456.jpg",
				},
				{
					OriginalFileName: "random_file.jpg",
					OriginalPath:     "photos/2023/vacation/random_file.jpg",
				},
			},
			criteria: []utils.TCriteria{
				{
					Key: "originalFileName",
					Regex: &utils.TRegex{
						Key:   `PXL_(\d{8})_\d{9}`,
						Index: 1, // Extract date, but won't match random_file.jpg
					},
				},
			},
			want: 1, // Only one group with the two PXL files (same date: 20230503)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up test criteria in environment
			t.Setenv("CRITERIA", mustMarshalJSON(t, tt.criteria))

			groups, err := StackBy(tt.assets, "", "", "", logrus.New())
			require.NoError(t, err)
			assert.Equal(t, tt.want, len(groups), "Expected %d groups but got %d", tt.want, len(groups))

			// Additional validation for specific test cases
			if tt.name == "group by date extracted with regex" && len(groups) == 2 {
				// Verify groups are correctly formed
				dates := make(map[string]int)
				for _, group := range groups {
					// All files in a group should have the same extracted date
					firstAsset := group[0]
					extractedDate, _, err := extractOriginalFileName(firstAsset, tt.criteria[0])
					require.NoError(t, err)
					dates[extractedDate] = len(group)
				}
				assert.Contains(t, dates, "20230503")
				assert.Contains(t, dates, "20230504")
				assert.Equal(t, 2, dates["20230503"]) // Two files with date 20230503
				assert.Equal(t, 2, dates["20230504"]) // Two files with date 20230504
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

/************************************************************************************************
** Test extractOriginalFileName with regex promotion functionality
************************************************************************************************/
func TestExtractOriginalFileNameRegexPromotion(t *testing.T) {
	type testCase struct {
		name         string
		filename     string
		criteria     utils.TCriteria
		expectedKey  string
		expectedProm string
		wantErr      bool
	}

	promoteIndex := 3
	tests := []testCase{
		{
			name:     "regex with promote_index extracts both values",
			filename: "PXL_20230503_152823814_MP.jpg",
			criteria: utils.TCriteria{
				Key: "originalFileName",
				Regex: &utils.TRegex{
					Key:          `PXL_(\d{8})_(\d{9})(_\w+)?\.jpg`,
					Index:        1,             // Date for grouping
					PromoteIndex: &promoteIndex, // Suffix for promotion
				},
			},
			expectedKey:  "20230503",
			expectedProm: "_MP",
			wantErr:      false,
		},
		{
			name:     "regex with promote_index - no suffix",
			filename: "PXL_20230503_152823814.jpg",
			criteria: utils.TCriteria{
				Key: "originalFileName",
				Regex: &utils.TRegex{
					Key:          `PXL_(\d{8})_(\d{9})(_\w+)?\.jpg`,
					Index:        1,
					PromoteIndex: &promoteIndex,
				},
			},
			expectedKey:  "20230503",
			expectedProm: "", // Optional group not matched
			wantErr:      false,
		},
		{
			name:     "regex without promote_index returns empty promotion",
			filename: "PXL_20230503_152823814_MP.jpg",
			criteria: utils.TCriteria{
				Key: "originalFileName",
				Regex: &utils.TRegex{
					Key:   `PXL_(\d{8})_(\d{9})(_\w+)?\.jpg`,
					Index: 1,
				},
			},
			expectedKey:  "20230503",
			expectedProm: "",
			wantErr:      false,
		},
		{
			name:     "promote_index out of range",
			filename: "PXL_20230503_152823814.jpg",
			criteria: utils.TCriteria{
				Key: "originalFileName",
				Regex: &utils.TRegex{
					Key:          `PXL_(\d{8})_(\d{9})\.jpg`,
					Index:        1,
					PromoteIndex: &promoteIndex, // Index 3 doesn't exist
				},
			},
			expectedKey:  "",
			expectedProm: "",
			wantErr:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			asset := utils.TAsset{OriginalFileName: tc.filename}
			key, prom, err := extractOriginalFileName(asset, tc.criteria)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedKey, key)
				assert.Equal(t, tc.expectedProm, prom)
			}
		})
	}
}

/************************************************************************************************
** Test stacking with regex promotion
************************************************************************************************/
func TestStackByWithRegexPromotion(t *testing.T) {
	promoteIndex := 3

	tests := []struct {
		name     string
		assets   []utils.TAsset
		criteria []utils.TCriteria
		expected []string // Expected order of filenames after sorting
	}{
		{
			name: "regex promotion with promote_keys",
			assets: []utils.TAsset{
				{ID: "1", OriginalFileName: "PXL_20230503_152823814.jpg"},
				{ID: "2", OriginalFileName: "PXL_20230503_152823814_edit.jpg"},
				{ID: "3", OriginalFileName: "PXL_20230503_152823814_MP.jpg"},
				{ID: "4", OriginalFileName: "PXL_20230503_152823814_crop.jpg"},
			},
			criteria: []utils.TCriteria{
				{
					Key: "originalFileName",
					Regex: &utils.TRegex{
						Key:          `PXL_(\d{8})_(\d{9})(_\w+)?\.jpg`,
						Index:        1,                                     // Group by date
						PromoteIndex: &promoteIndex,                         // Promote by suffix
						PromoteKeys:  []string{"_MP", "_edit", "_crop", ""}, // Order of promotion
					},
				},
			},
			expected: []string{
				"PXL_20230503_152823814_MP.jpg",   // _MP has highest priority
				"PXL_20230503_152823814_edit.jpg", // _edit is second
				"PXL_20230503_152823814_crop.jpg", // _crop is third
				"PXL_20230503_152823814.jpg",      // empty suffix is last
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up test criteria in environment
			t.Setenv("CRITERIA", mustMarshalJSON(t, tt.criteria))

			// Run stacking
			logger := logrus.New()
			stacks, err := StackBy(tt.assets, "", "", "", logger)
			require.NoError(t, err)
			require.Len(t, stacks, 1, "Expected exactly one stack")

			// Check the order of assets in the stack
			stack := stacks[0]
			require.Len(t, stack, len(tt.expected))

			for i, expectedFilename := range tt.expected {
				assert.Equal(t, expectedFilename, stack[i].OriginalFileName,
					"Asset at position %d should be %s", i, expectedFilename)
			}
		})
	}
}

/************************************************************************************************
** Test stacking with regex promotion prioritizing unedited files (empty string first)
************************************************************************************************/
func TestStackByWithRegexPromotion_UneditedFirst(t *testing.T) {
	logger := logrus.New()

	assets := []utils.TAsset{
		{ID: "1", OriginalFileName: "IMG_1234.jpg"},
		{ID: "2", OriginalFileName: "IMG_1234_edit.jpg"},
		{ID: "3", OriginalFileName: "IMG_1234-edited.jpg"},
		{ID: "4", OriginalFileName: "IMG_1234.cropped.jpg"},
		{ID: "5", OriginalFileName: "IMG_1234_crop.jpg"},
	}

	// Regex captures:
	//  - Group 1: base name without suffix and extension (e.g., IMG_1234)
	//  - Group 3: the edit token (crop|cropped|edit|edited) if present, else empty string
	//  - Group 4: file extension
	criteriaJSON := `[{"key":"originalFileName","regex":{"key":"^(.*?)([._-](crop|cropped|edit|edited).*)?\\.([^.]+)$","index":1,"promote_index":3,"promote_keys":["","edited","edit","cropped","crop"]}}]`
	t.Setenv("CRITERIA", criteriaJSON)

	stacks, err := StackBy(assets, "", "", "", logger)
	require.NoError(t, err)
	require.Len(t, stacks, 1, "Expected exactly one stack for same base name")

	stack := stacks[0]
	require.Len(t, stack, len(assets))

	// Expect unedited first, then edited variants in the order of promote_keys
	expected := []string{
		"IMG_1234.jpg",         // empty promote value
		"IMG_1234-edited.jpg",  // "edited"
		"IMG_1234_edit.jpg",    // "edit"
		"IMG_1234.cropped.jpg", // "cropped"
		"IMG_1234_crop.jpg",    // "crop"
	}

	for i, want := range expected {
		assert.Equal(t, want, stack[i].OriginalFileName, "position %d should be %s", i, want)
	}
}

/************************************************************************************************
** Test applyCriteriaWithPromote handles multiple regex promotions on same key without collision
************************************************************************************************/
func TestApplyCriteriaWithPromoteMultipleRegex(t *testing.T) {
	asset := utils.TAsset{
		ID:               "test-asset",
		OriginalFileName: "IMG_001.jpg",
	}

	// Two criteria with same key but different promotion configurations
	criteria := []utils.TCriteria{
		{
			Key: "originalFileName",
			Regex: &utils.TRegex{
				Key:          "^(IMG)_(\\d+)",
				Index:        1,                        // Extract "IMG" 
				PromoteIndex: &[]int{1}[0],             // Use captured group 1 for promotion
				PromoteKeys:  []string{"IMG", "PXL"},
			},
		},
		{
			Key: "originalFileName", 
			Regex: &utils.TRegex{
				Key:          "^([A-Z]+)_(\\d+)",
				Index:        2,                        // Extract "001"
				PromoteIndex: &[]int{2}[0],             // Use captured group 2 for promotion  
				PromoteKeys:  []string{"001", "002"},
			},
		},
	}

	values, promoteValues, err := applyCriteriaWithPromote(asset, criteria)
	require.NoError(t, err)

	// Verify both criteria extracted values
	assert.Equal(t, []string{"IMG", "001"}, values)

	// Verify both promotion values are stored with unique keys (no collision)
	assert.Len(t, promoteValues, 2, "Should have promotion values from both criteria")
	assert.Equal(t, "IMG", promoteValues["originalFileName:0"], "First criteria should store IMG")
	assert.Equal(t, "001", promoteValues["originalFileName:1"], "Second criteria should store 001")

	// Verify that neither promotion value overwrote the other
	assert.NotEqual(t, promoteValues["originalFileName:0"], promoteValues["originalFileName:1"],
		"Promotion values should be different (no collision)")
}


/************************************************************************************************
** Test unified PrecompileRegexes function with different source types
************************************************************************************************/
func TestPrecompileRegexes(t *testing.T) {
	tests := []struct {
		name           string
		criteriaSource interface{}
		expectError    bool
		errorPart      string
	}{
		{
			name: "legacy criteria slice with valid regex",
			criteriaSource: []utils.TCriteria{
				{Key: "originalFileName", Regex: &utils.TRegex{Key: "^IMG_.*"}},
			},
			expectError: false,
		},
		{
			name: "legacy criteria slice with invalid regex",
			criteriaSource: []utils.TCriteria{
				{Key: "originalFileName", Regex: &utils.TRegex{Key: "("}},
			},
			expectError: true,
			errorPart:   "failed to compile regex",
		},
		{
			name: "criteria groups with valid regex",
			criteriaSource: []utils.TCriteriaGroup{
				{
					Criteria: []utils.TCriteria{
						{Key: "originalFileName", Regex: &utils.TRegex{Key: "^IMG_.*"}},
					},
				},
			},
			expectError: false,
		},
		{
			name: "single criteria with valid regex",
			criteriaSource: utils.TCriteria{
				Key:   "originalFileName",
				Regex: &utils.TRegex{Key: "^IMG_.*"},
			},
			expectError: false,
		},
		{
			name: "expression with valid regex",
			criteriaSource: &utils.TCriteriaExpression{
				Criteria: &utils.TCriteria{
					Key:   "originalFileName",
					Regex: &utils.TRegex{Key: "^IMG_.*"},
				},
			},
			expectError: false,
		},
		{
			name:           "unsupported type",
			criteriaSource: "invalid_type",
			expectError:    true,
			errorPart:      "unsupported criteria source type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := PrecompileRegexes(tt.criteriaSource)
			
			if tt.expectError {
				assert.Error(t, err, "Expected error")
				if tt.errorPart != "" {
					assert.Contains(t, err.Error(), tt.errorPart)
				}
			} else {
				assert.NoError(t, err, "Expected no error")
			}
		})
	}
}

/************************************************************************************************
** Test ParseCriteria function (currently 0% coverage)
************************************************************************************************/
func TestParseCriteria(t *testing.T) {
	tests := []struct {
		name        string
		criteria    string
		expectMode  string
		expectError bool
	}{
		{
			name:        "valid legacy criteria",
			criteria:    `[{"key":"originalFileName","split":{"delimiters":["~","."],"index":0}}]`,
			expectMode:  "legacy",
			expectError: false,
		},
		{
			name:        "valid advanced criteria",
			criteria:    `{"mode":"advanced","groups":[{"criteria":[{"key":"originalFileName"}]}]}`,
			expectMode:  "advanced",
			expectError: false,
		},
		{
			name:        "empty criteria string uses default",
			criteria:    "",
			expectMode:  "legacy",
			expectError: false,
		},
		{
			name:        "invalid JSON returns error",
			criteria:    `{"invalid":json}`,
			expectMode:  "",
			expectError: true,
		},
		{
			name:        "valid expression criteria",
			criteria:    `{"mode":"expression","expression":{"criteria":{"key":"originalFileName"}}}`,
			expectMode:  "expression",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := ParseCriteria(tt.criteria)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectMode, config.Mode)
			}
		})
	}
}
