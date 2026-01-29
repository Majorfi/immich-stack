package stacker

import (
	"testing"
	"time"

	"github.com/majorfi/immich-stack/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/************************************************************************************************
** Tests corresponding to every scenario in EXAMPLES.md
** Each test validates both grouping (do files end up in the same stack?) and parent selection
** (which file is on top?).
************************************************************************************************/

func examplesLogger() *logrus.Logger {
	l := logrus.New()
	l.SetLevel(logrus.WarnLevel)
	return l
}

/************************************************************************************************
** RAW + JPEG Pairing
************************************************************************************************/

func TestExamples_CanonNikonSony_RawJpegGrouping(t *testing.T) {
	now := time.Now()
	assets := []utils.TAsset{
		assetFactory("IMG_1234.jpg", now),
		assetFactory("IMG_1234.CR2", now),
	}
	groups, err := StackBy(assets, "", "", "", examplesLogger())
	require.NoError(t, err)
	assert.Equal(t, 1, len(groups))
}

func TestExamples_CanonNikonSony_JpegOnTop(t *testing.T) {
	assets := []utils.TAsset{
		assetFactory("IMG_1234.CR2", time.Now()),
		assetFactory("IMG_1234.jpg", time.Now()),
	}
	sorted := sortStack(assets, utils.DefaultParentFilenamePromoteString, utils.DefaultParentExtPromoteString, []string{"~", "."}, utils.DefaultCriteria,
		&safePromoteData{data: make(map[string]map[string]string)}, make(map[int]map[string]int))
	assert.Equal(t, "IMG_1234.jpg", sorted[0].OriginalFileName)
}

func TestExamples_Fujifilm_RafJpegGrouping(t *testing.T) {
	now := time.Now()
	assets := []utils.TAsset{
		assetFactory("DSCF1234.jpg", now),
		assetFactory("DSCF1234.RAF", now),
	}
	groups, err := StackBy(assets, "", "", "", examplesLogger())
	require.NoError(t, err)
	assert.Equal(t, 1, len(groups))
}

func TestExamples_Fujifilm_JpegOnTop(t *testing.T) {
	assets := []utils.TAsset{
		assetFactory("DSCF1234.RAF", time.Now()),
		assetFactory("DSCF1234.jpg", time.Now()),
	}
	sorted := sortStack(assets, utils.DefaultParentFilenamePromoteString, utils.DefaultParentExtPromoteString, []string{"~", "."}, utils.DefaultCriteria,
		&safePromoteData{data: make(map[string]map[string]string)}, make(map[int]map[string]int))
	assert.Equal(t, "DSCF1234.jpg", sorted[0].OriginalFileName)
}

func TestExamples_Samsung_JpgDngGrouping(t *testing.T) {
	now := time.Now()
	assets := []utils.TAsset{
		assetFactory("20240115_143022.jpg", now),
		assetFactory("20240115_143022.dng", now),
	}
	groups, err := StackBy(assets, "", "", "", examplesLogger())
	require.NoError(t, err)
	assert.Equal(t, 1, len(groups))
}

func TestExamples_Samsung_JpgOnTop(t *testing.T) {
	assets := []utils.TAsset{
		assetFactory("20240115_143022.dng", time.Now()),
		assetFactory("20240115_143022.jpg", time.Now()),
	}
	sorted := sortStack(assets, utils.DefaultParentFilenamePromoteString, utils.DefaultParentExtPromoteString, []string{"~", "."}, utils.DefaultCriteria,
		&safePromoteData{data: make(map[string]map[string]string)}, make(map[int]map[string]int))
	assert.Equal(t, "20240115_143022.jpg", sorted[0].OriginalFileName)
}

func TestExamples_iPhoneProRaw_Grouping(t *testing.T) {
	now := time.Now()
	assets := []utils.TAsset{
		assetFactory("IMG_1234.HEIC", now),
		assetFactory("IMG_1234.DNG", now),
	}
	groups, err := StackBy(assets, "", "", "", examplesLogger())
	require.NoError(t, err)
	assert.Equal(t, 1, len(groups))
}

func TestExamples_iPhoneProRaw_HeicOnTop(t *testing.T) {
	assets := []utils.TAsset{
		assetFactory("IMG_1234.DNG", time.Now()),
		assetFactory("IMG_1234.HEIC", time.Now()),
	}
	sorted := sortStack(assets, utils.DefaultParentFilenamePromoteString, utils.DefaultParentExtPromoteString, []string{"~", "."}, utils.DefaultCriteria,
		&safePromoteData{data: make(map[string]map[string]string)}, make(map[int]map[string]int))
	assert.Equal(t, "IMG_1234.HEIC", sorted[0].OriginalFileName,
		"HEIC should win over DNG via extension promotion")
}

/************************************************************************************************
** Google Pixel Photos
************************************************************************************************/

func TestExamples_PixelRawJpeg_Grouping(t *testing.T) {
	now := time.Now()
	assets := []utils.TAsset{
		assetFactory("PXL_20260121_195958829.RAW-01.COVER.jpg", now),
		assetFactory("PXL_20260121_195958829.RAW-02.ORIGINAL.dng", now),
	}
	groups, err := StackBy(assets, "", "", "", examplesLogger())
	require.NoError(t, err)
	assert.Equal(t, 1, len(groups))
}

func TestExamples_PixelRawJpeg_JpegOnTop(t *testing.T) {
	assets := []utils.TAsset{
		assetFactory("PXL_20260121_195958829.RAW-02.ORIGINAL.dng", time.Now()),
		assetFactory("PXL_20260121_195958829.RAW-01.COVER.jpg", time.Now()),
	}
	sorted := sortStack(assets, utils.DefaultParentFilenamePromoteString, utils.DefaultParentExtPromoteString, []string{"~", "."}, utils.DefaultCriteria,
		&safePromoteData{data: make(map[string]map[string]string)}, make(map[int]map[string]int))
	assert.Equal(t, "PXL_20260121_195958829.RAW-01.COVER.jpg", sorted[0].OriginalFileName)
}

func TestExamples_PixelMotionPhotos_Grouping(t *testing.T) {
	now := time.Now()
	assets := []utils.TAsset{
		assetFactory("PXL_20240115_143022345.jpg", now),
		assetFactory("PXL_20240115_143022345.MP.jpg", now),
	}
	groups, err := StackBy(assets, "", "", "", examplesLogger())
	require.NoError(t, err)
	assert.Equal(t, 1, len(groups))
}

func TestExamples_PixelMotionPhotos_DefaultOrder(t *testing.T) {
	assets := []utils.TAsset{
		assetFactory("PXL_20240115_143022345.jpg", time.Now()),
		assetFactory("PXL_20240115_143022345.MP.jpg", time.Now()),
	}
	sorted := sortStack(assets, utils.DefaultParentFilenamePromoteString, utils.DefaultParentExtPromoteString, []string{"~", "."}, utils.DefaultCriteria,
		&safePromoteData{data: make(map[string]map[string]string)}, make(map[int]map[string]int))
	assert.Equal(t, "PXL_20240115_143022345.MP.jpg", sorted[0].OriginalFileName,
		"MP variant should be on top by default (uppercase M sorts before lowercase j)")
}

func TestExamples_PixelMotionPhotos_MpOnTopExplicit(t *testing.T) {
	assets := []utils.TAsset{
		assetFactory("PXL_20240115_143022345.jpg", time.Now()),
		assetFactory("PXL_20240115_143022345.MP.jpg", time.Now()),
	}
	sorted := sortStack(assets, "mp,cover,edit,crop,hdr,biggestNumber", utils.DefaultParentExtPromoteString, []string{"~", "."}, utils.DefaultCriteria,
		&safePromoteData{data: make(map[string]map[string]string)}, make(map[int]map[string]int))
	assert.Equal(t, "PXL_20240115_143022345.MP.jpg", sorted[0].OriginalFileName)
}

func TestExamples_PixelMotionPhotos_RegularOnTopWithNegativeMatch(t *testing.T) {
	assets := []utils.TAsset{
		assetFactory("PXL_20240115_143022345.MP.jpg", time.Now()),
		assetFactory("PXL_20240115_143022345.jpg", time.Now()),
	}
	sorted := sortStack(assets, ",mp,cover,edit,crop,hdr,biggestNumber", utils.DefaultParentExtPromoteString, []string{"~", "."}, utils.DefaultCriteria,
		&safePromoteData{data: make(map[string]map[string]string)}, make(map[int]map[string]int))
	assert.Equal(t, "PXL_20240115_143022345.jpg", sorted[0].OriginalFileName,
		"Leading empty string should promote files without mp keyword")
}

func TestExamples_Pixel10Pro_Grouping(t *testing.T) {
	now := time.Now()
	assets := []utils.TAsset{
		assetFactory("PXL_20260120_120000000.jpg", now),
		assetFactory("PXL_20260120_120000000.dng", now),
		assetFactory("PXL_20260120_120000000.NIGHT.jpg", now),
	}
	groups, err := StackBy(assets, "", "", "", examplesLogger())
	require.NoError(t, err)
	assert.Equal(t, 1, len(groups))
}

func TestExamples_Pixel10Pro_DefaultOrder(t *testing.T) {
	assets := []utils.TAsset{
		assetFactory("PXL_20260120_120000000.jpg", time.Now()),
		assetFactory("PXL_20260120_120000000.dng", time.Now()),
		assetFactory("PXL_20260120_120000000.NIGHT.jpg", time.Now()),
	}
	sorted := sortStack(assets, utils.DefaultParentFilenamePromoteString, utils.DefaultParentExtPromoteString, []string{"~", "."}, utils.DefaultCriteria,
		&safePromoteData{data: make(map[string]map[string]string)}, make(map[int]map[string]int))
	assert.Equal(t, "PXL_20260120_120000000.NIGHT.jpg", sorted[0].OriginalFileName,
		"NIGHT variant wins by default (uppercase N sorts before lowercase j)")
}

func TestExamples_Pixel10Pro_OriginalJpegOnTopWithNegativeMatch(t *testing.T) {
	assets := []utils.TAsset{
		assetFactory("PXL_20260120_120000000.NIGHT.jpg", time.Now()),
		assetFactory("PXL_20260120_120000000.dng", time.Now()),
		assetFactory("PXL_20260120_120000000.jpg", time.Now()),
	}
	sorted := sortStack(assets, ",night,cover,edit,crop,hdr,biggestNumber", utils.DefaultParentExtPromoteString, []string{"~", "."}, utils.DefaultCriteria,
		&safePromoteData{data: make(map[string]map[string]string)}, make(map[int]map[string]int))
	assert.Equal(t, "PXL_20260120_120000000.jpg", sorted[0].OriginalFileName,
		"Negative match should put the plain jpg on top")
	assert.Equal(t, "PXL_20260120_120000000.dng", sorted[1].OriginalFileName,
		"dng without night keyword should be second")
	assert.Equal(t, "PXL_20260120_120000000.NIGHT.jpg", sorted[2].OriginalFileName,
		"NIGHT variant should be last")
}

/************************************************************************************************
** Google Photos Edited Versions
************************************************************************************************/

func TestExamples_GooglePhotosEdited_Grouping(t *testing.T) {
	now := time.Now()
	assets := []utils.TAsset{
		assetFactory("vacation_sunset.jpg", now),
		assetFactory("vacation_sunset-edited.jpg", now),
	}
	criteria := `[{"key":"originalFileName","split":{"delimiters":["-","~","."],"index":0}},{"key":"localDateTime","delta":{"milliseconds":1000}}]`
	groups, err := StackBy(assets, criteria, "", "", examplesLogger())
	require.NoError(t, err)
	assert.Equal(t, 1, len(groups))
}

func TestExamples_GooglePhotosEdited_DifferentBaseNotGrouped(t *testing.T) {
	now := time.Now()
	assets := []utils.TAsset{
		assetFactory("beach_sunset.jpg", now),
		assetFactory("vacation_sunset-edited.jpg", now),
	}
	criteria := `[{"key":"originalFileName","split":{"delimiters":["-","~","."],"index":0}},{"key":"localDateTime","delta":{"milliseconds":1000}}]`
	groups, err := StackBy(assets, criteria, "", "", examplesLogger())
	require.NoError(t, err)
	assert.Equal(t, 0, len(groups))
}

func TestExamples_GooglePhotosEdited_EditedOnTop(t *testing.T) {
	assets := []utils.TAsset{
		assetFactory("vacation_sunset.jpg", time.Now()),
		assetFactory("vacation_sunset-edited.jpg", time.Now()),
	}
	sorted := sortStack(assets, "edit,cover,crop,hdr,biggestNumber", "", []string{"-", "~", "."}, utils.DefaultCriteria,
		&safePromoteData{data: make(map[string]map[string]string)}, make(map[int]map[string]int))
	assert.Equal(t, "vacation_sunset-edited.jpg", sorted[0].OriginalFileName,
		"edit keyword matches vacation_sunset-edited.jpg")
}

/************************************************************************************************
** RAW+JPEG with Lightroom Numeric Edits
************************************************************************************************/

func TestExamples_LightroomEdits_Grouping(t *testing.T) {
	now := time.Now()
	assets := []utils.TAsset{
		assetFactory("ABC001.ARW", now),
		assetFactory("ABC001.JPEG", now),
		assetFactory("ABC001-1.JPEG", now),
		assetFactory("ABC001-2.JPEG", now),
	}
	criteria := `[{"key":"originalFileName","split":{"delimiters":["-","~","."],"index":0}},{"key":"localDateTime","delta":{"milliseconds":1000}}]`
	groups, err := StackBy(assets, criteria, "", "", examplesLogger())
	require.NoError(t, err)
	assert.Equal(t, 1, len(groups))
	assert.Equal(t, 4, len(groups[0]))
}

func TestExamples_LightroomEdits_LatestEditOnTop(t *testing.T) {
	assets := []utils.TAsset{
		assetFactory("ABC001.ARW", time.Now()),
		assetFactory("ABC001.JPEG", time.Now()),
		assetFactory("ABC001-1.JPEG", time.Now()),
		assetFactory("ABC001-2.JPEG", time.Now()),
	}
	sorted := sortStack(assets, "cover,edit,crop,hdr,biggestNumber", ".jpg,.jpeg,.png,.dng,.arw",
		[]string{"-", "~", "."}, utils.DefaultCriteria,
		&safePromoteData{data: make(map[string]map[string]string)}, make(map[int]map[string]int))
	assert.Equal(t, "ABC001-2.JPEG", sorted[0].OriginalFileName,
		"biggestNumber should put the highest numeric suffix on top")
}

func TestExamples_LightroomEdits_JpegOverRawWhenNoNumericEdit(t *testing.T) {
	assets := []utils.TAsset{
		assetFactory("ABC001.ARW", time.Now()),
		assetFactory("ABC001.JPEG", time.Now()),
	}
	sorted := sortStack(assets, "cover,edit,crop,hdr,biggestNumber", ".jpg,.jpeg,.png,.dng,.arw",
		[]string{"-", "~", "."}, utils.DefaultCriteria,
		&safePromoteData{data: make(map[string]map[string]string)}, make(map[int]map[string]int))
	assert.Equal(t, "ABC001.JPEG", sorted[0].OriginalFileName,
		"JPEG should win over ARW via extension promotion")
}

func TestExamples_LightroomEdits_PixelRawAlsoGroups(t *testing.T) {
	now := time.Now()
	assets := []utils.TAsset{
		assetFactory("PXL_20250405_142136468.RAW-01.COVER.jpg", now),
		assetFactory("PXL_20250405_142136468.RAW-02.ORIGINAL.dng", now),
	}
	criteria := `[{"key":"originalFileName","split":{"delimiters":["-","~","."],"index":0}},{"key":"localDateTime","delta":{"milliseconds":1000}}]`
	groups, err := StackBy(assets, criteria, "", "", examplesLogger())
	require.NoError(t, err)
	assert.Equal(t, 1, len(groups))
}

/************************************************************************************************
** Photoshop Workflows
************************************************************************************************/

func TestExamples_Photoshop_RawJpegPsdGrouping(t *testing.T) {
	now := time.Now()
	assets := []utils.TAsset{
		assetFactory("IMG_1234.CR2", now),
		assetFactory("IMG_1234.jpg", now),
		assetFactory("IMG_1234.psd", now),
	}
	groups, err := StackBy(assets, "", "", "", examplesLogger())
	require.NoError(t, err)
	assert.Equal(t, 1, len(groups))
	assert.Equal(t, 3, len(groups[0]))
}

func TestExamples_Photoshop_RawJpegPsdWithFinalGrouping(t *testing.T) {
	now := time.Now()
	assets := []utils.TAsset{
		assetFactory("IMG_1234.CR2", now),
		assetFactory("IMG_1234.jpg", now),
		assetFactory("IMG_1234.psd", now),
		assetFactory("IMG_1234-final.jpg", now),
	}
	criteria := `[{"key":"originalFileName","split":{"delimiters":["-","~","."],"index":0}},{"key":"localDateTime","delta":{"milliseconds":1000}}]`
	groups, err := StackBy(assets, criteria, "", "", examplesLogger())
	require.NoError(t, err)
	assert.Equal(t, 1, len(groups))
	assert.Equal(t, 4, len(groups[0]))
}

func TestExamples_Photoshop_FinalJpegOnTop(t *testing.T) {
	assets := []utils.TAsset{
		assetFactory("IMG_1234.CR2", time.Now()),
		assetFactory("IMG_1234.jpg", time.Now()),
		assetFactory("IMG_1234.psd", time.Now()),
		assetFactory("IMG_1234-final.jpg", time.Now()),
	}
	sorted := sortStack(assets, "final,edit,cover,crop,hdr,biggestNumber", ".jpg,.jpeg,.png,.heic,.psd,.dng,.cr2",
		[]string{"-", "~", "."}, utils.DefaultCriteria,
		&safePromoteData{data: make(map[string]map[string]string)}, make(map[int]map[string]int))
	assert.Equal(t, "IMG_1234-final.jpg", sorted[0].OriginalFileName,
		"final keyword should promote the final export")
	assert.Equal(t, "IMG_1234.jpg", sorted[1].OriginalFileName,
		"jpg should be second via extension promotion")
}

func TestExamples_Photoshop_VersionedExportsGrouping(t *testing.T) {
	now := time.Now()
	assets := []utils.TAsset{
		assetFactory("portrait.psd", now),
		assetFactory("portrait_1.jpg", now),
		assetFactory("portrait_2.jpg", now),
		assetFactory("portrait_final.jpg", now),
	}
	criteria := `[{"key":"originalFileName","split":{"delimiters":["_","~","."],"index":0}},{"key":"localDateTime","delta":{"milliseconds":86400000}}]`
	groups, err := StackBy(assets, criteria, "", "", examplesLogger())
	require.NoError(t, err)
	assert.Equal(t, 1, len(groups))
	assert.Equal(t, 4, len(groups[0]))
}

func TestExamples_Photoshop_VersionedFinalOnTop(t *testing.T) {
	assets := []utils.TAsset{
		assetFactory("portrait.psd", time.Now()),
		assetFactory("portrait_1.jpg", time.Now()),
		assetFactory("portrait_2.jpg", time.Now()),
		assetFactory("portrait_final.jpg", time.Now()),
	}
	sorted := sortStack(assets, "final,biggestNumber", ".jpg,.jpeg,.png,.psd",
		[]string{"_", "~", "."}, utils.DefaultCriteria,
		&safePromoteData{data: make(map[string]map[string]string)}, make(map[int]map[string]int))
	assert.Equal(t, "portrait_final.jpg", sorted[0].OriginalFileName,
		"final keyword has highest priority")
}

func TestExamples_Photoshop_VersionedWithoutFinal(t *testing.T) {
	assets := []utils.TAsset{
		assetFactory("portrait.psd", time.Now()),
		assetFactory("portrait_1.jpg", time.Now()),
		assetFactory("portrait_2.jpg", time.Now()),
	}
	sorted := sortStack(assets, "final,biggestNumber", ".jpg,.jpeg,.png,.psd",
		[]string{"_", "~", "."}, utils.DefaultCriteria,
		&safePromoteData{data: make(map[string]map[string]string)}, make(map[int]map[string]int))
	assert.Equal(t, "portrait_2.jpg", sorted[0].OriginalFileName,
		"biggestNumber picks numeric suffix 2 over 1")
}

func TestExamples_Photoshop_VersionedWithVPrefixAlphabetical(t *testing.T) {
	assets := []utils.TAsset{
		assetFactory("portrait.psd", time.Now()),
		assetFactory("portrait_v1.jpg", time.Now()),
		assetFactory("portrait_v2.jpg", time.Now()),
	}
	sorted := sortStack(assets, "final,biggestNumber", ".jpg,.jpeg,.png,.psd",
		[]string{"_", "~", "."}, utils.DefaultCriteria,
		&safePromoteData{data: make(map[string]map[string]string)}, make(map[int]map[string]int))
	assert.Equal(t, "portrait_v1.jpg", sorted[0].OriginalFileName,
		"v1/v2 don't match biggestNumber pattern (requires pure numbers), falls to alphabetical")
}

func TestExamples_Photoshop_PsdOnTopWhenPreferred(t *testing.T) {
	assets := []utils.TAsset{
		assetFactory("portrait.psd", time.Now()),
		assetFactory("portrait_1.jpg", time.Now()),
		assetFactory("portrait_2.jpg", time.Now()),
	}
	sorted := sortStack(assets, "", ".psd,.jpg,.jpeg,.png",
		[]string{"_", "~", "."}, utils.DefaultCriteria,
		&safePromoteData{data: make(map[string]map[string]string)}, make(map[int]map[string]int))
	assert.Equal(t, "portrait.psd", sorted[0].OriginalFileName,
		"psd first in ext promote list makes it parent")
}

func TestExamples_Photoshop_ApertureVaultGrouping(t *testing.T) {
	now := time.Now()
	assets := []utils.TAsset{
		assetFactory("IMG_1234.CR2", now),
		assetFactory("IMG_1234.jpg", now),
		assetFactory("IMG_1234-Edit.psd", now),
		assetFactory("IMG_1234-Edit.jpg", now),
	}
	criteria := `[{"key":"originalFileName","split":{"delimiters":["-","~","."],"index":0}},{"key":"localDateTime","delta":{"milliseconds":86400000}}]`
	groups, err := StackBy(assets, criteria, "", "", examplesLogger())
	require.NoError(t, err)
	assert.Equal(t, 1, len(groups))
	assert.Equal(t, 4, len(groups[0]))
}

func TestExamples_Photoshop_ApertureVaultEditedJpegOnTop(t *testing.T) {
	assets := []utils.TAsset{
		assetFactory("IMG_1234.CR2", time.Now()),
		assetFactory("IMG_1234.jpg", time.Now()),
		assetFactory("IMG_1234-Edit.psd", time.Now()),
		assetFactory("IMG_1234-Edit.jpg", time.Now()),
	}
	sorted := sortStack(assets, "edit,cover,crop,hdr,biggestNumber", ".jpg,.jpeg,.png,.heic,.psd,.dng,.cr2",
		[]string{"-", "~", "."}, utils.DefaultCriteria,
		&safePromoteData{data: make(map[string]map[string]string)}, make(map[int]map[string]int))
	assert.Equal(t, "IMG_1234-Edit.jpg", sorted[0].OriginalFileName,
		"edit keyword matches, jpg extension wins over psd")
	assert.Equal(t, "IMG_1234-Edit.psd", sorted[1].OriginalFileName,
		"psd with edit keyword is second")
}

func TestExamples_Photoshop_DifferentBaseNotGrouped(t *testing.T) {
	now := time.Now()
	assets := []utils.TAsset{
		assetFactory("IMG_1234.jpg", now),
		assetFactory("IMG_5678.psd", now),
	}
	groups, err := StackBy(assets, "", "", "", examplesLogger())
	require.NoError(t, err)
	assert.Equal(t, 0, len(groups), "Different base filenames should not group")
}

/************************************************************************************************
** Burst Photos - Camera Bursts with Shared Timestamp
************************************************************************************************/

func TestExamples_CameraBurst_RegexGrouping(t *testing.T) {
	assets := []utils.TAsset{
		{OriginalFileName: "DSCPDC_0000_BURST20180828114700954.JPG", LocalDateTime: "2018-08-28T11:47:00.460Z"},
		{OriginalFileName: "DSCPDC_0001_BURST20180828114700954.JPG", LocalDateTime: "2018-08-28T11:47:00.608Z"},
		{OriginalFileName: "DSCPDC_0002_BURST20180828114700954.JPG", LocalDateTime: "2018-08-28T11:47:00.758Z"},
		{OriginalFileName: "DSCPDC_0003_BURST20180828114700954_COVER.JPG", LocalDateTime: "2018-08-28T11:47:00.910Z"},
	}
	criteria := `[{"key":"originalFileName","regex":{"key":"BURST(\\d+)","index":1}},{"key":"localDateTime","delta":{"milliseconds":1000}}]`
	groups, err := StackBy(assets, criteria, "", "", examplesLogger())
	require.NoError(t, err)
	assert.Equal(t, 1, len(groups))
	assert.Equal(t, 4, len(groups[0]))
}

func TestExamples_CameraBurst_CoverOnTopWithSequence(t *testing.T) {
	assets := []utils.TAsset{
		assetFactory("DSCPDC_0002_BURST20180828114700954.JPG", time.Now()),
		assetFactory("DSCPDC_0000_BURST20180828114700954.JPG", time.Now()),
		assetFactory("DSCPDC_0003_BURST20180828114700954_COVER.JPG", time.Now()),
		assetFactory("DSCPDC_0001_BURST20180828114700954.JPG", time.Now()),
	}
	sorted := sortStack(assets, "cover,sequence", "", []string{}, utils.DefaultCriteria,
		&safePromoteData{data: make(map[string]map[string]string)}, make(map[int]map[string]int))
	assert.Equal(t, "DSCPDC_0003_BURST20180828114700954_COVER.JPG", sorted[0].OriginalFileName,
		"COVER file should be parent")
	assert.Equal(t, "DSCPDC_0000_BURST20180828114700954.JPG", sorted[1].OriginalFileName,
		"Remaining should be in sequence order")
	assert.Equal(t, "DSCPDC_0001_BURST20180828114700954.JPG", sorted[2].OriginalFileName)
	assert.Equal(t, "DSCPDC_0002_BURST20180828114700954.JPG", sorted[3].OriginalFileName)
}

func TestExamples_CameraBurst_DifferentBurstsNotGrouped(t *testing.T) {
	assets := []utils.TAsset{
		{OriginalFileName: "DSCPDC_0000_BURST20180828114700954.JPG", LocalDateTime: "2018-08-28T11:47:00.460Z"},
		{OriginalFileName: "DSCPDC_0000_BURST20180828115000000.JPG", LocalDateTime: "2018-08-28T11:50:00.000Z"},
	}
	criteria := `[{"key":"originalFileName","regex":{"key":"BURST(\\d+)","index":1}},{"key":"localDateTime","delta":{"milliseconds":1000}}]`
	groups, err := StackBy(assets, criteria, "", "", examplesLogger())
	require.NoError(t, err)
	assert.Equal(t, 0, len(groups), "Different BURST timestamps should not group")
}

/************************************************************************************************
** Burst Photos - Sequential with Common Prefix
************************************************************************************************/

func TestExamples_SequentialBurst_RegexGrouping(t *testing.T) {
	now := time.Now()
	assets := []utils.TAsset{
		assetFactory("photo_0001.jpg", now),
		assetFactory("photo_0002.jpg", now),
		assetFactory("photo_0003.jpg", now),
	}
	criteria := `[{"key":"originalFileName","regex":{"key":"^(.+?)_\\d+\\.","index":1}},{"key":"localDateTime","delta":{"milliseconds":3000}}]`
	groups, err := StackBy(assets, criteria, "", "", examplesLogger())
	require.NoError(t, err)
	assert.Equal(t, 1, len(groups))
	assert.Equal(t, 3, len(groups[0]))
}

func TestExamples_SequentialBurst_SequenceOrders(t *testing.T) {
	assets := []utils.TAsset{
		assetFactory("photo_0003.jpg", time.Now()),
		assetFactory("photo_0001.jpg", time.Now()),
		assetFactory("photo_0002.jpg", time.Now()),
	}
	sorted := sortStack(assets, "sequence,cover,edit,crop,hdr", "", []string{}, utils.DefaultCriteria,
		&safePromoteData{data: make(map[string]map[string]string)}, make(map[int]map[string]int))
	assert.Equal(t, "photo_0001.jpg", sorted[0].OriginalFileName, "First in sequence is parent")
	assert.Equal(t, "photo_0002.jpg", sorted[1].OriginalFileName)
	assert.Equal(t, "photo_0003.jpg", sorted[2].OriginalFileName)
}

/************************************************************************************************
** Burst Photos - Limitation: Fully Sequential Filenames
************************************************************************************************/

func TestExamples_FullySequential_NotGroupedByDefault(t *testing.T) {
	now := time.Now()
	assets := []utils.TAsset{
		assetFactory("IMG_1234.jpg", now),
		assetFactory("IMG_1235.jpg", now),
		assetFactory("IMG_1236.jpg", now),
	}
	groups, err := StackBy(assets, "", "", "", examplesLogger())
	require.NoError(t, err)
	assert.Equal(t, 0, len(groups),
		"Files with different base names (IMG_1234 vs IMG_1235) should not group")
}

/************************************************************************************************
** Parent Selection Control
************************************************************************************************/

func TestExamples_ProcessedFilesOnTop(t *testing.T) {
	assets := []utils.TAsset{
		assetFactory("IMG_1234.CR2", time.Now()),
		assetFactory("IMG_1234.dng", time.Now()),
		assetFactory("IMG_1234.jpg", time.Now()),
		assetFactory("IMG_1234.png", time.Now()),
	}
	sorted := sortStack(assets, "", ".jpg,.jpeg,.png,.heic,.dng,.cr2,.cr3,.nef,.arw,.raf,.orf,.rw2",
		[]string{"~", "."}, utils.DefaultCriteria,
		&safePromoteData{data: make(map[string]map[string]string)}, make(map[int]map[string]int))
	assert.Equal(t, "IMG_1234.jpg", sorted[0].OriginalFileName, "jpg first in ext promote list")
	assert.Equal(t, "IMG_1234.png", sorted[1].OriginalFileName, "png third in ext promote list")
}

func TestExamples_RawOnTop(t *testing.T) {
	assets := []utils.TAsset{
		assetFactory("IMG_1234.jpg", time.Now()),
		assetFactory("IMG_1234.CR2", time.Now()),
		assetFactory("IMG_1234.dng", time.Now()),
	}
	sorted := sortStack(assets, "", ".dng,.cr2,.cr3,.nef,.arw,.raf,.orf,.rw2,.jpg,.jpeg,.png",
		[]string{"~", "."}, utils.DefaultCriteria,
		&safePromoteData{data: make(map[string]map[string]string)}, make(map[int]map[string]int))
	assert.Equal(t, "IMG_1234.dng", sorted[0].OriginalFileName, "dng first in RAW-priority ext promote")
}

func TestExamples_EditedOnTop(t *testing.T) {
	assets := []utils.TAsset{
		assetFactory("IMG_1234.jpg", time.Now()),
		assetFactory("IMG_1234_final.jpg", time.Now()),
		assetFactory("IMG_1234_edit.jpg", time.Now()),
	}
	sorted := sortStack(assets, "final,edit,crop,hdr,cover,biggestNumber", "",
		[]string{"~", "."}, utils.DefaultCriteria,
		&safePromoteData{data: make(map[string]map[string]string)}, make(map[int]map[string]int))
	assert.Equal(t, "IMG_1234_final.jpg", sorted[0].OriginalFileName, "final has highest priority")
	assert.Equal(t, "IMG_1234_edit.jpg", sorted[1].OriginalFileName, "edit has second priority")
	assert.Equal(t, "IMG_1234.jpg", sorted[2].OriginalFileName, "plain file last")
}

/************************************************************************************************
** Mixed Camera Setups
************************************************************************************************/

func TestExamples_MixedCameras_DefaultGrouping(t *testing.T) {
	now := time.Now()
	assets := []utils.TAsset{
		assetFactory("PXL_20260120_120000000.jpg", now),
		assetFactory("PXL_20260120_120000000.dng", now),
		assetFactory("IMG_1234.JPG", now),
		assetFactory("IMG_1234.CR2", now),
	}
	groups, err := StackBy(assets, "", "", "", examplesLogger())
	require.NoError(t, err)
	assert.Equal(t, 2, len(groups), "Should create two separate stacks")
}

func TestExamples_MixedCameras_AdvancedExpression(t *testing.T) {
	now := time.Now()
	assets := []utils.TAsset{
		{ID: "1", OriginalFileName: "PXL_20260120_120000000.jpg", LocalDateTime: now.Format("2006-01-02T15:04:05.000Z")},
		{ID: "2", OriginalFileName: "PXL_20260120_120000000.dng", LocalDateTime: now.Format("2006-01-02T15:04:05.000Z")},
		{ID: "3", OriginalFileName: "IMG_1234.JPG", LocalDateTime: now.Format("2006-01-02T15:04:05.000Z")},
		{ID: "4", OriginalFileName: "IMG_1234.CR2", LocalDateTime: now.Format("2006-01-02T15:04:05.000Z")},
		{ID: "5", OriginalFileName: "DSCF5678.jpg", LocalDateTime: now.Format("2006-01-02T15:04:05.000Z")},
		{ID: "6", OriginalFileName: "DSCF5678.RAF", LocalDateTime: now.Format("2006-01-02T15:04:05.000Z")},
	}
	criteria := `{
		"mode": "advanced",
		"expression": {
			"operator": "AND",
			"children": [
				{
					"operator": "OR",
					"children": [
						{"criteria": {"key": "originalFileName", "regex": {"key": "^(PXL_\\d+_\\d+)", "index": 1}}},
						{"criteria": {"key": "originalFileName", "regex": {"key": "^(IMG_\\d+)", "index": 1}}},
						{"criteria": {"key": "originalFileName", "split": {"delimiters": ["~", "."], "index": 0}}}
					]
				},
				{"criteria": {"key": "localDateTime", "delta": {"milliseconds": 1000}}}
			]
		}
	}`
	groups, err := StackBy(assets, criteria, "", "", examplesLogger())
	require.NoError(t, err)
	assert.Equal(t, 3, len(groups), "Should create three stacks: PXL pair, IMG pair, DSCF pair")
}

func TestExamples_MixedCameras_NoCrossContamination(t *testing.T) {
	now := time.Now()
	assets := []utils.TAsset{
		assetFactory("PXL_20260120_120000000.jpg", now),
		assetFactory("IMG_1234.JPG", now),
	}
	groups, err := StackBy(assets, "", "", "", examplesLogger())
	require.NoError(t, err)
	assert.Equal(t, 0, len(groups), "Different camera files should not cross-stack")
}
