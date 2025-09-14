# Configuration

## Basic Configuration

The basic configuration requires two environment variables:

```sh
API_KEY=your_immich_api_key
API_URL=http://your_immich_server:3001/api
```

## Run Modes

Immich Stack supports two run modes:

1. **Once Mode** (default)

   - Runs once and exits
   - Good for manual runs or scheduled tasks
   - Use: `RUN_MODE=once`

2. **Cron Mode**
   - Runs periodically
   - Good for continuous operation
   - Use: `RUN_MODE=cron`
   - Configure interval with `CRON_INTERVAL` (in seconds)

Example cron configuration:

```sh
RUN_MODE=cron
CRON_INTERVAL=3600  # Run every hour
```

## Stack Management

### Parent Selection

Control which files become stack parents using:

1. **Filename Promotion:**

   ```sh
   PARENT_FILENAME_PROMOTE=edit,raw,original
   ```

   Files containing these substrings will be promoted

2. **Extension Promotion:**
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

2. **Reset Stacks:**

```sh
RESET_STACKS=true
CONFIRM_RESET_STACK="I acknowledge all my current stacks will be deleted and new one will be created"
```

Deletes all existing stacks before processing. This requires `RUN_MODE=once`; using it in `cron` mode results in an error. The confirmation text must match exactly as shown above.

3. **Replace Stacks:**
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
REPLACE_STACKS=false

# Asset inclusion
WITH_ARCHIVED=false
WITH_DELETED=false

# Custom criteria
CRITERIA='[{"key":"originalFileName","split":{"delimiters":["~","."],"index":0}},{"key":"localDateTime","delta":{"milliseconds":1000}}]'
```
