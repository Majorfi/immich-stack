/**************************************************************************************************
** Main entry point for the Immich CLI application. This tool automatically groups
** similar photos into stacks within the Immich photo management system.
**************************************************************************************************/

package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/majorfi/immich-stack/pkg/immich"
	"github.com/majorfi/immich-stack/pkg/stacker"
	"github.com/majorfi/immich-stack/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var apiKey string
var apiURL string
var resetStacks bool
var dryRun bool
var replaceStacks bool
var criteria string
var parentFilenamePromote string
var parentExtPromote string

/**************************************************************************************************
** Loads environment variables and command-line flags, with flags taking precedence over env
** variables. Handles critical configuration like API credentials and operation modes.
**
** @param logger - Logger instance for outputting configuration status and errors
**************************************************************************************************/
func loadEnv(logger *logrus.Logger) {
	_ = godotenv.Load()
	if apiKey == "" {
		apiKey = os.Getenv("API_KEY")
	}
	if apiKey == "" {
		logger.Fatal("API_KEY is not set")
	}
	if apiURL == "" {
		apiURL = os.Getenv("API_URL")
	}
	if apiURL == "" {
		apiURL = "http://immich_server:3001/api"
	}
	if !resetStacks {
		resetStacks = os.Getenv("RESET_STACKS") == "true"
	}
	if resetStacks {
		logger.Info("RESET_STACKS is set to true, all existing stacks will be deleted")
		fmt.Print("Are you sure you want to delete all existing stacks? (y/N): ")
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			logger.Info("Operation cancelled by user")
			os.Exit(0)
		}
	}
	if !dryRun {
		dryRun = os.Getenv("DRY_RUN") == "true"
	}
	if dryRun {
		logger.Info("DRY_RUN is set to true, no changes will be applied")
	}
	if !replaceStacks {
		replaceStacks = os.Getenv("REPLACE_STACKS") == "true"
	}
}

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
	childrenIDs := make([]string, len(stack)-1)
	for i, asset := range stack[1:] {
		if asset.ID != parentID {
			childrenIDs[i] = asset.ID
		}
	}
	newStackIDs := append([]string{parentID}, childrenIDs...)
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
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
		FullTimestamp:    false,
	})
	loadEnv(logger)

	/**********************************************************************************************
	** Initialize clients and stacker.
	**********************************************************************************************/
	client := immich.NewClient(apiURL, apiKey, resetStacks, replaceStacks, dryRun, logger)

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
	stacks, err := stacker.StackBy(assets, criteria, parentFilenamePromote, parentExtPromote)
	if err != nil {
		logger.Fatalf("Error stacking assets: %v", err)
	}

	for i, stack := range stacks {
		_, _, newStackIDs := getParentAndChildrenIDs(stack)
		_, _, originalStackIDs := getOriginalStackIDs(stack)
		if !isValidStack(newStackIDs) {
			continue
		}
		if !needsStackUpdate(originalStackIDs, newStackIDs) {
			continue
		}
		childrenWithStack, hasChildrenWithStack := getChildrenWithStack(stack)
		if hasChildrenWithStack && !replaceStacks {
			continue
		}

		logger.Infof("--------------------------------")
		logger.Infof("%d/%d Key: %s", i+1, len(stacks), stack[0].OriginalFileName)

		/******************************************************************************************
		** Delete children stacks if replaceStacks is true.
		******************************************************************************************/
		if replaceStacks {
			for _, childID := range childrenWithStack {
				client.DeleteStack(childID, "replacing child stack with new one")
			}
		}

		/******************************************************************************************
		** Create the new stack.
		******************************************************************************************/
		logger.Infof("   Parent name: %-15s AT: %-32s (ID: %s)", stack[0].OriginalFileName, stack[0].LocalDateTime, stack[0].ID)
		for _, child := range stack[1:] {
			logger.Infof("   Child name: %-16s AT: %-32s (ID: %s)", child.OriginalFileName, child.LocalDateTime, child.ID)
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
** Application entry point. Sets up the CLI command structure using Cobra, including all
** available commands and their associated flags. Handles command execution and error
** reporting.
**************************************************************************************************/
func main() {
	var rootCmd = &cobra.Command{
		Use:   "immich-stack",
		Short: "Immich Stack CLI",
		Long:  "A tool to automatically stack Immich assets.",
	}

	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "API key (or set API_KEY env var)")
	rootCmd.PersistentFlags().StringVar(&apiURL, "api-url", "", "API URL (or set API_URL env var)")
	rootCmd.PersistentFlags().BoolVar(&resetStacks, "reset-stacks", false, "Delete all existing stacks (or set RESET_STACKS=true)")
	rootCmd.PersistentFlags().BoolVar(&replaceStacks, "replace-stacks", false, "Replace stacks for new groups (or set REPLACE_STACKS=true)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Dry run (or set DRY_RUN=true)")
	rootCmd.PersistentFlags().StringVar(&criteria, "criteria", "", "Criteria (or set CRITERIA env var)")
	rootCmd.PersistentFlags().StringVar(&parentFilenamePromote, "parent-filename-promote", "", "Parent filename promote (or set PARENT_FILENAME_PROMOTE env var)")
	rootCmd.PersistentFlags().StringVar(&parentExtPromote, "parent-ext-promote", "", "Parent ext promote (or set PARENT_EXT_PROMOTE env var)")

	var runCmd = &cobra.Command{
		Use:   "run",
		Short: "Run the stacker process",
		Run:   runStacker,
	}

	rootCmd.AddCommand(runCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
