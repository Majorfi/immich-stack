# Fix-Trash Command

The `fix-trash` command ensures stack consistency by moving related assets to trash when their stack members have been deleted.

## Overview

When you delete a photo in Immich that would stack with other photos (for example the JPG of a RAW+JPEG pair), the related assets stay behind in your library. This command finds them and moves them to trash too.

The command runs two passes:

1. **Stack cascade** — active assets that would stack with a trashed asset are moved to trash.
1. **Orphaned RAW cleanup** (opt-in via `--trash-orphaned-raws`) — active RAW files (DNG, NEF, ARW, CR2/CR3, ...) with no developed companion (JPG, HEIC, or any non-RAW file) are moved to trash, independently of what is in your trash.

!!! warning "RAW-only libraries"

    With `--trash-orphaned-raws` enabled, every RAW file without a developed companion is
    treated as orphaned, regardless of your trash contents. If you shoot RAW-only (no
    JPG/HEIC sidecars), this pass will flag those files for trash — restrict it with
    `--raw-orphan-extensions` (e.g. `dng`) if only part of your library pairs RAW with
    developed files. Always run with `--dry-run` first and review the summary.

## How It Works

### Pass 1: stack cascade

1. Fetches your trashed assets and your active assets. Partner-owned assets are dropped from both lists.
1. Skips trashed assets that appear to have been re-uploaded: if an active asset with the same filename has a newer created/modified/updated timestamp, cascading from the old copy would drag the new copy's companions into the trash.
1. Runs the stacking algorithm once over the remaining trashed assets plus all active assets, using the same criteria as the main command (`--criteria`, `--parent-filename-promote`, `--parent-ext-promote`).
1. Every active asset that lands in a group containing a trashed asset is marked for trash.

### Pass 2: orphaned RAW cleanup (opt-in)

This pass only runs with `--trash-orphaned-raws` (or `TRASH_ORPHANED_RAWS=true`).

1. Groups your active assets with the same stacking criteria as pass 1, so filename matching follows your `--criteria` configuration. Cameras that pair a RAW and a JPG under different names (for example Leica's `L1001336.dng` + `DO01001336.jpg`) need a regex criterion that maps both to the same key, such as `{"key":"originalFileName","regex":{"key":"^(?:L|DO0)(\\d+)","index":1}}`.
1. A RAW file whose group contains no developed file — or that groups with nothing at all — is marked for trash, unless it already sits in an Immich stack that contains a developed file.
1. `--raw-orphan-extensions` (or `RAW_ORPHAN_EXTENSIONS`) restricts which RAW extensions may be flagged, e.g. `dng` to clean up orphaned DNGs while never touching NEF/ARW/... files from a RAW-only workflow. It only restricts the candidates: another RAW file never counts as a developed companion, whatever the restriction. Unknown extensions are ignored with a warning.

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
🔍 Looking for orphaned RAW files...
📸 Found 2 orphaned RAW files without a developed companion
✅ Kept 1 RAW files already stacked with a developed file
📋 Summary of assets to trash (4):
	📸 Orphaned RAW files (no developed companion): L1000746.dng, L1000901.dng
	IMG_1234.jpg (in trash): IMG_1234.dng, IMG_1234~2.jpg
🗑️  Moving 4 assets to trash... done
```

This example runs with `--trash-orphaned-raws`; without it, the RAW lines are replaced by `⏭️  Orphaned RAW cleanup disabled (enable with --trash-orphaned-raws)`. With `--dry-run`, the last line becomes `🗑️  Moving 4 assets to trash... (dry run)` and nothing is modified. When nothing needs to move, the command prints `✅ No related assets need to be trashed.`

In debug mode, you additionally get the per-asset cascade and orphan decisions, and a count of assets to trash by file type.

## Flags

The command uses all global flags, particularly:

- `--dry-run` - Preview what would be deleted without making changes
- `--criteria` - Custom stacking criteria (uses same format as main command)
- `--parent-filename-promote` - Filename patterns for stacking
- `--trash-orphaned-raws` - Enable the orphaned RAW cleanup pass (off by default)
- `--raw-orphan-extensions` - Restrict the RAW pass to specific extensions, e.g. `dng,nef` (default: all RAW formats)
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

1. **Uses Stacking Criteria**: Both passes use the same criteria as the main stacking command, so the cascade and the orphan matching follow what the stacker would group.
1. **The RAW pass is opt-in**: The orphaned-RAW cleanup only runs with `--trash-orphaned-raws` — see the warning at the top before enabling it on a library that contains RAW files without developed companions.
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
