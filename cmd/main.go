/**************************************************************************************************
** Main entry point for the Immich CLI application. This tool automatically groups
** similar photos into stacks within the Immich photo management system.
**************************************************************************************************/

package main

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
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
var criteria string
var parentFilenamePromote string
var parentExtPromote string
var runMode string
var cronInterval int
var withArchived bool
var resetStacks bool
var dryRun bool
var replaceStacks bool
var withDeleted bool

/**************************************************************************************************
** Configures the logger based on environment variables. Sets up the log level and format
** according to LOG_LEVEL and LOG_FORMAT environment variables.
**
** @return *logrus.Logger - Configured logger instance
**************************************************************************************************/
func configureLogger() *logrus.Logger {
	logger := logrus.New()

	// Set log level from environment variable
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		if parsedLevel, err := logrus.ParseLevel(level); err == nil {
			logger.SetLevel(parsedLevel)
		} else {
			logger.Warnf("Invalid LOG_LEVEL '%s', using default 'info'", level)
			logger.SetLevel(logrus.InfoLevel)
		}
	} else {
		logger.SetLevel(logrus.InfoLevel)
	}

	// Set log format from environment variable
	if format := os.Getenv("LOG_FORMAT"); format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			DisableTimestamp: true,
			FullTimestamp:    false,
			TimestampFormat:  time.RFC3339,
		})
	}

	return logger
}

/**************************************************************************************************
** validateConfig validates the configuration parameters to ensure they are valid.
**
** @param apiURL - API URL to validate
** @param runMode - Run mode to validate
** @param cronInterval - Cron interval to validate
** @return error - Any validation error
**************************************************************************************************/
func validateConfig(apiURL string, runMode string, cronInterval int) error {
	if apiURL != "" {
		parsedURL, err := url.Parse(apiURL)
		if err != nil {
			return fmt.Errorf("invalid API_URL: %w", err)
		}
		if parsedURL.Scheme == "" || parsedURL.Host == "" {
			return fmt.Errorf("invalid API_URL: missing scheme or host")
		}
	}

	validModes := map[string]bool{"once": true, "cron": true}
	if !validModes[runMode] {
		return fmt.Errorf("invalid RUN_MODE: %s (must be 'once' or 'cron')", runMode)
	}

	if runMode == "cron" && cronInterval <= 0 {
		return fmt.Errorf("CRON_INTERVAL must be positive when RUN_MODE is 'cron'")
	}

	return nil
}

