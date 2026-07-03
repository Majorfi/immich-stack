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

func TestTimestampAfter(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want bool
	}{
		{name: "plainly newer", a: "2024-01-01T10:00:00Z", b: "2024-01-01T09:00:00Z", want: true},
		{name: "plainly older", a: "2024-01-01T08:00:00Z", b: "2024-01-01T09:00:00Z", want: false},
		{name: "equal", a: "2024-01-01T10:00:00Z", b: "2024-01-01T10:00:00Z", want: false},
		{name: "fractional seconds", a: "2024-01-01T10:00:00.500Z", b: "2024-01-01T10:00:00.400Z", want: true},
		{
			// Lexicographic comparison would wrongly say true here: "10:..." > "09:...",
			// but +02:00 puts a at 08:00 UTC, before b.
			name: "offset earlier despite larger string",
			a:    "2024-01-01T10:00:00+02:00",
			b:    "2024-01-01T09:00:00Z",
			want: false,
		},
		{name: "unparseable falls back to lexicographic", a: "b", b: "a", want: true},
		{name: "both empty", a: "", b: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := timestampAfter(tt.a, tt.b); got != tt.want {
				t.Fatalf("timestampAfter(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestIsReplacedByNewerCopy(t *testing.T) {
	t0 := "2024-01-01T10:00:00Z"
	older := "2023-12-01T10:00:00Z"
	newer := "2024-01-02T10:00:00Z"
	trashed := utils.TAsset{
		ID:               "trashed",
		OriginalFileName: "IMG_1234.jpg",
		FileCreatedAt:    t0,
		FileModifiedAt:   t0,
		UpdatedAt:        t0,
		IsTrashed:        true,
	}
	activeAsset := func(fileName, created, modified, updated string) utils.TAsset {
		return utils.TAsset{
			ID:               "active",
			OriginalFileName: fileName,
			FileCreatedAt:    created,
			FileModifiedAt:   modified,
			UpdatedAt:        updated,
		}
	}

	tests := []struct {
		name   string
		active utils.TAsset
		want   bool
	}{
		{name: "newer FileCreatedAt", active: activeAsset("IMG_1234.jpg", newer, t0, t0), want: true},
		{name: "newer FileModifiedAt only", active: activeAsset("IMG_1234.jpg", t0, newer, t0), want: true},
		{name: "newer UpdatedAt only", active: activeAsset("IMG_1234.jpg", t0, t0, newer), want: true},
		{name: "identical timestamps", active: activeAsset("IMG_1234.jpg", t0, t0, t0), want: false},
		{name: "all timestamps older", active: activeAsset("IMG_1234.jpg", older, older, older), want: false},
		{name: "newer but different filename", active: activeAsset("IMG_9999.jpg", newer, t0, t0), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			byFilename := map[string][]utils.TAsset{
				tt.active.OriginalFileName: {tt.active},
			}
			if got := isReplacedByNewerCopy(trashed, byFilename); got != tt.want {
				t.Fatalf("isReplacedByNewerCopy() = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("no active asset with that filename", func(t *testing.T) {
		if isReplacedByNewerCopy(trashed, map[string][]utils.TAsset{}) {
			t.Fatal("isReplacedByNewerCopy() = true with empty active map, want false")
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
	newerCopy := func(a utils.TAsset) utils.TAsset {
		a.FileCreatedAt = "2024-01-02T10:00:00Z"
		return a
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
			name: "replaced trashed asset does not cascade",
			trashed: []utils.TAsset{
				asset("t1", "IMG_1234.jpg", t0, true),
			},
			active: []utils.TAsset{
				newerCopy(asset("a1", "IMG_1234.jpg", t0, false)),
				asset("a2", "IMG_1234.dng", t0, false),
			},
			wantToTrash:  []string{},
			wantReplaced: 1,
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
