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
