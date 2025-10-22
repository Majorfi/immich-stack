# Edited Photo Promotion

This document explains how to properly configure Immich Stack to promote edited photos (with `~` or `.` suffixes) over original photos.

## Problem Description

When you edit a photo and save it with a numeric suffix (like `~2` for the second version), the edited version should typically be preferred over the original. For example:

- Original photo: `PXL_20250823_193751711.jpg`
- Edited photo: `PXL_20250823_193751711~2.jpg`

By default, without proper configuration, the original photo might be promoted as the stack parent instead of the edited version.

## Solution

To ensure edited photos are promoted over originals, you need to include `biggestNumber` in your `PARENT_FILENAME_PROMOTE` configuration.

### Configuration Options

#### Option 1: Use Default Configuration (Recommended)

The default configuration already includes `biggestNumber`:

```bash
# Default value (no need to set if using defaults)
PARENT_FILENAME_PROMOTE=cover,edit,crop,hdr,biggestNumber
```

If you're not setting `PARENT_FILENAME_PROMOTE` explicitly, the defaults will handle edited photos correctly.

#### Option 2: Explicit Configuration

If you're customizing the promote list, ensure you include `biggestNumber`:

```bash
# Example custom configuration that handles edited photos
PARENT_FILENAME_PROMOTE=cover,edit,biggestNumber
```

#### Option 3: Only Prioritize Edited Photos

If you only care about promoting edited photos with numeric suffixes:

```bash
PARENT_FILENAME_PROMOTE=biggestNumber
```

## How It Works

The `biggestNumber` keyword tells the stacker to:

1. Split filenames by delimiters (default: `~` and `.`)
2. Look for numeric suffixes in the last part
3. Promote files with higher numbers first

### Example Sorting

With `PARENT_FILENAME_PROMOTE=biggestNumber`:

```
Files:
- PXL_20250823_193751711.jpg
- PXL_20250823_193751711~2.jpg
- PXL_20250823_193751711~3.jpg
- PXL_20250823_193751711~5.jpg

Sorted order (parent first):
1. PXL_20250823_193751711~5.jpg  (highest edit)
2. PXL_20250823_193751711~3.jpg
3. PXL_20250823_193751711~2.jpg
4. PXL_20250823_193751711.jpg     (original)
```

## Common Configurations

### For Photos with Numeric Edits

```bash
# Prioritizes edits, crops, HDR, and then numbered versions
PARENT_FILENAME_PROMOTE=cover,edit,crop,hdr,biggestNumber
```

### For RAW+JPEG with Edits

```bash
# Prioritizes edited JPEGs over everything
PARENT_FILENAME_PROMOTE=biggestNumber
PARENT_EXT_PROMOTE=.jpg,.jpeg,.dng,.raw
```

### Mixed Priority

```bash
# COVER files first, then edited versions, then regular edits
PARENT_FILENAME_PROMOTE=COVER,biggestNumber,edit
```

## Verification

To verify your configuration is working:

1. Run with `--dry-run` flag to see what would happen:

```bash
immich-stack --dry-run --parent-filename-promote=biggestNumber
```

2. Check the logs for parent selection:

```
[INFO] Stack created with parent: PXL_20250823_193751711~2.jpg
```

## Troubleshooting

### Edited photos not being promoted?

1. **Check your configuration**: Ensure `biggestNumber` is in your `PARENT_FILENAME_PROMOTE` list
2. **Check delimiters**: The default delimiters are `~` and `.`. If your edited photos use different separators, you may need to adjust
3. **Check for conflicts**: If you have other promote patterns that match before `biggestNumber`, they take priority

### Want originals promoted instead?

Simply remove `biggestNumber` from your promote list:

```bash
PARENT_FILENAME_PROMOTE=edit,crop,hdr
```

## Technical Details

The `biggestNumber` feature:

- Splits filenames by delimiters to find numeric parts (default delimiters: `~` and `.`)
- Compares numeric values (not string comparison)
- Works with numeric suffixes after the delimiters (e.g., `photo~2`, `image.3`)
- Falls back to alphabetical sorting if no numbers are found

Note: The delimiters are configurable via the `CRITERIA` environment variable if you need to use different separators like `_` or `-`.
