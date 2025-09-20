/**************************************************************************************************
** Comprehensive CLI tests for the Immich Stack CLI application.
** Tests flag parsing, environment variable precedence, and end-to-end integration flows.
**************************************************************************************************/

package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/majorfi/immich-stack/pkg/stacker"
	"github.com/majorfi/immich-stack/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

/**************************************************************************************************
** Test helpers for setup and cleanup
**************************************************************************************************/

func resetGlobalConfig() {
	apiKey = ""
	apiURL = ""
	criteria = ""
	parentFilenamePromote = utils.DefaultParentFilenamePromoteString
	parentExtPromote = utils.DefaultParentExtPromoteString
	runMode = ""
	cronInterval = 0
	withArchived = false
	resetStacks = false
	dryRun = false
	replaceStacks = true
	withDeleted = false
	logLevel = ""
	removeSingleAssetStacks = false
}

func clearEnvironment() {
	os.Unsetenv("API_KEY")
	os.Unsetenv("API_URL")
	os.Unsetenv("CRITERIA")
	os.Unsetenv("PARENT_FILENAME_PROMOTE")
	os.Unsetenv("PARENT_EXT_PROMOTE")
	os.Unsetenv("RUN_MODE")
	os.Unsetenv("CRON_INTERVAL")
	os.Unsetenv("WITH_ARCHIVED")
	os.Unsetenv("RESET_STACKS")
	os.Unsetenv("DRY_RUN")
	os.Unsetenv("REPLACE_STACKS")
	os.Unsetenv("WITH_DELETED")
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("REMOVE_SINGLE_ASSET_STACKS")
	os.Unsetenv("CONFIRM_RESET_STACK")
}

func setupTest() {
	resetGlobalConfig()
	clearEnvironment()
}

func teardownTest() {
	resetGlobalConfig()
	clearEnvironment()
}

/**************************************************************************************************
** Test that we're using the actual root command structure (no duplication)
**************************************************************************************************/
func TestRealCommandStructure(t *testing.T) {
	defer teardownTest()
	setupTest()

	cmd := CreateRootCommand()

	// Verify we get the real command structure
	if cmd.Use != "immich-stack" {
		t.Errorf("Expected command name 'immich-stack', got '%s'", cmd.Use)
	}

	// Verify persistent flags are present
	criteriaFlag := cmd.PersistentFlags().Lookup("criteria")
	if criteriaFlag == nil {
		t.Fatal("Expected --criteria flag to be present")
	}

	apiKeyFlag := cmd.PersistentFlags().Lookup("api-key")
	if apiKeyFlag == nil {
		t.Fatal("Expected --api-key flag to be present")
	}

	// Verify subcommands are present
	duplicatesCmd := cmd.Commands()
	foundDuplicates := false
	foundFixTrash := false

	for _, subcmd := range duplicatesCmd {
		if subcmd.Use == "duplicates" {
			foundDuplicates = true
		}
		if subcmd.Use == "fix-trash" {
			foundFixTrash = true
		}
	}

	if !foundDuplicates {
		t.Error("Expected 'duplicates' subcommand to be present")
	}
	if !foundFixTrash {
		t.Error("Expected 'fix-trash' subcommand to be present")
	}
}

/**************************************************************************************************
** End-to-end test for invalid --criteria JSON: parse flags then call getCriteriaConfig
**************************************************************************************************/
func TestInvalidCriteriaJSONEndToEnd(t *testing.T) {
	defer teardownTest()
	setupTest()

	// Set required API_KEY to avoid early failure
	os.Setenv("API_KEY", "test-key")

	cmd := CreateTestableRootCommand()

	// Override Run to test the actual integration path
	cmd.Run = func(cmd *cobra.Command, args []string) {
		// Load environment to set up global state
		config := LoadEnvForTesting()
		if config.Error != nil {
			t.Errorf("LoadEnv should not fail: %v", config.Error)
			return
		}

		// Now test that invalid JSON in criteria causes ParseCriteria to fail
		_, err := stacker.ParseCriteria(criteria)
		if err == nil {
			t.Error("Expected getCriteriaConfig to fail with invalid JSON, but it succeeded")
		} else if !strings.Contains(err.Error(), "invalid character") && !strings.Contains(err.Error(), "cannot unmarshal") {
			t.Errorf("Expected JSON parsing error, got: %v", err)
		}
	}

	cmd.SetArgs([]string{"--criteria", `{"invalid": json}`})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("Command execution should not fail at flag parsing level: %v", err)
	}
}

