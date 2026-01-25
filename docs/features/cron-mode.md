# Cron Mode

Cron mode enables continuous, automated stacking operations that run periodically without manual intervention. This mode is ideal for production deployments where you want to keep stacks synchronized automatically.

## Configuration

Enable cron mode with environment variables or CLI flags:

```sh
RUN_MODE=cron
CRON_INTERVAL=3600  # Interval in seconds (1 hour)
```

Or using CLI flags:

```sh
./immich-stack --run-mode=cron --cron-interval=3600
```

## How It Works

### Execution Loop

When cron mode is enabled:

1. Application starts and immediately runs the first stacking operation
1. After completion, waits for `CRON_INTERVAL` seconds
1. Runs the next stacking operation
1. Repeats indefinitely until stopped

```
[Start] → [Run Stacker] → [Wait CRON_INTERVAL] → [Run Stacker] → [Wait] → ...
```

### State Management

**Important**: Cron mode is **stateless** between runs. Each execution:

- Fetches fresh data from Immich API
- Recalculates all groupings from scratch
- Makes independent stacking decisions
- Does not remember previous runs

This stateless design ensures:

- Resilience to Immich API changes
- Self-healing from transient errors
- Consistency with manually created stacks
- No risk of state corruption

### Timing Behavior

The interval timer starts **after** each run completes, not from the start time:

```
Run 1: [12:00:00 - 12:02:15] → Wait 3600s → Run 2: [13:02:15 - 13:04:30] → Wait 3600s → ...
```

**If a run takes longer than the interval**:

- The next run starts immediately after completion
- No runs are skipped
- A warning is logged if processing time exceeds 50% of the interval

Example with long processing:

```
CRON_INTERVAL=3600 (1 hour)

Run 1: 12:00:00 - 13:30:00 (90 minutes)
⚠️  Warning: Processing took 5400s, which exceeds interval of 3600s
Run 2: 13:30:00 - 15:00:00 (90 minutes)
⚠️  Warning: Processing took 5400s, which exceeds interval of 3600s
```

**Recommendation**: Set `CRON_INTERVAL` to at least 2× your expected processing time.

## Logging Behavior

### Structured Logging

Cron mode maintains the same logging format as once mode:

```sh
LOG_LEVEL=info    # Standard log level
LOG_FORMAT=text   # or "json" for structured logs
LOG_FILE=/app/logs/immich-stack.log  # Optional file logging
```

### Run Cycle Logging

Each cron cycle logs:

```
[12:00:00] INFO Starting cron cycle
[12:00:00] INFO Running for user: John Doe (john@example.com)
[12:00:05] INFO Processing 5,234 assets
...
[12:02:15] INFO Cron cycle completed in 2m 15s
[12:02:15] INFO Sleeping for 3600 seconds until next run
```

### Multi-User Logging

When using multiple API keys (comma-separated), each user is processed sequentially in each cycle:

```
API_KEYS=key1,key2,key3
```

Logs show clear separation:

```
[12:00:00] INFO Running for user: Alice (alice@example.com)
[12:01:30] INFO User Alice completed

[12:01:30] INFO Running for user: Bob (bob@example.com)
[12:03:00] INFO User Bob completed

[12:03:00] INFO Running for user: Carol (carol@example.com)
[12:04:15] INFO User Carol completed

[12:04:15] INFO Sleeping for 3600 seconds until next run
```

## Signal Handling

### Graceful Shutdown

Cron mode supports graceful shutdown via signals:

| Signal            | Behavior                                |
| ----------------- | --------------------------------------- |
| `SIGTERM`         | Completes current operation, then exits |
| `SIGINT` (Ctrl+C) | Completes current operation, then exits |
| `SIGKILL`         | Immediate termination (not graceful)    |

**Example graceful shutdown**:

```
[12:00:00] INFO Running cron cycle
[12:01:30] Received SIGTERM
[12:01:30] INFO Finishing current operation before shutdown
[12:02:15] INFO Operation completed
[12:02:15] INFO Shutting down gracefully
```

The application:

1. Receives the signal
1. Completes the current stacking operation
1. Does not start the next sleep cycle
1. Exits cleanly

