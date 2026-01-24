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
			},
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
		{
			name: "single comma (empty string as only promote element)",
			inputOrder: []string{
				"IMG_2482.xyz",
				"IMG_2482.abc",
				"IMG_2482_edited.jpg",
			},
			expectedOrder: []string{
				"IMG_2482_edited.jpg",
				"IMG_2482.abc",
				"IMG_2482.xyz",
			},
			promoteStr: ",",
		},
		{
			name: "empty string promotes files without suffixes",
			inputOrder: []string{
				"IMG_1234_edited.jpg",
				"IMG_1234.jpg",
				"IMG_1234_crop.jpg",
			},
			expectedOrder: []string{
				"IMG_1234.jpg",
				"IMG_1234_edited.jpg",
				"IMG_1234_crop.jpg",
			},
			promoteStr: ",_edited,_crop",
		},
		{
			name: "empty string with mixed priorities",
			inputOrder: []string{
				"IMG_1234_edited.jpg",
				"IMG_1234_cover.jpg",
				"IMG_1234.jpg",
				"IMG_1234_crop.jpg",
			},
			expectedOrder: []string{
				"IMG_1234_cover.jpg",
				"IMG_1234.jpg",
				"IMG_1234_edited.jpg",
				"IMG_1234_crop.jpg",
			},
			promoteStr: "cover,,_edited,_crop",
		},
		{
			name: "empty string with extension promotion",
			inputOrder: []string{
				"IMG_1234_edited.jpg",
				"IMG_1234.cr3",
				"IMG_1234.jpg",
				"IMG_1234_edited.cr3",
			},
			expectedOrder: []string{
				"IMG_1234.jpg",
				"IMG_1234.cr3",
				"IMG_1234_edited.jpg",
				"IMG_1234_edited.cr3",
			},
			promoteStr: ",_edited",
			promoteExt: ".jpg,.cr3",
		},
		{
			name: "empty string only - clean file first",
			inputOrder: []string{
				"IMG_1234_final_edited.jpg",
				"IMG_1234_crop_edited.jpg",
				"IMG_1234.jpg",
				"IMG_1234_edited.jpg",
			},
			expectedOrder: []string{
				"IMG_1234.jpg",
				"IMG_1234_crop_edited.jpg",
				"IMG_1234_edited.jpg",
				"IMG_1234_final_edited.jpg",
			},
			promoteStr: ",",
		},
		{
			name: "real world case - clean files promoted, then by extension",
			inputOrder: []string{
				"IMG_1234_edited.jpg",
				"IMG_1234.jpg",
				"IMG_1234.cr3",
			},
			expectedOrder: []string{
				"IMG_1234.jpg",
				"IMG_1234.cr3",
				"IMG_1234_edited.jpg",
			},
			promoteStr: ",_edited",
			promoteExt: ".jpg,.cr3",
		},
		{
			name: "biggestNumber with numeric suffixes",
			inputOrder: []string{
				"IMG_1234.jpg",
				"IMG_1234~2.jpg",
				"IMG_1234~5.jpg",
				"IMG_1234~3.jpg",
			},
			expectedOrder: []string{
				"IMG_1234~5.jpg",
				"IMG_1234~3.jpg",
				"IMG_1234~2.jpg",
				"IMG_1234.jpg",
			},
			promoteStr: "biggestNumber",
		},
		{
			name: "biggestNumber with empty string - clean files first then by number",
			inputOrder: []string{
				"IMG_1234_edited~2.jpg",
				"IMG_1234.jpg",
				"IMG_1234_edited~5.jpg",
				"IMG_1234~3.jpg",
			},
			expectedOrder: []string{
				"IMG_1234~3.jpg",
				"IMG_1234.jpg",
				"IMG_1234_edited~5.jpg",
				"IMG_1234_edited~2.jpg",
			},
			promoteStr: ",_edited,biggestNumber",
		},
		{
			name: "biggestNumber mixed with other promotes",
			inputOrder: []string{
				"IMG_1234~2.jpg",
				"IMG_1234_cover.jpg",
				"IMG_1234~5.jpg",
				"IMG_1234_edit.jpg",
				"IMG_1234.jpg",
			},
			expectedOrder: []string{
				"IMG_1234_cover.jpg",
				"IMG_1234_edit.jpg",
				"IMG_1234~5.jpg",
				"IMG_1234~2.jpg",
				"IMG_1234.jpg",
			},
			promoteStr: "cover,edit,biggestNumber",
		},
		{
			name: "biggestNumber with different delimiter patterns",
			inputOrder: []string{
				"IMG_1234.jpg",
				"IMG_1234.2.jpg",
				"IMG_1234.10.jpg",
				"IMG_1234.3.jpg",
			},
			expectedOrder: []string{
				"IMG_1234.10.jpg",
				"IMG_1234.3.jpg",
				"IMG_1234.2.jpg",
				"IMG_1234.jpg",
			},
			promoteStr: "biggestNumber",
		},
		{
			name: "biggestNumber only affects files at same promote level",
			inputOrder: []string{
				"IMG_1234~10.jpg",
				"IMG_1234_edit~2.jpg",
				"IMG_1234_edit~20.jpg",
				"IMG_1234~5.jpg",
			},
			expectedOrder: []string{
				"IMG_1234_edit~20.jpg",
				"IMG_1234_edit~2.jpg",
				"IMG_1234~10.jpg",
				"IMG_1234~5.jpg",
			},
			promoteStr: "edit,biggestNumber",
		},
		{
			name: "biggestNumber with no numeric suffixes - falls back to alphabetical",
			inputOrder: []string{
				"IMG_1234_c.jpg",
				"IMG_1234_a.jpg",
				"IMG_1234_b.jpg",
			},
			expectedOrder: []string{
				"IMG_1234_a.jpg",
				"IMG_1234_b.jpg",
				"IMG_1234_c.jpg",
			},
			promoteStr: "biggestNumber",
		},
		{
			name: "default promote list behavior",
			inputOrder: []string{
				"IMG_1234.jpg",
				"IMG_1234_edit.jpg",
				"IMG_1234_crop.jpg",
				"IMG_1234_hdr.jpg",
				"IMG_1234~5.jpg",
				"IMG_1234~2.jpg",
			},
			expectedOrder: []string{
				"IMG_1234_edit.jpg",
				"IMG_1234_crop.jpg",
				"IMG_1234_hdr.jpg",
				"IMG_1234~5.jpg",
				"IMG_1234~2.jpg",
				"IMG_1234.jpg",
			},
			promoteStr: "edit,crop,hdr,biggestNumber",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

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

			result := sortStack(assets, tt.promoteStr, tt.promoteExt, []string{"~", "."}, utils.DefaultCriteria, &safePromoteData{data: make(map[string]map[string]string)}, make(map[int]map[string]int))

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
			expectedGroups: 0,
		},
		{
			name: "same filename different datetime",
			assets: []utils.TAsset{
				assetFactory("test.jpg", time.Now()),
				assetFactory("test.jpg", time.Now().Add(time.Hour)),
			},
			expectedGroups: 0,
		},
		{
			name: "empty key handling",
			assets: []utils.TAsset{
				assetFactory("test.jpg", time.Now()),
				assetFactory("test.jpg", time.Time{}),
			},
			expectedGroups: 0,
			skipMatchMiss:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if tt.skipMatchMiss {
				os.Setenv("SKIP_MATCH_MISS", "true")
				defer os.Unsetenv("SKIP_MATCH_MISS")
			}

			groups, err := StackBy(tt.assets, "", "", "", logrus.New())

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

	parentFilenamePromote := "0000,0001,0002,0003"
	parentExtPromote := ""
	delimiters := []string{}

	sorted := sortStack(stack, parentFilenamePromote, parentExtPromote, delimiters, utils.DefaultCriteria, &safePromoteData{data: make(map[string]map[string]string)}, make(map[int]map[string]int))

	t.Logf("Sorted order:")
	for i, asset := range sorted {
		t.Logf("  [%d] %s", i, asset.OriginalFileName)
	}

	assert.Equal(t, "DSCPDC_0000_BURST20180828114700954.JPG", sorted[0].OriginalFileName)
	assert.Equal(t, "DSCPDC_0001_BURST20180828114700954.JPG", sorted[1].OriginalFileName)
	assert.Equal(t, "DSCPDC_0002_BURST20180828114700954.JPG", sorted[2].OriginalFileName)
	assert.Equal(t, "DSCPDC_0003_BURST20180828114700954_COVER.JPG", sorted[3].OriginalFileName)
}

