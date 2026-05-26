/**************************************************************************************************
** Shared helpers used by multiple cmd subcommands (stacker, duplicates, fix-trash).
**************************************************************************************************/

package main

import (
	"github.com/majorfi/immich-stack/pkg/utils"
	"github.com/sirupsen/logrus"
)

/**************************************************************************************************
** filterOutPartnerAssets removes assets not owned by the current user from a fetched list
** and logs how many were dropped. Partner-shared assets surfaced by /search/metadata cannot
** be modified via the Immich stack API (permission denied), so trying to stack them only
** generates noise — see issue #55.
**
** @param assets - Output of FetchAssets
** @param ownerID - Current user's UUID (from GetCurrentUser)
** @param logger - For logging the drop count
** @return []TAsset - Only the assets owned by ownerID
**************************************************************************************************/
func filterOutPartnerAssets(assets []utils.TAsset, ownerID string, logger *logrus.Logger) []utils.TAsset {
	filtered := utils.FilterAssetsByOwner(assets, ownerID)
	if dropped := len(assets) - len(filtered); dropped > 0 {
		logger.Infof("⏭️  Skipped %d assets owned by partners (not stackable via API)", dropped)
	}
	return filtered
}
