/**************************************************************************************************
** Test-only command creation utilities - only available during testing
**************************************************************************************************/

package main

import (
	"io"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
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

/************************************************************************************************
** Tests for filter flags binding and parsing
************************************************************************************************/

func TestFilterFlagsBinding(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedAlbum  []string
		expectedAfter  string
		expectedBefore string
	}{
		{
			name:           "album flag single value",
			args:           []string{"--filter-album-ids=album1"},
			expectedAlbum:  []string{"album1"},
			expectedAfter:  "",
			expectedBefore: "",
		},
		{
			name:           "album flag multiple values",
			args:           []string{"--filter-album-ids=album1,album2"},
			expectedAlbum:  []string{"album1", "album2"},
			expectedAfter:  "",
			expectedBefore: "",
		},
		{
			name:           "album flag with UUID",
			args:           []string{"--filter-album-ids=550e8400-e29b-41d4-a716-446655440000"},
			expectedAlbum:  []string{"550e8400-e29b-41d4-a716-446655440000"},
			expectedAfter:  "",
			expectedBefore: "",
		},
		{
			name:           "date flags both set",
			args:           []string{"--filter-taken-after=2024-01-01T00:00:00Z", "--filter-taken-before=2024-12-31T23:59:59Z"},
			expectedAlbum:  nil,
			expectedAfter:  "2024-01-01T00:00:00Z",
			expectedBefore: "2024-12-31T23:59:59Z",
		},
		{
			name:           "only taken-after flag",
			args:           []string{"--filter-taken-after=2024-06-15T12:00:00Z"},
			expectedAlbum:  nil,
			expectedAfter:  "2024-06-15T12:00:00Z",
			expectedBefore: "",
		},
		{
			name:           "only taken-before flag",
			args:           []string{"--filter-taken-before=2024-06-15T12:00:00Z"},
			expectedAlbum:  nil,
			expectedAfter:  "",
			expectedBefore: "2024-06-15T12:00:00Z",
		},
		{
			name:           "all filter flags combined",
			args:           []string{"--filter-album-ids=album1,album2", "--filter-taken-after=2024-01-01T00:00:00Z", "--filter-taken-before=2024-12-31T23:59:59Z"},
			expectedAlbum:  []string{"album1", "album2"},
			expectedAfter:  "2024-01-01T00:00:00Z",
			expectedBefore: "2024-12-31T23:59:59Z",
		},
		{
			name:           "no filter flags",
			args:           []string{},
			expectedAlbum:  nil,
			expectedAfter:  "",
			expectedBefore: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global variables
			filterAlbumIDs = nil
			filterTakenAfter = ""
			filterTakenBefore = ""

			// Create a fresh command
			cmd := CreateTestableRootCommand()

			// Set command to not run (we just want to test flag parsing)
			cmd.Run = nil
			cmd.RunE = func(c *cobra.Command, args []string) error {
				return nil
			}

			// Set args
			cmd.SetArgs(tt.args)

			// Suppress output
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)

			// Execute to parse flags
			err := cmd.Execute()
			assert.NoError(t, err)

			// Assert
			assert.Equal(t, tt.expectedAlbum, filterAlbumIDs, "filterAlbumIDs should match")
			assert.Equal(t, tt.expectedAfter, filterTakenAfter, "filterTakenAfter should match")
			assert.Equal(t, tt.expectedBefore, filterTakenBefore, "filterTakenBefore should match")
		})
	}
}

func TestFilterFlagsOverrideEnvVars(t *testing.T) {
	// This test verifies that CLI flags take precedence over environment variables

	// Set environment variables
	os.Setenv("FILTER_TAKEN_AFTER", "2023-01-01T00:00:00Z")
	os.Setenv("FILTER_TAKEN_BEFORE", "2023-12-31T23:59:59Z")
	defer func() {
		os.Unsetenv("FILTER_TAKEN_AFTER")
		os.Unsetenv("FILTER_TAKEN_BEFORE")
	}()

	// Reset global variables
	filterAlbumIDs = nil
	filterTakenAfter = ""
	filterTakenBefore = ""

	// Create command with CLI flags that should override env vars
	cmd := CreateTestableRootCommand()
	cmd.Run = nil
	cmd.RunE = func(c *cobra.Command, args []string) error {
		return nil
	}

	// Set args with different values than env vars
	cmd.SetArgs([]string{"--filter-taken-after=2024-06-01T00:00:00Z"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	// Execute
	err := cmd.Execute()
	assert.NoError(t, err)

	// CLI flag should take precedence
	assert.Equal(t, "2024-06-01T00:00:00Z", filterTakenAfter, "CLI flag should override env var")
}