func TestStackBy_SonyBurstWithRegex(t *testing.T) {

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

	t.Setenv("CRITERIA", `[{"key":"originalFileName","regex":{"key":"DSCPDC_(\\d{4})_(BURST\\d{17})(_COVER)?.JPG","index":2}}]`)

	parentFilenamePromote := "0000,0001,0002,0003"
	parentExtPromote := ""

	stacks, err := StackBy(assets, "", parentFilenamePromote, parentExtPromote, logger)
	assert.NoError(t, err)
	assert.Len(t, stacks, 1)

	stack := stacks[0]
	assert.Equal(t, "DSCPDC_0000_BURST20180828114700954.JPG", stack[0].OriginalFileName)
	assert.Equal(t, "DSCPDC_0001_BURST20180828114700954.JPG", stack[1].OriginalFileName)
	assert.Equal(t, "DSCPDC_0002_BURST20180828114700954.JPG", stack[2].OriginalFileName)
	assert.Equal(t, "DSCPDC_0003_BURST20180828114700954_COVER.JPG", stack[3].OriginalFileName)
}

func TestSortStack_BurstPhotoWithShuffledInput(t *testing.T) {

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

	parentFilenamePromote := "0000,0001,0002,0003"
	parentExtPromote := ""
	delimiters := []string{}

	sorted := sortStack(stack, parentFilenamePromote, parentExtPromote, delimiters, utils.DefaultCriteria, &safePromoteData{data: make(map[string]map[string]string)}, make(map[int]map[string]int))

	t.Logf("Sorted order with burst handling:")
	for i, asset := range sorted {
		t.Logf("  [%d] %s", i, asset.OriginalFileName)
	}

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
			filename:    "DSCPDC_0010_BURST20180828114700954.JPG",
			matchMode:   "sequence",
			expectedIdx: 10,
		},
		{
			filename:    "IMG_0001_OTHER.JPG",
			matchMode:   "sequence",
			expectedIdx: 1,
		},
		{
			filename:    "BURST_0001_20180828.JPG",
			matchMode:   "sequence",
			expectedIdx: 1,
		},
		{
			filename:    "DSCPDC_0001_BURST20180828114700954.JPG",
			matchMode:   "contains",
			expectedIdx: 1,
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
			expectedIdx: 1,
		},
		{
			name:        "Suffix pattern 1a,2a,3a",
			promoteList: []string{"1a", "2a", "3a"},
			filename:    "PHOTO_2a_BURST123.jpg",
			expectedIdx: 1,
		},
		{
			name:        "Complex pattern photo_001_final",
			promoteList: []string{"photo_001_final", "photo_002_final", "photo_003_final"},
			filename:    "PREFIX_photo_002_final_BURST.jpg",
			expectedIdx: 1,
		},
		{
			name:        "Extended sequence beyond promote list",
			promoteList: []string{"0000", "0001", "0002", "0003"},
			filename:    "DSCPDC_0010_BURST20180828114700954.JPG",
			expectedIdx: 10,
		},
		{
			name:        "Very high sequence number",
			promoteList: []string{"0000", "0001", "0002"},
			filename:    "DSCPDC_0999_BURST20180828114700954.JPG",
			expectedIdx: 999,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mode := detectPromoteMatchMode(tt.promoteList, tt.filename)
			assert.Equal(t, "sequence", mode, "Should detect sequence mode")

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

			assets := make([]utils.TAsset, len(tt.filenames))
			for i, filename := range tt.filenames {
				assets[i] = utils.TAsset{
					OriginalFileName: filename,
					LocalDateTime:    "2018-08-28T11:47:00.000Z",
				}
			}

			promoteStr := strings.Join(tt.promoteList, ",")
			sorted := sortStack(assets, promoteStr, "", []string{}, utils.DefaultCriteria, &safePromoteData{data: make(map[string]map[string]string)}, make(map[int]map[string]int))

			for i, expected := range tt.expected {
				assert.Equal(t, expected, sorted[i].OriginalFileName,
					"Position %d: expected %s but got %s", i, expected, sorted[i].OriginalFileName)
			}
		})
	}
}

