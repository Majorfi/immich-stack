/**************************************************************************************************
** Fix-trash command implementation for the Immich CLI application.
** Handles stack-aware trash operations to maintain consistency.
**************************************************************************************************/

package main

import (
	"io"
	"path/filepath"
	"strings"

	"github.com/majorfi/immich-stack/pkg/immich"
	"github.com/majorfi/immich-stack/pkg/stacker"
	"github.com/majorfi/immich-stack/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

/**************************************************************************************************
** Main execution logic for fixing incomplete trash operations. Identifies trashed assets
** and moves their stack-related assets to trash to maintain consistency.
**
** @param cmd - Cobra command instance
** @param args - Command line arguments
**************************************************************************************************/
func runFixTrash(cmd *cobra.Command, args []string) {
	logger := loadEnv()

	/**********************************************************************************************
	** Support multiple API keys (comma-separated).
	**********************************************************************************************/
	apiKeys := utils.RemoveEmptyStrings(func(keys []string) []string {
		for i, key := range keys {
			keys[i] = strings.TrimSpace(key)
		}
		return keys
	}(strings.Split(apiKey, ",")))
	if len(apiKeys) == 0 {
		logger.Fatalf("No API key(s) provided.")
	}

	for i, key := range apiKeys {
		if i > 0 {
			logger.Infof("\n")
		}
		client := immich.NewClient(apiURL, key, false, false, dryRun, withArchived, withDeleted, false, nil, "", "", logger)
		if client == nil {
			logger.Errorf("Invalid client for API key: %s", key)
			continue
		}
		user, err := client.GetCurrentUser()
		if err != nil {
			logger.Errorf("Failed to fetch user for API key: %s: %v", key, err)
			continue
		}
		logger.Infof("=====================================================================================")
		logger.Infof("Fixing trash for user: %s (%s)", user.Name, user.Email)
		logger.Infof("=====================================================================================")

		/**********************************************************************************************
		** Fetch trashed assets and all assets.
		**********************************************************************************************/
		trashedAssets, err := client.FetchTrashedAssets(1000)
		if err != nil {
			logger.Errorf("Error fetching trashed assets: %v", err)
			continue
		}

		if len(trashedAssets) == 0 {
			logger.Info("No trashed assets found. Nothing to fix.")
			continue
		}

		logger.Infof("🗑️  Found %d trashed assets", len(trashedAssets))

		existingStacks, err := client.FetchAllStacks()
		if err != nil {
			logger.Errorf("Error fetching stacks: %v", err)
			continue
		}

		allAssets, err := client.FetchAssets(1000, existingStacks)
		if err != nil {
			logger.Errorf("Error fetching all assets: %v", err)
			continue
		}

		/**********************************************************************************************
		** Find assets that should be trashed using reverse criteria matching.
		** For each trashed asset, combine it with all active assets and run stacker criteria
		** to find which active assets would group with the trashed ones.
		**********************************************************************************************/
		assetsToTrash := make(map[string]utils.TAsset)
		// Track which trashed asset caused each asset to be marked for deletion
		trashedAssetMapping := make(map[string]string) // assetID -> trashed asset filename

		// Filter active (non-trashed) assets
		activeAssets := make([]utils.TAsset, 0)
		for _, asset := range allAssets {
			if !asset.IsTrashed {
				activeAssets = append(activeAssets, asset)
			}
		}

		// Build a map of active assets by filename for replacement detection
		activeAssetsByFilename := make(map[string][]utils.TAsset)
		for _, asset := range activeAssets {
			activeAssetsByFilename[asset.OriginalFileName] = append(activeAssetsByFilename[asset.OriginalFileName], asset)
		}

		replacementCount := 0
		for idx, trashedAsset := range trashedAssets {
			// Show progress every 50 assets or in debug mode
			if logger.IsLevelEnabled(logrus.DebugLevel) || (idx > 0 && idx%50 == 0) {
				logger.Infof("   Analyzing trashed asset %d/%d...", idx+1, len(trashedAssets))
			}
			logger.Debugf("Analyzing trashed asset: %s", trashedAsset.OriginalFileName)

			// Check if this appears to be a replaced file
			// A file is considered replaced if:
			// 1. There's an active asset with the same filename
			// 2. The active asset was created/modified after the trashed asset
			if activeReplacements, exists := activeAssetsByFilename[trashedAsset.OriginalFileName]; exists && len(activeReplacements) > 0 {
				isReplacement := false
				for _, activeAsset := range activeReplacements {
					// Check if the active asset is newer (indicating it's a replacement)
					// Compare using FileCreatedAt or FileModifiedAt timestamps
					if activeAsset.FileCreatedAt > trashedAsset.FileCreatedAt ||
						activeAsset.FileModifiedAt > trashedAsset.FileModifiedAt ||
						activeAsset.UpdatedAt > trashedAsset.UpdatedAt {
						logger.Debugf("  🔄 Skipping %s - appears to be replaced (newer version exists)", trashedAsset.OriginalFileName)
						isReplacement = true
						break
					}
				}
				if isReplacement {
					replacementCount++
					continue // Skip this trashed asset as it's been replaced
				}
			}

			// Create a combined asset list: trashed asset + all active assets
			combinedAssets := make([]utils.TAsset, 0, len(activeAssets)+1)
			combinedAssets = append(combinedAssets, trashedAsset)
			combinedAssets = append(combinedAssets, activeAssets...)

			// Run stacker criteria on the combined list
			emptyLogrusLogger := logrus.New()
			emptyLogrusLogger.SetOutput(io.Discard)
			stacks, err := stacker.StackBy(combinedAssets, criteria, parentFilenamePromote, parentExtPromote, emptyLogrusLogger)
			if err != nil {
				logger.Errorf("Error using stacker criteria for asset %s: %v", trashedAsset.OriginalFileName, err)
				continue
			}

			// Find groups that contain our trashed asset
			for _, stack := range stacks {
				containsTrashedAsset := false
				for _, asset := range stack {
					if asset.ID == trashedAsset.ID {
						containsTrashedAsset = true
						break
					}
				}

				// If this group contains the trashed asset, all other assets in the group should be trashed
				if containsTrashedAsset && len(stack) > 1 {
					// Check if any assets in the stack are potential replacements
					hasReplacement := false
					for _, relatedAsset := range stack {
						if relatedAsset.ID != trashedAsset.ID && !relatedAsset.IsTrashed {
							// Check if this might be a replacement file (same name but newer)
							if relatedAsset.OriginalFileName == trashedAsset.OriginalFileName &&
								(relatedAsset.FileCreatedAt > trashedAsset.FileCreatedAt ||
									relatedAsset.FileModifiedAt > trashedAsset.FileModifiedAt ||
									relatedAsset.UpdatedAt > trashedAsset.UpdatedAt) {
								hasReplacement = true
								logger.Debugf("  🔄 Found replacement file %s, skipping stack", relatedAsset.OriginalFileName)
								break
							}
						}
					}

					if !hasReplacement {
						logger.Debugf("Stack found with %d assets (1 in trash, %d active):", len(stack), len(stack)-1)
						logger.Debugf("  🗑️  %s (already in trash)", trashedAsset.OriginalFileName)
						for _, relatedAsset := range stack {
							if relatedAsset.ID != trashedAsset.ID && !relatedAsset.IsTrashed {
								assetsToTrash[relatedAsset.ID] = relatedAsset
								trashedAssetMapping[relatedAsset.ID] = trashedAsset.OriginalFileName
								logger.Debugf("  ➡️  %s (active → will trash)", relatedAsset.OriginalFileName)
							}
						}
					}
				}
			}
		}

		/**********************************************************************************************
		** Find orphaned DNG files (DNG without corresponding JPG).
		**********************************************************************************************/
		logger.Info("🔍 Looking for orphaned DNG files...")

		// Helper function to normalize base names for edge cases like DO0xxxxxx -> Lxxxxxx
		normalizeBaseName := func(baseName string) string {
			// First, handle default delimiters used by the stacker (~ and _)
			// Split on common suffixes like _preview, _edited, etc.
			if idx := strings.Index(baseName, "_"); idx > 0 {
				baseName = baseName[:idx]
			}
			if idx := strings.Index(baseName, "~"); idx > 0 {
				baseName = baseName[:idx]
			}

			// Handle camera quirk: DO0xxxxxx, DLxxxxxx and Lxxxxxx are the same
			// Extract the numeric part after the prefix
			if strings.HasPrefix(baseName, "DO0") && len(baseName) > 3 {
				// DO01001336 -> 1001336
				return baseName[3:]
			}
			if strings.HasPrefix(baseName, "DL0") && len(baseName) > 3 {
				// DL01000491 -> 1000491
				return baseName[3:]
			}
			if strings.HasPrefix(baseName, "DL") && len(baseName) > 2 {
				// DL1000491 -> 1000491 (in case there's no leading zero)
				if len(baseName) > 2 && baseName[2] >= '0' && baseName[2] <= '9' {
					return baseName[2:]
				}
			}
			if strings.HasPrefix(baseName, "L") && len(baseName) > 1 {
				// L1001336 -> 1001336
				// But only if followed by numbers
				if len(baseName) > 1 && baseName[1] >= '0' && baseName[1] <= '9' {
					return baseName[1:]
				}
			}
			return baseName
		}

		// Build a map of base filenames (without extension) to their assets
		assetsByBaseName := make(map[string][]utils.TAsset)
		for _, asset := range activeAssets {
			// Get base filename without extension
			baseName := strings.TrimSuffix(asset.OriginalFileName, filepath.Ext(asset.OriginalFileName))
			// Also handle files with multiple extensions like .edit.jpg
			if strings.Contains(baseName, ".") {
				// Keep everything before the last dot sequence
				parts := strings.Split(asset.OriginalFileName, ".")
				if len(parts) > 2 {
					// For files like L1000746.edit.jpg, baseName becomes L1000746
					baseName = parts[0]
				}
			}

			// Normalize the base name for edge cases
			normalizedName := normalizeBaseName(baseName)
			assetsByBaseName[normalizedName] = append(assetsByBaseName[normalizedName], asset)

			// Debug log to see the normalization
			logger.Debugf("  Normalized %s -> %s (via base: %s)", asset.OriginalFileName, normalizedName, baseName)
		}

		// Check if assets are already in stacks
		isInStackWithJPG := func(dngAsset utils.TAsset) bool {
			if dngAsset.Stack == nil || dngAsset.Stack.ID == "" {
				return false
			}

			// Find all assets in the same stack
			for _, asset := range activeAssets {
				if asset.Stack != nil && asset.Stack.ID == dngAsset.Stack.ID && asset.ID != dngAsset.ID {
					ext := strings.ToLower(filepath.Ext(asset.OriginalFileName))
					if ext == ".jpg" || ext == ".jpeg" {
						// This DNG is in a stack that already has a JPG
						return true
					}
				}
			}
			return false
		}

		orphanedDNGCount := 0
		skippedStackedDNGCount := 0
		for _, assets := range assetsByBaseName {
			hasDNG := false
			hasJPG := false
			var dngAsset utils.TAsset

			for _, asset := range assets {
				ext := strings.ToLower(filepath.Ext(asset.OriginalFileName))
				if ext == ".dng" {
					hasDNG = true
					dngAsset = asset
				} else if ext == ".jpg" || ext == ".jpeg" {
					hasJPG = true
				}
			}

			// If we have a DNG but no JPG, it might be orphaned
			if hasDNG && !hasJPG {
				// But first check if it's already in a stack with a JPG
				if isInStackWithJPG(dngAsset) {
					skippedStackedDNGCount++
					logger.Debugf("  ✅ Skipping DNG %s - already in stack with JPG", dngAsset.OriginalFileName)
				} else {
					orphanedDNGCount++
					assetsToTrash[dngAsset.ID] = dngAsset
					trashedAssetMapping[dngAsset.ID] = "orphaned DNG"
					logger.Debugf("  🔍 Found orphaned DNG: %s (no corresponding JPG)", dngAsset.OriginalFileName)
				}
			}
		}

		if orphanedDNGCount > 0 {
			logger.Infof("📸 Found %d orphaned DNG files without corresponding JPG files", orphanedDNGCount)
		}
		if skippedStackedDNGCount > 0 {
			logger.Infof("✅ Skipped %d DNG files that are already in stacks with JPG files", skippedStackedDNGCount)
		}

		/**********************************************************************************************
		** Move the identified assets to trash.
		**********************************************************************************************/
		if len(assetsToTrash) == 0 {
			if replacementCount > 0 || orphanedDNGCount == 0 {
				logger.Info("✅ No related assets need to be trashed (replaced files were skipped, no orphaned DNGs found).")
			} else {
				logger.Info("✅ No related assets found that need to be moved to trash.")
			}
			continue
		}

		// Group by file extension for summary
		extensionCount := make(map[string]int)
		assetIDs := make([]string, 0, len(assetsToTrash))

		// In debug mode, show detailed mapping
		if logger.IsLevelEnabled(logrus.InfoLevel) {
			logger.Infof("📋 Summary of assets to trash (%d):", len(assetsToTrash))
			// Group by the trashed asset that caused them to be marked
			groupedByTrashed := make(map[string][]utils.TAsset)
			orphanedDNGs := make([]utils.TAsset, 0)

			for _, asset := range assetsToTrash {
				trashedName := trashedAssetMapping[asset.ID]
				if trashedName == "orphaned DNG" {
					orphanedDNGs = append(orphanedDNGs, asset)
				} else {
					groupedByTrashed[trashedName] = append(groupedByTrashed[trashedName], asset)
				}
			}

			// Show orphaned DNGs first if any
			if len(orphanedDNGs) > 0 {
				orphanedNames := make([]string, 0, len(orphanedDNGs))
				for _, asset := range orphanedDNGs {
					orphanedNames = append(orphanedNames, asset.OriginalFileName)
				}
				logger.Infof("\t📸 Orphaned DNG files (no JPG found): %s\n", strings.Join(orphanedNames, ", "))
			}

			// Then show regular stack-related assets
			for trashedName, relatedAssets := range groupedByTrashed {
				relatedAssetNames := make([]string, 0, len(relatedAssets))
				for _, asset := range relatedAssets {
					relatedAssetNames = append(relatedAssetNames, asset.OriginalFileName)
				}
				logger.Infof("\t%s (in trash): %s\n", trashedName, strings.Join(relatedAssetNames, ", "))
			}
		}

		for _, asset := range assetsToTrash {
			ext := filepath.Ext(asset.OriginalFileName)
			if ext == "" {
				ext = "(no extension)"
			}
			extensionCount[ext]++
			assetIDs = append(assetIDs, asset.ID)
		}

		// Show summary by file type
		if len(extensionCount) > 0 {
			logger.Debugf("📁 Assets to trash by type:")
			for ext, count := range extensionCount {
				logger.Debugf("   - %s files: %d", strings.ToUpper(strings.TrimPrefix(ext, ".")), count)
			}
		}

		if err := client.TrashAssets(assetIDs); err != nil {
			logger.Errorf("Error moving assets to trash: %v", err)
		}
	}
}
