/**************************************************************************************************
** Duplicates command implementation for the Immich CLI application.
** Handles duplicate asset detection and reporting functionality.
**************************************************************************************************/

package main

import (
	"strings"

	"github.com/majorfi/immich-stack/pkg/immich"
	"github.com/majorfi/immich-stack/pkg/utils"
	"github.com/spf13/cobra"
)

/**************************************************************************************************
** Main execution logic for duplicate detection. Fetches assets and calls the ListDuplicates
** function to identify and display duplicate assets based on filename and timestamp.
**
** @param cmd - Cobra command instance
** @param args - Command line arguments
**************************************************************************************************/
func runDuplicates(cmd *cobra.Command, args []string) {
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
		client := immich.NewClient(apiURL, key, false, false, true, withArchived, withDeleted, false, nil, "", "", logger)
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
		logger.Infof("Checking for duplicates for user: %s (%s)", user.Name, user.Email)
		logger.Infof("=====================================================================================")

		/**********************************************************************************************
		** Fetch all assets and check for duplicates.
		**********************************************************************************************/
		existingStacks, err := client.FetchAllStacks()
		if err != nil {
			logger.Errorf("Error fetching stacks: %v", err)
			continue
		}
		assets, err := client.FetchAssets(1000, existingStacks)
		if err != nil {
			logger.Errorf("Error fetching assets: %v", err)
			continue
		}

		/**********************************************************************************************
		** List duplicates using the existing function.
		**********************************************************************************************/
		if err := client.ListDuplicates(assets); err != nil {
			logger.Errorf("Error listing duplicates: %v", err)
		}
	}
}