func TestSortStack_SonyBurstPhotosWithSequenceKeyword(t *testing.T) {

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

	parentFilenamePromote := "sequence:4"
	parentExtPromote := ""
	delimiters := []string{}

	sorted := sortStack(stack, parentFilenamePromote, parentExtPromote, delimiters, utils.DefaultCriteria, &safePromoteData{data: make(map[string]map[string]string)}, make(map[int]map[string]int))

	t.Logf("Sorted order with sequence:4 keyword:")
	for i, asset := range sorted {
		t.Logf("  [%d] %s", i, asset.OriginalFileName)
	}

	assert.Equal(t, "DSCPDC_0000_BURST20180828114700954.JPG", sorted[0].OriginalFileName)
	assert.Equal(t, "DSCPDC_0001_BURST20180828114700954.JPG", sorted[1].OriginalFileName)
	assert.Equal(t, "DSCPDC_0002_BURST20180828114700954.JPG", sorted[2].OriginalFileName)
	assert.Equal(t, "DSCPDC_0003_BURST20180828114700954_COVER.JPG", sorted[3].OriginalFileName)
}

func TestSortStack_SonyBurstPhotosWithPrefixPattern(t *testing.T) {

	stack := []utils.TAsset{

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

	parentFilenamePromote := "sequence:DSCPDC_"
	parentExtPromote := ""
	delimiters := []string{}

	sorted := sortStack(stack, parentFilenamePromote, parentExtPromote, delimiters, utils.DefaultCriteria, &safePromoteData{data: make(map[string]map[string]string)}, make(map[int]map[string]int))

	t.Logf("Sorted order with sequence:DSCPDC_ pattern:")
	for i, asset := range sorted {
		t.Logf("  [%d] %s", i, asset.OriginalFileName)
	}

	assert.Equal(t, "DSCPDC_0000_BURST20180828114700954.JPG", sorted[0].OriginalFileName)
	assert.Equal(t, "DSCPDC_0001_BURST20180828114700954.JPG", sorted[1].OriginalFileName)
	assert.Equal(t, "DSCPDC_0002_BURST20180828114700954.JPG", sorted[2].OriginalFileName)
	assert.Equal(t, "DSCPDC_0003_BURST20180828114700954_COVER.JPG", sorted[3].OriginalFileName)

	assert.Equal(t, "DSC_0001.JPG", sorted[4].OriginalFileName)
	assert.Equal(t, "IMG_0001.JPG", sorted[5].OriginalFileName)
	assert.Equal(t, "PHOTO_001.JPG", sorted[6].OriginalFileName)
}

func TestStackBy_SonyBurstPhotosFullWorkflow(t *testing.T) {

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

	t.Setenv("CRITERIA", `[{"key":"originalFileName","regex":{"key":"DSCPDC_(\\d{4})_(BURST\\d{17})(_COVER)?.JPG","index":2}}]`)

	parentFilenamePromote := "sequence:4"
	parentExtPromote := ".jpg,.png,.jpeg,.dng"

	stacks, err := StackBy(assets, "", parentFilenamePromote, parentExtPromote, logger)
	assert.NoError(t, err)
	assert.Len(t, stacks, 2, "Should have 2 stacks")

	var firstStack []utils.TAsset
	for _, stack := range stacks {
		if strings.Contains(stack[0].OriginalFileName, "114700954") {
			firstStack = stack
			break
		}
	}

	assert.NotNil(t, firstStack, "Should find the first burst stack")
	assert.Len(t, firstStack, 4, "First stack should have 4 photos")

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
				"IMG_1000.jpg",
				"IMG_100.jpg",
				"IMG_10.jpg",
			},
			expected: []string{
				"IMG_001.jpg",
				"IMG_10.jpg",
				"IMG_010.jpg",
				"IMG_100.jpg",
				"IMG_1000.jpg",
			},
		},
		{
			name:        "2-digit sequence",
			promoteList: []string{"sequence:2"},
			filenames: []string{
				"photo_10.jpg",
				"photo_01.jpg",
				"photo_100.jpg",
				"photo_99.jpg",
				"photo_1.jpg",
			},
			expected: []string{
				"photo_01.jpg",
				"photo_1.jpg",
				"photo_10.jpg",
				"photo_100.jpg",
				"photo_99.jpg",
			},
		},
		{
			name:        "5-digit sequence",
			promoteList: []string{"sequence:5"},
			filenames: []string{
				"BURST_00100.jpg",
				"BURST_00001.jpg",
				"BURST_10000.jpg",
				"BURST_0001.jpg",
				"BURST_99999.jpg",
			},
			expected: []string{
				"BURST_00001.jpg",
				"BURST_0001.jpg",
				"BURST_00100.jpg",
				"BURST_10000.jpg",
				"BURST_99999.jpg",
			},
		},
		{
			name:        "Multiple sequence keywords with different digits",
			promoteList: []string{"HDR", "sequence:3", "EDIT", "sequence:4"},
			filenames: []string{
				"IMG_0001.jpg",
				"IMG_001_HDR.jpg",
				"IMG_010.jpg",
				"IMG_0010_EDIT.jpg",
				"IMG_100.jpg",
				"IMG_1000.jpg",
			},
			expected: []string{
				"IMG_001_HDR.jpg",
				"IMG_0001.jpg",
				"IMG_0010_EDIT.jpg",
				"IMG_010.jpg",
				"IMG_100.jpg",
				"IMG_1000.jpg",
			},
		},
		{
			name:        "Multiple sequence prefixes",
			promoteList: []string{"sequence:IMG_", "sequence:DSC_"},
			filenames: []string{
				"DSC_002.jpg",
				"IMG_002.jpg",
				"DSC_001.jpg",
				"PHOTO_001.jpg",
				"IMG_001.jpg",
				"DSC_003.jpg",
			},
			expected: []string{

				"IMG_001.jpg",
				"IMG_002.jpg",
				"DSC_001.jpg",
				"DSC_002.jpg",
				"DSC_003.jpg",
				"PHOTO_001.jpg",
			},
		},
		{
			name:        "Mixed sequence patterns in same promote list",
			promoteList: []string{"sequence:IMG_", "sequence:3", "sequence"},
			filenames: []string{
				"IMG_001.jpg",
				"PHOTO_100.jpg",
				"random_99999.jpg",
				"IMG_002.jpg",
				"test_010.jpg",
				"file_1.jpg",
			},
			expected: []string{

				"IMG_001.jpg",
				"IMG_002.jpg",
				"PHOTO_100.jpg",
				"file_1.jpg",
				"random_99999.jpg",
				"test_010.jpg",
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
				"OTHER_0001.jpg",
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
				"IMG_999.jpg",
				"IMG_0003.jpg",
			},
			expected: []string{
				"IMG_0001_COVER.jpg",
				"IMG_10_EDIT.jpg",
				"IMG_0002.jpg",
				"IMG_0003.jpg",
				"IMG_999.jpg",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			assets := make([]utils.TAsset, len(tt.filenames))
			for i, filename := range tt.filenames {
				assets[i] = utils.TAsset{
					OriginalFileName: filename,
					LocalDateTime:    "2018-08-28T11:47:00.000Z",
				}
			}

			promoteStr := strings.Join(tt.promoteList, ",")
			sorted := sortStack(assets, promoteStr, "", []string{}, utils.DefaultCriteria, &safePromoteData{data: make(map[string]map[string]string)}, make(map[int]map[string]int))

			for i, expected := range tt.expected {
				assert.Equal(t, expected, sorted[i].OriginalFileName,
					"Position %d: expected %s but got %s", i, expected, sorted[i].OriginalFileName)
			}
		})
	}
}

