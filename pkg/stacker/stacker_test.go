package stacker

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/majorfi/immich-stack/pkg/utils"
	"github.com/sirupsen/logrus"
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
			promoteExt: ".jpeg,.jpg,.png",
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
			result := sortStack(assets, tt.promoteStr, tt.promoteExt, []string{"~", "."})

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
			groups, err := StackBy(tt.assets, "", "", "", logrus.New())

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

func TestSortStack_SonyBurstPhotos(t *testing.T) {
	// Test Sony burst photo sorting issue from GitHub issue #18
	// Files should be sorted based on their numeric index (0000, 0001, 0002, 0003)
	// not by searching for those patterns anywhere in the filename

	// Test case where "0001" might match in the timestamp part BURST20180828114700954
	// The string "0001" appears in "114700" part of the timestamp
	// With the old contains-based matching, this would cause incorrect ordering
	stack := []utils.TAsset{
		{
			ID:               "7a733c19-a588-433c-9cd8-d621071e47c3",
			OriginalFileName: "DSCPDC_0000_BURST20180828114700954.JPG",
			LocalDateTime:    "2018-08-28T11:47:00.460Z",
		},
		{
			ID:               "26147f09-f6be-44c4-92e7-82b45313dc3c",
			OriginalFileName: "DSCPDC_0002_BURST20180828114700954.JPG",
			LocalDateTime:    "2018-08-28T11:47:00.758Z",
		},
		{
			ID:               "e964fcd7-8889-491d-aa08-ca54cfd716ab",
			OriginalFileName: "DSCPDC_0003_BURST20180828114700954_COVER.JPG",
			LocalDateTime:    "2018-08-28T11:47:00.910Z",
		},
		{
			ID:               "2dd4c37a-bc68-4f09-8150-bea904f30f51",
			OriginalFileName: "DSCPDC_0001_BURST20180828114700954.JPG",
			LocalDateTime:    "2018-08-28T11:47:00.608Z",
		},
	}

	// Test with promote list that should prefer 0000, then 0001, etc.
	parentFilenamePromote := "0000,0001,0002,0003"
	parentExtPromote := ""
	delimiters := []string{} // No delimiters, like in user's case

	sorted := sortStack(stack, parentFilenamePromote, parentExtPromote, delimiters)

	// Print actual order for debugging
	t.Logf("Sorted order:")
	for i, asset := range sorted {
		t.Logf("  [%d] %s", i, asset.OriginalFileName)
	}

	// Expected order: 0000, 0001, 0002, 0003
	assert.Equal(t, "DSCPDC_0000_BURST20180828114700954.JPG", sorted[0].OriginalFileName)
	assert.Equal(t, "DSCPDC_0001_BURST20180828114700954.JPG", sorted[1].OriginalFileName)
	assert.Equal(t, "DSCPDC_0002_BURST20180828114700954.JPG", sorted[2].OriginalFileName)
	assert.Equal(t, "DSCPDC_0003_BURST20180828114700954_COVER.JPG", sorted[3].OriginalFileName)
}

