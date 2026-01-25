# Troubleshooting Guide

This guide helps you resolve common issues with Immich Stack.

## Common Issues

### API Connection Issues

**Symptoms:**

- "Failed to connect to Immich API"
- "API request failed"
- "Invalid API key"

**Solutions:**

1. Verify API URL is correct
   ```sh
   API_URL=http://immich-server:2283/api
   ```
1. Check API key validity
   ```sh
   API_KEY=your_valid_api_key
   ```
1. Ensure network connectivity
   ```sh
   curl -I http://immich-server:2283/api
   ```

### Stack Creation Issues

**Symptoms:**

- "Failed to create stack"
- "Invalid stack data"
- "Stack already exists"

**Solutions:**

1. Enable dry run mode to test
   ```sh
   DRY_RUN=true
   ```
1. Check stack criteria
   ```sh
   CRITERIA='[{"key":"originalFileName","split":{"delimiters":["~","."],"index":0}}]'
   ```
1. Verify asset data
   ```sh
   WITH_ARCHIVED=true
   WITH_DELETED=false
   ```

### Grouping Issues

**Symptoms:**

- "Invalid grouping criteria"
- "No assets grouped"
- "Unexpected grouping results"

**Solutions:**

1. Review criteria configuration
   ```sh
   CRITERIA='[{"key":"localDateTime","delta":{"milliseconds":1000}}]'
   ```
1. Check parent selection
   ```sh
   PARENT_FILENAME_PROMOTE=edit,raw
   PARENT_EXT_PROMOTE=.jpg,.dng
   ```
1. Enable debug logging
   ```sh
   LOG_LEVEL=debug
   ```

### Infinite Re-stacking Loop (Issue #35)

**Fixed in**: Commit 2c3a75a (November 1, 2025)

**Symptoms:**

- Same assets processed repeatedly across runs
- Different queue positions for same asset IDs (e.g., 338/4275, then 772/4278)
- False "Success! Stack created" messages for stacks that already exist
- Cron mode infinite loop on same subset of photos
- No progress through entire photo library
- Stack count remains static across runs

**Root Cause:**

The stacksMap only indexed PRIMARY assets of each stack, not all child assets. When checking if an asset was already stacked, child assets were not found, causing the tool to repeatedly attempt to restack them.

**Resolution:**

The fix changed stack indexing from:

```go
// Old: Only indexed primary asset
stacksMap[stack.PrimaryAssetID] = stack
```

To:

```go
// New: Index ALL assets in the stack
for _, asset := range stack.Assets {
    stacksMap[asset.ID] = stack
}
```

**Verification:**

If you experienced this issue, update to the latest version and verify:

1. Check logs no longer show same asset IDs repeatedly
1. Stack count should increase steadily across runs
1. Queue positions should progress sequentially
1. "Success! Stack created" should only appear for genuinely new stacks

**Affected Users:**

- Large libraries (50k+ assets)
- Google Pixel camera files (RAW-01.COVER.jpg / RAW-02.ORIGINAL.dng patterns)
- Users running in cron mode with frequent intervals

**Related:**

- GitHub Issue: #35
- Commit: 2c3a75a

### Burst Photo Ordering Issues

**Symptoms:**

- Burst photos not ordered correctly (e.g., 0000, 0002, 0003, 0001 instead of 0000, 0001, 0002, 0003)
- Numeric promote strings matching in wrong places (e.g., "0001" matching in timestamps)
- Need to handle sequences with varying number of digits (1, 10, 100)

**Solutions:**

1. Use the `sequence` keyword for flexible sequence handling (Recommended)

   ```sh
   # Order any numeric sequence regardless of digits
   PARENT_FILENAME_PROMOTE=sequence

   # Prioritize COVER files, then order by sequence
   PARENT_FILENAME_PROMOTE=COVER,sequence

   # Only match 4-digit sequences (0001, 0002, etc.)
   PARENT_FILENAME_PROMOTE=sequence:4

   # Only match sequences with specific prefix
   PARENT_FILENAME_PROMOTE=sequence:IMG_
   ```

1. Use comma-separated numeric sequences for burst photos (Legacy)

   ```sh
   PARENT_FILENAME_PROMOTE=0000,0001,0002,0003
   ```

   The system will automatically detect this as a sequence and order photos correctly.