func TestPixelPhoneStacking(t *testing.T) {

	tests := []struct {
		name     string
		assets   []utils.TAsset
		criteria []utils.TCriteria
		want     int
		desc     string
	}{
		{
			name: "Pixel RAW files with same timestamp should stack",
			assets: []utils.TAsset{
				{
					OriginalFileName: "PXL_20250731_152626855.RAW-01.COVER.jpg",
					LocalDateTime:    "2025-07-31T15:26:26.855Z",
				},
				{
					OriginalFileName: "PXL_20250731_152626855.RAW-02.ORIGINAL.DNG",
					LocalDateTime:    "2025-07-31T15:26:26.855Z",
				},
			},
			criteria: []utils.TCriteria{
				{
					Key: "originalFileName",
					Regex: &utils.TRegex{
						Key:   `(PXL|IMG)_(\d{8})_(\d+)`,
						Index: 3,
					},
				},
				{
					Key: "localDateTime",
					Delta: &utils.TDelta{
						Milliseconds: 1000,
					},
				},
			},
			want: 1,
			desc: "Files with identical timestamps and matching regex should stack",
		},
		{
			name: "Pixel RAW files with slightly different timestamps within delta",
			assets: []utils.TAsset{
				{
					OriginalFileName: "PXL_20250731_152626855.RAW-01.COVER.jpg",
					LocalDateTime:    "2025-07-31T15:26:26.855Z",
				},
				{
					OriginalFileName: "PXL_20250731_152626855.RAW-02.ORIGINAL.DNG",
					LocalDateTime:    "2025-07-31T15:26:26.950Z",
				},
			},
			criteria: []utils.TCriteria{
				{
					Key: "originalFileName",
					Regex: &utils.TRegex{
						Key:   `(PXL|IMG)_(\d{8})_(\d+)`,
						Index: 3,
					},
				},
				{
					Key: "localDateTime",
					Delta: &utils.TDelta{
						Milliseconds: 1000,
					},
				},
			},
			want: 1,
			desc: "Files within 1 second delta should stack",
		},
		{
			name: "Pixel RAW files with timestamps outside delta",
			assets: []utils.TAsset{
				{
					OriginalFileName: "PXL_20250731_152626855.RAW-01.COVER.jpg",
					LocalDateTime:    "2025-07-31T15:26:26.855Z",
				},
				{
					OriginalFileName: "PXL_20250731_152626855.RAW-02.ORIGINAL.DNG",
					LocalDateTime:    "2025-07-31T15:26:28.000Z",
				},
			},
			criteria: []utils.TCriteria{
				{
					Key: "originalFileName",
					Regex: &utils.TRegex{
						Key:   `(PXL|IMG)_(\d{8})_(\d+)`,
						Index: 3,
					},
				},
				{
					Key: "localDateTime",
					Delta: &utils.TDelta{
						Milliseconds: 1000,
					},
				},
			},
			want: 0,
			desc: "Files outside 1 second delta should NOT stack",
		},
		{
			name: "Pixel files without localDateTime criterion",
			assets: []utils.TAsset{
				{
					OriginalFileName: "PXL_20250731_152626855.RAW-01.COVER.jpg",
					LocalDateTime:    "2025-07-31T15:26:26.855Z",
				},
				{
					OriginalFileName: "PXL_20250731_152626855.RAW-02.ORIGINAL.DNG",
					LocalDateTime:    "2025-07-31T15:26:28.000Z",
				},
			},
			criteria: []utils.TCriteria{
				{
					Key: "originalFileName",
					Regex: &utils.TRegex{
						Key:   `(PXL|IMG)_(\d{8})_(\d+)`,
						Index: 3,
					},
				},
			},
			want: 1,
			desc: "Without time criterion, files should stack based on filename alone",
		},
		{
			name: "Different Pixel burst sequences should not stack",
			assets: []utils.TAsset{
				{
					OriginalFileName: "PXL_20250731_152626855.RAW-01.COVER.jpg",
					LocalDateTime:    "2025-07-31T15:26:26.855Z",
				},
				{
					OriginalFileName: "PXL_20250731_152627900.RAW-01.COVER.jpg",
					LocalDateTime:    "2025-07-31T15:26:27.900Z",
				},
			},
			criteria: []utils.TCriteria{
				{
					Key: "originalFileName",
					Regex: &utils.TRegex{
						Key:   `(PXL|IMG)_(\d{8})_(\d+)`,
						Index: 3,
					},
				},
			},
			want: 0,
			desc: "Different burst sequences (different numbers) should not stack",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			t.Setenv("CRITERIA", mustMarshalJSON(t, tt.criteria))

			groups, err := StackBy(tt.assets, "", "", "", logrus.New())
			require.NoError(t, err)
			assert.Equal(t, tt.want, len(groups), "%s: Expected %d groups but got %d", tt.desc, tt.want, len(groups))

			if tt.want > 0 && len(groups) > 0 {

				t.Logf("Group contains %d assets", len(groups[0]))
				for _, asset := range groups[0] {
					t.Logf("  - %s at %s", asset.OriginalFileName, asset.LocalDateTime)
				}
			}
		})
	}
}