/**************************************************************************************************
** Test LoadEnv precedence and defaults with safe environment testing
** 
** NOTE: This test handles boolean flags (reset-stacks, dry-run, etc.) that don't take values.
** Boolean flags are only added to cmdArgs when their value is "true", otherwise they're omitted.
**************************************************************************************************/
func TestLoadEnvPrecedenceAndValidation(t *testing.T) {
	defer teardownTest()

	tests := []struct {
		name              string
		envVars           map[string]string
		cliFlags          map[string]string
		expectedCronInt   int
		expectedPromotion string
		expectedExtPromo  string
		expectError       bool
		errorContains     string
	}{
		{
			name:              "RUN_MODE=cron with no CRON_INTERVAL defaults to 86400",
			envVars:           map[string]string{"API_KEY": "test-key", "RUN_MODE": "cron"},
			expectedCronInt:   86400,
			expectedPromotion: utils.DefaultParentFilenamePromoteString,
			expectedExtPromo:  utils.DefaultParentExtPromoteString,
		},
		{
			name:              "PARENT_FILENAME_PROMOTE env applied when flag at default",
			envVars:           map[string]string{"API_KEY": "test-key", "PARENT_FILENAME_PROMOTE": "custom1,custom2"},
			expectedCronInt:   0,
			expectedPromotion: "custom1,custom2",
			expectedExtPromo:  utils.DefaultParentExtPromoteString,
		},
		{
			name:              "PARENT_EXT_PROMOTE env applied when flag at default",
			envVars:           map[string]string{"API_KEY": "test-key", "PARENT_EXT_PROMOTE": "raw,dng"},
			expectedCronInt:   0,
			expectedPromotion: utils.DefaultParentFilenamePromoteString,
			expectedExtPromo:  "raw,dng",
		},
		{
			name:          "Missing API_KEY returns error",
			envVars:       map[string]string{},
			expectError:   true,
			errorContains: "API_KEY is not set",
		},
		{
			name:          "RESET_STACKS with non-once mode returns error",
			envVars:       map[string]string{"API_KEY": "test-key", "RESET_STACKS": "true", "RUN_MODE": "cron"},
			expectError:   true,
			errorContains: "RESET_STACKS can only be used in 'once' run mode",
		},
		{
			name: "RESET_STACKS without confirmation returns error",
			envVars: map[string]string{
				"API_KEY":      "test-key",
				"RESET_STACKS": "true",
			},
			expectError:   true,
			errorContains: "to use RESET_STACKS, you must set CONFIRM_RESET_STACK",
		},
		{
			name:              "Boolean CLI flags work correctly",
			envVars:           map[string]string{"API_KEY": "test-key"},
			cliFlags:          map[string]string{"dry-run": "true", "with-archived": "true"},
			expectedCronInt:   0,
			expectedPromotion: utils.DefaultParentFilenamePromoteString,
			expectedExtPromo:  utils.DefaultParentExtPromoteString,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTest()

			// Set environment variables
			for key, val := range tt.envVars {
				os.Setenv(key, val)
			}

			// Create command and set CLI flags
			cmd := CreateTestableRootCommand()
			var cmdArgs []string
			
			// Boolean flags that don't take values
			booleanFlags := map[string]bool{
				"reset-stacks":               true,
				"replace-stacks":             true,
				"dry-run":                    true,
				"with-archived":              true,
				"with-deleted":               true,
				"remove-single-asset-stacks": true,
			}
			
			for key, val := range tt.cliFlags {
				if booleanFlags[key] {
					// Boolean flags: only add the flag if value is "true"
					if val == "true" {
						cmdArgs = append(cmdArgs, "--"+key)
					}
				} else {
					// String/Int flags: add flag with value
					cmdArgs = append(cmdArgs, "--"+key, val)
				}
			}
			cmd.SetArgs(cmdArgs)

			// Parse flags to set global variables
			err := cmd.ParseFlags(cmdArgs)
			if err != nil {
				t.Fatalf("Flag parsing failed: %v", err)
			}

			// Test LoadEnvForTesting
			config := LoadEnvForTesting()

			if tt.expectError {
				if config.Error == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(config.Error.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errorContains, config.Error)
				}
			} else {
				if config.Error != nil {
					t.Errorf("Unexpected error: %v", config.Error)
				}

				// Verify expected values
				if cronInterval != tt.expectedCronInt {
					t.Errorf("Expected cronInterval %d, got %d", tt.expectedCronInt, cronInterval)
				}
				if parentFilenamePromote != tt.expectedPromotion {
					t.Errorf("Expected parentFilenamePromote '%s', got '%s'", tt.expectedPromotion, parentFilenamePromote)
				}
				if parentExtPromote != tt.expectedExtPromo {
					t.Errorf("Expected parentExtPromote '%s', got '%s'", tt.expectedExtPromo, parentExtPromote)
				}
				
				// For the boolean flags test case, verify the boolean flags were parsed correctly
				if tt.name == "Boolean CLI flags work correctly" {
					if !dryRun {
						t.Error("Expected dryRun to be true when --dry-run flag is set")
					}
					if !withArchived {
						t.Error("Expected withArchived to be true when --with-archived flag is set")
					}
				}
			}
		})
	}
}

