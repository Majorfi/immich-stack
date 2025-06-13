/**************************************************************************************************
** Fix-trash command implementation for the Immich CLI application.
** Handles stack-aware trash operations to maintain consistency.
**************************************************************************************************/

package main

import (
	"strings"

	"github.com/majorfi/immich-stack/pkg/immich"
	"github.com/majorfi/immich-stack/pkg/stacker"
	"github.com/majorfi/immich-stack/pkg/utils"
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
		client := immich.NewClient(apiURL, key, false, false, dryRun, withArchived, withDeleted, logger)
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

		existingStacks, err := client.FetchAllStacks(false)
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

		// Filter active (non-trashed) assets
		activeAssets := make([]utils.TAsset, 0)
		for _, asset := range allAssets {
			if !asset.IsTrashed {
				activeAssets = append(activeAssets, asset)
			}
		}

		logger.Infof("Analyzing %d trashed assets against %d active assets using criteria matching", len(trashedAssets), len(activeAssets))

		for _, trashedAsset := range trashedAssets {
			logger.Debugf("Analyzing trashed asset: %s", trashedAsset.OriginalFileName)

			// Create a combined asset list: trashed asset + all active assets
			combinedAssets := make([]utils.TAsset, 0, len(activeAssets)+1)
			combinedAssets = append(combinedAssets, trashedAsset)
			combinedAssets = append(combinedAssets, activeAssets...)

			// Run stacker criteria on the combined list
			stacks, err := stacker.StackBy(combinedAssets, criteria, parentFilenamePromote, parentExtPromote, logger)
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
					logger.Debugf("Found group of %d assets containing trashed asset %s", len(stack), trashedAsset.OriginalFileName)
					for _, relatedAsset := range stack {
						if relatedAsset.ID != trashedAsset.ID && !relatedAsset.IsTrashed {
							assetsToTrash[relatedAsset.ID] = relatedAsset
							logger.Debugf("Found criteria-matched asset to trash: %s (matches with %s)", relatedAsset.OriginalFileName, trashedAsset.OriginalFileName)
						}
					}
				}
			}
		}

		/**********************************************************************************************
		** Move the identified assets to trash.
		**********************************************************************************************/
		if len(assetsToTrash) == 0 {
			logger.Info("No related assets found that need to be moved to trash.")
			continue
		}

		logger.Infof("Found %d assets that should be moved to trash:", len(assetsToTrash))
		assetIDs := make([]string, 0, len(assetsToTrash))
		for _, asset := range assetsToTrash {
			logger.Infof("  - %s (ID: %s)", asset.OriginalFileName, asset.ID)
			assetIDs = append(assetIDs, asset.ID)
		}

		if err := client.TrashAssets(assetIDs); err != nil {
			logger.Errorf("Error moving assets to trash: %v", err)
		}
	}
}