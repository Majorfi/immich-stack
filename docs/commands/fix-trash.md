# Fix-Trash Command

The `fix-trash` command ensures stack consistency by moving related assets to trash when their stack members have been deleted.

## Overview

When you delete a photo in Immich that would stack with other photos (for example the JPG of a RAW+JPEG pair), the related assets stay behind in your library. This command finds them and moves them to trash too.

The command runs two passes:

1. **Stack cascade** — active assets that would stack with a trashed asset are moved to trash.
1. **Orphaned DNG cleanup** — active DNG files with no JPG companion are moved to trash. This pass always runs, independently of what is in your trash.

!!! warning "RAW-only libraries"

    The orphaned-DNG pass treats every DNG without a matching `.jpg`/`.jpeg` as orphaned,
    regardless of your trash contents. If you shoot RAW-only (DNG files without JPG
    sidecars), this pass will flag those DNGs for trash. Other companion formats (HEIC,
    PNG, ...) are not recognized. Always run with `--dry-run` first and review the summary.

## How It Works

### Pass 1: stack cascade

1. Fetches your trashed assets and your active assets. Partner-owned assets are dropped from both lists.
1. Skips trashed assets that appear to have been re-uploaded: if an active asset with the same filename has a newer created/modified/updated timestamp, cascading from the old copy would drag the new copy's companions into the trash.
1. Runs the stacking algorithm once over the remaining trashed assets plus all active assets, using the same criteria as the main command (`--criteria`, `--parent-filename-promote`, `--parent-ext-promote`).
1. Every active asset that lands in a group containing a trashed asset is marked for trash.

### Pass 2: orphaned DNG cleanup

1. Groups active assets by base filename: the extension is stripped, suffixes after `_` or `~` are dropped, and the Leica prefixes `DO0`/`DL0`/`DL`/`L` are stripped when followed by digits (so `DO01001336.jpg` and `L1001336.dng` share the base `1001336`).
1. A DNG whose group contains no `.jpg`/`.jpeg` file is marked for trash, unless it already sits in an Immich stack that contains a JPG.

### Safety

- Assets are moved to trash (`force=false`), not deleted permanently. They can be restored from Immich's trash until Immich empties it. This tool has no undo command.
- If more than 10% of your active assets are about to be trashed, a warning is logged before the summary. The command does not stop — it is designed to run unattended.
- Deletion requests are sent in batches of 1000 assets.

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

```
🗑️  Found 5 trashed assets
🔍 Analyzing 5 trashed assets against 1000 active assets...
🔄 Skipped 2 trashed assets that appear to have been replaced
🔍 Looking for orphaned DNG files...
📸 Found 2 orphaned DNG files without corresponding JPG files
✅ Skipped 1 DNG files that are already in stacks with JPG files
📋 Summary of assets to trash (4):
	📸 Orphaned DNG files (no JPG found): L1000746.dng, L1000901.dng
	IMG_1234.jpg (in trash): IMG_1234.dng, IMG_1234~2.jpg
🗑️  Moving 4 assets to trash... done
```

With `--dry-run`, the last line becomes `🗑️  Moving 4 assets to trash... (dry run)` and nothing is modified. When nothing needs to move, the command prints `✅ No related assets need to be trashed.`

In debug mode, you additionally get the base-name normalization of every asset, the per-asset cascade decisions, and a count of assets to trash by file type.

## Flags

The command uses all global flags, particularly:

- `--dry-run` - Preview what would be deleted without making changes
- `--criteria` - Custom stacking criteria (uses same format as main command)
- `--parent-filename-promote` - Filename patterns for stacking
- `--with-archived` - Also look for archived assets in the trash scan
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

1. **Uses Stacking Criteria**: The command uses the same criteria as the main stacking command, so the cascade matches what the stacker would group.
1. **The DNG pass always runs**: There is currently no flag to disable the orphaned-DNG cleanup — see the warning at the top if your library contains DNGs without JPG companions.
1. **Trash, not deletion**: Assets go to Immich's trash and can be restored there; this tool has no undo command.
1. **Safety First**: Always use `--dry-run` first to preview changes.

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