**Note**: If you need immediate shutdown, use `SIGKILL` (not recommended):

```sh
docker kill -s SIGKILL immich-stack
```

### Docker Signal Handling

When running in Docker, ensure proper signal forwarding:

**Docker Compose** (recommended):

```yaml
services:
  immich-stack:
    image: majorfi/immich-stack:latest
    init: true # Ensures proper signal handling
    stop_grace_period: 5m # Allow time to finish current operation
```

**Docker run**:

```sh
docker run --init --stop-timeout 300 majorfi/immich-stack:latest
```

The `--init` flag ensures that the container properly forwards signals to the application.

## Operational Considerations

### Monitoring

Monitor cron mode health by:

1. **Log watching**: Track completion messages and error rates
1. **Process health**: Ensure container stays running
1. **API availability**: Verify Immich API is reachable
1. **Run duration**: Alert if processing time increases significantly

**Example monitoring script**:

```sh
# Check last successful run timestamp
docker logs immich-stack 2>&1 | grep "Cron cycle completed" | tail -1

# Alert if no completion in last 2 hours
if [ $(docker logs immich-stack 2>&1 | grep "Cron cycle completed" | tail -1 | cut -d' ' -f1) -lt $(date -d "2 hours ago" +%s) ]; then
  echo "ALERT: Cron hasn't completed in 2 hours"
fi
```

### Resource Usage

Cron mode resource usage patterns:

- **CPU**: Spikes during processing, idle during sleep
- **Memory**: Constant (holds asset data during processing)
- **Network**: Burst during API calls, idle during sleep
- **Disk**: Minimal (only for optional log files)

**Expected resource usage**:

| Library Size | Peak CPU | Memory | Network (per run) |
| ------------ | -------- | ------ | ----------------- |
| 10k assets   | 20-30%   | 200MB  | 50MB              |
| 50k assets   | 40-60%   | 800MB  | 250MB             |
| 100k assets  | 60-80%   | 1.5GB  | 500MB             |

### Error Handling

Cron mode is resilient to transient errors:

**Recoverable errors** (continues running):

- API connection failures
- Rate limiting (429 responses)
- Temporary network issues
- Invalid asset data

**Fatal errors** (stops running):

- Invalid API key
- Missing required configuration
- Out of memory
- Unrecoverable API errors

**Error logging**:

```
[12:00:00] ERROR Failed to fetch assets: connection timeout
[12:00:00] INFO Will retry in next cycle (3600s)
[13:00:00] INFO Retrying stacking operation
```

Errors are logged but don't stop the cron loop. The next cycle will retry the operation.

### Recommended Intervals

Choose `CRON_INTERVAL` based on your needs:

| Use Case                     | Recommended Interval | Reasoning                            |
| ---------------------------- | -------------------- | ------------------------------------ |
| Active photography studio    | 300-600s (5-10 min)  | Frequent uploads need quick stacking |
| Personal library             | 3600s (1 hour)       | Balance freshness and resource usage |
| Archival library             | 86400s (24 hours)    | Minimal changes, reduce API load     |
| Large library (100k+ assets) | 43200s (12 hours)    | Long processing time, avoid overlap  |

**Formula**: `CRON_INTERVAL = (expected_processing_time × 2) + buffer`

Example: If processing takes 10 minutes, set interval to at least 1800s (30 minutes).

## Best Practices

### 1. Start with Dry-Run

Test your configuration before enabling actual stacking:

```sh
RUN_MODE=cron
CRON_INTERVAL=600
DRY_RUN=true  # Test first!
```

Monitor logs to verify expected behavior, then disable dry-run:

```sh
DRY_RUN=false
```

### 2. Use File Logging

Enable persistent logs for troubleshooting:

```yaml
environment:
  - LOG_FILE=/app/logs/immich-stack.log
  - LOG_FORMAT=json # Easier to parse for monitoring
volumes:
  - ./logs:/app/logs
```

### 3. Set Appropriate Timeouts

Ensure your interval accounts for processing time:

