package main

import (
	"bytes"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestStartupConfigurationSummary(t *testing.T) {
	tests := []struct {
		name      string
		envVars   map[string]string
		wantInLog []string
	}{
		{
			name: "text format with basic config",
			envVars: map[string]string{
				"API_KEY":    "test-key",
				"RUN_MODE":   "once",
				"LOG_LEVEL":  "info",
				"LOG_FORMAT": "text",
				"DRY_RUN":    "true",
			},
			wantInLog: []string{
				"Starting with config:",
				"mode=once",
				"level=info",
				"format=text",
				"dry-run=true",
			},
		},
		{
			name: "json format with all flags",
			envVars: map[string]string{
				"API_KEY":                    "test-key",
				"RUN_MODE":                   "cron",
				"CRON_INTERVAL":              "3600",
				"LOG_LEVEL":                  "debug",
				"LOG_FORMAT":                 "json",
				"DRY_RUN":                    "true",
				"REPLACE_STACKS":             "true",
				"WITH_ARCHIVED":              "true",
				"WITH_DELETED":               "true",
				"REMOVE_SINGLE_ASSET_STACKS": "true",
			},
			wantInLog: []string{
				"Configuration loaded",
				`"runMode":"cron"`,
				`"cronInterval":3600`,
				`"logLevel":"debug"`,
				`"logFormat":"json"`,
				`"dryRun":true`,
				`"replaceStacks":true`,
				`"withArchived":true`,
				`"withDeleted":true`,
				`"removeSingleAssetStacks":true`,
			},
		},
		{
			name: "text format with filter fields",
			envVars: map[string]string{
				"API_KEY":             "test-key",
				"FILTER_ALBUM_IDS":    "album1,album2",
				"FILTER_TAKEN_AFTER":  "2024-01-01T00:00:00Z",
				"FILTER_TAKEN_BEFORE": "2024-12-31T23:59:59Z",
			},
			wantInLog: []string{
				"Starting with config:",
				"filter-albums=2",
				"filter-after=2024-01-01T00:00:00Z",
				"filter-before=2024-12-31T23:59:59Z",
			},
		},
		{
			name: "json format with filter fields",
			envVars: map[string]string{
				"API_KEY":             "test-key",
				"LOG_FORMAT":          "json",
				"FILTER_ALBUM_IDS":    "album1,album2,album3",
				"FILTER_TAKEN_AFTER":  "2024-06-01T00:00:00Z",
				"FILTER_TAKEN_BEFORE": "2024-06-30T23:59:59Z",
			},
			wantInLog: []string{
				"Configuration loaded",
				`"filterAlbumIDs":["album1","album2","album3"]`,
				`"filterTakenAfter":"2024-06-01T00:00:00Z"`,
				`"filterTakenBefore":"2024-06-30T23:59:59Z"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			resetTestEnv()

			// Set test environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}
			defer resetTestEnv()

			// Capture log output
			var buf bytes.Buffer
			config := LoadEnvForTesting()

			// Verify no error
			assert.NoError(t, config.Error)
			assert.NotNil(t, config.Logger)

			// Set output to buffer for testing
			config.Logger.SetOutput(&buf)

			// Trigger startup summary
			logStartupSummary(config.Logger)

			logOutput := buf.String()

			// Check that all expected strings are in the log
			for _, want := range tt.wantInLog {
				assert.Contains(t, logOutput, want, "Log should contain: %s", want)
			}
		})
	}
}

func TestResetStacksConfiguration(t *testing.T) {
	tests := []struct {
		name          string
		envVars       map[string]string
		wantError     bool
		errorContains string
	}{
		{
			name: "reset stacks with proper confirmation",
			envVars: map[string]string{
				"API_KEY":             "test-key",
				"RESET_STACKS":        "true",
				"RUN_MODE":            "once",
				"CONFIRM_RESET_STACK": "I acknowledge all my current stacks will be deleted and new one will be created",
			},
			wantError: false,
		},
		{
			name: "reset stacks with wrong run mode",
			envVars: map[string]string{
				"API_KEY":             "test-key",
				"RESET_STACKS":        "true",
				"RUN_MODE":            "cron",
				"CONFIRM_RESET_STACK": "I acknowledge all my current stacks will be deleted and new one will be created",
			},
			wantError:     true,
			errorContains: "RESET_STACKS can only be used in 'once' run mode",
		},
		{
			name: "reset stacks without confirmation",
			envVars: map[string]string{
				"API_KEY":      "test-key",
				"RESET_STACKS": "true",
				"RUN_MODE":     "once",
			},
			wantError:     true,
			errorContains: "to use RESET_STACKS, you must set CONFIRM_RESET_STACK",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			resetTestEnv()

			// Set test environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}
			defer resetTestEnv()

			config := LoadEnvForTesting()

			if tt.wantError {
				assert.Error(t, config.Error)
				if tt.errorContains != "" {
					assert.Contains(t, config.Error.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, config.Error)
				assert.NotNil(t, config.Logger)
				// Verify resetStacks was set correctly
				assert.True(t, resetStacks, "RESET_STACKS should be enabled")
			}
		})
	}
}

func TestFileLogging(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	logFile := tmpDir + "/test.log"

	tests := []struct {
		name       string
		envVars    map[string]string
		expectFile bool
		checkInLog string
	}{
		{
			name: "file logging enabled",
			envVars: map[string]string{
				"LOG_FILE": logFile,
			},
			expectFile: true,
			checkInLog: "Test message",
		},
		{
			name:       "file logging disabled",
			envVars:    map[string]string{},
			expectFile: false,
			checkInLog: "Test message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			resetTestEnv()

			// Set test environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}
			defer resetTestEnv()

			// Configure logger - note that configureLoggerWithOutput with nil
			// will read LOG_FILE from environment
			logger := configureLogger()

			// Log a test message
			logger.Info("Test message")

			if tt.expectFile {
				// Check if file was created and contains the message
				content, err := os.ReadFile(logFile)
				assert.NoError(t, err, "Log file should be readable")
				assert.Contains(t, string(content), tt.checkInLog, "Log file should contain test message")

				// Clean up
				os.Remove(logFile)
			}
		})
	}
}

func TestFileLoggingPermissionFallback(t *testing.T) {
	// This test verifies that the fallback mechanism works correctly
	// The actual warning logs go to the initial stdout before redirection,
	// so we test the behavior rather than the log output

	tests := []struct {
		name        string
		logFile     string
		description string
	}{
		{
			name:        "invalid directory permissions",
			logFile:     "/root/cannot-write-here/test.log",
			description: "Should gracefully handle unwritable directory",
		},
		{
			name:        "invalid file path",
			logFile:     "/dev/null/not-a-file.log",
			description: "Should gracefully handle invalid file path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			resetTestEnv()

			// Set LOG_FILE to an unwritable location
			os.Setenv("LOG_FILE", tt.logFile)
			defer resetTestEnv()

			// Configure logger - should fall back to stdout
			logger := configureLogger()

			// Logger should still be functional after fallback
			// We can't capture the initial warning but we can verify the logger works
			assert.NotNil(t, logger, "Logger should be created even with invalid LOG_FILE")

			// Test that logger is functional
			var buf bytes.Buffer
			logger.SetOutput(&buf)
			logger.Info("Test after fallback")
			assert.Contains(t, buf.String(), "Test after fallback", "Logger should work after fallback")

			// Verify that the log file was NOT created
			_, err := os.Stat(tt.logFile)
			assert.Error(t, err, "Log file should not exist when path is invalid")
		})
	}
}

func TestLogLevelConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		envLevel    string
		flagLevel   string
		expectLevel logrus.Level
	}{
		{
			name:        "default level",
			envLevel:    "",
			flagLevel:   "",
			expectLevel: logrus.InfoLevel,
		},
		{
			name:        "env variable set",
			envLevel:    "debug",
			flagLevel:   "",
			expectLevel: logrus.DebugLevel,
		},
		{
			name:        "flag overrides env",
			envLevel:    "debug",
			flagLevel:   "warn",
			expectLevel: logrus.WarnLevel,
		},
		{
			name:        "invalid level defaults to info",
			envLevel:    "invalid",
			flagLevel:   "",
			expectLevel: logrus.InfoLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			resetTestEnv()

			// Set test environment variables
			if tt.envLevel != "" {
				os.Setenv("LOG_LEVEL", tt.envLevel)
			}

			// Set flag value
			logLevel = tt.flagLevel

			defer func() {
				resetTestEnv()
				logLevel = ""
			}()

			// Configure logger
			logger := configureLogger()

			// Check log level
			assert.Equal(t, tt.expectLevel, logger.GetLevel())
		})
	}
}

// Helper function to reset test environment
func resetTestEnv() {
	envVars := []string{
		"API_KEY", "API_URL", "RUN_MODE", "CRON_INTERVAL",
		"LOG_LEVEL", "LOG_FORMAT", "LOG_FILE",
		"DRY_RUN", "RESET_STACKS", "CONFIRM_RESET_STACK",
		"REPLACE_STACKS", "WITH_ARCHIVED", "WITH_DELETED",
		"REMOVE_SINGLE_ASSET_STACKS", "CRITERIA",
		"PARENT_FILENAME_PROMOTE", "PARENT_EXT_PROMOTE",
		"FILTER_ALBUM_IDS", "FILTER_TAKEN_AFTER", "FILTER_TAKEN_BEFORE",
	}

	for _, env := range envVars {
		os.Unsetenv(env)
	}

	// Reset global variables
	apiKey = ""
	apiURL = ""
	criteria = ""
	parentFilenamePromote = ""
	parentExtPromote = ""
	runMode = ""
	cronInterval = 0
	withArchived = false
	resetStacks = false
	dryRun = false
	replaceStacks = false
	withDeleted = false
	logLevel = ""
	removeSingleAssetStacks = false
	filterAlbumIDs = nil
	filterTakenAfter = ""
	filterTakenBefore = ""
}

/************************************************************************************************
** Tests for FILTER_ALBUM_IDS environment variable parsing with whitespace handling
************************************************************************************************/

func TestFilterAlbumIDsParsing(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected []string
	}{
		{
			name:     "simple list",
			envValue: "album1,album2",
			expected: []string{"album1", "album2"},
		},
		{
			name:     "with spaces after comma",
			envValue: "album1, album2, album3",
			expected: []string{"album1", "album2", "album3"},
		},
		{
			name:     "with leading and trailing spaces",
			envValue: " album1 , album2 ",
			expected: []string{"album1", "album2"},
		},
		{
			name:     "empty entries filtered",
			envValue: "album1,,album2",
			expected: []string{"album1", "album2"},
		},
		{
			name:     "only spaces filtered",
			envValue: "album1,   ,album2",
			expected: []string{"album1", "album2"},
		},
		{
			name:     "single album",
			envValue: "album1",
			expected: []string{"album1"},
		},
		{
			name:     "single album with spaces",
			envValue: "  album1  ",
			expected: []string{"album1"},
		},
		{
			name:     "UUIDs with spaces",
			envValue: "550e8400-e29b-41d4-a716-446655440000 , 660e8400-e29b-41d4-a716-446655440001",
			expected: []string{"550e8400-e29b-41d4-a716-446655440000", "660e8400-e29b-41d4-a716-446655440001"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			resetTestEnv()

			// Set required env vars
			os.Setenv("API_KEY", "test-key")
			os.Setenv("FILTER_ALBUM_IDS", tt.envValue)
			defer resetTestEnv()

			// Load config
			config := LoadEnvForTesting()

			// Assert
			assert.NoError(t, config.Error)
			assert.Equal(t, tt.expected, filterAlbumIDs, "filterAlbumIDs should be correctly parsed and trimmed")
		})
	}
}

func TestFilterAlbumIDsEmptyEnv(t *testing.T) {
	// Clear environment
	resetTestEnv()

	// Set required env vars but NOT FILTER_ALBUM_IDS
	os.Setenv("API_KEY", "test-key")
	defer resetTestEnv()

	// Load config
	config := LoadEnvForTesting()

	// Assert
	assert.NoError(t, config.Error)
	assert.Nil(t, filterAlbumIDs, "filterAlbumIDs should be nil when env var is not set")
}

/************************************************************************************************
** Tests for date filter environment variable parsing with TrimSpace
************************************************************************************************/

func TestDateFilterEnvVarParsing(t *testing.T) {
	tests := []struct {
		name           string
		envAfter       string
		envBefore      string
		expectedAfter  string
		expectedBefore string
	}{
		{
			name:           "valid dates without spaces",
			envAfter:       "2024-01-01T00:00:00Z",
			envBefore:      "2024-12-31T23:59:59Z",
			expectedAfter:  "2024-01-01T00:00:00Z",
			expectedBefore: "2024-12-31T23:59:59Z",
		},
		{
			name:           "with leading space on after",
			envAfter:       " 2024-01-01T00:00:00Z",
			envBefore:      "",
			expectedAfter:  "2024-01-01T00:00:00Z",
			expectedBefore: "",
		},
		{
			name:           "with trailing space on after",
			envAfter:       "2024-01-01T00:00:00Z ",
			envBefore:      "",
			expectedAfter:  "2024-01-01T00:00:00Z",
			expectedBefore: "",
		},
		{
			name:           "with leading space on before",
			envAfter:       "",
			envBefore:      " 2024-12-31T23:59:59Z",
			expectedAfter:  "",
			expectedBefore: "2024-12-31T23:59:59Z",
		},
		{
			name:           "with trailing space on before",
			envAfter:       "",
			envBefore:      "2024-12-31T23:59:59Z ",
			expectedAfter:  "",
			expectedBefore: "2024-12-31T23:59:59Z",
		},
		{
			name:           "both with leading and trailing spaces",
			envAfter:       "  2024-01-01T00:00:00Z  ",
			envBefore:      "  2024-12-31T23:59:59Z  ",
			expectedAfter:  "2024-01-01T00:00:00Z",
			expectedBefore: "2024-12-31T23:59:59Z",
		},
		{
			name:           "empty values",
			envAfter:       "",
			envBefore:      "",
			expectedAfter:  "",
			expectedBefore: "",
		},
		{
			name:           "only whitespace becomes empty",
			envAfter:       "   ",
			envBefore:      "   ",
			expectedAfter:  "",
			expectedBefore: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			resetTestEnv()

			// Set required env vars
			os.Setenv("API_KEY", "test-key")
			if tt.envAfter != "" {
				os.Setenv("FILTER_TAKEN_AFTER", tt.envAfter)
			}
			if tt.envBefore != "" {
				os.Setenv("FILTER_TAKEN_BEFORE", tt.envBefore)
			}
			defer resetTestEnv()

			// Load config
			config := LoadEnvForTesting()

			// Assert
			assert.NoError(t, config.Error)
			assert.Equal(t, tt.expectedAfter, filterTakenAfter, "filterTakenAfter should be trimmed")
			assert.Equal(t, tt.expectedBefore, filterTakenBefore, "filterTakenBefore should be trimmed")
		})
	}
}
