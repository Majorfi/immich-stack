package main

import (
	"testing"

	"github.com/majorfi/immich-stack/pkg/utils"
)

func TestIsRAWFile(t *testing.T) {
	tests := []struct {
		fileName string
		want     bool
	}{
		{fileName: "L1001336.dng", want: true},
		{fileName: "IMG_0001.NEF", want: true},
		{fileName: "shot.arw", want: true},
		{fileName: "shot.cr3", want: true},
		{fileName: "photo.jpg", want: false},
		{fileName: "photo.heic", want: false},
		{fileName: "photo.png", want: false},
		{fileName: "clip.mov", want: false},
		{fileName: "noext", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.fileName, func(t *testing.T) {
			if got := isRAWFile(tt.fileName); got != tt.want {
				t.Fatalf("isRAWFile(%q) = %v, want %v", tt.fileName, got, tt.want)
			}
		})
	}
}

func TestFindOrphanedRAWs(t *testing.T) {
	t0 := "2024-01-01T10:00:00.000000000Z"
	asset := func(id, fileName string) utils.TAsset {
		return utils.TAsset{ID: id, OriginalFileName: fileName, LocalDateTime: t0}
	}
	stackedAsset := func(id, fileName, stackID string) utils.TAsset {
		a := asset(id, fileName)
		a.Stack = &utils.TStack{ID: stackID}
		return a
	}
	leicaCriteria := `{"mode":"advanced","groups":[{"operator":"AND","criteria":[{"key":"originalFileName","regex":{"key":"^(?:L|DO0)(\\d+)","index":1}},{"key":"localDateTime","delta":{"milliseconds":1000}}]}]}`

	tests := []struct {
		name        string
		active      []utils.TAsset
		criteria    string
		wantOrphans []string
		wantKept    int
	}{
		{
			name:        "lone DNG is orphaned",
			active:      []utils.TAsset{asset("d1", "L1001336.dng")},
			wantOrphans: []string{"d1"},
		},
		{
			name:        "lone NEF is orphaned (any RAW format)",
			active:      []utils.TAsset{asset("n1", "IMG_0001.nef")},
			wantOrphans: []string{"n1"},
		},
		{
			name: "DNG with JPG companion is kept",
			active: []utils.TAsset{
				asset("d1", "IMG_1234.dng"),
				asset("j1", "IMG_1234.jpg"),
			},
			wantOrphans: []string{},
		},
		{
			name: "DNG with HEIC companion is kept",
			active: []utils.TAsset{
				asset("d1", "IMG_1234.dng"),
				asset("h1", "IMG_1234.heic"),
			},
			wantOrphans: []string{},
		},
		{
			name: "all-RAW group: every member is orphaned",
			active: []utils.TAsset{
				asset("d1", "IMG_1234~1.dng"),
				asset("d2", "IMG_1234~2.dng"),
			},
			wantOrphans: []string{"d1", "d2"},
		},
		{
			name: "RAW in an Immich stack with a developed file is kept",
			active: []utils.TAsset{
				stackedAsset("d1", "L1001336.dng", "stack-1"),
				stackedAsset("j1", "UNRELATED.jpg", "stack-1"),
			},
			wantOrphans: []string{},
			wantKept:    1,
		},
		{
			name: "RAW in an all-RAW Immich stack is still orphaned",
			active: []utils.TAsset{
				stackedAsset("d1", "L1001336.dng", "stack-1"),
				stackedAsset("n1", "UNRELATED.nef", "stack-1"),
			},
			wantOrphans: []string{"d1", "n1"},
		},
		{
			name:        "developed files alone are never flagged",
			active:      []utils.TAsset{asset("j1", "IMG_1234.jpg"), asset("h1", "OTHER.heic")},
			wantOrphans: []string{},
		},
		{
			// With default criteria, Leica DNG+JPG variants have different base names, so
			// the DNG counts as orphaned. The regex criteria below is what pairs them.
			name: "Leica pair with default criteria is not matched",
			active: []utils.TAsset{
				asset("d1", "L1001336.dng"),
				asset("j1", "DO01001336.jpg"),
			},
			wantOrphans: []string{"d1"},
		},
		{
			name: "Leica pair grouped via regex criteria is kept",
			active: []utils.TAsset{
				asset("d1", "L1001336.dng"),
				asset("j1", "DO01001336.jpg"),
			},
			criteria:    leicaCriteria,
			wantOrphans: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orphans, kept, err := findOrphanedRAWs(tt.active, tt.criteria, "", "", quietFixTrashLogger())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if kept != tt.wantKept {
				t.Fatalf("kept = %d, want %d", kept, tt.wantKept)
			}
			if len(orphans) != len(tt.wantOrphans) {
				t.Fatalf("found %d orphans (%v), want %d (%v)", len(orphans), orphans, len(tt.wantOrphans), tt.wantOrphans)
			}
			for _, id := range tt.wantOrphans {
				if _, ok := orphans[id]; !ok {
					t.Fatalf("asset %s not flagged as orphan; got %v", id, orphans)
				}
			}
		})
	}
}

func TestFindOrphanedRAWsInvalidCriteria(t *testing.T) {
	_, _, err := findOrphanedRAWs(
		[]utils.TAsset{{ID: "d1", OriginalFileName: "a.dng"}},
		`{"mode":"advanced"}`, "", "", quietFixTrashLogger())
	if err == nil {
		t.Fatal("expected an error for advanced criteria without groups or expression")
	}
}