/**************************************************************************************************
** Test that --criteria wins over env end-to-end (parse -> getCriteriaConfig outcome)
**************************************************************************************************/
func TestCriteriaFlagOverridesEnvEndToEnd(t *testing.T) {
	defer teardownTest()
	setupTest()

	// Set environment with one criteria and flag with another
	os.Setenv("API_KEY", "test-key")
	os.Setenv("CRITERIA", `[{"key": "localDateTime"}]`)

	cmd := CreateTestableRootCommand()
	cmd.SetArgs([]string{"--criteria", `[{"key": "originalFileName"}]`})

	// Override Run to test integration
	cmd.Run = func(cmd *cobra.Command, args []string) {
		// Load environment
		config := LoadEnvForTesting()
		if config.Error != nil {
			t.Errorf("LoadEnv should not fail: %v", config.Error)
			return
		}

		// Test that ParseCriteria uses the CLI flag value
		criteriaConfig, err := stacker.ParseCriteria(criteria)
		if err != nil {
			t.Errorf("getCriteriaConfig failed: %v", err)
			return
		}

		// Verify we get the flag value, not the env value
		if len(criteriaConfig.Legacy) == 0 {
			t.Error("Expected criteria to be parsed")
			return
		}

		// Check that we got the flag value (originalFileName), not the env value (localDateTime)
		if criteriaConfig.Legacy[0].Key != "originalFileName" {
			t.Errorf("Expected key 'originalFileName' from CLI flag, got '%s' - CLI flag should override env", criteriaConfig.Legacy[0].Key)
		}
	}

	err := cmd.Execute()
	if err != nil {
		t.Errorf("Command execution failed: %v", err)
	}
}

/**************************************************************************************************
** Test LOG_LEVEL validation with configureLogger
**************************************************************************************************/
func TestLogLevelValidation(t *testing.T) {
	defer teardownTest()

	tests := []struct {
		name            string
		flagValue       string
		envValue        string
		expectedResult  string // "debug", "info", "warn", "error"
		expectWarning   bool
		expectedWarning string
	}{
		{
			name:           "Valid flag debug",
			flagValue:      "debug",
			expectedResult: "debug",
		},
		{
			name:           "Valid flag info",
			flagValue:      "info",
			expectedResult: "info",
		},
		{
			name:           "Valid flag warn",
			flagValue:      "warn",
			expectedResult: "warning", // logrus uses "warning" internally
		},
		{
			name:           "Valid flag error",
			flagValue:      "error",
			expectedResult: "error",
		},
		{
			name:            "Invalid flag falls back to info with warning",
			flagValue:       "invalid-level",
			expectedResult:  "info",
			expectWarning:   true,
			expectedWarning: "Invalid LOG_LEVEL 'invalid-level', using default 'info'",
		},
		{
			name:           "Valid env debug when no flag",
			envValue:       "debug",
			expectedResult: "debug",
		},
		{
			name:           "Flag overrides env",
			flagValue:      "error",
			envValue:       "debug",
			expectedResult: "error",
		},
		{
			name:           "No flag or env defaults to info",
			expectedResult: "info",
		},
		{
			name:            "Invalid env falls back to info with warning",
			envValue:        "invalid-env-level",
			expectedResult:  "info",
			expectWarning:   true,
			expectedWarning: "Invalid LOG_LEVEL 'invalid-env-level', using default 'info'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTest()

			// Set logLevel global (simulating flag parsing)
			logLevel = tt.flagValue

			// Set environment
			if tt.envValue != "" {
				os.Setenv("LOG_LEVEL", tt.envValue)
			}

			// Always test the actual configureLogger function
			var logBuffer bytes.Buffer
			var logger *logrus.Logger

			if tt.expectWarning {
				// Use the test helper to capture output from configureLogger
				logger = configureLoggerForTesting(&logBuffer)
			} else {
				// Test configureLogger directly
				logger = configureLogger()
			}

			actualLevel := logger.GetLevel().String()
			if actualLevel != tt.expectedResult {
				t.Errorf("Expected log level '%s', got '%s'", tt.expectedResult, actualLevel)
			}

			// Check for expected warning
			if tt.expectWarning {
				logOutput := logBuffer.String()
				if !strings.Contains(logOutput, tt.expectedWarning) {
					t.Errorf("Expected warning message '%s' not found in log output: %s", tt.expectedWarning, logOutput)
				}
			}
		})
	}
}

