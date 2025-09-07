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
** CreateTestableRootCommand mirrors CreateRootCommand but is kept separate for tests that want
** to override Run or inject args without affecting the real command symbol.
**************************************************************************************************/
func CreateTestableRootCommand() *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:   "immich-stack",
		Short: "Immich Stack CLI",
		Long:  "A tool to automatically stack Immich assets.",
		Run:   runStacker,
	}

	bindFlags(rootCmd)
	addSubcommands(rootCmd)
	return rootCmd
}

/**************************************************************************************************
** configureLoggerForTesting allows tests to capture log output from configureLogger.
** This enables proper testing of warning messages.
**************************************************************************************************/
func configureLoggerForTesting(output io.Writer) *logrus.Logger {
	return configureLoggerWithOutput(output)
}
