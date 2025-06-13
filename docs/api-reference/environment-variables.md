# Environment Variables

This document provides a complete reference of all environment variables supported by Immich Stack.

## Required Variables

| Variable  | Description         | Example                          |
| --------- | ------------------- | -------------------------------- |
| `API_KEY` | Immich API key(s)   | `API_KEY=key1,key2`              |
| `API_URL` | Immich API base URL | `API_URL=http://immich:2283/api` |

## Run Mode Configuration

| Variable        | Description                  | Default | Example |
| --------------- | ---------------------------- | ------- | ------- |
| `RUN_MODE`      | Run mode: "once" or "cron"   | "once"  | `cron`  |
| `CRON_INTERVAL` | Interval in seconds for cron | 60      | `3600`  |

## Stack Management

| Variable              | Description                                  | Default | Example              |
| --------------------- | -------------------------------------------- | ------- | -------------------- |
| `RESET_STACKS`        | Delete all existing stacks before processing | false   | `true`               |
| `CONFIRM_RESET_STACK` | Confirmation message for reset               | -       | `"I acknowledge..."` |
| `REPLACE_STACKS`      | Replace stacks for new groups                | false   | `true`               |
| `DRY_RUN`             | Simulate actions without making changes      | false   | `true`               |

## Parent Selection

| Variable                  | Description                                                                                                                   | Default | Example                                                 |
| ------------------------- | ----------------------------------------------------------------------------------------------------------------------------- | ------- | ------------------------------------------------------- |
| `PARENT_FILENAME_PROMOTE` | Substrings to promote as parent filenames. Supports the `sequence` keyword and automatic sequence detection for burst photos. | -       | `edit,raw` or `COVER,sequence` or `0000,0001,0002,0003` |
| `PARENT_EXT_PROMOTE`      | Extensions to promote as parent files                                                                                         | -       | `.jpg,.dng`                                             |

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

## Custom Criteria

| Variable   | Description                   | Default | Example                                               |
| ---------- | ----------------------------- | ------- | ----------------------------------------------------- |
| `CRITERIA` | Custom grouping criteria JSON | -       | See [Custom Criteria](../features/custom-criteria.md) |

## Logging

| Variable     | Description                       | Default | Example |
| ------------ | --------------------------------- | ------- | ------- |
| `LOG_LEVEL`  | Log level (debug,info,warn,error) | info    | `debug` |
| `LOG_FORMAT` | Log format (json,text)            | text    | `json`  |

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
```

### Parent Selection

```sh
PARENT_FILENAME_PROMOTE=edit,raw
PARENT_EXT_PROMOTE=.jpg,.dng
```

### Custom Criteria

```sh
CRITERIA='[{"key":"originalFileName","split":{"delimiters":["~","."],"index":0}},{"key":"localDateTime","delta":{"milliseconds":1000}}]'
```

## Best Practices

1. **Security**

   - Never commit API keys to version control
   - Use environment-specific .env files
   - Rotate API keys regularly

2. **Configuration**

   - Use specific versions in production
   - Document all custom configurations
   - Test changes in development first

3. **Monitoring**

   - Enable debug logging when needed
   - Monitor cron job execution
   - Check stack operation results

4. **Maintenance**
   - Review and update configurations regularly
   - Clean up old stacks periodically
   - Monitor API usage and limits
