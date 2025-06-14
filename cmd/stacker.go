/**************************************************************************************************
** Stacker command implementation for the Immich CLI application.
** Handles the main stacking operations, including asset grouping and stack management.
**************************************************************************************************/

package main

import (
	"strings"
	"time"

	"github.com/majorfi/immich-stack/pkg/immich"
	"github.com/majorfi/immich-stack/pkg/stacker"
	"github.com/majorfi/immich-stack/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

/**************************************************************************************************
** Extracts parent and child asset IDs from a stack of assets. The first asset is considered
** the parent, while subsequent assets are treated as children. This function is used when
** creating new stacks or modifying existing ones.
**
** @param stack - Array of assets to process
** @return parentID - ID of the parent asset
** @return childrenIDs - Array of child asset IDs
** @return newStackIDs - Combined array of parent and child IDs
**************************************************************************************************/
func getParentAndChildrenIDs(stack []utils.TAsset) (string, []string, []string) {
	parentID := stack[0].ID
	childrenIDs := make([]string, 0, len(stack)-1)
	for _, asset := range stack[1:] {
		if asset.ID != parentID {
			childrenIDs = append(childrenIDs, asset.ID)
		}
	}
	newStackIDs := append([]string{parentID}, utils.RemoveEmptyStrings(childrenIDs)...)
	return parentID, childrenIDs, newStackIDs
}

/**************************************************************************************************
** Retrieves the original stack configuration from Immich for a given stack of assets.
** This is used to compare existing stacks with proposed new configurations.
**
** @param stack - Array of assets to process
** @return parentID - ID of the parent asset in existing stack
** @return childrenIDs - Array of child asset IDs in existing stack
** @return originalStackIDs - Combined array of existing parent and child IDs
**************************************************************************************************/
func getOriginalStackIDs(stack []utils.TAsset) (string, []string, []string) {
	if len(stack) == 0 || stack[0].Stack == nil {
		return "", nil, nil
	}
	parentID := stack[0].Stack.PrimaryAssetID
	childrenIDs := make([]string, len(stack[0].Stack.Assets)-1)
	for i, asset := range stack[0].Stack.Assets[1:] {
		childrenIDs[i] = asset.ID
	}
	originalStackIDs := append([]string{parentID}, childrenIDs...)
	return parentID, childrenIDs, originalStackIDs
}

/**************************************************************************************************
** Validates if a proposed stack configuration is valid. A valid stack must have at least
** one child asset and the parent asset must not be listed as a child.
**
** @param newStackIDs - Array of asset IDs to validate
** @return bool - True if the stack configuration is valid
**************************************************************************************************/
func isValidStack(newStackIDs []string) bool {
	newStackIDs = utils.RemoveEmptyStrings(newStackIDs)
	if len(newStackIDs) <= 1 {
		return false
	}
	parentID := newStackIDs[0]
	for _, childID := range newStackIDs[1:] {
		if childID == parentID {
			return false
		}
	}
	return true
}

/**************************************************************************************************
** Determines if a stack needs to be updated by comparing original and expected configurations.**
** Takes into account the replaceStacks flag to decide whether to force updates.
**
** @param originalStack - Array of IDs from existing stack
** @param expectedStack - Array of IDs from proposed new stack
** @return bool - True if the stack needs to be updated
**************************************************************************************************/
func needsStackUpdate(originalStack, expectedStack []string) bool {
	if len(expectedStack) <= 1 {
		return false
	}
	if len(originalStack) != len(expectedStack) {
		return true
	}

	if !utils.AreArraysEqual(originalStack, expectedStack) && replaceStacks {
		return true
	}
	return false
}

/**************************************************************************************************
** Identifies any child assets that are already part of existing stacks. This is used to
** prevent conflicts when creating new stacks and to handle stack replacement scenarios.
**
** @param stack - Array of assets to check
** @return []string - Array of stack IDs where conflicts were found
** @return bool - True if any conflicts were found
**************************************************************************************************/
func getChildrenWithStack(stack []utils.TAsset) ([]string, bool) {
	childrenWithStack := make([]string, 0)
	for _, asset := range stack[1:] {
		if asset.Stack != nil {
			childrenWithStack = append(childrenWithStack, asset.Stack.ID)
		}
	}
	return childrenWithStack, len(childrenWithStack) > 0
}

/**************************************************************************************************
** Main execution logic for the stacker process. Handles the core workflow of fetching assets,
** grouping them into stacks, and applying updates to Immich. Includes detailed logging and
** error handling throughout the process.
**
** @param cmd - Cobra command instance
** @param args - Command line arguments
**************************************************************************************************/
func runStacker(cmd *cobra.Command, args []string) {
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
		client := immich.NewClient(apiURL, key, resetStacks, replaceStacks, dryRun, withArchived, withDeleted, removeSingleAssetStacks, logger)
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
		logger.Infof("Running for user: %s (%s)", user.Name, user.Email)
		logger.Infof("=====================================================================================")
		if runMode == "cron" {
			logger.Infof("Running in cron mode with interval of %d seconds", cronInterval)
			runCronLoop(client, logger)
		} else {
			logger.Info("Running in once mode")
			runStackerOnce(client, logger)
		}
	}
}

