# Environment Variables

This document provides a complete reference of all environment variables supported by Immich Stack.

## Required Variables

| Variable  | Description         | Example                          |
| --------- | ------------------- | -------------------------------- |
| `API_KEY` | Immich API key(s)   | `API_KEY=key1,key2`              |
| `API_URL` | Immich API base URL | `API_URL=http://immich:2283/api` |

## Run Mode Configuration

| Variable        | Description                  | Default                       | Example |
| --------------- | ---------------------------- | ----------------------------- | ------- |
| `RUN_MODE`      | Run mode: "once" or "cron"   | "once"                        | `cron`  |
| `CRON_INTERVAL` | Interval in seconds for cron | 86400 (when RUN_MODE is cron) | `3600`  |

## Stack Management

| Variable                     | Description                                                            | Default | Example              |
| ---------------------------- | ---------------------------------------------------------------------- | ------- | -------------------- |
| `RESET_STACKS`               | Delete all existing stacks before processing (only in `RUN_MODE=once`) | false   | `true`               |
| `CONFIRM_RESET_STACK`        | Confirmation message for reset                                         | -       | `"I acknowledge..."` |
| `REPLACE_STACKS`             | Replace stacks for new groups                                          | true    | `false`              |
| `DRY_RUN`                    | Simulate actions without making changes                                | false   | `true`               |
| `REMOVE_SINGLE_ASSET_STACKS` | Remove stacks containing only one asset                                | false   | `true`               |

Note:

- `RESET_STACKS` can only be used when `RUN_MODE=once`. Using it in `cron` mode results in an error.
- `CONFIRM_RESET_STACK` must match the exact confirmation phrase shown in the examples.

## Parent Selection

| Variable                  | Description                                                                                                                                                       | Default                             | Example                                                               |
| ------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------- | --------------------------------------------------------------------- |
| `PARENT_FILENAME_PROMOTE` | Substrings to promote as parent filenames. Supports empty string for negative matching, the `sequence` keyword and automatic sequence detection for burst photos. | `cover,edit,crop,hdr,biggestNumber` | `,_edited` or `edit,raw` or `COVER,sequence` or `0000,0001,0002,0003` |
| `PARENT_EXT_PROMOTE`      | Extensions to promote as parent files                                                                                                                             | `.jpg,.png,.jpeg,.heic,.dng`        | `.jpg,.dng`                                                           |

### Empty String for Negative Matching

An empty string (`""`) in the promote list acts as a negative match - it matches files that **don't** contain any of the other non-empty substrings in the list:

| Example          | Description                | Effect                                                                  |
| ---------------- | -------------------------- | ----------------------------------------------------------------------- |
| `,_edited`       | Prioritize unedited files  | Files without "\_edited" are promoted first                             |
| `,_edited,_crop` | Prioritize clean filenames | Files without "\_edited" or "\_crop" come first                         |
| `COVER,,_edited` | Complex priority           | COVER files first, then files without "\_edited", then "\_edited" files |

**Examples:**

```sh
# Promote unedited JPGs over edited ones
PARENT_FILENAME_PROMOTE=,_edited
# Result: IMG_1234.jpg > IMG_1234_edited.jpg

# Multiple exclusions
PARENT_FILENAME_PROMOTE=,_edited,_crop,_cropped
# Result: IMG_1234.jpg > IMG_1234_edited.jpg > IMG_1234_crop.jpg > IMG_1234_cropped.jpg
```

### Sequence Keyword

The `sequence` keyword provides flexible handling of sequential files (like burst photos):

| Syntax          | Description                            | Example Files                                  | Result                                 |
| --------------- | -------------------------------------- | ---------------------------------------------- | -------------------------------------- |
| `sequence`      | Matches any numeric sequence           | `IMG_0001.jpg`, `IMG_0010.jpg`, `IMG_0100.jpg` | Orders by numeric value: 1, 10, 100    |
| `sequence:4`    | Matches exactly 4-digit sequences      | `IMG_0001.jpg`, `IMG_0010.jpg`                 | Only matches 4-digit numbers           |
| `sequence:IMG_` | Matches sequences with specific prefix | `IMG_001.jpg`, `PHOTO_001.jpg`                 | Only orders files starting with `IMG_` |

**Mixed Promote Lists:**

```sh
# Prioritize COVER files, then order remaining by sequence
PARENT_FILENAME_PROMOTE=COVER,sequence

# Prioritize edited files, then 4-digit sequences
PARENT_FILENAME_PROMOTE=edit,sequence:4
```

### Automatic Sequence Detection (Legacy)

When `PARENT_FILENAME_PROMOTE` contains a numeric sequence pattern (e.g., `0000,0001,0002,0003`), the system automatically:

- Detects the sequence pattern (prefix, number, suffix)
- Matches files that follow the same pattern
- Orders files by their numeric value, even beyond the listed values
- Works with various formats: `IMG_0001`, `photo001`, `1a`, etc.

**Note:** The `sequence` keyword is recommended over listing individual numbers for better flexibility.

## Asset Inclusion

