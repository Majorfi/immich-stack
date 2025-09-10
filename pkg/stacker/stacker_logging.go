package stacker

import "github.com/sirupsen/logrus"

/**************************************************************************************************
** logStackingResults logs stacking results with consistent format across different stacking modes.
**
** @param mode - The stacking mode used (e.g., "Legacy criteria stacking")
** @param stackCount - Number of stacks formed
** @param assetCount - Total number of assets processed
** @param logger - Logger instance to use
**************************************************************************************************/
func logStackingResults(mode string, stackCount, assetCount int, logger *logrus.Logger) {
	logger.Infof("%s formed %d stacks from %d assets", mode, stackCount, assetCount)
}
