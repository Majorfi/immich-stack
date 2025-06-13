/**************************************************************************************************
** Configuration and environment management for the Immich CLI application.
** Handles logger configuration, environment variable loading, and global configuration state.
**************************************************************************************************/

package main

import (
	"os"
	"strconv"
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
		utils.Pretty(`hello`)
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