| Variable        | Description                           | Default | Example |
| --------------- | ------------------------------------- | ------- | ------- |
| `WITH_ARCHIVED` | Include archived assets in processing | false   | `true`  |
| `WITH_DELETED`  | Include deleted assets in processing  | false   | `true`  |

## Asset Filtering

| Variable              | Description                                           | Default | Example                  |
| --------------------- | ----------------------------------------------------- | ------- | ------------------------ |
| `FILTER_ALBUM_IDS`    | Filter by album IDs or names (comma-separated)        | -       | `album-uuid-1,My Photos` |
| `FILTER_TAKEN_AFTER`  | Only process assets taken after this date (ISO 8601)  | -       | `2024-01-01T00:00:00Z`   |
| `FILTER_TAKEN_BEFORE` | Only process assets taken before this date (ISO 8601) | -       | `2024-12-31T23:59:59Z`   |

### Album Filtering

Album filters accept both UUIDs and album names:

```sh
# By UUID
FILTER_ALBUM_IDS=550e8400-e29b-41d4-a716-446655440000

# By name
FILTER_ALBUM_IDS=Vacation Photos

# Multiple albums (OR logic - assets from ANY album)
FILTER_ALBUM_IDS=album-uuid-1,Vacation Photos,Family
```

When multiple albums are specified, assets from **any** of the albums are processed (OR logic).

### Date Range Filtering

Date filters use ISO 8601 (RFC3339) format:

```sh
# Assets from 2024 only
FILTER_TAKEN_AFTER=2024-01-01T00:00:00Z
FILTER_TAKEN_BEFORE=2024-12-31T23:59:59Z

# Assets from last month
FILTER_TAKEN_AFTER=2024-11-01T00:00:00Z
```

Valid date formats:

- `2024-01-15T10:30:00Z` (UTC)
- `2024-01-15T10:30:00+00:00` (with timezone offset)
- `2024-01-15T10:30:00-05:00` (EST timezone)

## Custom Criteria

| Variable   | Description                   | Default   | Example                                               |
| ---------- | ----------------------------- | --------- | ----------------------------------------------------- |
| `CRITERIA` | Custom grouping criteria JSON | See below | See [Custom Criteria](../features/custom-criteria.md) |

### Default Criteria

When `CRITERIA` is not set, the following default is used:

```json
[
  {
    "key": "originalFileName",
    "split": { "delimiters": ["~", "."], "index": 0 }
  },
  {
    "key": "localDateTime",
    "delta": { "milliseconds": 1000 }
  }
]
```

This groups photos by:

- Base filename (before `~` or `.`)
- Time captured (within 1 second tolerance)

### Custom Criteria Formats

The `CRITERIA` environment variable supports three formats for flexible asset stacking:

### Legacy Array Format

Simple array of criteria where ALL conditions must match (AND logic):

```json
[
  {
    "key": "originalFileName",
    "split": { "delimiters": ["~", "."], "index": 0 }
  },
  { "key": "localDateTime", "delta": { "milliseconds": 1000 } }
]
```

### Advanced Groups Format

Flexible format supporting multiple grouping strategies with OR/AND logic:

```json
{
  "mode": "advanced",
  "groups": [
    {
      "operator": "AND",
      "criteria": [
        {
          "key": "originalFileName",
          "regex": { "key": "PXL_\\d{8}_\\d+", "index": 0 }
        },
        {
          "key": "localDateTime",
          "delta": { "milliseconds": 1000 }
        }
      ]
    }
  ]
}
```

### Advanced Expression Format

Most powerful format supporting unlimited nested logical expressions with AND, OR, and NOT operations:

```json
{
  "mode": "advanced",
  "expression": {
    "operator": "AND",
    "children": [
      {
        "operator": "OR",
        "children": [
          {
            "criteria": {
              "key": "originalFileName",
              "regex": { "key": "PXL_\\d{8}_\\d+", "index": 0 }
            }
          },
          {
            "criteria": {
              "key": "originalPath",
              "split": { "delimiters": ["/"], "index": 2 }
            }
          }
        ]
      },
      {
        "criteria": {
          "key": "localDateTime",
          "delta": { "milliseconds": 1000 }
        }
      }
    ]
  }
}
```

**Format Features:**

| Format         | Complexity | Use Case                     | Logic Support                     |
| -------------- | ---------- | ---------------------------- | --------------------------------- |
| **Legacy**     | Simple     | Basic grouping               | AND only                          |
| **Groups**     | Medium     | Multiple grouping strategies | AND/OR per group                  |
| **Expression** | Advanced   | Complex logical conditions   | Unlimited nesting with AND/OR/NOT |

**Expression Format Benefits:**

- **Unlimited Nesting**: Create complex logical trees with multiple levels
- **Full Logic Support**: AND, OR, and NOT operators at any level
- **Precise Control**: Express any logical combination of criteria
- **Backward Compatible**: All legacy formats continue to work unchanged

**Complex Expression Example:**

