package stacker

import (
	"testing"
	"time"

	"github.com/majorfi/immich-stack/pkg/utils"
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
			groups, err := StackBy(assets, "", "", "")

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