func TestStackBy_SonyBurstWithRegex(t *testing.T) {
	// Test the full stacking process with regex criteria as used by the user
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	assets := []utils.TAsset{
		{
			ID:               "7a733c19-a588-433c-9cd8-d621071e47c3",
			OriginalFileName: "DSCPDC_0000_BURST20180828114700954.JPG",
			LocalDateTime:    "2018-08-28T11:47:00.460Z",
		},
		{
			ID:               "2dd4c37a-bc68-4f09-8150-bea904f30f51",
			OriginalFileName: "DSCPDC_0001_BURST20180828114700954.JPG",
			LocalDateTime:    "2018-08-28T11:47:00.608Z",
		},
		{
			ID:               "26147f09-f6be-44c4-92e7-82b45313dc3c",
			OriginalFileName: "DSCPDC_0002_BURST20180828114700954.JPG",
			LocalDateTime:    "2018-08-28T11:47:00.758Z",
		},
		{
			ID:               "e964fcd7-8889-491d-aa08-ca54cfd716ab",
			OriginalFileName: "DSCPDC_0003_BURST20180828114700954_COVER.JPG",
			LocalDateTime:    "2018-08-28T11:47:00.910Z",
		},
	}

	// Set up environment variable for criteria
	t.Setenv("CRITERIA", `[{"key":"originalFileName","regex":{"key":"DSCPDC_(\\d{4})_(BURST\\d{17})(_COVER)?.JPG","index":2}}]`)

	parentFilenamePromote := "0000,0001,0002,0003"
	parentExtPromote := ""

	stacks, err := StackBy(assets, "", parentFilenamePromote, parentExtPromote, logger)
	assert.NoError(t, err)
	assert.Len(t, stacks, 1)

	// Check that the stack is ordered correctly
	stack := stacks[0]
	assert.Equal(t, "DSCPDC_0000_BURST20180828114700954.JPG", stack[0].OriginalFileName)
	assert.Equal(t, "DSCPDC_0001_BURST20180828114700954.JPG", stack[1].OriginalFileName)
	assert.Equal(t, "DSCPDC_0002_BURST20180828114700954.JPG", stack[2].OriginalFileName)
	assert.Equal(t, "DSCPDC_0003_BURST20180828114700954_COVER.JPG", stack[3].OriginalFileName)
}

func TestSortStack_BurstPhotoWithShuffledInput(t *testing.T) {
	// Test burst photo handling with shuffled input
	stack := []utils.TAsset{
		{
			ID:               "2dd4c37a-bc68-4f09-8150-bea904f30f51",
			OriginalFileName: "DSCPDC_0001_BURST20180828114700954.JPG",
			LocalDateTime:    "2018-08-28T11:47:00.608Z",
		},
		{
			ID:               "e964fcd7-8889-491d-aa08-ca54cfd716ab",
			OriginalFileName: "DSCPDC_0003_BURST20180828114700954_COVER.JPG",
			LocalDateTime:    "2018-08-28T11:47:00.910Z",
		},
		{
			ID:               "7a733c19-a588-433c-9cd8-d621071e47c3",
			OriginalFileName: "DSCPDC_0000_BURST20180828114700954.JPG",
			LocalDateTime:    "2018-08-28T11:47:00.460Z",
		},
		{
			ID:               "26147f09-f6be-44c4-92e7-82b45313dc3c",
			OriginalFileName: "DSCPDC_0002_BURST20180828114700954.JPG",
			LocalDateTime:    "2018-08-28T11:47:00.758Z",
		},
	}

	// Test with promote list that should be detected as burst numbers
	parentFilenamePromote := "0000,0001,0002,0003"
	parentExtPromote := ""
	delimiters := []string{} // No delimiters, like in user's case

	sorted := sortStack(stack, parentFilenamePromote, parentExtPromote, delimiters)

	t.Logf("Sorted order with burst handling:")
	for i, asset := range sorted {
		t.Logf("  [%d] %s", i, asset.OriginalFileName)
	}

	// Should be in correct order
	assert.Equal(t, "DSCPDC_0000_BURST20180828114700954.JPG", sorted[0].OriginalFileName)
	assert.Equal(t, "DSCPDC_0001_BURST20180828114700954.JPG", sorted[1].OriginalFileName)
	assert.Equal(t, "DSCPDC_0002_BURST20180828114700954.JPG", sorted[2].OriginalFileName)
	assert.Equal(t, "DSCPDC_0003_BURST20180828114700954_COVER.JPG", sorted[3].OriginalFileName)
}