/**************************************************************************************************
** Loads environment variables and command-line flags, with flags taking precedence over env
** variables. Handles critical configuration like API credentials and operation modes.
**
** @param logger - Logger instance for outputting configuration status and errors
**************************************************************************************************/
func loadEnv() *logrus.Logger {
	_ = godotenv.Load()
	logger := configureLogger()
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
	if runMode == "" {
		runMode = os.Getenv("RUN_MODE")
	}
	if runMode == "" {
		runMode = "once"
	}
	if cronInterval == 0 {
		if val := os.Getenv("CRON_INTERVAL"); val != "" {
			if intVal, err := strconv.Atoi(val); err == nil {
				cronInterval = intVal
			} else {
				logger.Warnf("Invalid CRON_INTERVAL '%s', using default", val)
			}
		}
	}
	if cronInterval == 0 && runMode == "cron" {
		cronInterval = 86400
	}

	// Validate configuration
	if err := validateConfig(apiURL, runMode, cronInterval); err != nil {
		logger.Fatalf("Configuration error: %v", err)
	}

	if !resetStacks {
		resetStacks = os.Getenv("RESET_STACKS") == "true"
	}
	if resetStacks {
		if runMode != "once" {
			logger.Fatal("RESET_STACKS can only be used in 'once' run mode. Aborting.")
		}
		confirmReset := os.Getenv("CONFIRM_RESET_STACK")
		const requiredConfirm = "I acknowledge all my current stacks will be deleted and new one will be created"
		if confirmReset != requiredConfirm {
			logger.Fatalf("To use RESET_STACKS, you must set CONFIRM_RESET_STACK to: '%s'", requiredConfirm)
		}
		logger.Info("RESET_STACKS is set to true, all existing stacks will be deleted")
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
	if !withArchived {
		withArchived = os.Getenv("WITH_ARCHIVED") == "true"
	}
	if !withDeleted {
		withDeleted = os.Getenv("WITH_DELETED") == "true"
	}
	return logger
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
	if len(stack) == 0 {
		return "", nil, nil
	}
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
** processAPIKey processes a single API key, handling all operations for that user.
** This function encapsulates all processing for a single API key to avoid state sharing.
**
** @param key - API key to process
** @param index - Index of the API key (for logging)
** @param logger - Logger instance
**************************************************************************************************/
func processAPIKey(key string, index int, logger *logrus.Logger) {
	// Create client for this API key
	client := immich.NewClient(apiURL, key, resetStacks, replaceStacks, dryRun, withArchived, withDeleted, logger)
	if client == nil {
		logger.Errorf("Invalid client for API key at index %d", index)
		return
	}

	user, err := client.GetCurrentUser()
	if err != nil {
		logger.Errorf("Failed to fetch user for API key at index %d: %v", index, err)
		return
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

	// Process each API key sequentially
	// NOTE: This implementation processes API keys sequentially. If concurrent processing
	// is needed in the future, ensure proper synchronization of shared resources.
	for i, key := range apiKeys {
		if i > 0 {
			logger.Infof("\n")
		}

		// Process each API key in isolation to avoid state sharing issues
		processAPIKey(key, i, logger)
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
		** Adding info logs, bug only if we are not in debug mode.
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
		** Modify the stack with retry logic for better reliability.
		******************************************************************************************/
		const maxStackRetries = 3
		var stackErr error
		for attempt := 1; attempt <= maxStackRetries; attempt++ {
			stackErr = client.ModifyStack(newStackIDs)
			if stackErr == nil {
				break
			}

			if attempt < maxStackRetries {
				waitTime := time.Duration(attempt) * time.Second
				logger.Warnf("Stack modification failed (attempt %d/%d), retrying in %v: %v",
					attempt, maxStackRetries, waitTime, stackErr)
				time.Sleep(waitTime)
			} else {
				logger.Errorf("Stack modification failed after %d attempts: %v", maxStackRetries, stackErr)
			}
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
	iteration := 0
	for {
		iteration++
		// Log the iteration for debugging
		logger.WithField("iteration", iteration).Info("Starting new iteration")
		runStackerOnce(client, logger)
		logger.Infof("Sleeping for %d seconds until next run", cronInterval)
		time.Sleep(time.Duration(cronInterval) * time.Second)
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
		Run:   runStacker,
	}

	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "API key (or set API_KEY env var)")
	rootCmd.PersistentFlags().StringVar(&apiURL, "api-url", "", "API URL (or set API_URL env var)")
	rootCmd.PersistentFlags().BoolVar(&resetStacks, "reset-stacks", false, "Delete all existing stacks (or set RESET_STACKS=true)")
	rootCmd.PersistentFlags().BoolVar(&replaceStacks, "replace-stacks", true, "Replace stacks for new groups (or set REPLACE_STACKS=true)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Dry run (or set DRY_RUN=true)")
	rootCmd.PersistentFlags().StringVar(&criteria, "criteria", "", "Criteria (or set CRITERIA env var)")
	rootCmd.PersistentFlags().StringVar(&parentFilenamePromote, "parent-filename-promote", utils.DefaultParentFilenamePromoteString, "Parent filename promote (or set PARENT_FILENAME_PROMOTE env var)")
	rootCmd.PersistentFlags().StringVar(&parentExtPromote, "parent-ext-promote", utils.DefaultParentExtPromoteString, "Parent ext promote (or set PARENT_EXT_PROMOTE env var)")
	rootCmd.PersistentFlags().BoolVar(&withArchived, "with-archived", false, "Include archived assets (or set WITH_ARCHIVED=true)")
	rootCmd.PersistentFlags().BoolVar(&withDeleted, "with-deleted", false, "Include deleted assets (or set WITH_DELETED=true)")
	rootCmd.PersistentFlags().StringVar(&runMode, "run-mode", os.Getenv("RUN_MODE"), "Run mode (or set RUN_MODE env var)")
	rootCmd.PersistentFlags().IntVar(&cronInterval, "cron-interval", 0, "Cron interval (or set CRON_INTERVAL env var)")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
