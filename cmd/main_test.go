/**************************************************************************************************
** Test-only command creation utilities - only available during testing
**************************************************************************************************/

package main

import (
	"io"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

/**************************************************************************************************
** CreateTestableRootCommand creates a command structure identical to CreateRootCommand but with
** no-op Run functions to avoid calling loadEnv() which would Fatal in tests. For testing only.
**************************************************************************************************/
func CreateTestableRootCommand() *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:   "immich-stack",
		Short: "Immich Stack CLI",
		Long:  "A tool to automatically stack Immich assets.",
		// No Run function - tests will override as needed
	}

	// Use the same flag binding to ensure no drift
	bindFlags(rootCmd)
	
	// Add testable subcommands (no run functions)
	addTestableSubcommands(rootCmd)
	
	return rootCmd
}

/**************************************************************************************************
** addTestableSubcommands adds subcommands without run functions for testing
**************************************************************************************************/
func addTestableSubcommands(rootCmd *cobra.Command) {
	var duplicatesCmd = &cobra.Command{
		Use:   "duplicates",
		Short: "List duplicate assets",
		Long:  "Scan your Immich library and list duplicate assets based on filename and timestamp.",
		// No Run function - tests will override as needed
	}

	var fixTrashCmd = &cobra.Command{
		Use:   "fix-trash",
		Short: "Fix incomplete stack trash operations",
		Long:  "Scan trash for assets and move related stack members to trash to maintain consistency.",
		// No Run function - tests will override as needed
	}

	rootCmd.AddCommand(duplicatesCmd)
	rootCmd.AddCommand(fixTrashCmd)
}

/**************************************************************************************************
** configureLoggerForTesting allows tests to capture log output from configureLogger.
** This enables proper testing of warning messages.
**************************************************************************************************/
func configureLoggerForTesting(output io.Writer) *logrus.Logger {
	return configureLoggerWithOutput(output)
}