func TestEditedPhotoPromotion(t *testing.T) {
	tests := []struct {
		name          string
		inputOrder    []string
		expectedOrder []string
		promoteStr    string
		desc          string
	}{
		{
			name: "Edited photo with ~2 should be promoted over original",
			inputOrder: []string{
				"PXL_20250823_193751711.jpg",
				"PXL_20250823_193751711~2.jpg",
			},
			expectedOrder: []string{
				"PXL_20250823_193751711~2.jpg",
				"PXL_20250823_193751711.jpg",
			},
			promoteStr: "biggestNumber",
			desc:       "Edited photo with ~2 should come before original",
		},
		{
			name: "Multiple edited versions - highest number first",
			inputOrder: []string{
				"PXL_20250823_193751711.jpg",
				"PXL_20250823_193751711~2.jpg",
				"PXL_20250823_193751711~3.jpg",
				"PXL_20250823_193751711~5.jpg",
			},
			expectedOrder: []string{
				"PXL_20250823_193751711~5.jpg",
				"PXL_20250823_193751711~3.jpg",
				"PXL_20250823_193751711~2.jpg",
				"PXL_20250823_193751711.jpg",
			},
			promoteStr: "biggestNumber",
			desc:       "Multiple edits should be sorted by highest number first",
		},
		{
			name: "Edited with explicit ~ promote pattern",
			inputOrder: []string{
				"PXL_20250823_193751711.jpg",
				"PXL_20250823_193751711~2.jpg",
			},
			expectedOrder: []string{
				"PXL_20250823_193751711~2.jpg",
				"PXL_20250823_193751711.jpg",
			},
			promoteStr: "~2",
			desc:       "Explicit ~2 promote should work",
		},
		{
			name: "Real-world test from issue",
			inputOrder: []string{
				"PXL_20250628_123043121.RAW-01.COVER.jpg",
				"PXL_20250628_123043121.RAW-01.COVER~2.jpg",
			},
			expectedOrder: []string{
				"PXL_20250628_123043121.RAW-01.COVER~2.jpg",
				"PXL_20250628_123043121.RAW-01.COVER.jpg",
			},
			promoteStr: "biggestNumber",
			desc:       "Real example from issue should promote edited version",
		},
		{
			name: "Test with default promote list",
			inputOrder: []string{
				"PXL_20250823_193751711.jpg",
				"PXL_20250823_193751711~2.jpg",
				"PXL_20250823_193751711_edit.jpg",
			},
			expectedOrder: []string{
				"PXL_20250823_193751711_edit.jpg",
				"PXL_20250823_193751711~2.jpg",
				"PXL_20250823_193751711.jpg",
			},
			promoteStr: "edit,crop,hdr,biggestNumber",
			desc:       "Default promote list should handle edits properly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			assets := make([]utils.TAsset, len(tt.inputOrder))
			for i, f := range tt.inputOrder {
				assets[i] = assetFactory(f, time.Now())
			}

			result := sortStack(assets, tt.promoteStr, "", []string{"~", "."}, utils.DefaultCriteria, &safePromoteData{data: make(map[string]map[string]string)}, make(map[int]map[string]int))

			for i, expectedFile := range tt.expectedOrder {
				assert.Equal(t, expectedFile, result[i].OriginalFileName,
					"%s: Position %d expected %s but got %s",
					tt.desc, i, expectedFile, result[i].OriginalFileName)
			}
		})
	}
}