```sh
# Bad: Interval shorter than processing time
CRON_INTERVAL=300  # 5 minutes
# If processing takes 10 minutes, runs overlap!

# Good: Interval > 2x processing time
CRON_INTERVAL=1800  # 30 minutes for 10-minute processing
```

### 4. Monitor Resource Limits

Set Docker resource limits to prevent runaway memory usage:

```yaml
services:
  immich-stack:
    deploy:
      resources:
        limits:
          memory: 2G # Adjust based on library size
          cpus: "1.0"
```

### 5. Plan for Maintenance

Schedule maintenance windows for:

- Application updates
- Configuration changes
- Immich server maintenance

Use `docker stop` (not `docker kill`) for graceful shutdowns:

```sh
# Graceful stop (waits for current operation)
docker stop immich-stack

# Forced stop (immediate, may cause issues)
docker kill immich-stack  # Avoid if possible
```

### 6. Separate Concerns

Don't combine cron mode with one-time operations:

```sh
# Bad: Mixing modes
RUN_MODE=cron
RESET_STACKS=true  # This only works in "once" mode!

# Good: Use once mode for resets
RUN_MODE=once
RESET_STACKS=true
CONFIRM_RESET_STACK="I acknowledge..."
```

After reset completes, switch back to cron mode.

## Troubleshooting

### Issue: Cron runs too frequently

**Symptom**: Logs show runs starting immediately after previous completion

**Cause**: Processing time exceeds `CRON_INTERVAL`

**Solution**: Increase the interval:

```sh
CRON_INTERVAL=7200  # Double the interval
```

### Issue: Cron stops running

**Symptom**: No new log entries after initial runs

**Possible causes**:

1. Container crashed (check `docker ps`)
1. Fatal error occurred (check logs: `docker logs immich-stack`)
1. API key became invalid (verify key in Immich settings)

**Solution**: Check logs and restart with `docker restart immich-stack`

### Issue: High memory usage

**Symptom**: Container OOM killed or system slowdown

**Cause**: Library too large for available memory

**Solutions**:

1. Increase Docker memory limit
1. Use simpler criteria (Legacy mode instead of Expression mode)
1. Filter assets with `WITH_ARCHIVED=false` and `WITH_DELETED=false`
1. Increase interval to reduce memory pressure

### Issue: Inconsistent stacking results

**Symptom**: Same assets grouped differently across runs

**Cause**: Non-deterministic criteria or race conditions

**Solution**: Ensure criteria are deterministic:

- Use specific time deltas (not relative times)
- Avoid criteria that depend on external state
- Use `--replace-stacks=true` for consistency

## Example Configurations

### Basic Cron Setup

```yaml
version: "3"
services:
  immich-stack:
    image: majorfi/immich-stack:latest
    init: true
    environment:
      - API_KEY=your_key_here
      - API_URL=http://immich:2283/api
      - RUN_MODE=cron
      - CRON_INTERVAL=3600
      - REPLACE_STACKS=true
      - LOG_LEVEL=info
    restart: unless-stopped
```

### Advanced Cron with Logging

```yaml
version: "3"
services:
  immich-stack:
    image: majorfi/immich-stack:latest
    init: true
    stop_grace_period: 5m
    environment:
      - API_KEY=your_key_here
      - API_URL=http://immich:2283/api
      - RUN_MODE=cron
      - CRON_INTERVAL=1800
      - REPLACE_STACKS=true
      - LOG_LEVEL=info
      - LOG_FORMAT=json
      - LOG_FILE=/app/logs/immich-stack.log
    volumes:
      - ./logs:/app/logs
    deploy:
      resources:
        limits:
          memory: 1G
          cpus: "0.5"
    restart: unless-stopped
```

### Multi-User Cron

```yaml
version: "3"
services:
  immich-stack:
    image: majorfi/immich-stack:latest
    init: true
    environment:
      - API_KEY=user1_key,user2_key,user3_key
      - API_URL=http://immich:2283/api
      - RUN_MODE=cron
      - CRON_INTERVAL=3600
      - REPLACE_STACKS=true
      - LOG_LEVEL=info
      - PARENT_FILENAME_PROMOTE=edit,raw
    restart: unless-stopped
```
