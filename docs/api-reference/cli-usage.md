# CLI Usage

The main entrypoint is `cmd/main.go`, which provides a Cobra-based CLI.

## Basic Usage

```sh
# Using the binary
./immich-stack

# Or if installed in PATH
immich-stack
```

## Command Line Flags

| Flag                        | Env Var                   | Description                                                                                                                  |
| --------------------------- | ------------------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| `--api-key`                 | `API_KEY`                 | Immich API key (comma-separated for multiple users)                                                                          |
| `--api-url`                 | `API_URL`                 | Immich API base URL                                                                                                          |
| `--reset-stacks`            | `RESET_STACKS`            | Delete all existing stacks before processing                                                                                 |
| `--confirm-reset-stack`     | `CONFIRM_RESET_STACK`     | Required for RESET_STACKS. Must be set to: 'I acknowledge all my current stacks will be deleted and new one will be created' |
| `--replace-stacks`          | `REPLACE_STACKS`          | Replace stacks for new groups                                                                                                |
| `--dry-run`                 | `DRY_RUN`                 | Simulate actions without making changes                                                                                      |
| `--criteria`                | `CRITERIA`                | Custom grouping criteria                                                                                                     |
| `--parent-filename-promote` | `PARENT_FILENAME_PROMOTE` | Substrings to promote as parent filenames                                                                                    |
| `--parent-ext-promote`      | `PARENT_EXT_PROMOTE`      | Extensions to promote as parent files                                                                                        |
| `--with-archived`           | `WITH_ARCHIVED`           | Include archived assets in processing                                                                                        |
| `--with-deleted`            | `WITH_DELETED`            | Include deleted assets in processing                                                                                         |
| `--run-mode`                | `RUN_MODE`                | Run mode: "once" (default) or "cron"                                                                                         |
| `--cron-interval`           | `CRON_INTERVAL`           | Interval in seconds for cron mode                                                                                            |

## Examples

### Basic Run

```sh
immich-stack --api-key your_key --api-url http://immich-server:2283/api
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