func TestPixelEditedPhotoPromotion(t *testing.T) {

	tests := []struct {
		name          string
		assets        []utils.TAsset
		promoteStr    string
		expectedFirst string
		desc          string
	}{
		{
			name: "Pixel edited photo should be promoted with default settings",
			assets: []utils.TAsset{
				{
					OriginalFileName: "PXL_20250823_193751711.jpg",
					LocalDateTime:    "2025-08-23T19:37:51.000Z",
				},
				{
					OriginalFileName: "PXL_20250823_193751711~2.jpg",
					LocalDateTime:    "2025-08-23T19:37:51.000Z",
				},
			},
			promoteStr:    "edit,crop,hdr,biggestNumber",
			expectedFirst: "PXL_20250823_193751711~2.jpg",
			desc:          "With default settings, edited photo should be promoted",
		},
		{
			name: "Pixel RAW edited photo from real issue",
			assets: []utils.TAsset{
				{
					OriginalFileName: "PXL_20250628_123043121.RAW-01.COVER.jpg",
					LocalDateTime:    "2025-06-28T12:30:43.121Z",
				},
				{
					OriginalFileName: "PXL_20250628_123043121.RAW-01.COVER~2.jpg",
					LocalDateTime:    "2025-06-28T12:30:43.121Z",
				},
			},
			promoteStr:    "edit,crop,hdr,biggestNumber",
			expectedFirst: "PXL_20250628_123043121.RAW-01.COVER~2.jpg",
			desc:          "Real example from issue should promote edited version",
		},
		{
			name: "Multiple edits - highest number should be first",
			assets: []utils.TAsset{
				{
					OriginalFileName: "PXL_20250823_193751711.jpg",
					LocalDateTime:    "2025-08-23T19:37:51.000Z",
				},
				{
					OriginalFileName: "PXL_20250823_193751711~2.jpg",
					LocalDateTime:    "2025-08-23T19:37:51.000Z",
				},
				{
					OriginalFileName: "PXL_20250823_193751711~3.jpg",
					LocalDateTime:    "2025-08-23T19:37:51.000Z",
				},
				{
					OriginalFileName: "PXL_20250823_193751711~5.jpg",
					LocalDateTime:    "2025-08-23T19:37:51.000Z",
				},
			},
			promoteStr:    "edit,crop,hdr,biggestNumber",
			expectedFirst: "PXL_20250823_193751711~5.jpg",
			desc:          "Highest numbered edit should be promoted first",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			delimiters := []string{"~", "."}
			sorted := sortStack(tt.assets, tt.promoteStr, "", delimiters, utils.DefaultCriteria, &safePromoteData{data: make(map[string]map[string]string)}, make(map[int]map[string]int))

			assert.Equal(t, tt.expectedFirst, sorted[0].OriginalFileName,
				"%s: Expected %s to be first but got %s",
				tt.desc, tt.expectedFirst, sorted[0].OriginalFileName)

			t.Logf("Sorted order for %s:", tt.name)
			for i, asset := range sorted {
				t.Logf("  [%d] %s", i, asset.OriginalFileName)
			}
		})
	}
}