func TestDetectPromoteMatchMode(t *testing.T) {
	tests := []struct {
		name           string
		promoteList    []string
		sampleFilename string
		expectedMode   string
	}{
		{
			name:           "Burst photo with 4-digit numbers",
			promoteList:    []string{"0000", "0001", "0002", "0003"},
			sampleFilename: "DSCPDC_0001_BURST20180828114700954.JPG",
			expectedMode:   "sequence",
		},
		{
			name:           "Burst photo with different sequence",
			promoteList:    []string{"img1", "img2", "img3"},
			sampleFilename: "DSCPDC_img2_BURST20180828114700954.JPG",
			expectedMode:   "sequence",
		},
		{
			name:           "Regular promote list",
			promoteList:    []string{"edit", "crop", "hdr"},
			sampleFilename: "IMG_1234_edit.jpg",
			expectedMode:   "contains",
		},
		{
			name:           "Mixed promote list",
			promoteList:    []string{"0001", "edit", "crop"},
			sampleFilename: "DSCPDC_0001_BURST20180828114700954.JPG",
			expectedMode:   "contains",
		},
		{
			name:           "Sequence pattern but unrelated filename",
			promoteList:    []string{"0000", "0001", "0002", "0003"},
			sampleFilename: "IMG_1234.jpg",
			expectedMode:   "contains",
		},
		{
			name:           "Sequence pattern with matching structure",
			promoteList:    []string{"IMG_0001", "IMG_0002", "IMG_0003"},
			sampleFilename: "IMG_0045.jpg",
			expectedMode:   "sequence",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode := detectPromoteMatchMode(tt.promoteList, tt.sampleFilename)
			assert.Equal(t, tt.expectedMode, mode)
		})
	}
}

func TestGetPromoteIndexWithMatchMode(t *testing.T) {
	promoteList := []string{"0000", "0001", "0002", "0003"}

	tests := []struct {
		filename    string
		matchMode   string
		expectedIdx int
	}{
		{
			filename:    "DSCPDC_0000_BURST20180828114700954.JPG",
			matchMode:   "sequence",
			expectedIdx: 0,
		},
		{
			filename:    "DSCPDC_0001_BURST20180828114700954.JPG",
			matchMode:   "sequence",
			expectedIdx: 1,
		},
		{
			filename:    "DSCPDC_0010_BURST20180828114700954.JPG", // Higher number
			matchMode:   "sequence",
			expectedIdx: 10, // Should return the numeric value
		},
		{
			filename:    "IMG_0001_OTHER.JPG", // Different pattern
			matchMode:   "sequence",
			expectedIdx: 1, // Should still match if the number is in position 1
		},
		{
			filename:    "BURST_0001_20180828.JPG", // Number not in expected position
			matchMode:   "sequence",
			expectedIdx: 1, // Will match in position 1
		},
		{
			filename:    "DSCPDC_0001_BURST20180828114700954.JPG",
			matchMode:   "contains",
			expectedIdx: 1, // Standard contains matching
		},
	}

	for _, tt := range tests {
		t.Run(tt.filename+"_"+tt.matchMode, func(t *testing.T) {
			idx := getPromoteIndexWithMode(tt.filename, promoteList, tt.matchMode)
			assert.Equal(t, tt.expectedIdx, idx, "For filename %s with mode %s", tt.filename, tt.matchMode)
		})
	}
}

func TestGetPromoteIndexWithSequencePatterns(t *testing.T) {
	tests := []struct {
		name        string
		promoteList []string
		filename    string
		expectedIdx int
	}{
		{
			name:        "Prefix pattern img1,img2,img3",
			promoteList: []string{"img1", "img2", "img3"},
			filename:    "PHOTO_img2_BURST123.jpg",
			expectedIdx: 1, // index in promote list (img2 is at index 1)
		},
		{
			name:        "Suffix pattern 1a,2a,3a",
			promoteList: []string{"1a", "2a", "3a"},
			filename:    "PHOTO_2a_BURST123.jpg",
			expectedIdx: 1, // index in promote list (2a is at index 1)
		},
		{
			name:        "Complex pattern photo_001_final",
			promoteList: []string{"photo_001_final", "photo_002_final", "photo_003_final"},
			filename:    "PREFIX_photo_002_final_BURST.jpg",
			expectedIdx: 1, // index in promote list (photo_002_final is at index 1)
		},
		{
			name:        "Extended sequence beyond promote list",
			promoteList: []string{"0000", "0001", "0002", "0003"},
			filename:    "DSCPDC_0010_BURST20180828114700954.JPG",
			expectedIdx: 10, // Should handle numbers beyond the list
		},
		{
			name:        "Very high sequence number",
			promoteList: []string{"0000", "0001", "0002"},
			filename:    "DSCPDC_0999_BURST20180828114700954.JPG",
			expectedIdx: 999, // Should handle any number in the sequence
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Detect mode first
			mode := detectPromoteMatchMode(tt.promoteList, tt.filename)
			assert.Equal(t, "sequence", mode, "Should detect sequence mode")

			// Test the index
			idx := getPromoteIndexWithMode(tt.filename, tt.promoteList, mode)
			assert.Equal(t, tt.expectedIdx, idx, "For filename %s with promoteList %v", tt.filename, tt.promoteList)
		})
	}
}

func TestShouldUseSequenceMatchingEdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		filename       string
		promoteList    []string
		expectedResult bool
	}{
		{
			name:           "Empty promote list",
			filename:       "DSCPDC_0001_BURST.jpg",
			promoteList:    []string{},
			expectedResult: false,
		},
		{
			name:           "Non-sequence promote list",
			filename:       "IMG_edit.jpg",
			promoteList:    []string{"edit", "crop", "hdr"},
			expectedResult: false,
		},
		{
			name:           "Sequence in promote but wrong prefix",
			filename:       "PHOTO_0001.jpg",
			promoteList:    []string{"IMG_0001", "IMG_0002", "IMG_0003"},
			expectedResult: false,
		},
		{
			name:           "Sequence in promote with matching prefix",
			filename:       "IMG_9999.jpg",
			promoteList:    []string{"IMG_0001", "IMG_0002", "IMG_0003"},
			expectedResult: true,
		},
		{
			name:           "Pure number sequence with burst photo",
			filename:       "DSCPDC_0999_BURST20180828114700954.JPG",
			promoteList:    []string{"0000", "0001", "0002", "0003"},
			expectedResult: true,
		},
		{
			name:           "Pure number sequence with non-matching file",
			filename:       "vacation_photo.jpg",
			promoteList:    []string{"0000", "0001", "0002", "0003"},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldUseSequenceMatching(tt.filename, tt.promoteList)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestSequenceKeywordHandling(t *testing.T) {
	tests := []struct {
		name        string
		promoteList []string
		filenames   []string
		expected    []string
	}{
		{
			name:        "COVER first then sequence",
			promoteList: []string{"COVER", "sequence"},
			filenames: []string{
				"DSCPDC_0002_BURST20180828114700954.JPG",
				"DSCPDC_0000_BURST20180828114700954.JPG",
				"DSCPDC_0003_BURST20180828114700954_COVER.JPG",
				"DSCPDC_0001_BURST20180828114700954.JPG",
			},
			expected: []string{
				"DSCPDC_0003_BURST20180828114700954_COVER.JPG",
				"DSCPDC_0000_BURST20180828114700954.JPG",
				"DSCPDC_0001_BURST20180828114700954.JPG",
				"DSCPDC_0002_BURST20180828114700954.JPG",
			},
		},
		{
			name:        "Edit first then sequence with 4 digits",
			promoteList: []string{"edit", "sequence:4"},
			filenames: []string{
				"IMG_0002.jpg",
				"IMG_0001_edit.jpg",
				"IMG_0003.jpg",
				"IMG_0001.jpg",
			},
			expected: []string{
				"IMG_0001_edit.jpg",
				"IMG_0001.jpg",
				"IMG_0002.jpg",
				"IMG_0003.jpg",
			},
		},
		{
			name:        "Sequence with prefix pattern",
			promoteList: []string{"sequence:IMG_"},
			filenames: []string{
				"PHOTO_0001.jpg",
				"IMG_0002.jpg",
				"IMG_0001.jpg",
				"PHOTO_0002.jpg",
			},
			expected: []string{
				"IMG_0001.jpg",
				"IMG_0002.jpg",
				"PHOTO_0001.jpg",
				"PHOTO_0002.jpg",
			},
		},
		{
			name:        "Only sequence keyword",
			promoteList: []string{"sequence"},
			filenames: []string{
				"DSCPDC_0002_BURST.jpg",
				"DSCPDC_0000_BURST.jpg",
				"DSCPDC_0001_BURST.jpg",
			},
			expected: []string{
				"DSCPDC_0000_BURST.jpg",
				"DSCPDC_0001_BURST.jpg",
				"DSCPDC_0002_BURST.jpg",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create assets from filenames
			assets := make([]utils.TAsset, len(tt.filenames))
			for i, filename := range tt.filenames {
				assets[i] = utils.TAsset{
					OriginalFileName: filename,
					LocalDateTime:    "2018-08-28T11:47:00.000Z",
				}
			}

			// Sort using our stacker
			promoteStr := strings.Join(tt.promoteList, ",")
			sorted := sortStack(assets, promoteStr, "", []string{})

			// Check the order
			for i, expected := range tt.expected {
				assert.Equal(t, expected, sorted[i].OriginalFileName,
					"Position %d: expected %s but got %s", i, expected, sorted[i].OriginalFileName)
			}
		})
	}
}

func TestSortStack_SonyBurstPhotosWithSequenceKeyword(t *testing.T) {
	// Test case from GitHub issue #18 using the recommended sequence:4 pattern
	// This should correctly order Sony burst photos based on their 4-digit sequence numbers
	stack := []utils.TAsset{
		{
			ID:               "7a733c19-a588-433c-9cd8-d621071e47c3",
			OriginalFileName: "DSCPDC_0000_BURST20180828114700954.JPG",
			LocalDateTime:    "2018-08-28T11:47:00.460Z",
		},
		{
			ID:               "26147f09-f6be-44c4-92e7-82b45313dc3c",
			OriginalFileName: "DSCPDC_0002_BURST20180828114700954.JPG",
			LocalDateTime:    "2018-08-28T11:47:00.758Z",
		},
		{
			ID:               "e964fcd7-8889-491d-aa08-ca54cfd716ab",
			OriginalFileName: "DSCPDC_0003_BURST20180828114700954_COVER.JPG",
			LocalDateTime:    "2018-08-28T11:47:00.910Z",
		},
		{
			ID:               "2dd4c37a-bc68-4f09-8150-bea904f30f51",
			OriginalFileName: "DSCPDC_0001_BURST20180828114700954.JPG",
			LocalDateTime:    "2018-08-28T11:47:00.608Z",
		},
	}

	// Test with sequence:4 keyword that should match 4-digit patterns
	parentFilenamePromote := "sequence:4"
	parentExtPromote := ""
	delimiters := []string{} // No delimiters, like in user's case

	sorted := sortStack(stack, parentFilenamePromote, parentExtPromote, delimiters)

	// Log for debugging
	t.Logf("Sorted order with sequence:4 keyword:")
	for i, asset := range sorted {
		t.Logf("  [%d] %s", i, asset.OriginalFileName)
	}

	// Expected order: 0000, 0001, 0002, 0003
	assert.Equal(t, "DSCPDC_0000_BURST20180828114700954.JPG", sorted[0].OriginalFileName)
	assert.Equal(t, "DSCPDC_0001_BURST20180828114700954.JPG", sorted[1].OriginalFileName)
	assert.Equal(t, "DSCPDC_0002_BURST20180828114700954.JPG", sorted[2].OriginalFileName)
	assert.Equal(t, "DSCPDC_0003_BURST20180828114700954_COVER.JPG", sorted[3].OriginalFileName)
}

func TestSortStack_SonyBurstPhotosWithPrefixPattern(t *testing.T) {
	// Test that sequence:DSCPDC_ correctly orders only Sony burst photos
	// and ignores other files without the DSCPDC_ prefix
	stack := []utils.TAsset{
		// Mix of Sony burst photos and other files
		{
			ID:               "1",
			OriginalFileName: "IMG_0001.JPG",
			LocalDateTime:    "2018-08-28T11:47:00.000Z",
		},
		{
			ID:               "2",
			OriginalFileName: "DSCPDC_0002_BURST20180828114700954.JPG",
			LocalDateTime:    "2018-08-28T11:47:00.758Z",
		},
		{
			ID:               "3",
			OriginalFileName: "PHOTO_001.JPG",
			LocalDateTime:    "2018-08-28T11:47:00.100Z",
		},
		{
			ID:               "4",
			OriginalFileName: "DSCPDC_0000_BURST20180828114700954.JPG",
			LocalDateTime:    "2018-08-28T11:47:00.460Z",
		},
		{
			ID:               "5",
			OriginalFileName: "DSCPDC_0003_BURST20180828114700954_COVER.JPG",
			LocalDateTime:    "2018-08-28T11:47:00.910Z",
		},
		{
			ID:               "6",
			OriginalFileName: "DSCPDC_0001_BURST20180828114700954.JPG",
			LocalDateTime:    "2018-08-28T11:47:00.608Z",
		},
		{
			ID:               "7",
			OriginalFileName: "DSC_0001.JPG",
			LocalDateTime:    "2018-08-28T11:47:00.200Z",
		},
	}

	// Use sequence:DSCPDC_ to only order Sony burst photos
	parentFilenamePromote := "sequence:DSCPDC_"
	parentExtPromote := ""
	delimiters := []string{}

	sorted := sortStack(stack, parentFilenamePromote, parentExtPromote, delimiters)

	// Log for debugging
	t.Logf("Sorted order with sequence:DSCPDC_ pattern:")
	for i, asset := range sorted {
		t.Logf("  [%d] %s", i, asset.OriginalFileName)
	}

	// Expected: DSCPDC_ files ordered by sequence first, then others
	assert.Equal(t, "DSCPDC_0000_BURST20180828114700954.JPG", sorted[0].OriginalFileName)
	assert.Equal(t, "DSCPDC_0001_BURST20180828114700954.JPG", sorted[1].OriginalFileName)
	assert.Equal(t, "DSCPDC_0002_BURST20180828114700954.JPG", sorted[2].OriginalFileName)
	assert.Equal(t, "DSCPDC_0003_BURST20180828114700954_COVER.JPG", sorted[3].OriginalFileName)

	// Other files should come after, sorted alphabetically
	assert.Equal(t, "DSC_0001.JPG", sorted[4].OriginalFileName)
	assert.Equal(t, "IMG_0001.JPG", sorted[5].OriginalFileName)
	assert.Equal(t, "PHOTO_001.JPG", sorted[6].OriginalFileName)
}

func TestStackBy_SonyBurstPhotosFullWorkflow(t *testing.T) {
	// Test the complete workflow from GitHub issue #18
	// Using both the regex criteria and sequence:4 promote pattern
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	assets := []utils.TAsset{
		{
			ID:               "7a733c19-a588-433c-9cd8-d621071e47c3",
			OriginalFileName: "DSCPDC_0000_BURST20180828114700954.JPG",
			LocalDateTime:    "2018-08-28T11:47:00.460Z",
		},
		{
			ID:               "2dd4c37a-bc68-4f09-8150-bea904f30f51",
			OriginalFileName: "DSCPDC_0001_BURST20180828114700954.JPG",
			LocalDateTime:    "2018-08-28T11:47:00.608Z",
		},
		{
			ID:               "26147f09-f6be-44c4-92e7-82b45313dc3c",
			OriginalFileName: "DSCPDC_0002_BURST20180828114700954.JPG",
			LocalDateTime:    "2018-08-28T11:47:00.758Z",
		},
		{
			ID:               "e964fcd7-8889-491d-aa08-ca54cfd716ab",
			OriginalFileName: "DSCPDC_0003_BURST20180828114700954_COVER.JPG",
			LocalDateTime:    "2018-08-28T11:47:00.910Z",
		},
		// Add another burst sequence to ensure proper grouping
		{
			ID:               "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
			OriginalFileName: "DSCPDC_0000_BURST20180828115000000.JPG",
			LocalDateTime:    "2018-08-28T11:50:00.000Z",
		},
		{
			ID:               "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
			OriginalFileName: "DSCPDC_0001_BURST20180828115000000.JPG",
			LocalDateTime:    "2018-08-28T11:50:00.100Z",
		},
	}

	// Set up the exact environment from the issue
	t.Setenv("CRITERIA", `[{"key":"originalFileName","regex":{"key":"DSCPDC_(\\d{4})_(BURST\\d{17})(_COVER)?.JPG","index":2}}]`)

	// Use sequence:4 for proper ordering
	parentFilenamePromote := "sequence:4"
	parentExtPromote := ".jpg,.png,.jpeg,.dng"

	stacks, err := StackBy(assets, "", parentFilenamePromote, parentExtPromote, logger)
	assert.NoError(t, err)
	assert.Len(t, stacks, 2, "Should have 2 stacks")

	// Find the first burst stack
	var firstStack []utils.TAsset
	for _, stack := range stacks {
		if strings.Contains(stack[0].OriginalFileName, "114700954") {
			firstStack = stack
			break
		}
	}

	assert.NotNil(t, firstStack, "Should find the first burst stack")
	assert.Len(t, firstStack, 4, "First stack should have 4 photos")

	// Verify correct ordering
	assert.Equal(t, "DSCPDC_0000_BURST20180828114700954.JPG", firstStack[0].OriginalFileName)
	assert.Equal(t, "DSCPDC_0001_BURST20180828114700954.JPG", firstStack[1].OriginalFileName)
	assert.Equal(t, "DSCPDC_0002_BURST20180828114700954.JPG", firstStack[2].OriginalFileName)
	assert.Equal(t, "DSCPDC_0003_BURST20180828114700954_COVER.JPG", firstStack[3].OriginalFileName)
}

func TestSequenceKeywordVariations(t *testing.T) {
	tests := []struct {
		name        string
		promoteList []string
		filenames   []string
		expected    []string
	}{
		{
			name:        "3-digit sequence",
			promoteList: []string{"sequence:3"},
			filenames: []string{
				"IMG_010.jpg",
				"IMG_001.jpg",
				"IMG_1000.jpg", // Has "100" as first 3 digits - will match as 100
				"IMG_100.jpg",
				"IMG_10.jpg", // No 3-digit match, falls back to finding any 3 digits
			},
			expected: []string{
				"IMG_001.jpg",  // index 1
				"IMG_10.jpg",   // No 3-digit sequence after _, but has no fallback match -> index 1
				"IMG_010.jpg",  // index 10
				"IMG_100.jpg",  // index 100
				"IMG_1000.jpg", // Regex finds "100" -> index 100
			},
		},
		{
			name:        "2-digit sequence",
			promoteList: []string{"sequence:2"},
			filenames: []string{
				"photo_10.jpg",
				"photo_01.jpg",
				"photo_100.jpg", // 3 digits, but regex will find "10"
				"photo_99.jpg",
				"photo_1.jpg", // 1 digit, no 2-digit match
			},
			expected: []string{
				"photo_01.jpg",  // index 1
				"photo_1.jpg",   // No match, index len(promoteList) = 1
				"photo_10.jpg",  // index 10 (first one)
				"photo_100.jpg", // Regex finds "10", index 10 (second one, sorted alphabetically)
				"photo_99.jpg",  // index 99
			},
		},
		{
			name:        "5-digit sequence",
			promoteList: []string{"sequence:5"},
			filenames: []string{
				"BURST_00100.jpg",
				"BURST_00001.jpg",
				"BURST_10000.jpg",
				"BURST_0001.jpg", // 4 digits, no 5-digit match
				"BURST_99999.jpg",
			},
			expected: []string{
				"BURST_00001.jpg", // index 1 (matches after _)
				"BURST_0001.jpg",  // No 5-digit match, index 1 (sorted alphabetically after 00001)
				"BURST_00100.jpg", // index 100
				"BURST_10000.jpg", // index 10000
				"BURST_99999.jpg", // index 99999
			},
		},
		{
			name:        "Multiple sequence keywords with different digits",
			promoteList: []string{"HDR", "sequence:3", "EDIT", "sequence:4"},
			filenames: []string{
				"IMG_0001.jpg",      // 4-digit after _, but sequence:3 is found first
				"IMG_001_HDR.jpg",   // Has HDR
				"IMG_010.jpg",       // 3-digit after _
				"IMG_0010_EDIT.jpg", // Has EDIT
				"IMG_100.jpg",       // 3-digit after _
				"IMG_1000.jpg",      // 4-digit, but regex will find "100" for sequence:3
			},
			expected: []string{
				"IMG_001_HDR.jpg",   // HDR is index 0
				"IMG_0001.jpg",      // No exact 3-digit match, regex finds "000", index 1+0 = 1
				"IMG_0010_EDIT.jpg", // EDIT is index 2
				"IMG_010.jpg",       // sequence:3 match, index 1+10 = 11
				"IMG_100.jpg",       // sequence:3 match, index 1+100 = 101
				"IMG_1000.jpg",      // Regex finds "100", index 1+100 = 101
			},
		},
		{
			name:        "Multiple sequence prefixes",
			promoteList: []string{"sequence:IMG_", "sequence:DSC_"},
			filenames: []string{
				"DSC_002.jpg",
				"IMG_002.jpg",
				"DSC_001.jpg",
				"PHOTO_001.jpg", // Different prefix, should not match
				"IMG_001.jpg",
				"DSC_003.jpg",
			},
			expected: []string{
				// Only the first sequence keyword is used (sequence:IMG_)
				"IMG_001.jpg",   // index 0+1 = 1
				"IMG_002.jpg",   // index 0+2 = 2
				"DSC_001.jpg",   // No IMG_ prefix, gets high index
				"DSC_002.jpg",   // No IMG_ prefix, gets high index
				"DSC_003.jpg",   // No IMG_ prefix, gets high index
				"PHOTO_001.jpg", // No IMG_ prefix, gets high index
			},
		},
		{
			name:        "Mixed sequence patterns in same promote list",
			promoteList: []string{"sequence:IMG_", "sequence:3", "sequence"},
			filenames: []string{
				"IMG_001.jpg",      // Matches sequence:IMG_
				"PHOTO_100.jpg",    // No IMG_ prefix
				"random_99999.jpg", // No IMG_ prefix
				"IMG_002.jpg",      // Matches sequence:IMG_
				"test_010.jpg",     // No IMG_ prefix
				"file_1.jpg",       // No IMG_ prefix
			},
			expected: []string{
				// Only first sequence keyword is used (sequence:IMG_)
				"IMG_001.jpg",      // index 0+1 = 1
				"IMG_002.jpg",      // index 0+2 = 2
				"PHOTO_100.jpg",    // No IMG_ prefix, high index
				"file_1.jpg",       // No IMG_ prefix, high index
				"random_99999.jpg", // No IMG_ prefix, high index
				"test_010.jpg",     // No IMG_ prefix, high index
			},
		},
		{
			name:        "Sequence with complex prefix patterns",
			promoteList: []string{"sequence:DSCPDC_", "sequence:DSC_"},
			filenames: []string{
				"DSCPDC_0002_BURST.jpg",
				"DSC_0001.jpg",
				"DSCPDC_0001_BURST.jpg",
				"DSC_0002.jpg",
				"OTHER_0001.jpg", // Should not match
			},
			expected: []string{
				"DSCPDC_0001_BURST.jpg",
				"DSCPDC_0002_BURST.jpg",
				"DSC_0001.jpg",
				"DSC_0002.jpg",
				"OTHER_0001.jpg",
			},
		},
		{
			name:        "Multiple sequences with priority mixing",
			promoteList: []string{"COVER", "sequence:4", "EDIT", "sequence"},
			filenames: []string{
				"IMG_0002.jpg",
				"IMG_10_EDIT.jpg",
				"IMG_0001_COVER.jpg",
				"IMG_999.jpg", // 3 digits, not 4
				"IMG_0003.jpg",
			},
			expected: []string{
				"IMG_0001_COVER.jpg", // COVER is index 0
				"IMG_10_EDIT.jpg",    // EDIT is index 2
				"IMG_0002.jpg",       // sequence:4 match, index 1+2 = 3
				"IMG_0003.jpg",       // sequence:4 match, index 1+3 = 4
				"IMG_999.jpg",        // No 4-digit match, gets len(promoteList) = 4
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create assets from filenames
			assets := make([]utils.TAsset, len(tt.filenames))
			for i, filename := range tt.filenames {
				assets[i] = utils.TAsset{
					OriginalFileName: filename,
					LocalDateTime:    "2018-08-28T11:47:00.000Z",
				}
			}

			// Sort using our stacker
			promoteStr := strings.Join(tt.promoteList, ",")
			sorted := sortStack(assets, promoteStr, "", []string{})

			// Check the order
			for i, expected := range tt.expected {
				assert.Equal(t, expected, sorted[i].OriginalFileName,
					"Position %d: expected %s but got %s", i, expected, sorted[i].OriginalFileName)
			}
		})
	}
}
