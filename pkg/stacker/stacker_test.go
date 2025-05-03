package stacker

import (
	"os"
	"testing"
	"time"

	"github.com/majorfi/immich-stack/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/************************************************************************************************
** Test helper functions and types
************************************************************************************************/

func assetFactory(filename string, dateTime time.Time) utils.TAsset {
	return utils.TAsset{
		OriginalFileName: filename,
		LocalDateTime:    dateTime.Format(time.RFC3339),
	}
}

/************************************************************************************************
** Test cases for parent criteria
************************************************************************************************/

func TestSortStack(t *testing.T) {
	tests := []struct {
		name          string
		inputOrder    []string
		expectedOrder []string
		promoteStr    string
		promoteExt    string
	}{
		{
			name: "alphabetical sort",
			inputOrder: []string{
				"IMG_2482.xyz",
				"IMG_2482.XYZ",
				"IMG_2482.xyzz",
			},
			expectedOrder: []string{
				"IMG_2482.XYZ",
				"IMG_2482.xyz",
				"IMG_2482.xyzz",
			},
		},
		{
			name: "prioritize jpg jpeg png",
			inputOrder: []string{
				"IMG_2482.xyz",
				"IMG_2482.jpg",
				"IMG_2482.png",
				"IMG_2482.abc",
				"IMG_2482.jpeg",
			},
			expectedOrder: []string{
				"IMG_2482.jpeg",
				"IMG_2482.jpg",
				"IMG_2482.png",
				"IMG_2482.abc",
				"IMG_2482.xyz",
			},
		},
		{
			name: "promote override",
			inputOrder: []string{
				"testIMG_2482.xyz",
				"IMG_2482.jpg",
				"IMG_2482.test.png",
			},
			expectedOrder: []string{
				"IMG_2482.test.png",
				"testIMG_2482.xyz",
				"IMG_2482.jpg",
			}, // promote string first, then extension rank, then alpha
			promoteStr: "test",
		},
		{
			name: "filename promote takes priority over extension promote",
			inputOrder: []string{
				"L1010229.JPG",
				"L1010229.edit.jpg",
				"L1010229.DNG",
			},
			expectedOrder: []string{
				"L1010229.edit.jpg",
				"L1010229.JPG",
				"L1010229.DNG",
			},
			promoteStr: "edit",
			promoteExt: ".jpg,.dng",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			assets := make([]utils.TAsset, len(tt.inputOrder))
			for i, f := range tt.inputOrder {
				assets[i] = assetFactory(f, time.Now())
			}

			if tt.promoteStr != "" {
				os.Setenv("PARENT_FILENAME_PROMOTE", tt.promoteStr)
				defer os.Unsetenv("PARENT_FILENAME_PROMOTE")
			}
			if tt.promoteExt != "" {
				os.Setenv("PARENT_EXT_PROMOTE", tt.promoteExt)
				defer os.Unsetenv("PARENT_EXT_PROMOTE")
			}

			// Act
			result := sortStack(assets, tt.promoteStr, tt.promoteExt)

			// Assert
			expectedAssets := make([]utils.TAsset, len(tt.expectedOrder))
			for i, f := range tt.expectedOrder {
				expectedAssets[i] = assetFactory(f, time.Now())
			}
			assert.Equal(t, expectedAssets, result)
		})
	}
}

/************************************************************************************************
** Test cases for stackBy
************************************************************************************************/

func TestStackBy(t *testing.T) {
	tests := []struct {
		name           string
		assets         []utils.TAsset
		expectedGroups int
		skipMatchMiss  bool
	}{
		{
			name: "different filenames",
			assets: []utils.TAsset{
				assetFactory("test1.jpg", time.Now()),
				assetFactory("test2.jpg", time.Now()),
			},
			expectedGroups: 0, // No groups since they don't match criteria
		},
		{
			name: "same filename different datetime",
			assets: []utils.TAsset{
				assetFactory("test.jpg", time.Now()),
				assetFactory("test.jpg", time.Now().Add(time.Hour)),
			},
			expectedGroups: 0, // No groups since the datetime is different
		},
		{
			name: "empty key handling",
			assets: []utils.TAsset{
				assetFactory("test.jpg", time.Now()),
				assetFactory("test.jpg", time.Time{}),
			},
			expectedGroups: 0, // No groups since the datetime is different
			skipMatchMiss:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			if tt.skipMatchMiss {
				os.Setenv("SKIP_MATCH_MISS", "true")
				defer os.Unsetenv("SKIP_MATCH_MISS")
			}

			// Act
			groups, err := StackBy(tt.assets, "", "", "")

			// Assert
			if tt.skipMatchMiss {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedGroups, len(groups))
			} else if tt.expectedGroups == 0 {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedGroups, len(groups))
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedGroups, len(groups))
			}
		})
	}
}
