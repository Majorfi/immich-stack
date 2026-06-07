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

### Large Library: 5xx on `GET /stacks`

**Symptoms:**

```
level=warning msg="⚠️  GET /stacks failed: error response: 500 Internal Server Error - {...}"
level=warning msg="    A 500 here is typically the JSON.stringify limit on large libraries (Immich issue #15332)."
level=warning msg="    Falling back to per-asset hybrid lookup. Slower but bypasses the failing endpoint."
```

**Cause:**

The Immich `GET /stacks` endpoint is unpaginated. On libraries with roughly >100k stacks, the
JSON response exceeds Node.js's max string length (~512 MB) and the Immich server crashes at
`JSON.stringify`, returning HTTP 500. See [immich-app/immich#15332](https://github.com/immich-app/immich/issues/15332).

**What the tool does:**

`FetchAllStacks` detects 5xx responses and automatically falls back to a per-asset hybrid
lookup that bypasses the failing endpoint:

1. Enumerate all asset IDs via paginated `POST /search/metadata`
2. Phase 1 — parallel `GET /assets/{id}` per asset to read each asset's stack reference
3. Phase 2 — for assets where Phase 1 reported "no stack" (typically archived primaries that
   Immich strips from `/assets/{id}` responses), call `GET /stacks?primaryAssetId=X` to
   recover the missing stack metadata

The result is a stacks map equivalent to what `GET /stacks` would have returned, reconstructed
from many small HTTP calls instead of one giant one. No user action is required.

**Performance implications:**

- Healthy responses on small/medium libraries are unaffected — `/stacks` is still the primary
  path (~1 s for ~6k stacks).
- When the fallback fires, expect on the order of `total_assets / 800` seconds with the default
  concurrency of 10 in-flight requests. For 450k assets, that's roughly 10 minutes per run.
- Transient nginx upstream errors (502/503/504) during the fallback are retried automatically.
  Persistent failures surface as a `PartialResultError` that the caller treats as a soft
  failure (returns the partial map and continues).

**When to be concerned:**

- A 4xx (401, 403, 404) on `/stacks` is **not** the JSON-limit bug — those propagate as
  hard errors with no fallback. Check your API key and URL.
- If the warning fires repeatedly on a small library, the underlying Immich server may be
  unhealthy for another reason. Check Immich server logs.

### Partner Sharing Assets Skipped

**Symptoms:**

```
level=info msg="⏭️  Skipped 8755 assets owned by partners (not stackable via API)"
```

**Cause:**

When a partner has shared their library with you and you have "Show in timeline" enabled
for that share, Immich's `/search/metadata` endpoint surfaces the partner's assets in your
timeline. immich-stack used to attempt to stack those assets and get rejected by Immich with
permission errors (the stack write API requires `AssetUpdate` permission, which you don't
have on partner-owned assets).

**What the tool does:**

`GET /users/me`) and drops anything you don't own. When any assets are dropped, an info-level
log line reports the count (see the symptom above). When nothing is dropped (the normal
case if you have no incoming partner shares), no log line is emitted. Either way, only owned
assets reach the stacking pipeline, so no partner-related write attempts ever leave the
client.

**When to be concerned:**

- A skip count that surprises you may indicate an unexpected partner share — check Immich's
  Account Settings → Partner Sharing to confirm what's coming in.
- A skip count of `0` is normal when you have no incoming partner shares or have `Show in
timeline` disabled.

This resolves [issue #55](https://github.com/Majorfi/immich-stack/issues/55).

### Stacking Video Files

**Symptoms:**

- Stacking criteria match but `.mov`/`.mp4`/other video files are never picked up
- Live Photos (`.HEIC` + `.MOV` pairs) only stack the photo, not the motion file
- Edited videos (trimmed, cropped) cannot be stacked with their originals

**Cause:**

By default, immich-stack restricts `/search/metadata` calls to `type=IMAGE`. Video assets
are excluded from the candidate pool entirely, regardless of whether your stacking criteria
would otherwise match them.

**Solution:**

Enable `INCLUDE_VIDEOS=true` (or the CLI flag `--include-videos`). When set, every asset
fetch runs twice (once for `IMAGE`, once for `VIDEO`) and results are deduplicated.
Existing stacking criteria (filename patterns, time deltas, regex, etc.) work on videos
the same way they work on images.

```sh
INCLUDE_VIDEOS=true
```

**Examples of use cases this unlocks:**

- iPhone Live Photos: `IMG_1234.HEIC` paired with `IMG_1234.MOV` (same base name)
- Trimmed/edited videos paired with the original file
- Android burst videos with identical timestamps

**Performance implications:**

- A second pagination round runs for VIDEO. Wall-clock latency increases by the time the
  VIDEO scan takes — proportional to how many videos you have. A library that's mostly
  images sees a small bump; a library with many videos sees more.
- The `OTHER` and `AUDIO` Immich types are still excluded — only `IMAGE` and `VIDEO` are
  pulled in.

This resolves [issue #54](https://github.com/Majorfi/immich-stack/issues/54).

### Slow Stacking on Large Libraries

**Symptoms:**

- A single stacking run takes 30 minutes or more on a library with thousands of stacks
- CPU usage during the run is near 0% (the tool is waiting, not computing)
- Reset operations (`RESET_STACKS=true`) take comparable time on big libraries

**Cause:**

Historically the tool inserted a 100 ms pause before every `POST /stacks` call to avoid
hammering Immich. For libraries with 10k+ stacks, that single pause dominated wall-clock
time (e.g., 21 000 stacks × 100 ms ≈ 35 minutes of pure sleep). See [issue #53](https://github.com/Majorfi/immich-stack/issues/53).

**What changed:**

- The preemptive sleep is no longer applied by default — empirically Immich has no rate
  limit on `POST /stacks` and handles bursts of hundreds of requests per second cleanly.
- Stack writes can now run in parallel via `STACK_CONCURRENCY` (default `1`, i.e., sequential).
- Both the main stacking loop AND the `RESET_STACKS` / `REMOVE_SINGLE_ASSET_STACKS` cleanup
  paths respect the same concurrency setting.

**Configuration:**

```sh
# Default — sequential writes, no artificial delay
STACK_CONCURRENCY=1

# Recommended for libraries above 10k stacks — 10× speedup typical
STACK_CONCURRENCY=10

# Aggressive — use only if your Immich host is sized for it
STACK_CONCURRENCY=20

# Safety throttle — opt in if you observe upstream errors on a slow host
# (inserts a 50 ms pause before each write; combinable with STACK_CONCURRENCY)
PREVENT_SELF_REKT=true
```

**Expect interleaved logs when `STACK_CONCURRENCY > 1`:**

Each individual log line stays intact (the logger is goroutine-safe), but the per-stack
sequence ("1/N Key: …", "Parent …", "Child …", "Creating new stack") is no longer printed
together for a given stack — lines from different stacks interleave. This is the visible
trade-off for the speedup. Drop back to `STACK_CONCURRENCY=1` if you need sequential logs
for debugging a specific stack.

**When to enable `PREVENT_SELF_REKT`:**

- Self-hosted Immich on a single low-power machine (e.g., Raspberry Pi) where the database
  starts queueing under sustained writes
- You see repeated `502 Bad Gateway` errors during stack writes
- You're combining `STACK_CONCURRENCY > 10` with a constrained host and want a per-write
  cooldown

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
