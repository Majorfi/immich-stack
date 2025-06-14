# CLI Usage

The main entrypoint is `cmd/main.go`, which provides a Cobra-based CLI with multiple commands.

## Command Structure

```sh
immich-stack [command] [flags]
```

### Available Commands

- _(default)_ - Main stacking functionality (when no command is specified)
- `duplicates` - Find and list duplicate assets
- `fix-trash` - Fix incomplete trash operations for stacks
- `help` - Display help information

## Basic Usage

```sh
# Run the main stacking command
./immich-stack --api-key your_key --api-url http://immich:2283

# Run duplicates command
./immich-stack duplicates --api-key your_key

# Run fix-trash command
./immich-stack fix-trash --api-key your_key

# Get help
./immich-stack --help

# Get help for a specific command
./immich-stack duplicates --help
```

## Command Line Flags

### Global Flags (All Commands)

| Flag           | Env Var      | Description                                   |
| -------------- | ------------ | --------------------------------------------- |
| `--api-key`    | `API_KEY`    | Immich API key (comma-separated for multiple) |
| `--api-url`    | `API_URL`    | Immich API base URL                           |
| `--log-level`  | `LOG_LEVEL`  | Log verbosity: debug, info, warn, error       |
| `--log-format` | `LOG_FORMAT` | Log format: text or json                      |

### Stack Command Flags

| Flag                           | Env Var                      | Description                                                                                                                  |
| ------------------------------ | ---------------------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| `--reset-stacks`               | `RESET_STACKS`               | Delete all existing stacks before processing                                                                                 |
| `--confirm-reset-stack`        | `CONFIRM_RESET_STACK`        | Required for RESET_STACKS. Must be set to: 'I acknowledge all my current stacks will be deleted and new one will be created' |
| `--replace-stacks`             | `REPLACE_STACKS`             | Replace stacks for new groups                                                                                                |
| `--dry-run`                    | `DRY_RUN`                    | Simulate actions without making changes                                                                                      |
| `--criteria`                   | `CRITERIA`                   | Custom grouping criteria                                                                                                     |
| `--parent-filename-promote`    | `PARENT_FILENAME_PROMOTE`    | Substrings to promote as parent filenames                                                                                    |
| `--parent-ext-promote`         | `PARENT_EXT_PROMOTE`         | Extensions to promote as parent files                                                                                        |
| `--with-archived`              | `WITH_ARCHIVED`              | Include archived assets in processing                                                                                        |
| `--with-deleted`               | `WITH_DELETED`               | Include deleted assets in processing                                                                                         |
| `--run-mode`                   | `RUN_MODE`                   | Run mode: "once" (default) or "cron"                                                                                         |
| `--cron-interval`              | `CRON_INTERVAL`              | Interval in seconds for cron mode                                                                                            |
| `--log-level`                  | `LOG_LEVEL`                  | Log level: debug, info, warn, error                                                                                          |
| `--remove-single-asset-stacks` | `REMOVE_SINGLE_ASSET_STACKS` | Remove stacks containing only one asset                                                                                      |

### Command-Specific Notes

- **duplicates**: Uses global flags only, particularly `--with-archived` and `--with-deleted` to control which assets are checked
- **fix-trash**: Uses global flags plus the stacking criteria flags (`--criteria`, `--parent-filename-promote`, etc.) to determine which assets to move to trash

## Examples

### Main Stacking Command

```sh
immich-stack --api-key your_key --api-url http://immich-server:2283/api
```

### Find Duplicates

```sh
immich-stack duplicates --api-key your_key --api-url http://immich-server:2283/api
```

### Fix Trash Issues

```sh
immich-stack fix-trash --api-key your_key --dry-run
```

### Dry Run

```sh
immich-stack --dry-run --api-key your_key
```

### Custom Parent Selection

```sh
immich-stack \
  --parent-filename-promote edit,raw \
  --parent-ext-promote .jpg,.dng \
  --api-key your_key
```

### Include Archived/Deleted

```sh
immich-stack \
  --with-archived \
  --with-deleted \
  --api-key your_key
```

### Custom Criteria

```sh
immich-stack \
  --criteria '[{"key":"originalFileName","split":{"delimiters":["~","."],"index":0}},{"key":"localDateTime","delta":{"milliseconds":1000}}]' \
  --api-key your_key
```

### Reset Stacks

```sh
immich-stack \
  --reset-stacks \
  --confirm-reset-stack "I acknowledge all my current stacks will be deleted and new one will be created" \
  --api-key your_key
```

### Remove Single-Asset Stacks

```sh
immich-stack \
  --remove-single-asset-stacks \
  --api-key your_key
```

## Flag Precedence

- Command line flags take precedence over environment variables
- If both are set, the command line flag value is used

## Error Handling

The CLI provides clear error messages for:

- Missing required flags
- Invalid flag values
- API connection issues
- Stack operation failures

## Exit Codes

| Code | Description           |
| ---- | --------------------- |
| 0    | Success               |
| 1    | General error         |
| 2    | Configuration error   |
| 3    | API error             |
| 4    | Stack operation error |