func TestPixelRegexExtraction(t *testing.T) {
	testCases := []struct {
		filename string
		regex    string
		index    int
		expected string
	}{
		{
			filename: "PXL_20250731_152626855.RAW-01.COVER.jpg",
			regex:    `(PXL|IMG)_(\d{8})_(\d+)`,
			index:    3,
			expected: "152626855",
		},
		{
			filename: "PXL_20250731_152626855.RAW-02.ORIGINAL.DNG",
			regex:    `(PXL|IMG)_(\d{8})_(\d+)`,
			index:    3,
			expected: "152626855",
		},
		{
			filename: "PXL_20250628_123043121.RAW-01.COVER~2.jpg",
			regex:    `(PXL|IMG)_(\d{8})_(\d+)`,
			index:    3,
			expected: "123043121",
		},
		{
			filename: "PXL_20250628_123043121.RAW-01.COVER.jpg",
			regex:    `(PXL|IMG)_(\d{8})_(\d+)`,
			index:    3,
			expected: "123043121",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			criteria := utils.TCriteria{
				Key: "originalFileName",
				Regex: &utils.TRegex{
					Key:   tc.regex,
					Index: tc.index,
				},
			}

			asset := utils.TAsset{OriginalFileName: tc.filename}
			result, _, err := extractOriginalFileName(asset, criteria)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, result, "Expected %s but got %s for %s", tc.expected, result, tc.filename)
		})
	}
}

func TestGetExtensionRank(t *testing.T) {
	tests := []struct {
		name     string
		ext      string
		expected int
	}{
		{
			name:     "jpeg has highest rank",
			ext:      ".jpeg",
			expected: 4,
		},
		{
			name:     "jpg has second highest rank",
			ext:      ".jpg",
			expected: 3,
		},
		{
			name:     "png has third highest rank",
			ext:      ".png",
			expected: 2,
		},
		{
			name:     "dng has default rank",
			ext:      ".dng",
			expected: 1,
		},
		{
			name:     "cr2 has default rank",
			ext:      ".cr2",
			expected: 1,
		},
		{
			name:     "arw has default rank",
			ext:      ".arw",
			expected: 1,
		},
		{
			name:     "empty extension has default rank",
			ext:      "",
			expected: 1,
		},
		{
			name:     "unknown extension has default rank",
			ext:      ".xyz",
			expected: 1,
		},
		{
			name:     "uppercase JPG treated as default (case sensitive)",
			ext:      ".JPG",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getExtensionRank(tt.ext)
			assert.Equal(t, tt.expected, result, "getExtensionRank(%q) should return %d", tt.ext, tt.expected)
		})
	}
}

func TestGetPromoteIndex(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		promoteList []string
		expected    int
	}{
		{
			name:        "empty promote list",
			value:       "photo.jpg",
			promoteList: []string{},
			expected:    0,
		},
		{
			name:        "match first in list",
			value:       "photo_edited.jpg",
			promoteList: []string{"_edited", "_crop", "_raw"},
			expected:    0,
		},
		{
			name:        "match second in list",
			value:       "photo_crop.jpg",
			promoteList: []string{"_edited", "_crop", "_raw"},
			expected:    1,
		},
		{
			name:        "match last in list",
			value:       "photo_raw.dng",
			promoteList: []string{"_edited", "_crop", "_raw"},
			expected:    2,
		},
		{
			name:        "no match returns list length",
			value:       "photo.jpg",
			promoteList: []string{"_edited", "_crop"},
			expected:    2,
		},
		{
			name:        "case insensitive match",
			value:       "photo_EDITED.jpg",
			promoteList: []string{"_edited"},
			expected:    0,
		},
		{
			name:        "nil promote list",
			value:       "photo.jpg",
			promoteList: nil,
			expected:    0,
		},
		{
			name:        "empty string as first priority - no other matches",
			value:       "photo.jpg",
			promoteList: []string{"", "_edited", "_crop"},
			expected:    0,
		},
		{
			name:        "empty string as first priority - has other match",
			value:       "photo_edited.jpg",
			promoteList: []string{"", "_edited", "_crop"},
			expected:    1,
		},
		{
			name:        "empty string only in list",
			value:       "photo.jpg",
			promoteList: []string{""},
			expected:    0,
		},
		{
			name:        "biggestNumber keyword",
			value:       "photo.jpg",
			promoteList: []string{"_edited", "biggestNumber"},
			expected:    1,
		},
		{
			name:        "biggestNumber keyword - has match before",
			value:       "photo_edited.jpg",
			promoteList: []string{"_edited", "biggestNumber"},
			expected:    0,
		},
		{
			name:        "multiple empty strings - first one wins",
			value:       "photo.jpg",
			promoteList: []string{"_edited", "", "_crop", ""},
			expected:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPromoteIndex(tt.value, tt.promoteList)
			assert.Equal(t, tt.expected, result, "getPromoteIndex(%q, %v) should return %d", tt.value, tt.promoteList, tt.expected)
		})
	}
}

