package stacker

import (
	"testing"
	"time"

	"github.com/majorfi/immich-stack/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/************************************************************************************************
** Test cases for criteria matching
************************************************************************************************/

func TestApplyCriteria(t *testing.T) {
	tests := []struct {
		name     string
		fileList []string
		want     int // number of groups
	}{
		{
			name: "same file different folder",
			fileList: []string{
				"IMG_2482.jpg",
				"IMG_2482.jpg",
			},
			want: 1,
		},
		{
			name: "same base different extension",
			fileList: []string{
				"IMG_2482.jpg",
				"IMG_2482.cr2",
			},
			want: 1, // Should group by base name, regardless of extension
		},
		{
			name: "different files",
			fileList: []string{
				"IMG_2482.jpg",
				"IMG_2483.cr2",
			},
			want: 0, // Different base names, so no group (per implementation)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			assets := make([]utils.TAsset, len(tt.fileList))
			for i, f := range tt.fileList {
				assets[i] = assetFactory(f, time.Now())
			}

			// Act
			groups, err := StackBy(assets, "", "", "")

			// Assert
			require.NoError(t, err)
			assert.Equal(t, tt.want, len(groups))
		})
	}
}
