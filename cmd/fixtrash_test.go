package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/majorfi/immich-stack/pkg/utils"
	"github.com/sirupsen/logrus"
)

func captureTrashSummary(level logrus.Level, assetsToTrash map[string]utils.TAsset, triggeredBy map[string]string) string {
	var buf bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&buf)
	logger.SetLevel(level)
	logger.SetFormatter(&logrus.TextFormatter{DisableColors: true, DisableTimestamp: true})
	logTrashSummary(logger, assetsToTrash, triggeredBy)
	return buf.String()
}

func TestLogTrashSummary(t *testing.T) {
	assetsToTrash := map[string]utils.TAsset{
		"a1": {ID: "a1", OriginalFileName: "IMG_1234.dng"},
		"a2": {ID: "a2", OriginalFileName: "IMG_1234~2.jpg"},
		"o1": {ID: "o1", OriginalFileName: "L1000746.dng"},
		"n1": {ID: "n1", OriginalFileName: "noext"},
	}
	triggeredBy := map[string]string{
		"a1": "IMG_1234.jpg",
		"a2": "IMG_1234.jpg",
		"o1": orphanedRAWTrigger,
		"n1": orphanedRAWTrigger,
	}

	t.Run("info level groups by trigger with orphans separated", func(t *testing.T) {
		out := captureTrashSummary(logrus.InfoLevel, assetsToTrash, triggeredBy)
		for _, want := range []string{
			"Summary of assets to trash (4)",
			"Orphaned RAW files (no developed companion):",
			"L1000746.dng",
			"noext",
			"IMG_1234.jpg (in trash):",
			"IMG_1234.dng",
			"IMG_1234~2.jpg",
		} {
			if !strings.Contains(out, want) {
				t.Fatalf("info output missing %q\n---\n%s", want, out)
			}
		}
		if strings.Contains(out, "Assets to trash by type") {
			t.Fatalf("extension histogram must not appear at info level\n---\n%s", out)
		}
	})

	t.Run("debug level adds the extension histogram", func(t *testing.T) {
		out := captureTrashSummary(logrus.DebugLevel, assetsToTrash, triggeredBy)
		for _, want := range []string{
			"Assets to trash by type",
			"DNG files: 2",
			"JPG files: 1",
			"(NO EXTENSION) files: 1",
		} {
			if !strings.Contains(out, want) {
				t.Fatalf("debug output missing %q\n---\n%s", want, out)
			}
		}
	})

	t.Run("above info level prints nothing", func(t *testing.T) {
		out := captureTrashSummary(logrus.WarnLevel, assetsToTrash, triggeredBy)
		if out != "" {
			t.Fatalf("expected no output at warn level, got:\n%s", out)
		}
	})
}
