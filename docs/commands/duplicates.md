# Duplicates Command

The `duplicates` command helps you identify duplicate assets in your Immich library based on filename and timestamp.

## Overview

This command scans your entire Immich library and groups assets that have identical:

- Original filename
- Local date/time

This is useful for finding:

- Multiple uploads of the same photo
- Photos that were accidentally imported multiple times
- Duplicate files from different sources that have the same name and timestamp

## Usage

```bash
immich-stack duplicates [flags]
```

## Examples

### Basic Usage

Find duplicates in your library:

```bash
immich-stack duplicates --api-key your_key --api-url http://immich:2283
```

### Include Archived Assets

```bash
immich-stack duplicates --api-key your_key --with-archived
```

### Multi-User Scan

Check duplicates for multiple users:

```bash
immich-stack duplicates --api-key "user1_key,user2_key"
```

### With Debug Logging

Get detailed information during the scan:

```bash
immich-stack duplicates --api-key your_key --log-level debug
```

## Output

The command will output groups of duplicate assets. For example:

```
Duplicate group: IMG_1234.jpg|2024-01-15T10:30:00 (3 assets)
  - ID: abc123, FileName: IMG_1234.jpg, LocalDateTime: 2024-01-15T10:30:00
  - ID: def456, FileName: IMG_1234.jpg, LocalDateTime: 2024-01-15T10:30:00
  - ID: ghi789, FileName: IMG_1234.jpg, LocalDateTime: 2024-01-15T10:30:00
```

If no duplicates are found:

```
No duplicates found based on OriginalFileName and LocalDateTime.
```

## Flags

The `duplicates` command inherits all global flags, particularly:

- `--api-key` - Required for authentication
- `--api-url` - Immich server URL
- `--with-archived` - Include archived assets in the scan
- `--with-deleted` - Include deleted assets in the scan
- `--log-level` - Control verbosity of output

## Use Cases

### 1. Pre-Cleanup Audit

Before running cleanup operations, identify duplicates:

```bash
# First, find duplicates
immich-stack duplicates --api-key your_key

# Then manually review and delete duplicates in Immich UI
```

### 2. Regular Maintenance

Run periodically to check for new duplicates:

```bash
# Add to a monthly maintenance script
immich-stack duplicates --api-key your_key --log-level warn
```

### 3. Migration Verification

After migrating photos from another system:

```bash
# Check if migration created duplicates
immich-stack duplicates --api-key your_key --with-archived --with-deleted
```

## Important Notes

1. **Read-Only Operation**: This command only reports duplicates; it does not delete or modify any assets
1. **Exact Matching**: Only assets with identical filename AND timestamp are considered duplicates
1. **Performance**: For large libraries, this command may take several minutes to complete
1. **Stack-Aware**: The command fetches stack information but duplicates are detected independently of stack membership

## See Also

- [Main Stacking Command](../getting-started/quick-start.md) - Create stacks from your assets
- [Fix-Trash Command](fix-trash.md) - Maintain stack consistency when deleting
- [Environment Variables](../api-reference/environment-variables.md) - Configuration options
