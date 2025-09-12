package stacker

import (
	"sort"

	"github.com/majorfi/immich-stack/pkg/utils"
	"github.com/sirupsen/logrus"
)

/**************************************************************************************************
** buildConnectedComponents creates connected components of assets based on shared grouping keys.
** Assets that share any grouping key are considered connected and will be placed in the same
** component, implementing union semantics for OR groups.
**
** @param assets - List of assets that matched at least one group
** @param assetKeys - Map from asset ID to list of grouping keys for that asset
** @param logger - Logger for debug output
** @return [][]utils.TAsset - List of connected components (each is a potential stack)
**************************************************************************************************/
func buildConnectedComponents(assets []utils.TAsset, assetKeys map[string][]string, logger *logrus.Logger) [][]utils.TAsset {
	if len(assets) == 0 {
		return nil
	}

	// Build a map from grouping keys to assets that have that key
	keyToAssets := make(map[string][]string) // grouping key -> list of asset IDs
	for assetID, keys := range assetKeys {
		for _, key := range keys {
			keyToAssets[key] = append(keyToAssets[key], assetID)
		}
	}

	// Build adjacency list for the connectivity graph
	assetIDToIndex := make(map[string]int)
	for i, asset := range assets {
		assetIDToIndex[asset.ID] = i
	}

	// Convert keyToAssets from string IDs to indices to avoid repeated lookups
	keyToIndices := make(map[string][]int)
	for key, assetIDs := range keyToAssets {
		indices := make([]int, 0, len(assetIDs))
		for _, assetID := range assetIDs {
			if idx, ok := assetIDToIndex[assetID]; ok {
				indices = append(indices, idx)
			}
		}
		if len(indices) > 1 { // Only store keys that connect multiple assets
			keyToIndices[key] = indices
		}
	}

	// Create adjacency list where assets are connected if they share any grouping key
	// Use map[int]bool per node to deduplicate neighbors and reduce memory usage for dense graphs
	neighborSets := make([]map[int]bool, len(assets))
	for i := range neighborSets {
		neighborSets[i] = make(map[int]bool)
	}

	for _, indices := range keyToIndices {
		// Connect all assets that share this grouping key
		// Use i < j loop to set both directions once, halving map writes
		for i := 0; i < len(indices); i++ {
			for j := i + 1; j < len(indices); j++ {
				idx1, idx2 := indices[i], indices[j]
				// Set both directions for undirected connectivity
				neighborSets[idx1][idx2] = true
				neighborSets[idx2][idx1] = true
			}
		}
	}

	// Convert neighbor sets to adjacency lists with deterministic order
	adjacency := make([][]int, len(assets))
	for i, neighborSet := range neighborSets {
		adjacency[i] = make([]int, 0, len(neighborSet))
		for neighbor := range neighborSet {
			adjacency[i] = append(adjacency[i], neighbor)
		}
		// Sort neighbors for deterministic DFS traversal order
		sort.Ints(adjacency[i])
	}

	// Find connected components using DFS
	visited := make([]bool, len(assets))
	var components [][]utils.TAsset

	for i := 0; i < len(assets); i++ {
		if !visited[i] {
			// Start a new component from this unvisited node
			var component []utils.TAsset
			dfsConnectedComponents(i, adjacency, visited, assets, &component)

			if len(component) > 0 {
				components = append(components, component)
			}
		}
	}

	if logger.IsLevelEnabled(logrus.DebugLevel) {
		logger.Debugf("Built %d connected components from %d assets", len(components), len(assets))
		for i, comp := range components {
			logger.Debugf("Component %d has %d assets", i, len(comp))
		}
	}

	return components
}

/**************************************************************************************************
** dfsConnectedComponents performs depth-first search to find all assets in a connected component.
**
** @param node - Current asset index to explore
** @param adjacency - Adjacency list representation of the connectivity graph
** @param visited - Array tracking which assets have been visited
** @param assets - Original list of assets
** @param component - Current component being built (modified in-place)
**************************************************************************************************/
func dfsConnectedComponents(node int, adjacency [][]int, visited []bool, assets []utils.TAsset, component *[]utils.TAsset) {
	if visited[node] || node >= len(assets) {
		return
	}

	visited[node] = true
	*component = append(*component, assets[node])

	// Visit all connected nodes
	for _, neighbor := range adjacency[node] {
		if !visited[neighbor] {
			dfsConnectedComponents(neighbor, adjacency, visited, assets, component)
		}
	}
}
