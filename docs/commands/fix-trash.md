# Fix-Trash Command

The `fix-trash` command ensures stack consistency by moving related assets to trash when their stack members have been deleted.

## Overview

When you delete a photo in Immich that's part of a stack (e.g., one photo from a burst sequence), the other photos in that stack should typically be deleted too. However, if photos are deleted through the Immich UI or other means, related stack members might remain in your library.

This command:

1. Scans your trash for deleted assets
2. Identifies which active assets would stack with the trashed ones
3. Moves those related assets to trash to maintain consistency

## How It Works

The command uses the same stacking criteria as the main stacking command. For each trashed asset:

1. It combines the trashed asset with all active assets
2. Runs the stacking algorithm to find matches
3. Any active assets that would group with the trashed asset are marked for deletion

## Usage

```bash
immich-stack fix-trash [flags]
```

## Examples

### Basic Usage

```bash
immich-stack fix-trash --api-key your_key --api-url http://immich:2283
```

### Dry Run Mode

See what would be deleted without making changes:

```bash
immich-stack fix-trash --api-key your_key --dry-run
```

### With Custom Criteria

Use specific stacking criteria for matching:

```bash
immich-stack fix-trash --api-key your_key --criteria '[{"key":"originalFileName","regex":{"pattern":"BURST(\\d+)","index":1}}]'
```

### Debug Mode

Get detailed information about the matching process:

```bash
immich-stack fix-trash --api-key your_key --log-level debug
```

## Output

The command provides detailed feedback:

```
🗑️  Found 5 trashed assets
📊 Analyzing against 1000 active assets...
✅ Analysis complete: 5 trashed → 15 related assets to trash

📁 Assets to trash by type:
   - JPG files: 10
   - DNG files: 5

🗑️  Moving 15 assets to trash... done
```

In debug mode, you'll see detailed stack information:

```
📋 Summary of assets to trash:
Stack with DSC_0001_BURST.jpg (in trash): DSC_0002_BURST.jpg, DSC_0003_BURST.jpg
Stack with IMG_1234.jpg (in trash): IMG_1234.dng
```

## Flags

The command uses all global flags, particularly:

- `--dry-run` - Preview what would be deleted without making changes
- `--criteria` - Custom stacking criteria (uses same format as main command)
- `--parent-filename-promote` - Filename patterns for stacking
- `--log-level` - Set to `debug` for detailed matching information

## Use Cases

### 1. Post-Deletion Cleanup

After deleting photos through Immich UI:

```bash
# Clean up related assets after manual deletion
immich-stack fix-trash --api-key your_key
```

### 2. Burst Photo Management

When you delete one photo from a burst sequence:

```bash
# Ensure all burst photos are deleted together
immich-stack fix-trash --api-key your_key --parent-filename-promote "sequence"
```

### 3. RAW+JPEG Cleanup

After deleting JPEG files, remove orphaned RAW files:

```bash
# First check what would be deleted
immich-stack fix-trash --api-key your_key --dry-run

# Then execute if correct
immich-stack fix-trash --api-key your_key
```

### 4. Scheduled Maintenance

Add to a cron job for automatic cleanup:

```bash
# Run weekly to maintain consistency
0 2 * * 0 immich-stack fix-trash --api-key your_key --log-level warn
```

## Important Notes

1. **Uses Stacking Criteria**: The command uses the same criteria as the main stacking command
2. **Irreversible**: Moving assets to trash cannot be undone through this tool
3. **Performance**: For large libraries, analysis may take several minutes
4. **Safety First**: Always use `--dry-run` first to preview changes

## Best Practices

1. **Test with Dry Run**: Always run with `--dry-run` first
2. **Review Debug Output**: Use `--log-level debug` to understand matching logic
3. **Backup Important Data**: Ensure you have backups before running
4. **Regular Maintenance**: Run periodically to maintain library consistency

## Common Scenarios

### Incomplete Burst Deletion

```bash
# You deleted DSC_0001_BURST but DSC_0002_BURST remains
immich-stack fix-trash --api-key your_key
# Output: Will move DSC_0002_BURST to trash
```

### Orphaned RAW Files

```bash
# You deleted IMG_1234.jpg but IMG_1234.dng remains
immich-stack fix-trash --api-key your_key
# Output: Will move IMG_1234.dng to trash
```

## See Also

- [Main Stacking Command](../getting-started/quick-start.md) - Create stacks from your assets
- [Duplicates Command](duplicates.md) - Find duplicate assets
- [Stacking Logic](../features/stacking-logic.md) - Understand how assets are grouped
- [Custom Criteria](../features/custom-criteria.md) - Configure matching rules
