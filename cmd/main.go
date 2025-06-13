/**************************************************************************************************
** Main entry point for the Immich CLI application. This tool automatically groups
** similar photos into stacks within the Immich photo management system.
**************************************************************************************************/

package main

import (
	"os"

	"github.com/majorfi/immich-stack/pkg/utils"
	"github.com/spf13/cobra"
)

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

	var duplicatesCmd = &cobra.Command{
		Use:   "duplicates",
		Short: "List duplicate assets",
		Long:  "Scan your Immich library and list duplicate assets based on filename and timestamp.",
		Run:   runDuplicates,
	}

	var fixTrashCmd = &cobra.Command{
		Use:   "fix-trash",
		Short: "Fix incomplete stack trash operations",
		Long:  "Scan trash for assets and move related stack members to trash to maintain consistency.",
		Run:   runFixTrash,
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

	rootCmd.AddCommand(duplicatesCmd)
	rootCmd.AddCommand(fixTrashCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
