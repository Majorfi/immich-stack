package stacker

import (
	"testing"

	"github.com/majorfi/immich-stack/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func TestDfsConnectedComponents(t *testing.T) {
	tests := []struct {
		name              string
		assets            []utils.TAsset
		adjacency         [][]int
		startNode         int
		expectedComponent []string
	}{
		{
			name: "single node",
			assets: []utils.TAsset{
				{ID: "1", OriginalFileName: "a.jpg"},
			},
			adjacency:         [][]int{{}},
			startNode:         0,
			expectedComponent: []string{"1"},
		},
		{
			name: "two connected nodes",
			assets: []utils.TAsset{
				{ID: "1", OriginalFileName: "a.jpg"},
				{ID: "2", OriginalFileName: "b.jpg"},
			},
			adjacency:         [][]int{{1}, {0}},
			startNode:         0,
			expectedComponent: []string{"1", "2"},
		},
		{
			name: "chain of nodes",
			assets: []utils.TAsset{
				{ID: "1", OriginalFileName: "a.jpg"},
				{ID: "2", OriginalFileName: "b.jpg"},
				{ID: "3", OriginalFileName: "c.jpg"},
			},
			adjacency:         [][]int{{1}, {0, 2}, {1}},
			startNode:         0,
			expectedComponent: []string{"1", "2", "3"},
		},
		{
			name: "already visited node",
			assets: []utils.TAsset{
				{ID: "1", OriginalFileName: "a.jpg"},
				{ID: "2", OriginalFileName: "b.jpg"},
			},
			adjacency:         [][]int{{1}, {0}},
			startNode:         0,
			expectedComponent: []string{"1", "2"},
		},
		{
			name: "disconnected graph - only start component",
			assets: []utils.TAsset{
				{ID: "1", OriginalFileName: "a.jpg"},
				{ID: "2", OriginalFileName: "b.jpg"},
				{ID: "3", OriginalFileName: "c.jpg"},
			},
			adjacency:         [][]int{{}, {2}, {1}},
			startNode:         0,
			expectedComponent: []string{"1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			visited := make([]bool, len(tt.assets))
			component := []utils.TAsset{}

			dfsConnectedComponents(tt.startNode, tt.adjacency, visited, tt.assets, &component)

			componentIDs := make([]string, len(component))
			for i, asset := range component {
				componentIDs[i] = asset.ID
			}

			assert.Equal(t, tt.expectedComponent, componentIDs)
		})
	}
}

