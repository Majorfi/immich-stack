package stacker

import (
	"testing"

	"github.com/majorfi/immich-stack/pkg/utils"
)

func TestMatchedAssetIDs(t *testing.T) {
	t0 := "2024-01-01T10:00:00.000000000Z"
	asset := func(id, fileName, localDateTime string) utils.TAsset {
		return utils.TAsset{ID: id, OriginalFileName: fileName, LocalDateTime: localDateTime}
	}

	tests := []struct {
		name        string
		assets      []utils.TAsset
		criteria    string
		wantMatched []string
	}{
		{
			name: "default criteria match named assets",
			assets: []utils.TAsset{
				asset("a1", "IMG_1234.jpg", t0),
				asset("a2", "IMG_5678.dng", t0),
			},
			criteria:    "",
			wantMatched: []string{"a1", "a2"},
		},
		{
			name: "legacy regex criteria only match the pattern",
			assets: []utils.TAsset{
				asset("a1", "BURST001.jpg", t0),
				asset("a2", "IMG_1234.jpg", t0),
			},
			criteria:    `[{"key":"originalFileName","regex":{"key":"BURST(\\d+)","index":1}}]`,
			wantMatched: []string{"a1"},
		},
		{
			name: "AND group expression requires every leaf",
			assets: []utils.TAsset{
				asset("a1", "L1001336.dng", t0),
				asset("a2", "L1001337.dng", ""),
				asset("a3", "HOLIDAY.jpg", t0),
			},
			criteria:    `{"mode":"advanced","groups":[{"operator":"AND","criteria":[{"key":"originalFileName","regex":{"key":"^L(\\d+)","index":1}},{"key":"localDateTime","delta":{"milliseconds":1000}}]}]}`,
			wantMatched: []string{"a1"},
		},
		{
			name: "OR groups match through the connectivity path",
			assets: []utils.TAsset{
				asset("a1", "BURST001.jpg", t0),
				asset("a2", "IMG_1234.jpg", t0),
			},
			criteria:    `{"mode":"advanced","groups":[{"operator":"OR","criteria":[{"key":"originalFileName","regex":{"key":"BURST(\\d+)","index":1}}]}]}`,
			wantMatched: []string{"a1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, err := MatchedAssetIDs(tt.assets, tt.criteria)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(matched) != len(tt.wantMatched) {
				t.Fatalf("matched %d assets (%v), want %d (%v)", len(matched), matched, len(tt.wantMatched), tt.wantMatched)
			}
			for _, id := range tt.wantMatched {
				if !matched[id] {
					t.Fatalf("asset %s not matched; got %v", id, matched)
				}
			}
		})
	}
}

func TestMatchedAssetIDsInvalidCriteria(t *testing.T) {
	_, err := MatchedAssetIDs(
		[]utils.TAsset{{ID: "a1", OriginalFileName: "a.jpg"}},
		`{"mode":"advanced"}`)
	if err == nil {
		t.Fatal("expected an error for advanced criteria without groups or expression")
	}
}