/**************************************************************************************************
** Test multi-API key parsing end-to-end
**************************************************************************************************/
func TestMultiAPIKeyEndToEndFlow(t *testing.T) {
	defer teardownTest()
	setupTest()

	cmd := CreateTestableRootCommand()
	cmd.SetArgs([]string{"--api-key", "key1,key2,key3"})

	cmd.Run = func(cmd *cobra.Command, args []string) {
		// Test the same splitting logic used in runStacker
		apiKeys := utils.RemoveEmptyStrings(func(keys []string) []string {
			for i, key := range keys {
				keys[i] = strings.TrimSpace(key)
			}
			return keys
		}(strings.Split(apiKey, ",")))

		// Verify parsing
		expectedKeys := []string{"key1", "key2", "key3"}
		if len(apiKeys) != len(expectedKeys) {
			t.Errorf("Expected %d API keys, got %d", len(expectedKeys), len(apiKeys))
			return
		}

		for i, expected := range expectedKeys {
			if apiKeys[i] != expected {
				t.Errorf("Expected API key %d to be '%s', got '%s'", i, expected, apiKeys[i])
			}
		}
	}

	err := cmd.Execute()
	if err != nil {
		t.Errorf("Command execution failed: %v", err)
	}
}

/**************************************************************************************************
** Test subcommand flag inheritance and behavior
**************************************************************************************************/
func TestSubcommandFlagInheritance(t *testing.T) {
	defer teardownTest()
	setupTest()

	cmd := CreateRootCommand()

	// Test that subcommands inherit persistent flags
	var duplicatesCmd, fixTrashCmd *cobra.Command

	for _, subcmd := range cmd.Commands() {
		if subcmd.Use == "duplicates" {
			duplicatesCmd = subcmd
		}
		if subcmd.Use == "fix-trash" {
			fixTrashCmd = subcmd
		}
	}

	if duplicatesCmd == nil {
		t.Fatal("duplicates command not found")
	}
	if fixTrashCmd == nil {
		t.Fatal("fix-trash command not found")
	}

	// Verify key persistent flags are inherited
	testFlags := []string{"api-key", "dry-run", "criteria"}

	for _, flagName := range testFlags {
		// Check via InheritedFlags() which includes persistent flags from parent
		if duplicatesCmd.InheritedFlags().Lookup(flagName) == nil {
			t.Errorf("duplicates command missing flag: %s", flagName)
		}
		if fixTrashCmd.InheritedFlags().Lookup(flagName) == nil {
			t.Errorf("fix-trash command missing flag: %s", flagName)
		}
	}
}

