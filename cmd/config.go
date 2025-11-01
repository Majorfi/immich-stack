/**************************************************************************************************
** Configuration and environment management for the Immich CLI application.
** Handles logger configuration, environment variable loading, and global configuration state.
**************************************************************************************************/

package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/majorfi/immich-stack/pkg/utils"
	"github.com/sirupsen/logrus"
)

// Global configuration variables
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
var logLevel string
var removeSingleAssetStacks bool

/**************************************************************************************************
** Configures the logger based on command-line flags and environment variables. Sets up the
** log level and format. The --log-level flag takes precedence over the LOG_LEVEL environment
** variable.
**
** @return *logrus.Logger - Configured logger instance
**************************************************************************************************/
func configureLogger() *logrus.Logger {
	return configureLoggerWithOutput(nil)
}

/**************************************************************************************************
** configureLoggerWithOutput is like configureLogger but accepts an optional output writer
** for testing purposes. If output is nil, uses the default output.
**
** @param output - Optional output writer for testing
** @return *logrus.Logger - Configured logger instance
**************************************************************************************************/
func configureLoggerWithOutput(output io.Writer) *logrus.Logger {
	logger := logrus.New()

	// Set output - file logging if LOG_FILE is set, otherwise stdout
	if output != nil {
		// Testing mode - use provided output
		logger.SetOutput(output)
	} else if logFile := os.Getenv("LOG_FILE"); logFile != "" {
		// File logging enabled - write to both stdout and file
		if err := os.MkdirAll(utils.GetDir(logFile), 0755); err != nil {
			logger.Warnf("Failed to create log directory: %v, falling back to stdout only", err)
			logger.SetOutput(os.Stdout)
		} else {
			file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				logger.Warnf("Failed to open log file %s: %v, falling back to stdout only", logFile, err)
				logger.SetOutput(os.Stdout)
			} else {
				// Write to both stdout and file
				multiWriter := io.MultiWriter(os.Stdout, file)
				logger.SetOutput(multiWriter)
				logger.Infof("Logging to file: %s", logFile)
			}
		}
	} else {
		// Default to stdout only
		logger.SetOutput(os.Stdout)
	}

	// Set log level - flag takes precedence over environment variable
	level := logLevel
	if level == "" {
		level = os.Getenv("LOG_LEVEL")
	}

	if level != "" {
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
** LoadEnvConfig represents the result of environment loading, including any validation errors.
**************************************************************************************************/
type LoadEnvConfig struct {
	Logger *logrus.Logger
	Error  error
}

/**************************************************************************************************
** logStartupSummary logs a concise summary of the current configuration at startup.
** Shows the resolved configuration values for all major settings.
**
** @param logger - Logger instance to output the summary
**************************************************************************************************/
func logStartupSummary(logger *logrus.Logger) {
	// Build summary based on format
	if format := os.Getenv("LOG_FORMAT"); format == "json" {
		logger.WithFields(logrus.Fields{
			"runMode":                 runMode,
			"cronInterval":            cronInterval,
			"logLevel":                logger.GetLevel().String(),
			"logFormat":               "json",
			"logFile":                 os.Getenv("LOG_FILE"),
			"dryRun":                  dryRun,
			"replaceStacks":           replaceStacks,
			"resetStacks":             resetStacks,
			"withArchived":            withArchived,
			"withDeleted":             withDeleted,
			"removeSingleAssetStacks": removeSingleAssetStacks,
			"criteria":                criteria,
			"parentFilenamePromote":   parentFilenamePromote,
			"parentExtPromote":        parentExtPromote,
		}).Info("Configuration loaded")
	} else {
		// Build human-readable summary
		var summary []string
		summary = append(summary, fmt.Sprintf("mode=%s", runMode))
		if runMode == "cron" {
			summary = append(summary, fmt.Sprintf("interval=%ds", cronInterval))
		}
		summary = append(summary, fmt.Sprintf("level=%s", logger.GetLevel().String()))
		summary = append(summary, fmt.Sprintf("format=%s", "text"))
		if logFile := os.Getenv("LOG_FILE"); logFile != "" {
			summary = append(summary, fmt.Sprintf("file=%s", logFile))
		}
		if dryRun {
			summary = append(summary, "dry-run=true")
		}
		if replaceStacks {
			summary = append(summary, "replace=true")
		}
		if resetStacks {
			summary = append(summary, "reset=true")
		}
		if withArchived {
			summary = append(summary, "archived=true")
		}
		if withDeleted {
			summary = append(summary, "deleted=true")
		}
		if removeSingleAssetStacks {
			summary = append(summary, "remove-single=true")
		}
		if criteria != "" {
			summary = append(summary, fmt.Sprintf("criteria=%s", criteria))
		}

		logger.Infof("Starting with config: %s", strings.Join(summary, ", "))
	}
}

/**************************************************************************************************
** LoadEnvForTesting loads environment variables and validates configuration without calling Fatal().
** Returns errors instead of terminating, allowing tests to verify error conditions.
**
** @return LoadEnvConfig - Configuration result with logger and any validation error
**************************************************************************************************/
func LoadEnvForTesting() LoadEnvConfig {
	godotenv.Load()

	logger := configureLogger()
	if criteria == "" {
		criteria = os.Getenv("CRITERIA")
	}
	if apiKey == "" {
		apiKey = os.Getenv("API_KEY")
	}
	if apiKey == "" {
		return LoadEnvConfig{Logger: logger, Error: fmt.Errorf("API_KEY is not set")}
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
			}
		}
	}
	if cronInterval == 0 && runMode == "cron" {
		cronInterval = 86400
	}
	if !resetStacks {
		resetStacks = os.Getenv("RESET_STACKS") == "true"
	}
	if resetStacks {
		if runMode != "once" {
			return LoadEnvConfig{Logger: logger, Error: fmt.Errorf("RESET_STACKS can only be used in 'once' run mode")}
		}
		confirmReset := os.Getenv("CONFIRM_RESET_STACK")
		const requiredConfirm = "I acknowledge all my current stacks will be deleted and new one will be created"
		if confirmReset != requiredConfirm {
			return LoadEnvConfig{Logger: logger, Error: fmt.Errorf("to use RESET_STACKS, you must set CONFIRM_RESET_STACK to: '%s'", requiredConfirm)}
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
	if !removeSingleAssetStacks {
		removeSingleAssetStacks = os.Getenv("REMOVE_SINGLE_ASSET_STACKS") == "true"
	}
	if parentFilenamePromote == "" || parentFilenamePromote == utils.DefaultParentFilenamePromoteString {
		if envVal := os.Getenv("PARENT_FILENAME_PROMOTE"); envVal != "" {
			parentFilenamePromote = envVal
		}
	}
	if parentExtPromote == "" || parentExtPromote == utils.DefaultParentExtPromoteString {
		if envVal := os.Getenv("PARENT_EXT_PROMOTE"); envVal != "" {
			parentExtPromote = envVal
		}
	}

	// Log startup configuration summary
	logStartupSummary(logger)

	return LoadEnvConfig{Logger: logger, Error: nil}
}

/**************************************************************************************************
** Loads environment variables and command-line flags, with flags taking precedence over env
** variables. Handles critical configuration like API credentials and operation modes.
**
** @param logger - Logger instance for outputting configuration status and errors
**************************************************************************************************/
func loadEnv() *logrus.Logger {
	config := LoadEnvForTesting()
	if config.Error != nil {
		config.Logger.Fatal(config.Error.Error())
	}
	return config.Logger
}
