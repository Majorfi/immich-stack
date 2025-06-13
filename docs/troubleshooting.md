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
2. Check API key validity
   ```sh
   API_KEY=your_valid_api_key
   ```
3. Ensure network connectivity
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
2. Check stack criteria
   ```sh
   CRITERIA='[{"key":"originalFileName","split":{"delimiters":["~","."],"index":0}}]'
   ```
3. Verify asset data
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
2. Check parent selection
   ```sh
   PARENT_FILENAME_PROMOTE=edit,raw
   PARENT_EXT_PROMOTE=.jpg,.dng
   ```
3. Enable debug logging
   ```sh
   LOG_LEVEL=debug
   ```

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

2. Use comma-separated numeric sequences for burst photos (Legacy)

   ```sh
   PARENT_FILENAME_PROMOTE=0000,0001,0002,0003
   ```

   The system will automatically detect this as a sequence and order photos correctly.

3. The sequence detection works with various patterns:

   ```sh
   # Pure numbers
   PARENT_FILENAME_PROMOTE=0000,0001,0002,0003

   # Prefixed numbers
   PARENT_FILENAME_PROMOTE=IMG_0001,IMG_0002,IMG_0003

   # Suffixed numbers
   PARENT_FILENAME_PROMOTE=1a,2a,3a
   ```

4. Files with numbers beyond your promote list are handled automatically:

   - If you specify `0000,0001,0002,0003` but have files up to `0999`, they will be sorted correctly at position 999

5. Understanding `sequence:X` behavior:
   - `sequence` - Matches any numeric sequence (1, 2, 10, 100, etc.)
   - `sequence:4` - Matches ONLY 4-digit numbers (0001, 0002, not 1, 10, 100)
   - `sequence:IMG_` - Matches only files with IMG\_ prefix followed by numbers

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
2. Check interval setting
   ```sh
   CRON_INTERVAL=3600
   ```
3. Monitor logs
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

2. Test with minimal criteria

   ```sh
   CRITERIA='[{"key":"originalFileName"}]'
   ```

3. Verify API connection
   ```sh
   curl -I $API_URL
   ```

## Performance Issues

### High Memory Usage

**Solutions:**

1. Process fewer assets at once
2. Use more specific criteria
3. Enable pagination

### Slow Processing

**Solutions:**

1. Optimize criteria
2. Use appropriate delta values
3. Consider batch processing

## Best Practices

1. **Testing**

   - Always use dry run mode first
   - Test with small asset sets
   - Verify criteria before production

2. **Monitoring**

   - Enable debug logging
   - Monitor resource usage
   - Check operation results

3. **Maintenance**

   - Regular stack cleanup
   - API key rotation
   - Configuration review

4. **Security**
   - Secure API keys
   - Regular updates
   - Access control