func TestIsSequencePattern(t *testing.T) {
	tests := []struct {
		name        string
		promoteList []string
		expected    bool
	}{
		{
			name:        "empty list",
			promoteList: []string{},
			expected:    false,
		},
		{
			name:        "single item",
			promoteList: []string{"0001"},
			expected:    false,
		},
		{
			name:        "sequential numbers",
			promoteList: []string{"0001", "0002", "0003"},
			expected:    true,
		},
		{
			name:        "sequential with prefix",
			promoteList: []string{"img1", "img2", "img3"},
			expected:    true,
		},
		{
			name:        "sequential with suffix",
			promoteList: []string{"1_edit", "2_edit", "3_edit"},
			expected:    true,
		},
		{
			name:        "sequential with prefix and suffix",
			promoteList: []string{"IMG_001.jpg", "IMG_002.jpg", "IMG_003.jpg"},
			expected:    true,
		},
		{
			name:        "non-sequential numbers",
			promoteList: []string{"0001", "0003", "0002"},
			expected:    true, // After sorting, it becomes sequential
		},
		{
			name:        "non-numeric items",
			promoteList: []string{"abc", "def", "ghi"},
			expected:    false,
		},
		{
			name:        "mixed numeric and non-numeric",
			promoteList: []string{"img1", "abc", "img3"},
			expected:    false,
		},
		{
			name:        "different prefixes",
			promoteList: []string{"img1", "photo2", "img3"},
			expected:    false,
		},
		{
			name:        "different suffixes",
			promoteList: []string{"1.jpg", "2.png", "3.jpg"},
			expected:    false,
		},
		{
			name:        "biggestNumber keyword skipped",
			promoteList: []string{"0001", "biggestNumber", "0002"},
			expected:    true,
		},
		{
			name:        "only biggestNumber",
			promoteList: []string{"biggestNumber"},
			expected:    false,
		},
		{
			name:        "duplicate numbers",
			promoteList: []string{"001", "001", "002"},
			expected:    false, // Same number appears twice
		},
		{
			name:        "large gap in sequence",
			promoteList: []string{"0001", "0100"},
			expected:    true, // Gaps are allowed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSequencePattern(tt.promoteList)
			assert.Equal(t, tt.expected, result, "isSequencePattern(%v) should return %v", tt.promoteList, tt.expected)
		})
	}
}

func TestExtractSequencePattern(t *testing.T) {
	tests := []struct {
		name           string
		keyword        string
		expectedPrefix string
		expectedDigits int
	}{
		{
			name:           "plain sequence keyword",
			keyword:        "sequence",
			expectedPrefix: "",
			expectedDigits: 0,
		},
		{
			name:           "sequence with digit count",
			keyword:        "sequence:4",
			expectedPrefix: "",
			expectedDigits: 4,
		},
		{
			name:           "sequence with prefix",
			keyword:        "sequence:IMG_",
			expectedPrefix: "IMG_",
			expectedDigits: 0,
		},
		{
			name:           "sequence with complex prefix",
			keyword:        "sequence:PXL_20250731_",
			expectedPrefix: "PXL_20250731_",
			expectedDigits: 0,
		},
		{
			name:           "not a sequence keyword",
			keyword:        "other",
			expectedPrefix: "",
			expectedDigits: 0,
		},
		{
			name:           "empty string",
			keyword:        "",
			expectedPrefix: "",
			expectedDigits: 0,
		},
		{
			name:           "sequence with zero",
			keyword:        "sequence:0",
			expectedPrefix: "",
			expectedDigits: 0,
		},
		{
			name:           "sequence with large number",
			keyword:        "sequence:10",
			expectedPrefix: "",
			expectedDigits: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix, digits := extractSequencePattern(tt.keyword)
			assert.Equal(t, tt.expectedPrefix, prefix, "extractSequencePattern(%q) prefix should be %q", tt.keyword, tt.expectedPrefix)
			assert.Equal(t, tt.expectedDigits, digits, "extractSequencePattern(%q) digits should be %d", tt.keyword, tt.expectedDigits)
		})
	}
}
