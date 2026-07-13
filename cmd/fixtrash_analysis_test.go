package main

import (
	"io"
	"testing"

	"github.com/majorfi/immich-stack/pkg/utils"
	"github.com/sirupsen/logrus"
)

func quietFixTrashLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	return logger
}

func TestHasActiveCopy(t *testing.T) {
	t0 := "2024-01-01T10:00:00.000000000Z"
	otherDay := "2025-05-05T10:00:00.000000000Z"
	trashed := utils.TAsset{
		ID:               "trashed",
		OriginalFileName: "IMG_1234.jpg",
		LocalDateTime:    t0,
		IsTrashed:        true,
	}
	activeAsset := func(fileName, localDateTime string) utils.TAsset {
		return utils.TAsset{ID: "active", OriginalFileName: fileName, LocalDateTime: localDateTime}
	}

	tests := []struct {
		name   string
		active utils.TAsset
		want   bool
	}{
		{name: "same name and capture time is a copy", active: activeAsset("IMG_1234.jpg", t0), want: true},
		{name: "recycled name from another photo is not a copy", active: activeAsset("IMG_1234.jpg", otherDay), want: false},
		{name: "different filename is not a copy", active: activeAsset("IMG_9999.jpg", t0), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			byFilename := map[string][]utils.TAsset{
				tt.active.OriginalFileName: {tt.active},
			}
			if got := hasActiveCopy(trashed, byFilename); got != tt.want {
				t.Fatalf("hasActiveCopy() = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("no active asset with that filename", func(t *testing.T) {
		if hasActiveCopy(trashed, map[string][]utils.TAsset{}) {
			t.Fatal("hasActiveCopy() = true with empty active map, want false")
		}
	})
}

func TestFindStackRelatedAssets(t *testing.T) {
	// Helper matching the default criteria: base name before "~"/"." plus 1s time buckets.
	asset := func(id, fileName, localDateTime string, trashed bool) utils.TAsset {
		return utils.TAsset{
			ID:               id,
			OriginalFileName: fileName,
			LocalDateTime:    localDateTime,
			FileCreatedAt:    "2024-01-01T10:00:00Z",
			FileModifiedAt:   "2024-01-01T10:00:00Z",
			UpdatedAt:        "2024-01-01T10:00:00Z",
			IsTrashed:        trashed,
		}
	}
	t0 := "2024-01-01T10:00:00.000000000Z"

	tests := []struct {
		name          string
		trashed       []utils.TAsset
		active        []utils.TAsset
		wantToTrash   []string
		wantReplaced  int
		wantTriggerOf map[string]string
	}{
		{
			name:          "trashed JPG pulls its DNG companion",
			trashed:       []utils.TAsset{asset("t1", "IMG_1234.jpg", t0, true)},
			active:        []utils.TAsset{asset("a1", "IMG_1234.dng", t0, false)},
			wantToTrash:   []string{"a1"},
			wantTriggerOf: map[string]string{"a1": "IMG_1234.jpg"},
		},
		{
			// The HIGH regression from the holistic review: trashing one duplicate must
			// not cascade the surviving copy or its RAW companion.
			name: "trashed duplicate keeps the surviving copy and its companions",
			trashed: []utils.TAsset{
				asset("t1", "IMG_1234.jpg", t0, true),
			},
			active: []utils.TAsset{
				asset("a1", "IMG_1234.jpg", t0, false),
				asset("a2", "IMG_1234.dng", t0, false),
			},
			wantToTrash:  []string{},
			wantReplaced: 1,
		},
		{
			// A recycled filename from an unrelated photo must not suppress the cascade.
			name: "recycled filename does not suppress the cascade",
			trashed: []utils.TAsset{
				asset("t1", "DSC_0001.jpg", t0, true),
			},
			active: []utils.TAsset{
				asset("a1", "DSC_0001.dng", t0, false),
				asset("b1", "DSC_0001.jpg", "2025-05-05T10:00:00.000000000Z", false),
			},
			wantToTrash:   []string{"a1"},
			wantTriggerOf: map[string]string{"a1": "DSC_0001.jpg"},
		},
		{
			name: "archived group members are never cascaded",
			trashed: []utils.TAsset{
				asset("t1", "IMG_1234.jpg", t0, true),
			},
			active: []utils.TAsset{
				asset("a1", "IMG_1234.dng", t0, false),
				func() utils.TAsset {
					a := asset("a2", "IMG_1234.heic", t0, false)
					a.IsArchived = true
					return a
				}(),
			},
			wantToTrash: []string{"a1"},
		},
		{
			name:        "trashed asset without companions",
			trashed:     []utils.TAsset{asset("t1", "IMG_1234.jpg", t0, true)},
			active:      []utils.TAsset{asset("a1", "IMG_9999.jpg", t0, false)},
			wantToTrash: []string{},
		},
		{
			name: "two trashed variants mark the shared companion once",
			trashed: []utils.TAsset{
				asset("t1", "PIC~1.jpg", t0, true),
				asset("t2", "PIC~2.jpg", t0, true),
			},
			active:      []utils.TAsset{asset("a1", "PIC.dng", t0, false)},
			wantToTrash: []string{"a1"},
		},
		{
			name:    "active-only groups stay untouched",
			trashed: []utils.TAsset{asset("t1", "IMG_1234.jpg", t0, true)},
			active: []utils.TAsset{
				asset("a1", "IMG_1234.dng", t0, false),
				asset("b1", "OTHER.jpg", t0, false),
				asset("b2", "OTHER.dng", t0, false),
			},
			wantToTrash: []string{"a1"},
		},
		{
			// Combined-run semantics: t2 bridges the 800ms gaps, so a1 at +1.6s joins
			// the group even though it is more than 1s away from t1.
			name: "trashed assets bridge time buckets",
			trashed: []utils.TAsset{
				asset("t1", "PIC~1.jpg", "2024-01-01T10:00:00.000000000Z", true),
				asset("t2", "PIC~2.jpg", "2024-01-01T10:00:00.800000000Z", true),
			},
			active:      []utils.TAsset{asset("a1", "PIC~3.jpg", "2024-01-01T10:00:01.600000000Z", false)},
			wantToTrash: []string{"a1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toTrash, triggeredBy, replaced, err := findStackRelatedAssets(
				tt.trashed, tt.active, "", "", "", quietFixTrashLogger())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if replaced != tt.wantReplaced {
				t.Fatalf("replacedCount = %d, want %d", replaced, tt.wantReplaced)
			}
			if len(toTrash) != len(tt.wantToTrash) {
				t.Fatalf("marked %d assets (%v), want %d (%v)", len(toTrash), toTrash, len(tt.wantToTrash), tt.wantToTrash)
			}
			for _, id := range tt.wantToTrash {
				if _, ok := toTrash[id]; !ok {
					t.Fatalf("asset %s not marked for trash; got %v", id, toTrash)
				}
				if triggeredBy[id] == "" {
					t.Fatalf("asset %s has no trigger recorded", id)
				}
			}
			for id, wantTrigger := range tt.wantTriggerOf {
				if triggeredBy[id] != wantTrigger {
					t.Fatalf("triggeredBy[%s] = %q, want %q", id, triggeredBy[id], wantTrigger)
				}
			}
		})
	}
}

func TestFindStackRelatedAssetsInvalidCriteria(t *testing.T) {
	trashed := []utils.TAsset{{ID: "t1", OriginalFileName: "a.jpg", IsTrashed: true}}
	active := []utils.TAsset{{ID: "a1", OriginalFileName: "a.dng"}}

	_, _, _, err := findStackRelatedAssets(trashed, active, `{"mode":"advanced"}`, "", "", quietFixTrashLogger())
	if err == nil {
		t.Fatal("expected an error for advanced criteria without groups or expression")
	}
}

func TestFindStackRelatedAssetsNeverMarksTrashedOrTriggers(t *testing.T) {
	t0 := "2024-01-01T10:00:00.000000000Z"
	trashed := []utils.TAsset{
		{ID: "t1", OriginalFileName: "PIC~1.jpg", LocalDateTime: t0, IsTrashed: true},
		{ID: "t2", OriginalFileName: "PIC~2.jpg", LocalDateTime: t0, IsTrashed: true},
	}
	active := []utils.TAsset{
		{ID: "a1", OriginalFileName: "PIC.dng", LocalDateTime: t0},
	}

	toTrash, _, _, err := findStackRelatedAssets(trashed, active, "", "", "", quietFixTrashLogger())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, id := range []string{"t1", "t2"} {
		if _, ok := toTrash[id]; ok {
			t.Fatalf("trashed trigger %s must never be re-marked for trash", id)
		}
	}
}
