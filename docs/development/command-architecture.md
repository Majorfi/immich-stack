# Command Architecture

Immich Stack uses the [Cobra](https://github.com/spf13/cobra) framework to provide a modular, extensible command-line interface.

## Architecture Overview

The application is structured as a multi-command CLI tool with the following hierarchy:

```
immich-stack (root command)
├── stack (default command)
├── duplicates
├── fix-trash
├── help
└── version
```

## File Structure

```
cmd/
├── main.go          # Entry point and root command setup
├── config.go        # Shared configuration and environment loading
├── stack.go         # Main stacking command implementation
├── duplicates.go    # Duplicate detection command
└── fixtrash.go      # Fix trash consistency command
```

## Command Implementation Pattern

Each command follows a consistent pattern:

### 1. Command Definition

```go
var duplicatesCmd = &cobra.Command{
    Use:   "duplicates",
    Short: "Find duplicate assets",
    Long:  `Detailed description...`,
    Run:   runDuplicates,
}
```

### 2. Flag Registration

```go
func init() {
    duplicatesCmd.Flags().BoolVar(&withArchived, "with-archived", false, "Include archived assets")
    // ... more flags
}
```

### 3. Execution Function

```go
func runDuplicates(cmd *cobra.Command, args []string) {
    logger := loadEnv()

    // Multi-user support
    apiKeys := parseAPIKeys(apiKey)

    for _, key := range apiKeys {
        client := immich.NewClient(...)
        // Command-specific logic
    }
}
```

## Shared Components

### Configuration Loading

The `loadEnv()` function in `config.go` handles:

- Environment variable loading
- Logger initialization
- Configuration validation
- Flag/environment precedence

### Multi-User Support

All commands support processing multiple users:

- API keys are comma-separated
- Each user is processed sequentially
- Errors for one user don't affect others

### Client Initialization

Each command creates its own Immich client with appropriate settings:

```go
client := immich.NewClient(
    apiURL,
    key,
    resetStacks,      // Command-specific
    replaceStacks,    // Command-specific
    dryRun,          // Global option
    withArchived,    // Global option
    withDeleted,     // Global option
    withPartners,    // Command-specific
    logger
)
```

## Adding New Commands

To add a new command:

### 1. Create Command File

Create `cmd/newcommand.go`:

```go
package main

import (
    "github.com/spf13/cobra"
    "github.com/majorfi/immich-stack/pkg/immich"
)

var newCmd = &cobra.Command{
    Use:   "new-command",
    Short: "Brief description",
    Long:  `Detailed description`,
    Run:   runNewCommand,
}

func init() {
    // Register command-specific flags
    newCmd.Flags().StringVar(&someFlag, "some-flag", "", "Flag description")
}

func runNewCommand(cmd *cobra.Command, args []string) {
    logger := loadEnv()

    // Implementation
}
```

### 2. Register Command

In `cmd/main.go`, add to `init()`:

```go
func init() {
    // ... existing commands
    rootCmd.AddCommand(newCmd)
}
```

### 3. Update Documentation

- Add command documentation in `docs/commands/`
- Update `docs/api-reference/cli-usage.md`
- Update `docs/commands/index.md`
- Add to changelog

## Best Practices

### 1. Command Design

- Keep commands focused on a single purpose
- Use descriptive names and help text
- Follow existing flag naming conventions
- Support dry-run where applicable

### 2. Error Handling

- Log errors with appropriate levels
- Continue processing other users on error
- Return meaningful error messages
- Use structured logging

### 3. Code Reuse

- Use shared configuration loading
- Leverage common client initialization
- Share utility functions via packages
- Avoid duplicating logic

### 4. Testing

- Unit test command logic separately
- Test flag parsing and validation
- Mock API calls for integration tests
- Test multi-user scenarios

## Environment Variables

All commands respect environment variables with flag precedence:

```go
viper.BindEnv("api_key", "API_KEY")
viper.BindEnv("api_url", "API_URL")
// ... more bindings

// Flags take precedence over environment
if cmd.Flags().Changed("api-key") {
    viper.Set("api_key", apiKey)
}
```

## Logging

Commands use the shared logger configuration:

```go
logger := loadEnv() // Initializes logger with LOG_LEVEL and LOG_FORMAT
logger.Info("Starting command...")
logger.Debug("Detailed information...")
logger.Error("Error occurred: %v", err)
```

## Future Considerations

### Potential Enhancements

1. **Interactive Mode**: Add interactive prompts for configuration
1. **Config Files**: Support for configuration files
1. **Plugin System**: Allow external command plugins
1. **Parallel Processing**: Process multiple users concurrently
1. **Progress Indicators**: Add progress bars for long operations

### Command Ideas

- `immich-stack stats` - Show stacking statistics
- `immich-stack validate` - Validate stack integrity
- `immich-stack export` - Export stack information
- `immich-stack migrate` - Migrate from other stacking solutions
