/**************************************************************************************************
** Stacker command implementation for the Immich CLI application.
** Handles the main stacking operations, including asset grouping and stack management.
**************************************************************************************************/

package main

import (
	"sync"
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
	if len(stack) == 0 {
		return "", nil, nil
	}

	var existingStack *utils.TStack
	for _, asset := range stack {
		if asset.Stack != nil {
			existingStack = asset.Stack
			break
		}
	}

	if existingStack == nil {
		return "", nil, nil
	}

	parentID := existingStack.PrimaryAssetID

	if len(existingStack.Assets) == 0 {
		return parentID, nil, []string{parentID}
	}

	childrenIDs := make([]string, 0, len(existingStack.Assets)-1)
	for _, asset := range existingStack.Assets {
		if asset.ID != parentID {
			childrenIDs = append(childrenIDs, asset.ID)
		}
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
	** The chained fix-trash run never goes through runFixTrash, so its inert-setting
	** diagnostic must also live here.
	**********************************************************************************************/
	if fixTrashAfterStacking && !trashOrphanedRAWs && rawOrphanExtensions != "" {
		logger.Warnf("--raw-orphan-extensions (RAW_ORPHAN_EXTENSIONS) has no effect without --trash-orphaned-raws")
	}

	/**********************************************************************************************
	** Support multiple API keys (comma-separated).
	**********************************************************************************************/
	apiKeys := splitCommaList(apiKey)
	if len(apiKeys) == 0 {
		logger.Fatalf("No API key(s) provided.")
	}

	if runMode == "cron" {
		logger.Infof("Running in cron mode with interval of %d seconds", cronInterval)
		runCronLoopForAllUsers(apiKeys, apiURL, logger)
	} else {
		for i, key := range apiKeys {
			if i > 0 {
				logger.Infof("\n")
			}
			client := immich.NewClient(apiURL, key, resetStacks, replaceStacks, dryRun, withArchived, withDeleted, removeSingleAssetStacks, includeVideos, stackConcurrency, filterAlbumIDs, filterTakenAfter, filterTakenBefore, logger)
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
			logger.Info("Running in once mode")
			runStackerOnce(client, logger, user.ID)
			if fixTrashAfterStacking {
				logger.Infof("\n")
				fixTrashForAPIKey(key, logger)
			}
		}
	}
}

/**************************************************************************************************
** Runs the stacker process once, handling all the core functionality of fetching assets,
** grouping them into stacks, and applying updates to Immich.
**
** @param client - Immich client instance
** @param logger - Logger instance for outputting status and errors
** @param ownerID - Current user's UUID; non-owned assets (e.g., partner-shared with timeline
**                  visibility) are filtered out before stacking since the API rejects writes
**                  on them.
**************************************************************************************************/
func runStackerOnce(client *immich.Client, logger *logrus.Logger, ownerID string) {
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
	assets = filterOutPartnerAssets(assets, ownerID, logger)

	/**********************************************************************************************
	** Group the assets into stacks.
	**********************************************************************************************/
	stacks, err := stacker.StackBy(assets, criteria, parentFilenamePromote, parentExtPromote, logger)
	if err != nil {
		logger.Fatalf("Error stacking assets: %v", err)
	}

	/**********************************************************************************************
	** Process each candidate stack. When stackConcurrency > 1, multiple stacks are written in
	** parallel using a bounded semaphore — the sequence WITHIN each stack (delete children →
	** modify) stays ordered, only different stacks run concurrently. Default concurrency = 1
	** preserves the historical sequential behavior. See issue #53.
	**********************************************************************************************/
	concurrency := max(stackConcurrency, 1)
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	for i, stack := range stacks {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, stack []utils.TAsset) {
			defer wg.Done()
			defer func() { <-sem }()
			processStack(client, logger, i, len(stacks), stack)
		}(i, stack)
	}
	wg.Wait()
}

