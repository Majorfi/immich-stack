package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/majorfi/immich-stack/pkg/utils"
	"github.com/sirupsen/logrus"
)

func TestFilterOutPartnerAssets(t *testing.T) {
	me := "11111111-1111-1111-1111-111111111111"
	partner := "22222222-2222-2222-2222-222222222222"
	mine := func(id string) utils.TAsset { return utils.TAsset{ID: id, OwnerID: me} }
	theirs := func(id string) utils.TAsset { return utils.TAsset{ID: id, OwnerID: partner} }

	tests := []struct {
		name          string
		assets        []utils.TAsset
		ownerID       string
		wantLen       int
		wantLogSubstr string
	}{
		{
			name:          "drops partner assets and logs the count",
			assets:        []utils.TAsset{mine("a"), theirs("b"), mine("c"), theirs("d"), theirs("e")},
			ownerID:       me,
			wantLen:       2,
			wantLogSubstr: "Skipped 3 assets owned by partners",
		},
		{
			name:          "no partner assets: no log line emitted",
			assets:        []utils.TAsset{mine("a"), mine("b")},
			ownerID:       me,
			wantLen:       2,
			wantLogSubstr: "",
		},
		{
			name:          "empty input: no log line emitted",
			assets:        []utils.TAsset{},
			ownerID:       me,
			wantLen:       0,
			wantLogSubstr: "",
		},
		{
			name:          "empty ownerID is a no-op (defensive default) — no log",
			assets:        []utils.TAsset{mine("a"), theirs("b")},
			ownerID:       "",
			wantLen:       2,
			wantLogSubstr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := logrus.New()
			logger.SetOutput(&buf)
			logger.SetFormatter(&logrus.TextFormatter{DisableColors: true, DisableTimestamp: true})

			got := filterOutPartnerAssets(tt.assets, tt.ownerID, logger)
			if len(got) != tt.wantLen {
				t.Errorf("filterOutPartnerAssets returned %d assets, want %d", len(got), tt.wantLen)
			}

			logOutput := buf.String()
			if tt.wantLogSubstr == "" {
				if strings.Contains(logOutput, "Skipped") {
					t.Errorf("expected no Skipped log, got: %q", logOutput)
				}
			} else {
				if !strings.Contains(logOutput, tt.wantLogSubstr) {
					t.Errorf("log output %q does not contain %q", logOutput, tt.wantLogSubstr)
				}
			}
		})
	}
}

func TestSplitCommaList(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{name: "plain list", input: "a,b,c", want: []string{"a", "b", "c"}},
		{name: "trims whitespace", input: " a , b ,c ", want: []string{"a", "b", "c"}},
		{name: "drops empty entries", input: "a,,b,", want: []string{"a", "b"}},
		{name: "drops whitespace-only entries", input: "a, ,b", want: []string{"a", "b"}},
		{name: "empty input yields empty list", input: "", want: []string{}},
		{name: "single value", input: "key", want: []string{"key"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitCommaList(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("splitCommaList(%q) = %v, want %v", tt.input, got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("splitCommaList(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}