1. The sequence detection works with various patterns:

   ```sh
   # Pure numbers
   PARENT_FILENAME_PROMOTE=0000,0001,0002,0003

   # Prefixed numbers
   PARENT_FILENAME_PROMOTE=IMG_0001,IMG_0002,IMG_0003

   # Suffixed numbers
   PARENT_FILENAME_PROMOTE=1a,2a,3a
   ```

1. Files with numbers beyond your promote list are handled automatically:

   - If you specify `0000,0001,0002,0003` but have files up to `0999`, they will be sorted correctly at position 999

1. Understanding `sequence:X` behavior:

   - `sequence` - Matches any numeric sequence (1, 2, 10, 100, etc.)
   - `sequence:4` - Matches ONLY 4-digit numbers (0001, 0002, not 1, 10, 100)
   - `sequence:IMG_` - Matches only files with IMG\_ prefix followed by numbers

### Stack Recovery Procedures

**When to Use:**

- After failed stack operations
- When migrating between criteria
- After database issues
- When cleaning up corrupted stacks

**Complete Stack Reset:**

```sh
# CAUTION: This will delete ALL existing stacks
RUN_MODE=once
RESET_STACKS=true
CONFIRM_RESET_STACK="I acknowledge all my current stacks will be deleted and new one will be created"

# Run the stacker
./immich-stack
```

**Important Notes:**

- RESET_STACKS only works with RUN_MODE=once
- Using RESET_STACKS in cron mode results in an error
- Confirmation text must match exactly
- Always test with DRY_RUN=true first

**Recovering from Partial Failures:**

1. Enable replace stacks mode to fix existing stacks:

   ```sh
   REPLACE_STACKS=true
   DRY_RUN=false
   ```

1. Remove single-asset stacks (cleanup):

   ```sh
   REMOVE_SINGLE_ASSET_STACKS=true
   ```

1. Process incrementally with filters:

   ```sh
   WITH_ARCHIVED=false
   WITH_DELETED=false
   ```

**Safe Recovery Workflow:**

1. First, run in dry-run mode to preview changes:

   ```sh
   DRY_RUN=true
   REPLACE_STACKS=true
   LOG_LEVEL=debug
   ```

1. Review the logs carefully to verify expected behavior

1. Execute the actual operation:

   ```sh
   DRY_RUN=false
   REPLACE_STACKS=true
   ```

1. Monitor logs for errors:

   ```sh
   docker logs -f immich-stack
   ```

**Rolling Back Changes:**

If you need to revert to a previous state:

1. Use Immich database backups (if available)
1. Run complete reset with previous criteria configuration
1. Manually adjust stacks via Immich UI for specific cases

### Cron Mode Issues

**Symptoms:**

- "Cron job not running"
- "Invalid interval"
- "Unexpected execution"

**Solutions:**

1. Verify run mode
   ```sh
   RUN_MODE=cron
   ```
1. Check interval setting
   ```sh
   CRON_INTERVAL=3600
   ```
1. Monitor logs
   ```sh
   LOG_LEVEL=debug
   LOG_FORMAT=json
   ```

## Debugging

### Enable Debug Logging

```sh
LOG_LEVEL=debug
LOG_FORMAT=json
```

### Check Logs

```sh
# View logs
docker logs immich-stack

# Follow logs
docker logs -f immich-stack
```

### Test Configuration

1. Use dry run mode

   ```sh
   DRY_RUN=true
   ```

1. Test with minimal criteria

   ```sh
   CRITERIA='[{"key":"originalFileName"}]'
   ```

1. Verify API connection

   ```sh
   curl -I $API_URL
   ```

## Performance Issues

### High Memory Usage

**Solutions:**

1. Process fewer assets at once
1. Use more specific criteria
1. Enable pagination

### Slow Processing

**Solutions:**

1. Optimize criteria
1. Use appropriate delta values
1. Consider batch processing

## Best Practices

1. **Testing**

   - Always use dry run mode first
   - Test with small asset sets
   - Verify criteria before production

1. **Monitoring**

   - Enable debug logging
   - Monitor resource usage
   - Check operation results

1. **Maintenance**

   - Regular stack cleanup
   - API key rotation
   - Configuration review

1. **Security**

   - Secure API keys
   - Regular updates
   - Access control