/**************************************************************************************************
** processStack runs the per-stack pipeline: compare against existing state, optionally delete
** child stacks (for replace mode), throttle if requested, and call ModifyStack. Designed to be
** safe to invoke from many goroutines in parallel — operates on its own stack slice and uses
** the client/logger which are themselves goroutine-safe.
**************************************************************************************************/
func processStack(client *immich.Client, logger *logrus.Logger, i int, total int, stack []utils.TAsset) {
	_, _, newStackIDs := getParentAndChildrenIDs(stack)
	_, _, originalStackIDs := getOriginalStackIDs(stack)

	/**********************************************************************************************
	** Doing standard stacker checks. These early-returns only emit debug-level skip reasons, so
	** they never produce interleaved info output.
	**********************************************************************************************/
	if !isValidStack(newStackIDs) {
		logger.Debugf("\t⚠️ Invalid stack: %s", stack[0].OriginalFileName)
		return
	}
	if !needsStackUpdate(originalStackIDs, newStackIDs) {
		logger.Debugf("\tℹ️ No update needed for stack: %s", stack[0].OriginalFileName)
		return
	}
	childrenWithStack, hasChildrenWithStack := getChildrenWithStack(stack)
	if hasChildrenWithStack && !replaceStacks {
		logger.Debugf("\tℹ️ No replaceStacks, skipping stack: %s", stack[0].OriginalFileName)
		return
	}

	/**********************************************************************************************
	** Delete children stacks if replaceStacks is true. The deletes run here (concurrently with
	** other stacks) but their result lines are collected so they can be printed inside the
	** contiguous block below rather than interleaving with other in-flight stacks.
	**********************************************************************************************/
	var deleteMsgs []string
	if replaceStacks {
		for _, childID := range childrenWithStack {
			msg, err := client.DeleteStackCollect(childID, utils.REASON_REPLACE_CHILD_STACK_WITH_NEW_ONE)
			if err != nil {
				continue
			}
			deleteMsgs = append(deleteMsgs, msg)
		}
	}

	/**********************************************************************************************
	** Determine action type for logging.
	**********************************************************************************************/
	var actionMsg string
	if len(originalStackIDs) == 0 {
		actionMsg = "\t🆕 Creating new stack"
	} else if replaceStacks && len(childrenWithStack) > 0 {
		actionMsg = "\t🔄 Replacing existing stack (deleted child stacks)"
	} else {
		actionMsg = "\t✏️  Updating stack configuration"
	}

	emitStackReport(logger, i, total, stack, originalStackIDs, newStackIDs, deleteMsgs, actionMsg)

	/**********************************************************************************************
	** Optional throttle between stack writes. Default is no delay — empirically Immich has no
	** rate limit on POST /stacks and handles bursts fine. Users with very large libraries on
	** slow hosting can opt in to a 50ms gap via PREVENT_SELF_REKT=true if they observe upstream
	** errors. See issue #53.
	**********************************************************************************************/
	if preventSelfRekt {
		time.Sleep(50 * time.Millisecond)
	}
	if err := client.ModifyStack(newStackIDs); err != nil {
		logger.Errorf("Error modifying stack: %v", err)
	}
}

/**************************************************************************************************
** stackLogMu serializes the multi-line per-stack report so that, when STACK_CONCURRENCY > 1,
** the lines of one stack stay contiguous instead of interleaving with other in-flight stacks.
** Only the (fast) logging is serialized — the API work in processStack still runs concurrently.
**************************************************************************************************/
var stackLogMu sync.Mutex

/**************************************************************************************************
** emitStackReport prints one stack's full report as a contiguous block. Each line is a separate
** logrus entry (preserving structured fields and text/json formatting); the mutex is what keeps
** the block together under concurrency. The block is emitted at info level so it shows in normal
** runs; the extra comparison line is debug-only.
**************************************************************************************************/
func emitStackReport(logger *logrus.Logger, i int, total int, stack []utils.TAsset, originalStackIDs []string, newStackIDs []string, deleteMsgs []string, actionMsg string) {
	stackLogMu.Lock()
	defer stackLogMu.Unlock()

	logger.Info("--------------------------------")
	logger.Infof("%d/%d Key: %s", i+1, total, stack[0].OriginalFileName)
	logger.WithFields(logrus.Fields{
		"Name": stack[0].OriginalFileName,
		"ID":   stack[0].ID,
		"Time": stack[0].LocalDateTime,
	}).Info("\tParent")
	for _, child := range stack[1:] {
		logger.WithFields(logrus.Fields{
			"Name": child.OriginalFileName,
			"ID":   child.ID,
			"Time": child.LocalDateTime,
		}).Info("\tChild")
	}
	if logger.IsLevelEnabled(logrus.DebugLevel) {
		logger.Debugf("\tStack comparison: Original=%v Expected=%v REPLACE_STACKS=%v", originalStackIDs, newStackIDs, replaceStacks)
	}
	for _, msg := range deleteMsgs {
		logger.Info(msg)
	}
	logger.Info(actionMsg)
}

/**************************************************************************************************
** Runs the stacker process in a continuous loop for all users. Processes each user sequentially
** in each iteration to ensure all users are handled.
**
** @param apiKeys - Array of API keys for each user
** @param apiURL - Base URL for the Immich API
** @param logger - Logger instance for outputting status and errors
**************************************************************************************************/
func runCronLoopForAllUsers(apiKeys []string, apiURL string, logger *logrus.Logger) {
	for {
		for i, key := range apiKeys {
			if i > 0 {
				logger.Infof("\n")
			}
			client := immich.NewClient(apiURL, key, resetStacks, replaceStacks, dryRun, withArchived, withDeleted, removeSingleAssetStacks, includeVideos, stackConcurrency, filterAlbumIDs, filterTakenAfter, filterTakenBefore, logger)
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
			runStackerOnce(client, logger, user.ID)
			if fixTrashAfterStacking {
				logger.Infof("\n")
				fixTrashForAPIKey(key, logger)
			}
		}
		logger.Infof("Sleeping for %d seconds until next run", cronInterval)
		time.Sleep(time.Duration(cronInterval) * time.Second)
	}
}