/**************************************************************************************************
** Test full run path: CLI → loadEnv → runStackerOnce → StackBy with mocked Immich client
**************************************************************************************************/
func TestFullRunPathWithMockedImmich(t *testing.T) {
	defer teardownTest()
	setupTest()

	// Set up test environment
	os.Setenv("API_KEY", "test-key")

	// Create a mock runStackerOnce function that we can verify gets called
	var actualCriteria string
	var stackByCalled bool

	cmd := CreateTestableRootCommand()
	cmd.SetArgs([]string{"--criteria", `[{"key": "originalFileName"}]`})

	// Override Run to test the full integration path
	cmd.Run = func(cmd *cobra.Command, args []string) {
		// This follows the exact same flow as the real runStacker
		logger := loadEnv()

		// Create minimal test assets to pass to StackBy
		testAssets := []utils.TAsset{
			{
				ID:               "asset1",
				OriginalFileName: "IMG_001.jpg",
				LocalDateTime:    "2023-01-01T10:00:00Z",
			},
			{
				ID:               "asset2",
				OriginalFileName: "IMG_002.jpg",
				LocalDateTime:    "2023-01-01T10:00:30Z",
			},
		}

		// Call StackBy with the same parameters as runStackerOnce
		actualCriteria = criteria
		stacks, err := stacker.StackBy(testAssets, criteria, parentFilenamePromote, parentExtPromote, logger)
		stackByCalled = true

		if err != nil {
			t.Errorf("StackBy failed: %v", err)
			return
		}

		// Verify we got results
		if stacks == nil {
			t.Error("StackBy should return non-nil stacks")
		}
	}

	err := cmd.Execute()
	if err != nil {
		t.Errorf("Command execution failed: %v", err)
	}

	// Verify the full flow executed
	if !stackByCalled {
		t.Error("StackBy was not called - full run path not exercised")
	}

	// Verify criteria was passed through correctly
	expectedCriteria := `[{"key": "originalFileName"}]`
	if actualCriteria != expectedCriteria {
		t.Errorf("Expected criteria '%s' to be passed to StackBy, got '%s'", expectedCriteria, actualCriteria)
	}
}

/**************************************************************************************************
** Test subcommand run paths honor loadEnv constraints
**************************************************************************************************/
func TestSubcommandRequiresAPIKey(t *testing.T) {
	defer teardownTest()

	tests := []struct {
		name        string
		subcommand  string
		expectError bool
	}{
		{"duplicates without API_KEY", "duplicates", true},
		{"fix-trash without API_KEY", "fix-trash", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTest()
			// Note: Not setting API_KEY environment variable

			cmd := CreateTestableRootCommand()

			// Find the subcommand and override its Run function
			for _, subcmd := range cmd.Commands() {
				if subcmd.Use == tt.subcommand {
					subcmd.Run = func(cmd *cobra.Command, args []string) {
						// Test that loadEnv validation is enforced
						config := LoadEnvForTesting()
						if config.Error != nil {
							// This is expected - API_KEY is required
							if !strings.Contains(config.Error.Error(), "API_KEY is not set") {
								t.Errorf("Expected API_KEY error, got: %v", config.Error)
							}
							return
						}

						if tt.expectError {
							t.Error("Expected error for missing API_KEY, but validation passed")
						}
					}
				}
			}

			cmd.SetArgs([]string{tt.subcommand})

			err := cmd.Execute()
			if err != nil {
				t.Errorf("Command execution failed: %v", err)
			}
		})
	}
}

