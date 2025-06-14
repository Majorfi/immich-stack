# Commands Overview

Immich Stack provides multiple commands for different photo management operations.

## Available Commands

### Main Stacking Command

```bash
immich-stack [flags]
```

The default command that processes your photo library and creates stacks based on your configured criteria. This is the primary functionality of the tool.

[Learn more about stacking →](../getting-started/quick-start.md)

### Duplicates Detection

```bash
immich-stack duplicates [flags]
```

Finds and reports duplicate assets in your library based on filename and timestamp matching.

[Full documentation →](duplicates.md)

### Fix Trash Operations

```bash
immich-stack fix-trash [flags]
```

Maintains stack consistency by moving related assets to trash when their stack members have been deleted.

[Full documentation →](fix-trash.md)

## Common Workflows

### 1. Initial Library Organization

```bash
# First, check for duplicates
immich-stack duplicates --api-key your_key

# Then create stacks
immich-stack --api-key your_key --dry-run

# If satisfied, run without dry-run
immich-stack --api-key your_key
```

### 2. Regular Maintenance

```bash
# Weekly maintenance routine
immich-stack duplicates --api-key your_key
immich-stack fix-trash --api-key your_key
immich-stack --api-key your_key
```

### 3. Post-Deletion Cleanup

```bash
# After deleting photos in Immich UI
immich-stack fix-trash --api-key your_key --dry-run
immich-stack fix-trash --api-key your_key
```

## Global Flags

All commands share these common flags:

- `--api-key` - Immich API key (required)
- `--api-url` - Immich server URL
- `--dry-run` - Preview changes without applying
- `--log-level` - Set logging verbosity
- `--with-archived` - Include archived assets
- `--with-deleted` - Include deleted assets

See [CLI Usage](../api-reference/cli-usage.md) for complete flag documentation.

## Best Practices

1. **Always use dry-run first** - Preview changes before applying them
2. **Regular maintenance** - Run commands periodically to keep library organized
3. **Check duplicates before stacking** - Avoid creating stacks with duplicate assets
4. **Fix trash after deletions** - Maintain consistency when deleting stacked photos

## See Also

- [Environment Variables](../api-reference/environment-variables.md) - Configure via environment
- [Stacking Logic](../features/stacking-logic.md) - Understand grouping criteria
- [Custom Criteria](../features/custom-criteria.md) - Advanced configuration