/**************************************************************************************************
** Runs the stacker process once, handling all the core functionality of fetching assets,
** grouping them into stacks, and applying updates to Immich.
**
** @param client - Immich client instance
** @param logger - Logger instance for outputting status and errors
**************************************************************************************************/
func runStackerOnce(client *immich.Client, logger *logrus.Logger) {
	/**********************************************************************************************
	** Fetch all the assets from Immich.
	**********************************************************************************************/
	existingStacks, err := client.FetchAllStacks()
	if err != nil {
		logger.Fatalf("Error fetching stacks: %v", err)
	}
	assets, err := client.FetchAssets(1000, existingStacks)
	if err != nil {
		logger.Fatalf("Error fetching assets: %v", err)
	}

	/**********************************************************************************************
	** Group the assets into stacks.
	**********************************************************************************************/
	stacks, err := stacker.StackBy(assets, criteria, parentFilenamePromote, parentExtPromote, logger)
	if err != nil {
		logger.Fatalf("Error stacking assets: %v", err)
	}

	for i, stack := range stacks {
		_, _, newStackIDs := getParentAndChildrenIDs(stack)
		_, _, originalStackIDs := getOriginalStackIDs(stack)

		/******************************************************************************************
		** Adding debug logs
		******************************************************************************************/
		{
			logger.Debugf("--------------------------------")
			logger.Debugf("%d/%d Key: %s", i+1, len(stacks), stack[0].OriginalFileName)
			logger.WithFields(logrus.Fields{
				"Name": stack[0].OriginalFileName,
				"ID":   stack[0].ID,
				"Time": stack[0].LocalDateTime,
			}).Debugf("\tParent")
			for _, child := range stack[1:] {
				logger.WithFields(logrus.Fields{
					"Name": child.OriginalFileName,
					"ID":   child.ID,
					"Time": child.LocalDateTime,
				}).Debugf("\tChild")
			}
		}

		/******************************************************************************************
		** Doing standard stacker checks.
		******************************************************************************************/
		if !isValidStack(newStackIDs) {
			logger.Debugf("\t⚠️ Invalid stack: %s", stack[0].OriginalFileName)
			continue
		}
		if !needsStackUpdate(originalStackIDs, newStackIDs) {
			logger.Debugf("\tℹ️ No update needed for stack: %s", stack[0].OriginalFileName)
			continue
		}
		childrenWithStack, hasChildrenWithStack := getChildrenWithStack(stack)
		if hasChildrenWithStack && !replaceStacks {
			logger.Debugf("\tℹ️ No replaceStacks, skipping stack: %s", stack[0].OriginalFileName)
			continue
		}

		/******************************************************************************************
		** Adding info logs, but only if we are not in debug mode.
		******************************************************************************************/
		{
			if logger.Level != logrus.DebugLevel {
				logger.Infof("--------------------------------")
				logger.Infof("%d/%d Key: %s", i+1, len(stacks), stack[0].OriginalFileName)
			}
			if logger.Level != logrus.DebugLevel {
				logger.WithFields(logrus.Fields{
					"Name": stack[0].OriginalFileName,
					"ID":   stack[0].ID,
					"Time": stack[0].LocalDateTime,
				}).Infof("\tParent")
				for _, child := range stack[1:] {
					logger.WithFields(logrus.Fields{
						"Name": child.OriginalFileName,
						"ID":   child.ID,
						"Time": child.LocalDateTime,
					}).Infof("\tChild")
				}
			}
		}

		/******************************************************************************************
		** Delete children stacks if replaceStacks is true.
		******************************************************************************************/
		if replaceStacks {
			for _, childID := range childrenWithStack {
				client.DeleteStack(childID, utils.REASON_REPLACE_CHILD_STACK_WITH_NEW_ONE)
			}
		}

		/******************************************************************************************
		** Modify the stack after a little delay to avoid self-rekt.
		******************************************************************************************/
		time.Sleep(100 * time.Millisecond)
		if err := client.ModifyStack(newStackIDs); err != nil {
			logger.Errorf("Error modifying stack: %v", err)
		}
	}
}

/**************************************************************************************************
** Runs the stacker process in a loop with the specified interval. Handles graceful shutdown
** and error recovery between runs.
**
** @param client - Immich client instance
** @param logger - Logger instance for outputting status and errors
**************************************************************************************************/
func runCronLoop(client *immich.Client, logger *logrus.Logger) {
	for {
		runStackerOnce(client, logger)
		logger.Infof("Sleeping for %d seconds until next run", cronInterval)
		time.Sleep(time.Duration(cronInterval) * time.Second)
	}
}
