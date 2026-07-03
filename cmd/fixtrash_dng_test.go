package main

import (
	"testing"

	"github.com/majorfi/immich-stack/pkg/utils"
)

func TestNormalizeDNGBaseName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "Leica DO0 prefix", input: "DO01001336", want: "1001336"},
		{name: "Leica DL0 prefix", input: "DL01000491", want: "1000491"},
		{name: "Leica DL prefix without leading zero", input: "DL1000491", want: "1000491"},
		{name: "Leica L prefix", input: "L1001336", want: "1001336"},
		{name: "L not followed by digit untouched", input: "Lab", want: "Lab"},
		{name: "DL not followed by digit untouched", input: "DLab", want: "DLab"},
		{name: "underscore suffix dropped", input: "IMG_1234", want: "IMG"},
		{name: "tilde suffix dropped", input: "A~2", want: "A"},
		{name: "plain name untouched", input: "photo", want: "photo"},
		{name: "leading underscore untouched", input: "_leading", want: "_leading"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeDNGBaseName(tt.input); got != tt.want {
				t.Fatalf("normalizeDNGBaseName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDNGGroupingBaseName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "single extension", input: "IMG.jpg", want: "IMG"},
		{name: "multi extension keeps first part", input: "L1000746.edit.jpg", want: "L1000746"},
		{name: "no extension", input: "noext", want: "noext"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dngGroupingBaseName(tt.input); got != tt.want {
				t.Fatalf("dngGroupingBaseName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFindOrphanedDNGs(t *testing.T) {
	asset := func(id, fileName string) utils.TAsset {
		return utils.TAsset{ID: id, OriginalFileName: fileName}
	}
	stackedAsset := func(id, fileName, stackID string) utils.TAsset {
		a := asset(id, fileName)
		a.Stack = &utils.TStack{ID: stackID}
		return a
	}

	tests := []struct {
		name        string
		active      []utils.TAsset
		wantOrphans []string
		wantSkipped int
	}{
		{
			name:        "DNG alone is orphaned",
			active:      []utils.TAsset{asset("d1", "L1001336.dng")},
			wantOrphans: []string{"d1"},
		},
		{
			name: "DNG with JPG companion is kept",
			active: []utils.TAsset{
				asset("d1", "L1001336.dng"),
				asset("j1", "L1001336.jpg"),
			},
			wantOrphans: []string{},
		},
		{
			name: "DNG with jpeg companion is kept",
			active: []utils.TAsset{
				asset("d1", "L1001336.dng"),
				asset("j1", "L1001336.jpeg"),
			},
			wantOrphans: []string{},
		},
		{
			// Pins the current quirk: only .jpg/.jpeg count as companions, so a
			// DNG+HEIC pair loses its DNG. Revisit with the DNG-pass discussion.
			name: "DNG with HEIC companion is still flagged",
			active: []utils.TAsset{
				asset("d1", "L1001336.dng"),
				asset("h1", "L1001336.heic"),
			},
			wantOrphans: []string{"d1"},
		},
		{
			name: "Leica prefixes map JPG and DNG to the same base",
			active: []utils.TAsset{
				asset("d1", "L1001336.dng"),
				asset("j1", "DO01001336.jpg"),
			},
			wantOrphans: []string{},
		},
		{
			name: "DNG already stacked with a JPG is skipped",
			active: []utils.TAsset{
				stackedAsset("d1", "L1001336.dng", "stack-1"),
				stackedAsset("j1", "UNRELATED.jpg", "stack-1"),
			},
			wantOrphans: []string{},
			wantSkipped: 1,
		},
		{
			name: "DNG stacked without JPG is still orphaned",
			active: []utils.TAsset{
				stackedAsset("d1", "L1001336.dng", "stack-1"),
				stackedAsset("h1", "UNRELATED.heic", "stack-1"),
			},
			wantOrphans: []string{"d1"},
		},
		{
			name:        "non-DNG assets alone are ignored",
			active:      []utils.TAsset{asset("j1", "L1001336.jpg"), asset("h1", "OTHER.heic")},
			wantOrphans: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orphans, skipped := findOrphanedDNGs(tt.active, quietFixTrashLogger())
			if skipped != tt.wantSkipped {
				t.Fatalf("skipped = %d, want %d", skipped, tt.wantSkipped)
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