```json
{
  "mode": "advanced",
  "expression": {
    "operator": "AND",
    "children": [
      {
        "operator": "OR",
        "children": [
          {
            "criteria": {
              "key": "originalFileName",
              "regex": { "key": "PXL_", "index": 0 }
            }
          },
          {
            "criteria": {
              "key": "originalFileName",
              "regex": { "key": "IMG_", "index": 0 }
            }
          }
        ]
      },
      {
        "operator": "NOT",
        "children": [{ "criteria": { "key": "isArchived" } }]
      },
      {
        "criteria": {
          "key": "localDateTime",
          "delta": { "milliseconds": 2000 }
        }
      }
    ]
  }
}
```

This example groups assets that:

- Have filenames starting with "PXL\*" OR "IMG\*"
- AND are NOT archived
- AND were taken within 2 seconds of each other

## Logging

| Variable     | Description                                | Default | Example                      |
| ------------ | ------------------------------------------ | ------- | ---------------------------- |
| `LOG_LEVEL`  | Log level (trace,debug,info,warn,error)    | info    | `debug`                      |
| `LOG_FORMAT` | Log format (json,text)                     | text    | `json`                       |
| `LOG_FILE`   | Optional file path for dual logging output | -       | `/app/logs/immich-stack.log` |

### File Logging

When `LOG_FILE` is set, logs are written to both stdout (visible in `docker logs`) and the specified file. This is useful for:

- Persistent log storage across container restarts
- Log aggregation and analysis
- Debugging issues after they occur

To enable file logging with Docker:

1. Set the `LOG_FILE` environment variable to a path inside the container
1. Mount a host directory to persist logs

Example Docker Compose configuration:

```yaml
environment:
  - LOG_FILE=/app/logs/immich-stack.log
volumes:
  - ./logs:/app/logs
```

If the log file cannot be created (e.g., permission issues), the application gracefully falls back to stdout-only logging.

## Examples

### Basic Configuration

```sh
API_KEY=your_key
API_URL=http://immich:2283/api
```

### Cron Mode

```sh
RUN_MODE=cron
CRON_INTERVAL=3600
```

### Stack Management

```sh
RESET_STACKS=true
CONFIRM_RESET_STACK="I acknowledge all my current stacks will be deleted and new one will be created"
REPLACE_STACKS=true
DRY_RUN=false
REMOVE_SINGLE_ASSET_STACKS=true
```

This operation requires `RUN_MODE=once`.

### Parent Selection

```sh
PARENT_FILENAME_PROMOTE=edit,raw
PARENT_EXT_PROMOTE=.jpg,.dng
```

### Custom Criteria - Legacy Array Format

```sh
CRITERIA='[{"key":"originalFileName","split":{"delimiters":["~","."],"index":0}},{"key":"localDateTime","delta":{"milliseconds":1000}}]'
```

### Custom Criteria - Advanced Groups Format

```sh
# Filter PXL files AND group by timestamp
CRITERIA='{
  "mode": "advanced",
  "groups": [
    {
      "operator": "AND",
      "criteria": [
        {"key": "originalFileName", "regex": {"key": "PXL_\\d{8}_\\d+", "index": 0}},
        {"key": "localDateTime", "delta": {"milliseconds": 1000}}
      ]
    }
  ]
}'

# Group by same directory OR same timestamp
CRITERIA='{
  "mode": "advanced",
  "groups": [
    {
      "operator": "OR",
      "criteria": [
        {"key": "originalPath", "split": {"delimiters": ["/"], "index": 2}},
        {"key": "localDateTime", "delta": {"milliseconds": 1000}}
      ]
    }
  ]
}'
```

### Custom Criteria - Advanced Expression Format

```sh
# Complex multi-camera setup with exclusions
CRITERIA='{
  "mode": "advanced",
  "expression": {
    "operator": "AND",
    "children": [
      {
        "operator": "OR",
        "children": [
          {"criteria": {"key": "originalFileName", "regex": {"key": "PXL_", "index": 0}}},
          {"criteria": {"key": "originalFileName", "regex": {"key": "IMG_", "index": 0}}}
        ]
      },
      {
        "operator": "NOT",
        "children": [
          {"criteria": {"key": "isArchived"}}
        ]
      },
      {"criteria": {"key": "localDateTime", "delta": {"milliseconds": 2000}}}
    ]
  }
}'
```

### Asset Filtering

```sh
# Filter by specific album
FILTER_ALBUM_IDS=550e8400-e29b-41d4-a716-446655440000

# Filter by multiple albums (OR logic)
FILTER_ALBUM_IDS=album-1,Vacation Photos,Family Events

# Filter by date range
FILTER_TAKEN_AFTER=2024-01-01T00:00:00Z
FILTER_TAKEN_BEFORE=2024-12-31T23:59:59Z

# Combined: specific album and date range
FILTER_ALBUM_IDS=My Photos
FILTER_TAKEN_AFTER=2024-06-01T00:00:00Z
```

## Best Practices

1. **Security**

   - Never commit API keys to version control
   - Use environment-specific .env files
   - Rotate API keys regularly

1. **Configuration**

   - Use specific versions in production
   - Document all custom configurations
   - Test changes in development first

1. **Monitoring**

   - Enable debug logging when needed
   - Monitor cron job execution
   - Check stack operation results

1. **Maintenance**

   - Review and update configurations regularly
   - Clean up old stacks periodically
   - Monitor API usage and limits
