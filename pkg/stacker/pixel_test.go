package stacker

import (
	"testing"

	"github.com/majorfi/immich-stack/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPixelPhoneStacking(t *testing.T) {
	// Test case for the reported issue with Pixel phone images
	tests := []struct {
		name     string
		assets   []utils.TAsset
		criteria []utils.TCriteria
		want     int // number of groups expected
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
					LocalDateTime:    "2025-07-31T15:26:26.950Z", // 95ms later
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
					LocalDateTime:    "2025-07-31T15:26:28.000Z", // 1.145 seconds later
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
					LocalDateTime:    "2025-07-31T15:26:28.000Z", // Different time
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
					OriginalFileName: "PXL_20250731_152627900.RAW-01.COVER.jpg", // Different burst
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
			// Set up test criteria in environment
			t.Setenv("CRITERIA", mustMarshalJSON(t, tt.criteria))

			groups, err := StackBy(tt.assets, "", "", "", logrus.New())
			require.NoError(t, err)
			assert.Equal(t, tt.want, len(groups), "%s: Expected %d groups but got %d", tt.desc, tt.want, len(groups))

			if tt.want > 0 && len(groups) > 0 {
				// Verify all assets in the group have matching criteria
				t.Logf("Group contains %d assets", len(groups[0]))
				for _, asset := range groups[0] {
					t.Logf("  - %s at %s", asset.OriginalFileName, asset.LocalDateTime)
				}
			}
		})
	}
}

// Test to verify the exact regex extraction
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
			result, err := extractOriginalFileName(asset, criteria)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, result, "Expected %s but got %s for %s", tc.expected, result, tc.filename)
		})
	}
}