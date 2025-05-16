# Stacking Logic

## Grouping

- **Default Criteria:** Groups by base filename (before extension) and local capture time
- **Custom Criteria:** Override with the `--criteria` flag or `CRITERIA` environment variable

## Sorting

- **Parent Promotion:** Use `--parent-filename-promote` or `PARENT_FILENAME_PROMOTE` (comma-separated substrings) to promote files as stack parents
- **Extension Promotion:** Use `--parent-ext-promote` or `PARENT_EXT_PROMOTE` (comma-separated extensions) to further prioritize
- **Extension Rank:** Built-in priority: `.jpeg` > `.jpg` > `.png` > others
- **Alphabetical:** Final tiebreaker

## Example

For files: `L1010229.JPG`, `L1010229.edit.jpg`, `L1010229.DNG`

With `PARENT_FILENAME_PROMOTE=edit` and `PARENT_EXT_PROMOTE=.jpg,.dng` in your .env file, or with `--parent-filename-promote=edit` and `--parent-ext-promote=.jpg,.dng`, the order will be:

```
L1010229.edit.jpg
L1010229.JPG
L1010229.DNG
```

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
