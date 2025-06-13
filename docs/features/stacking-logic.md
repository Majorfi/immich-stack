# Stacking Logic

## Grouping

- **Default Criteria:** Groups by base filename (before extension) and local capture time
- **Custom Criteria:** Override with the `--criteria` flag or `CRITERIA` environment variable

## Sorting

- **Parent Promotion:** Use `--parent-filename-promote` or `PARENT_FILENAME_PROMOTE` (comma-separated substrings) to promote files as stack parents
- **Sequence Keyword:** Use the `sequence` keyword for flexible sequential file handling (e.g., `sequence`, `sequence:4`, `sequence:IMG_`)
- **Sequence Detection:** Automatically detects numeric sequences in promote lists (e.g., `0000,0001,0002`) and uses intelligent matching for burst photos
- **Extension Promotion:** Use `--parent-ext-promote` or `PARENT_EXT_PROMOTE` (comma-separated extensions) to further prioritize
- **Extension Rank:** Built-in priority: `.jpeg` > `.jpg` > `.png` > others
- **Alphabetical:** Final tiebreaker

## Examples

### Standard Promotion Example

For files: `L1010229.JPG`, `L1010229.edit.jpg`, `L1010229.DNG`

With `PARENT_FILENAME_PROMOTE=edit` and `PARENT_EXT_PROMOTE=.jpg,.dng` in your .env file, or with `--parent-filename-promote=edit` and `--parent-ext-promote=.jpg,.dng`, the order will be:

```
L1010229.edit.jpg
L1010229.JPG
L1010229.DNG
```

### Burst Photo Sequence Example

For burst photo files: `DSCPDC_0000_BURST20180828114700954.JPG`, `DSCPDC_0001_BURST20180828114700954.JPG`, `DSCPDC_0002_BURST20180828114700954.JPG`, `DSCPDC_0003_BURST20180828114700954_COVER.JPG`

With `PARENT_FILENAME_PROMOTE=0000,0001,0002,0003`, the system automatically detects this as a numeric sequence and orders them correctly:

```
DSCPDC_0000_BURST20180828114700954.JPG
DSCPDC_0001_BURST20180828114700954.JPG
DSCPDC_0002_BURST20180828114700954.JPG
DSCPDC_0003_BURST20180828114700954_COVER.JPG
```

The sequence detection works even with numbers beyond your promote list. For example, if you have `PARENT_FILENAME_PROMOTE=0000,0001,0002,0003` but your files include `DSCPDC_0999_BURST...`, it will be sorted at position 999 automatically.

### Sequence Keyword Examples

The `sequence` keyword provides powerful and flexible sequence handling:

```sh
# Order any numeric sequence (1, 2, 10, 100, etc.)
PARENT_FILENAME_PROMOTE=sequence

# Order only 4-digit sequences (0001, 0002, 0010, 0100, etc.)
PARENT_FILENAME_PROMOTE=sequence:4

# Order sequences with specific prefix
PARENT_FILENAME_PROMOTE=sequence:IMG_

# Mix with other promote values - COVER files first, then sequence
PARENT_FILENAME_PROMOTE=COVER,sequence

# Multiple criteria - edited files first, then sequences
PARENT_FILENAME_PROMOTE=edit,hdr,sequence
```

### Supported Sequence Patterns (Legacy Method)

The system can detect various sequence patterns when using comma-separated numbers:

- Pure numbers: `0000,0001,0002,0003`
- Prefixed numbers: `img1,img2,img3` or `IMG_0001,IMG_0002,IMG_0003`
- Suffixed numbers: `1a,2a,3a` or `001_final,002_final,003_final`
- Complex patterns: `photo_001_v2,photo_002_v2,photo_003_v2`

**Note:** The `sequence` keyword is more flexible and recommended over listing individual sequence numbers.

## Stacking Process

1. **Fetch all stacks and assets** from Immich
2. **Group assets** into stacks using criteria
3. **Sort each stack** to determine the parent and children
4. **Apply changes** via the Immich API (create, update, or delete stacks as needed)
5. **Log all actions** and optionally run in dry-run mode for safety

## Safe Operations

The stacker includes several safety features:

- **Dry Run Mode:** Use `--dry-run` or `DRY_RUN=true` to simulate actions without making changes
- **Stack Replacement:** Use `--replace-stacks` or `REPLACE_STACKS=true` to replace existing stacks
- **Stack Reset:** Use `--reset-stacks` or `RESET_STACKS=true` with confirmation to delete all stacks
- **Confirmation Required:** Stack reset requires explicit confirmation via `CONFIRM_RESET_STACK`

## Logging

The stacker provides comprehensive logging:

- Colorful, structured logs for all operations
- Clear indication of actions taken
- Error reporting with context
- Progress updates for long-running operations
