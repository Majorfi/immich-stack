# Configuration

## Basic Configuration

The basic configuration requires two environment variables:

```sh
API_KEY=your_immich_api_key
API_URL=http://your_immich_server:3001/api
```

## API key permissions

When you create the Immich API key, the permissions you tick determine which
commands will work. If a run fails with `403 Forbidden — Missing required permission: X`, add `X` to the key and try again.

### Minimum for `stacker`

| Permission     | Why                                                                                                     |
| -------------- | ------------------------------------------------------------------------------------------------------- |
| `user.read`    | Identify the current user (needed for multi-user and partner filtering)                                 |
| `asset.read`   | Search and fetch assets                                                                                 |
| `stack.read`   | Read existing stacks                                                                                    |
| `stack.create` | Create new stacks                                                                                       |
| `stack.update` | Modify existing stacks (Immich enforces this even though replacement is implemented as delete + create) |
| `stack.delete` | Delete stacks (`RESET_STACKS`, `REPLACE_STACKS`, single-asset cleanup)                                  |

Add `album.read` if you filter by album (`ALBUM_NAMES` or `--filter-album-ids`).

### Other commands

`duplicates` needs nothing extra. It runs the detection client-side and
groups assets by `OriginalFileName` and `LocalDateTime`.

`fix-trash` needs `asset.delete` on top of the stacker set. It uses
`DELETE /assets` to move matched assets to the trash.

This list was confirmed by the maintainer in
[discussion #29](https://github.com/Majorfi/immich-stack/discussions/29#discussioncomment-16000078).

## Run Modes

Immich Stack supports two run modes:

1. **Once Mode** (default)

   - Runs once and exits
   - Good for manual runs or scheduled tasks
   - Use: `RUN_MODE=once`

1. **Cron Mode**

   - Runs periodically
   - Good for continuous operation
   - Use: `RUN_MODE=cron`
   - Configure interval with `CRON_INTERVAL` (in seconds)

Example cron configuration:

```sh
RUN_MODE=cron
CRON_INTERVAL=3600  # Run every hour
```

For detailed information about cron mode including state management, signal handling, monitoring, and best practices, see the [Cron Mode documentation](../features/cron-mode.md).

## Stack Management

### Parent Selection

Control which files become stack parents using:

1. **Filename Promotion:**

   ```sh
   PARENT_FILENAME_PROMOTE=edit,raw,original
   ```

   Files containing these substrings will be promoted

1. **Extension Promotion:**

   ```sh
   PARENT_EXT_PROMOTE=.jpg,.dng
   ```

   Files with these extensions will be promoted

### Stack Operations

1. **Dry Run:**

   ```sh
   DRY_RUN=true
   ```

   Simulates actions without making changes

1. **Reset Stacks:**

```sh
RESET_STACKS=true
CONFIRM_RESET_STACK="I acknowledge all my current stacks will be deleted and new one will be created"
```

Deletes all existing stacks before processing. This requires `RUN_MODE=once`; using it in `cron` mode results in an error. The confirmation text must match exactly as shown above.

1. **Replace Stacks:**
   ```sh
   REPLACE_STACKS=true
   ```
   Replaces existing stacks with new groups

## Asset Inclusion

Control which assets are processed:

```sh
WITH_ARCHIVED=true  # Include archived assets
WITH_DELETED=true   # Include deleted assets
```

## Asset Filtering

Limit which assets are processed using album and date filters:

### Filter by Album

```sh
# Single album by UUID
FILTER_ALBUM_IDS=550e8400-e29b-41d4-a716-446655440000

# Single album by name
FILTER_ALBUM_IDS=Vacation Photos

# Multiple albums (OR logic - processes assets from any of these)
FILTER_ALBUM_IDS=album-uuid-1,Vacation Photos,Family Events
```

### Filter by Date Range

```sh
# Process only assets from 2024
FILTER_TAKEN_AFTER=2024-01-01T00:00:00Z
FILTER_TAKEN_BEFORE=2024-12-31T23:59:59Z
```

Dates must use ISO 8601 format (e.g., `2024-01-15T10:30:00Z`).

## Logging

Configure logging output and verbosity:

```sh
LOG_LEVEL=info      # Options: trace, debug, info, warn, error
LOG_FORMAT=text     # Options: text, json
LOG_FILE=/app/logs/immich-stack.log  # Optional: enable dual logging (stdout + file)
```

### File Logging with Docker

When using Docker, you can persist logs to a file by setting `LOG_FILE` and mounting a volume:

```yaml
services:
  immich-stack:
    image: majorfi/immich-stack:latest
    environment:
      - LOG_FILE=/app/logs/immich-stack.log
      - LOG_LEVEL=info
      - LOG_FORMAT=text
    volumes:
      - ./logs:/app/logs
```

The application automatically creates the log directory if it doesn't exist. If file logging fails (e.g., permission issues), it gracefully falls back to stdout-only logging.

## Custom Criteria

Configure custom grouping criteria using the `CRITERIA` environment variable. See [Custom Criteria](../features/custom-criteria.md) for details.

## Example Configuration

```sh
# Required
API_KEY=your_immich_api_key
API_URL=http://immich-server:2283/api

# Run mode
RUN_MODE=cron
CRON_INTERVAL=3600

# Stack management
PARENT_FILENAME_PROMOTE=edit,raw
PARENT_EXT_PROMOTE=.jpg,.dng
DRY_RUN=false
RESET_STACKS=false
REPLACE_STACKS=true

# Asset inclusion
WITH_ARCHIVED=false
WITH_DELETED=false

# Asset filtering (optional)
FILTER_ALBUM_IDS=
FILTER_TAKEN_AFTER=
FILTER_TAKEN_BEFORE=

# Logging
LOG_LEVEL=info
LOG_FORMAT=text
LOG_FILE=/app/logs/immich-stack.log

# Custom criteria
CRITERIA='[{"key":"originalFileName","split":{"delimiters":["~","."],"index":0}},{"key":"localDateTime","delta":{"milliseconds":1000}}]'
```