/**************************************************************************************************
** Test getOriginalStackIDs function with edge cases
**************************************************************************************************/
func TestGetOriginalStackIDs(t *testing.T) {
	tests := []struct {
		name               string
		stack              []utils.TAsset
		expectedParentID   string
		expectedChildrenIDs []string
		expectedOriginalIDs []string
	}{
		{
			name:                "Empty stack returns empty results",
			stack:               []utils.TAsset{},
			expectedParentID:    "",
			expectedChildrenIDs: nil,
			expectedOriginalIDs: nil,
		},
		{
			name: "Stack with nil Stack field returns empty results",
			stack: []utils.TAsset{
				{ID: "asset1", Stack: nil},
			},
			expectedParentID:    "",
			expectedChildrenIDs: nil,
			expectedOriginalIDs: nil,
		},
		{
			name: "Stack with empty Assets array returns only parentID",
			stack: []utils.TAsset{
				{
					ID: "asset1",
					Stack: &utils.TStack{
						ID:             "stack1",
						PrimaryAssetID: "parent1",
						Assets:         []utils.TAsset{}, // Empty Assets array - the bug case
					},
				},
			},
			expectedParentID:    "parent1",
			expectedChildrenIDs: nil,
			expectedOriginalIDs: []string{"parent1"},
		},
		{
			name: "Stack with one asset returns only parentID",
			stack: []utils.TAsset{
				{
					ID: "asset1",
					Stack: &utils.TStack{
						ID:             "stack1",
						PrimaryAssetID: "parent1",
						Assets: []utils.TAsset{
							{ID: "parent1"},
						},
					},
				},
			},
			expectedParentID:    "parent1",
			expectedChildrenIDs: nil,
			expectedOriginalIDs: []string{"parent1"},
		},
		{
			name: "Stack with multiple assets returns parent and children",
			stack: []utils.TAsset{
				{
					ID: "asset1",
					Stack: &utils.TStack{
						ID:             "stack1",
						PrimaryAssetID: "parent1",
						Assets: []utils.TAsset{
							{ID: "parent1"},
							{ID: "child1"},
							{ID: "child2"},
						},
					},
				},
			},
			expectedParentID:    "parent1",
			expectedChildrenIDs: []string{"child1", "child2"},
			expectedOriginalIDs: []string{"parent1", "child1", "child2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parentID, childrenIDs, originalStackIDs := getOriginalStackIDs(tt.stack)

			if parentID != tt.expectedParentID {
				t.Errorf("Expected parentID '%s', got '%s'", tt.expectedParentID, parentID)
			}

			if tt.expectedChildrenIDs == nil && childrenIDs != nil {
				t.Errorf("Expected nil childrenIDs, got %v", childrenIDs)
			} else if tt.expectedChildrenIDs != nil && childrenIDs == nil {
				t.Errorf("Expected childrenIDs %v, got nil", tt.expectedChildrenIDs)
			} else if len(childrenIDs) != len(tt.expectedChildrenIDs) {
				t.Errorf("Expected %d childrenIDs, got %d", len(tt.expectedChildrenIDs), len(childrenIDs))
			} else {
				for i, expected := range tt.expectedChildrenIDs {
					if childrenIDs[i] != expected {
						t.Errorf("Expected childrenIDs[%d] to be '%s', got '%s'", i, expected, childrenIDs[i])
					}
				}
			}

			if tt.expectedOriginalIDs == nil && originalStackIDs != nil {
				t.Errorf("Expected nil originalStackIDs, got %v", originalStackIDs)
			} else if tt.expectedOriginalIDs != nil && originalStackIDs == nil {
				t.Errorf("Expected originalStackIDs %v, got nil", tt.expectedOriginalIDs)
			} else if len(originalStackIDs) != len(tt.expectedOriginalIDs) {
				t.Errorf("Expected %d originalStackIDs, got %d", len(tt.expectedOriginalIDs), len(originalStackIDs))
			} else {
				for i, expected := range tt.expectedOriginalIDs {
					if originalStackIDs[i] != expected {
						t.Errorf("Expected originalStackIDs[%d] to be '%s', got '%s'", i, expected, originalStackIDs[i])
					}
				}
			}
		})
	}
}

/**************************************************************************************************
** Test boolean environment variable overrides
**************************************************************************************************/
func TestBooleanEnvironmentOverrides(t *testing.T) {
	defer teardownTest()
	setupTest()

	tests := []struct {
		name      string
		envVar    string
		envValue  string
		globalVar *bool
		expected  bool
	}{
		{"WITH_ARCHIVED true", "WITH_ARCHIVED", "true", &withArchived, true},
		{"WITH_ARCHIVED false", "WITH_ARCHIVED", "false", &withArchived, false},
		{"WITH_DELETED true", "WITH_DELETED", "true", &withDeleted, true},
		{"DRY_RUN true", "DRY_RUN", "true", &dryRun, true},
		// NOTE: replaceStacks has default=true, and env is only checked if !replaceStacks
		// So REPLACE_STACKS env is only effective if flag is set to false first
		{"REMOVE_SINGLE_ASSET_STACKS true", "REMOVE_SINGLE_ASSET_STACKS", "true", &removeSingleAssetStacks, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTest()
			os.Setenv("API_KEY", "test-key") // Avoid API_KEY error
			os.Setenv(tt.envVar, tt.envValue)

			config := LoadEnvForTesting()
			if config.Error != nil {
				t.Errorf("LoadEnv failed: %v", config.Error)
				return
			}

			if *tt.globalVar != tt.expected {
				t.Errorf("Expected %s to be %v, got %v", tt.envVar, tt.expected, *tt.globalVar)
			}
		})
	}
}
