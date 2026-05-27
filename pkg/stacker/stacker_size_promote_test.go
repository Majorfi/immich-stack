package stacker

import (
	"encoding/json"
	"testing"

	"github.com/majorfi/immich-stack/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/**************************************************************************************************
** sizedAssetFactory builds a TAsset with the given filename and size in bytes. Used here rather
** than assetFactory because size-promotion tests need to control ExifInfo.FileSizeInByte directly.
**************************************************************************************************/
func sizedAssetFactory(filename string, sizeInByte int64) utils.TAsset {
	return utils.TAsset{
		ID:               filename,
		OriginalFileName: filename,
		ExifInfo:         &utils.TExifInfo{FileSizeInByte: sizeInByte},
	}
}

/**************************************************************************************************
** emptyPromoteData returns an empty thread-safe promote data store + empty promotion maps.
** Used to satisfy sortStack's signature when no regex-promotion criteria are exercised.
**************************************************************************************************/
func emptyPromoteData() (*safePromoteData, map[int]map[string]int) {
	return &safePromoteData{data: make(map[string]map[string]string)}, make(map[int]map[string]int)
}

/**************************************************************************************************
** TestSortStack_BiggestSize covers the new biggestSize magic keyword: when present in the
** promote list, the asset with the largest exifInfo.fileSizeInByte wins the parent slot.
**************************************************************************************************/
func TestSortStack_BiggestSize(t *testing.T) {
	data, maps := emptyPromoteData()
	assets := []utils.TAsset{
		sizedAssetFactory("IMG_1234.jpg", 2_500_000),  // small original
		sizedAssetFactory("IMG_1234_b.jpg", 8_400_000), // large exported
		sizedAssetFactory("IMG_1234_a.jpg", 5_100_000), // medium
	}
	sorted := sortStack(assets, "biggestSize", "", []string{"_", "."},
		utils.DefaultCriteria, data, maps)
	assert.Equal(t, "IMG_1234_b.jpg", sorted[0].OriginalFileName,
		"largest file should be promoted as parent")
	assert.Equal(t, "IMG_1234_a.jpg", sorted[1].OriginalFileName)
	assert.Equal(t, "IMG_1234.jpg", sorted[2].OriginalFileName)
}

/**************************************************************************************************
** TestSortStack_SmallestSize is the symmetric counterpart — useful for stacks where the
** thumbnail/reduced variant should win (less common but supported for symmetry).
**************************************************************************************************/
func TestSortStack_SmallestSize(t *testing.T) {
	data, maps := emptyPromoteData()
	assets := []utils.TAsset{
		sizedAssetFactory("IMG_a.jpg", 8_000_000),
		sizedAssetFactory("IMG_b.jpg", 1_200_000),
		sizedAssetFactory("IMG_c.jpg", 4_500_000),
	}
	sorted := sortStack(assets, "smallestSize", "", []string{"_", "."},
		utils.DefaultCriteria, data, maps)
	assert.Equal(t, "IMG_b.jpg", sorted[0].OriginalFileName,
		"smallest file should be promoted as parent")
}

/**************************************************************************************************
** TestSortStack_BiggestSize_WithSubstringMatch: substring promotes still take priority over the
** size tie-breaker. This is the realistic case where a user wants edits to win, with size only
** breaking ties within the matched (or unmatched) buckets.
**************************************************************************************************/
func TestSortStack_BiggestSize_WithSubstringMatch(t *testing.T) {
	data, maps := emptyPromoteData()
	assets := []utils.TAsset{
		sizedAssetFactory("IMG_1234.jpg", 9_000_000),         // unmatched, huge
		sizedAssetFactory("IMG_1234_edited.jpg", 3_000_000),  // matches "_edited", small
		sizedAssetFactory("IMG_1234_edited2.jpg", 5_500_000), // matches "_edited", medium
	}
	sorted := sortStack(assets, "_edited,biggestSize", "", []string{"_", "."},
		utils.DefaultCriteria, data, maps)
	// "_edited" match wins over the larger unmatched original, then size tie-breaks
	// inside the matched bucket.
	assert.Equal(t, "IMG_1234_edited2.jpg", sorted[0].OriginalFileName,
		"largest _edited file should be parent")
	assert.Equal(t, "IMG_1234_edited.jpg", sorted[1].OriginalFileName)
	assert.Equal(t, "IMG_1234.jpg", sorted[2].OriginalFileName,
		"unmatched original is last despite being largest")
}

/**************************************************************************************************
** TestSortStack_BiggestSize_MissingExif verifies the "no exif" fall-through: when neither asset
** has exif data we must NOT pin them to size=0=0 (which would short-circuit the tie-break);
** the alphabetical fallback must still kick in.
**************************************************************************************************/
func TestSortStack_BiggestSize_MissingExif(t *testing.T) {
	data, maps := emptyPromoteData()
	assets := []utils.TAsset{
		{ID: "b", OriginalFileName: "b.jpg"},
		{ID: "a", OriginalFileName: "a.jpg"},
	}
	sorted := sortStack(assets, "biggestSize", "", []string{"_", "."},
		utils.DefaultCriteria, data, maps)
	assert.Equal(t, "a.jpg", sorted[0].OriginalFileName,
		"with no exif data, fallback alphabetical sort must still apply")
}

/**************************************************************************************************
** TestSortStack_BiggestSize_PartialExif: when only one asset has exif we should not flip the
** alphabetical order just because the other defaults to 0. The size tie-break is skipped when
** EITHER asset is missing data, so the relative order falls through to the next sort rule.
**************************************************************************************************/
func TestSortStack_BiggestSize_PartialExif(t *testing.T) {
	data, maps := emptyPromoteData()
	assets := []utils.TAsset{
		sizedAssetFactory("a.jpg", 5_000_000),
		{ID: "b", OriginalFileName: "b.jpg"}, // no exif
	}
	sorted := sortStack(assets, "biggestSize", "", []string{"_", "."},
		utils.DefaultCriteria, data, maps)
	// Without partial-exif protection, "a" (size=5MB) would beat "b" (size=0) and we'd
	// promote a.jpg even though b.jpg has no comparable metadata. Alphabetical wins instead.
	assert.Equal(t, "a.jpg", sorted[0].OriginalFileName,
		"alphabetical resolves the tie when one side has no exif")
}

/**************************************************************************************************
** TestSortStack_BiggestSize_DSLRScenario reproduces the exact use case from issue #47: an original
** JPG (camera output) + RAW/CR2 + a substantially larger exported JPG from Lightroom. The
** exported JPG should win as the stack parent.
**************************************************************************************************/
func TestSortStack_BiggestSize_DSLRScenario(t *testing.T) {
	data, maps := emptyPromoteData()
	assets := []utils.TAsset{
		sizedAssetFactory("IMG_5821.JPG", 4_800_000),  // camera JPG
		sizedAssetFactory("IMG_5821.CR2", 28_000_000), // RAW
		sizedAssetFactory("IMG_5821.jpg", 12_400_000), // exported JPG (the desired parent)
	}
	// PARENT_EXT_PROMOTE prefers .jpg over .cr2; biggestSize tie-breaks between the two .jpg files.
	sorted := sortStack(assets, "biggestSize", ".jpg,.jpeg,.png", []string{"_", "."},
		utils.DefaultCriteria, data, maps)
	assert.Equal(t, "IMG_5821.jpg", sorted[0].OriginalFileName,
		"exported JPG (largest .jpg) should be the stack parent")
}

/**************************************************************************************************
** TestSortStack_BiggestSize_EqualSizes: when sizes match exactly we should fall through to the
** next tie-breaker (extension rank / alphabetical). Guards against accidentally claiming the
** tie when the comparison is a no-op.
**************************************************************************************************/
func TestSortStack_BiggestSize_EqualSizes(t *testing.T) {
	data, maps := emptyPromoteData()
	assets := []utils.TAsset{
		sizedAssetFactory("z.jpg", 5_000_000),
		sizedAssetFactory("a.jpg", 5_000_000),
	}
	sorted := sortStack(assets, "biggestSize", "", []string{"_", "."},
		utils.DefaultCriteria, data, maps)
	assert.Equal(t, "a.jpg", sorted[0].OriginalFileName,
		"equal sizes fall through to alphabetical")
}

/**************************************************************************************************
** TestExifInfo_JSONUnmarshal confirms the API plumbing: an Immich-shaped JSON payload with the
** exifInfo.fileSizeInByte field is correctly deserialized into TAsset.ExifInfo. Without this,
** the size tie-break above can never engage in production.
**************************************************************************************************/
func TestExifInfo_JSONUnmarshal(t *testing.T) {
	payload := []byte(`{
		"id": "abc",
		"originalFileName": "IMG.jpg",
		"exifInfo": {
			"fileSizeInByte": 12345678
		}
	}`)
	var asset utils.TAsset
	require.NoError(t, json.Unmarshal(payload, &asset))
	require.NotNil(t, asset.ExifInfo, "exifInfo should be deserialized")
	assert.Equal(t, int64(12345678), asset.ExifInfo.FileSizeInByte)
}

/**************************************************************************************************
** TestExifInfo_JSONUnmarshal_Missing covers Immich responses that omit exifInfo entirely.
** ExifInfo must stay nil (not be auto-initialized) so the size tie-breaker can skip cleanly.
**************************************************************************************************/
func TestExifInfo_JSONUnmarshal_Missing(t *testing.T) {
	payload := []byte(`{"id": "abc", "originalFileName": "IMG.jpg"}`)
	var asset utils.TAsset
	require.NoError(t, json.Unmarshal(payload, &asset))
	assert.Nil(t, asset.ExifInfo, "missing exifInfo must leave ExifInfo nil")
}
