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
** TestSortStack_BiggestSize_PartialExif: assets with a positive size always rank ahead of
** assets without exif data. The "has size" predicate is per-asset, not per-pair, which keeps
** the comparator transitive even though the alphabetical fall-through is non-monotonic with
** respect to size.
**************************************************************************************************/
func TestSortStack_BiggestSize_PartialExif(t *testing.T) {
	data, maps := emptyPromoteData()
	assets := []utils.TAsset{
		{ID: "z", OriginalFileName: "z.jpg"}, // no exif
		sizedAssetFactory("a.jpg", 5_000_000),
	}
	sorted := sortStack(assets, "biggestSize", "", []string{"_", "."},
		utils.DefaultCriteria, data, maps)
	assert.Equal(t, "a.jpg", sorted[0].OriginalFileName,
		"asset with exif data ranks ahead of asset without, even if alphabetically later")
	assert.Equal(t, "z.jpg", sorted[1].OriginalFileName,
		"missing-exif asset goes to the back bucket")
}

/**************************************************************************************************
** TestSortStack_BiggestSize_TransitivityWithMissingExif is the regression test for the bug
** Copilot caught on PR #64. The original implementation applied the size tie-break pairwise
** (only when both sides had positive size, falling through to alphabetical otherwise), which
** produced a non-transitive comparator with 3+ assets:
**
**   - z.jpg (10MB) vs a.jpg (5MB)  → both have size → z first by size
**   - z.jpg (10MB) vs m.jpg (none) → alphabetical fall-through → m first
**   - a.jpg (5MB)  vs m.jpg (none) → alphabetical fall-through → a first
**
** => z<a, a<m, m<z forms a cycle, breaking sort.SliceStable's total-order contract.
**
** The fix partitions assets into two buckets by a per-asset predicate (has size / no size).
** Assets with size go first (sorted by size); assets without size go to the back. This is a
** total order: bucket membership is determined by a single asset, not by the pair, so all
** triples are consistent.
**************************************************************************************************/
func TestSortStack_BiggestSize_TransitivityWithMissingExif(t *testing.T) {
	data, maps := emptyPromoteData()
	assets := []utils.TAsset{
		sizedAssetFactory("z.jpg", 10_000_000),
		sizedAssetFactory("a.jpg", 5_000_000),
		{ID: "m", OriginalFileName: "m.jpg"}, // no exif → goes to the back bucket
	}
	sorted := sortStack(assets, "biggestSize", "", []string{"_", "."},
		utils.DefaultCriteria, data, maps)
	got := []string{sorted[0].OriginalFileName, sorted[1].OriginalFileName, sorted[2].OriginalFileName}
	assert.Equal(t, []string{"z.jpg", "a.jpg", "m.jpg"}, got,
		"sized assets sort by size in front; the no-exif asset goes last (transitive)")
}

/**************************************************************************************************
** TestSortStack_SmallestSize_PartialExif: symmetric to biggestSize — assets with exif still
** beat assets without, regardless of the direction. A missing-size asset is treated as "no
** data", not "smallest possible", so it can never win as parent.
**************************************************************************************************/
func TestSortStack_SmallestSize_PartialExif(t *testing.T) {
	data, maps := emptyPromoteData()
	assets := []utils.TAsset{
		{ID: "a", OriginalFileName: "a.jpg"}, // no exif
		sizedAssetFactory("z.jpg", 5_000_000),
	}
	sorted := sortStack(assets, "smallestSize", "", []string{"_", "."},
		utils.DefaultCriteria, data, maps)
	assert.Equal(t, "z.jpg", sorted[0].OriginalFileName,
		"sized asset wins under smallestSize too — no-exif is not 'size 0'")
}

/**************************************************************************************************
** TestSortStack_SmallestSize_TransitivityWithMissingExif is the symmetric companion to the
** biggestSize transitivity test. With 3+ assets where one is missing exif, the bucket
** partition must still produce a strict total order in the smallest-first direction:
** sized assets at the front (ascending by size), no-exif at the back. Without this case the
** "no cycles in smallest direction" property would only be proven mathematically, not by a
** regression test that would catch a future refactor mistake.
**************************************************************************************************/
func TestSortStack_SmallestSize_TransitivityWithMissingExif(t *testing.T) {
	data, maps := emptyPromoteData()
	assets := []utils.TAsset{
		sizedAssetFactory("z.jpg", 10_000_000),
		sizedAssetFactory("a.jpg", 5_000_000),
		{ID: "m", OriginalFileName: "m.jpg"}, // no exif → back bucket
	}
	sorted := sortStack(assets, "smallestSize", "", []string{"_", "."},
		utils.DefaultCriteria, data, maps)
	got := []string{sorted[0].OriginalFileName, sorted[1].OriginalFileName, sorted[2].OriginalFileName}
	assert.Equal(t, []string{"a.jpg", "z.jpg", "m.jpg"}, got,
		"smallest first within sized bucket; no-exif goes last (transitive in smallest direction)")
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
