package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// AppSettings holds all runtime-configurable settings persisted to disk.
// API credentials are intentionally excluded (env-var only, never written to file).
type AppSettings struct {
	// Stacking
	DeltaMs               int    `json:"deltaMs"`
	ParentFilenamePromote string `json:"parentFilenamePromote"`
	ParentExtPromote      string `json:"parentExtPromote"`
	Criteria              string `json:"criteria"`
	// Schedule
	CronInterval int `json:"cronInterval"`
	// Behaviour
	DryRun                  bool `json:"dryRun"`
	ReplaceStacks           bool `json:"replaceStacks"`
	RemoveSingleAssetStacks bool `json:"removeSingleAssetStacks"`
	WithArchived            bool `json:"withArchived"`
	WithDeleted             bool `json:"withDeleted"`
	// Filters
	FilterAlbumIDs    string `json:"filterAlbumIDs"`
	FilterTakenAfter  string `json:"filterTakenAfter"`
	FilterTakenBefore string `json:"filterTakenBefore"`
	// Logging
	LogLevel string `json:"logLevel"`
	// Metadata Sync
	SyncMetadataEnabled bool `json:"syncMetadataEnabled"`
	SyncDate            bool `json:"syncDate"`
	SyncTags            bool `json:"syncTags"`
	SyncPeople          bool `json:"syncPeople"`
}

var settingsMu sync.RWMutex

func defaultAppSettings() AppSettings {
	return AppSettings{
		DeltaMs:               5000,
		ParentFilenamePromote: ",a,b",
		ParentExtPromote:      ".jpg,.png,.jpeg,.heic,.dng",
		CronInterval:          3600,
		ReplaceStacks:         true,
		LogLevel:              "info",
	}
}

// loadAppSettings reads the settings file, falling back to defaults if missing or invalid.
func loadAppSettings() AppSettings {
	s, _ := loadAppSettingsWithOK()
	return s
}

// loadAppSettingsWithOK reads the settings file and returns whether it was successfully loaded.
// Returns (defaults, false) if the file doesn't exist or can't be parsed.
func loadAppSettingsWithOK() (AppSettings, bool) {
	path := settingsFilePath()
	if path == "" {
		return defaultAppSettings(), false
	}
	settingsMu.RLock()
	defer settingsMu.RUnlock()
	data, err := os.ReadFile(path)
	if err != nil {
		return defaultAppSettings(), false
	}
	s := defaultAppSettings()
	if err := json.Unmarshal(data, &s); err != nil {
		return defaultAppSettings(), false
	}
	return s, true
}

// saveAppSettings writes settings to the JSON file.
func saveAppSettings(s AppSettings) error {
	path := settingsFilePath()
	if path == "" {
		return fmt.Errorf("CONFIG_FILE not set")
	}
	settingsMu.Lock()
	defer settingsMu.Unlock()
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func settingsFilePath() string {
	if v := os.Getenv("CONFIG_FILE"); v != "" {
		return v
	}
	return "/app/config.json"
}
