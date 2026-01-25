# Architecture Documentation

This document describes the internal architecture, state management, error handling, and technical design decisions of Immich Stack.

## System Overview

Immich Stack is a stateless CLI application that synchronizes photo stacks between computed groupings and the Immich photo management system via its REST API.

```
┌──────────────┐      ┌───────────────┐      ┌──────────────┐
│   CLI Tool   │ ───> │ Stacker Logic │ ───> │  Immich API  │
│  (Commands)  │      │  (Grouping)   │      │   (Stacks)   │
└──────────────┘      └───────────────┘      └──────────────┘
       │                      │                      │
       └──────────────────────┴──────────────────────┘
                        Configuration
                    (Criteria, Flags, Env)
```

### Core Components

1. **Command Layer** (`cmd/`): CLI interface and command orchestration
1. **Stacker Logic** (`pkg/stacker/`): Grouping algorithm and parent selection
1. **API Client** (`pkg/immich/`): HTTP client with retry logic and error handling
1. **Utilities** (`pkg/utils/`): Shared types, logging, and helpers

## State Management

### Stateless Design Philosophy

Immich Stack is **intentionally stateless** between runs:

- No persistent database or state files
- Each run fetches fresh data from Immich API
- Computed groupings are derived from criteria on each execution
- No memory of previous runs or decisions

### Why Stateless?

**Advantages**:

- Resilient to Immich API changes (always uses current state)
- Self-healing from transient errors (retry on next run)
- Consistent with manually created stacks (no drift from external state)
- No risk of state corruption or inconsistency
- Simpler to reason about and debug

**Trade-offs**:

- Must re-fetch all data on each run
- Cannot track incremental progress within a run
- No built-in idempotency tracking (relies on API state comparison)

### State Lifecycle Per Run

Each execution follows this lifecycle:

```
1. Initialize
   ├─ Load configuration (env vars, CLI flags)
   ├─ Create logger
   └─ Create API client

2. Fetch Current State
   ├─ GET /stacks (all existing stacks)
   │  └─ Build stacksMap (asset ID → stack)
   ├─ GET /assets (all assets to process)
   │  └─ Enrich with stack information
   └─ GET /me (current user information)

3. Compute Desired State
   ├─ Apply grouping criteria to assets
   ├─ Form groups (potential stacks)
   └─ Determine parent for each group

4. Compare States
   ├─ Identify new stacks to create
   ├─ Identify stacks to delete
   └─ Identify stacks to update/replace

5. Apply Changes
   ├─ DELETE /stacks/{id} (remove old stacks)
   ├─ PUT /stacks (create/update stacks)
   └─ Log all actions

6. Cleanup
   └─ Exit (no state persisted)
```

### Stack State Representation

**Current State** (from Immich):

```go
type TStack struct {
    ID             string
    PrimaryAssetID string
    Assets         []TAsset
}
```

**Desired State** (computed):

```go
type Group struct {
    Key    string
    Assets []TAsset  // First asset is desired parent
}
```

### Stack Comparison Logic

Determines if existing stack matches desired stack:

```go
func needsUpdate(existing TStack, desired Group) bool {
    // Different parent?
    if existing.PrimaryAssetID != desired.Assets[0].ID {
        return true
    }

    // Different asset membership?
    if !sameAssets(existing.Assets, desired.Assets) {
        return true
    }

    return false  // Stack is already correct
}
```

## Dry-Run Verification

### How Dry-Run Works

Dry-run mode (`DRY_RUN=true`) simulates all operations without making API changes:

```go
func (c *Client) ModifyStack(assetIDs []string) error {
    if c.dryRun {
        c.logger.Info("[DRY RUN] Would create stack")
        return nil  // No-op, just log
    }

    // Real API call
    return c.doRequest(http.MethodPut, "/stacks", payload, nil)
}
```

### Dry-Run Guarantees

1. **No API Writes**: Only GET requests executed, no PUT/POST/DELETE
1. **Full Simulation**: All grouping and comparison logic runs normally
1. **Accurate Logging**: Shows exactly what would happen in real run
1. **Safe Testing**: Can test dangerous operations (RESET_STACKS, REPLACE_STACKS)

### Dry-Run Workflow

```
User Request
    │
    ├─ DRY_RUN=true
    │   ├─ Fetch current state (READ)
    │   ├─ Compute desired state
    │   ├─ Compare states
    │   ├─ Log all planned actions
    │   └─ Exit (no writes)
    │
    └─ DRY_RUN=false
        ├─ Fetch current state (READ)
        ├─ Compute desired state
        ├─ Compare states
        ├─ Execute actions (WRITE)
        └─ Exit
```

### Verifying Dry-Run Output

Look for these log patterns:

```
[DRY RUN] Would create stack with 3 assets
[DRY RUN] Would delete stack abc-123-def
[DRY RUN] Would replace stack xyz-456-uvw
```

Real runs show:

```
✅ Success! Stack created
🗑️  Deleted stack abc-123-def - replacing child stack with new one
🔄 Updated stack xyz-456-uvw
```

## Error Recovery Mechanisms

### Error Classification

Errors are classified into three categories:

1. **Transient Errors** (retry automatically):

   - Network failures (connection timeout, DNS resolution)
   - Server errors (5xx responses)
   - Rate limiting (429 responses)

1. **Permanent Errors** (fail immediately):

   - Authentication failures (401, 403)
   - Invalid request format (400)
   - Resource not found (404)

1. **Application Errors** (log and continue):

   - Invalid asset data
   - Criteria parsing errors
   - Individual stack operation failures

### Error Handling Strategy

```
┌────────────────┐
│  API Request   │
└───────┬────────┘
        │
        ├─ Success (2xx)
        │  └─> Return data
        │
        ├─ Transient Error (5xx, timeout, 429)
        │  ├─> Retry with exponential backoff
        │  └─> Max 3 retries, then fail
        │
        ├─ Permanent Error (4xx except 429)
        │  └─> Fail immediately, log error
        │
        └─ Application Error
           └─> Log error, continue processing
```

### Graceful Degradation

When errors occur during processing:

1. **Individual Asset Failure**: Skip asset, continue with others
1. **Stack Operation Failure**: Log error, continue with remaining stacks
1. **API Client Failure**: Retry automatically, then fail entire run
1. **Criteria Parsing Failure**: Fail fast (cannot proceed without valid criteria)

### Recovery Actions

**For Transient Errors**:

- Automatic retry with exponential backoff (500ms, 1s, 2s)
- Log retry attempts for debugging
- Fail entire operation after max retries

**For Permanent Errors**:

- Log detailed error message with context
- Provide actionable remediation steps
- Exit with non-zero status code

**For Application Errors**:

- Log error with asset/stack context
- Continue processing remaining items
- Report summary at end of run

## API Retry Logic and Backoff Strategy

### Retry Configuration

```go
const (
    maxRetries  = 3
    baseDelay   = 500 * time.Millisecond
)
```

### Exponential Backoff

Retry delays follow exponential pattern:

```
Attempt 1: Wait 500ms  (baseDelay × 2^0)
Attempt 2: Wait 1s     (baseDelay × 2^1)
Attempt 3: Wait 2s     (baseDelay × 2^2)
Fail: No more retries
```

### Retry Implementation

```go
func (c *Client) doRequest(method, path string, body, response interface{}) error {
    for attempt := 0; attempt < maxRetries; attempt++ {
        err := c.makeRequest(method, path, body, response)

        if err == nil {
            return nil  // Success
        }

        if !isRetriable(err) {
            return err  // Permanent error, don't retry
        }

        if attempt < maxRetries-1 {
            delay := baseDelay * time.Duration(1<<attempt)
            c.logger.Warnf("Retry %d/%d after %v", attempt+1, maxRetries, delay)
            time.Sleep(delay)
        }
    }

    return fmt.Errorf("max retries exceeded")
}
```

### Retriable Conditions

```go
func isRetriable(err error) bool {
    // Network errors
    if isNetworkError(err) {
        return true
    }

    // HTTP status codes
    if statusCode == 429 {  // Rate limited
        return true
    }
    if statusCode >= 500 && statusCode < 600 {  // Server errors
        return true
    }

    return false  // Client errors (4xx) are not retriable
}
```

### Backoff Jitter

To prevent thundering herd, random jitter can be added:

```go
delay := baseDelay * time.Duration(1<<attempt)
jitter := time.Duration(rand.Int63n(int64(delay / 2)))
time.Sleep(delay + jitter)
```

### Rate Limiting Handling

When receiving 429 (Too Many Requests):

1. Check `Retry-After` header if present
1. Use exponential backoff if header absent
1. Log rate limit event for monitoring
1. Respect server's requested delay

## Concurrency Handling

### Multi-User Operations

When processing multiple API keys:

```sh
API_KEY=user1_key,user2_key,user3_key
```

Processing is **sequential**, not concurrent:

```go
apiKeys := strings.Split(os.Getenv("API_KEY"), ",")

for _, key := range apiKeys {
    client := immich.NewClient(apiURL, key, ...)

    user, err := client.GetCurrentUser()
    if err != nil {
        logger.Errorf("Failed for key: %v", err)
        continue  // Skip this user, continue with others
    }

    logger.Infof("Processing user: %s", user.Name)

    // Process stacks for this user
    if err := processStacks(client); err != nil {
        logger.Errorf("Error for user %s: %v", user.Name, err)
        continue
    }
}
```

### Why Sequential Processing?

**Design Choice**: Sequential processing per user to:

1. **Avoid API Rate Limits**: Concurrent requests could exceed limits
1. **Maintain Clear Logs**: User-by-user logging is easier to follow
1. **Prevent Resource Contention**: Single HTTP client per user
1. **Ensure Isolation**: Errors in one user don't affect others

### Within-User Parallelism

Within a single user's processing, operations are sequential:

```
Fetch Stacks → Fetch Assets → Group Assets → Apply Changes
    ↓             ↓               ↓              ↓
  Serial        Serial          Serial         Serial
```

**Rationale**:

- Stacks depend on assets (must fetch stacks first)
- Grouping requires all assets (can't parallelize)
- Stack operations have dependencies (delete before create)

### Thread Safety

HTTP client is **not** shared across goroutines:

```go
// Safe: New client per user
for _, key := range apiKeys {
    client := immich.NewClient(...)  // Fresh instance
    // Use client for this user only
}

// Unsafe: Sharing client across goroutines
client := immich.NewClient(...)
for _, key := range apiKeys {
    go func() {
        // DON'T DO THIS - not thread-safe
        client.SetAPIKey(key)
    }()
}
```

### Signal Handling

Graceful shutdown for cron mode:

```go
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

go func() {
    <-sigChan
    logger.Info("Received shutdown signal")
    shutdownFlag.Set(true)  // Set atomic flag
}()

for !shutdownFlag.Get() {
    runStacker()
    time.Sleep(cronInterval)
}
```

## API Client Architecture

### HTTP Client Configuration

```go
client := &http.Client{
    Timeout: 600 * time.Second,  // 10 minutes
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 100,
        IdleConnTimeout:     90 * time.Second,
    },
}
```

### Request/Response Flow

```
1. Build Request
   ├─ Set method (GET, POST, PUT, DELETE)
   ├─ Build URL (baseURL + path)
   ├─ Marshal JSON body (if present)
   ├─ Set headers (Content-Type, x-api-key)
   └─ Create http.Request

2. Execute Request (with retries)
   ├─ Attempt 1: Send request
   │  ├─ Success? Return response
   │  └─ Retriable error? Continue
   ├─ Wait with exponential backoff
   ├─ Attempt 2: Send request
   │  └─ ...
   └─ Attempt 3: Send request
      └─ Fail if still erroring

3. Handle Response
   ├─ Check status code
   ├─ Read response body
   ├─ Unmarshal JSON (if expected)
   └─ Return data or error
```

### Connection Pooling

Benefits of connection pooling:

- **Reduced Latency**: Reuse existing TCP connections
- **Lower Overhead**: Avoid handshake for each request
- **Better Performance**: Especially for many small requests

Configuration:

```go
MaxIdleConns: 100          // Total idle connections across all hosts
MaxIdleConnsPerHost: 100   // Idle connections per host
IdleConnTimeout: 90s       // Close idle connections after 90s
```

## Grouping Algorithm

### High-Level Flow

```
Assets (unsorted) → Group By Criteria → Sort Within Groups → Stacks
```

### Grouping Process

1. **Initialize Empty Groups**:

   ```go
   groups := make(map[string][]TAsset)
   ```

1. **Iterate All Assets**:

   ```go
   for _, asset := range assets {
       key := computeGroupKey(asset, criteria)
       groups[key] = append(groups[key], asset)
   }
   ```

1. **Compute Group Key**:

   ```go
   func computeGroupKey(asset TAsset, criteria []Criterion) string {
       keys := []string{}
       for _, crit := range criteria {
           switch crit.Key {
           case "originalFileName":
               keys = append(keys, extractFilename(asset, crit))
           case "localDateTime":
               keys = append(keys, formatTime(asset, crit))
           // ... other criteria
           }
       }
       return strings.Join(keys, "|")
   }
   ```

### Parent Selection Within Group

1. **Sort Group by Promotion Rules**:

   ```go
   sort.Slice(group, func(i, j int) bool {
       return compareByPromotionRules(group[i], group[j])
   })
   ```

1. **Promotion Rule Precedence**:

   ```
   1. PARENT_FILENAME_PROMOTE list order (left to right)
   2. PARENT_EXT_PROMOTE list order (left to right)
   3. Built-in extension rank (.jpeg > .jpg > .png > others)
   4. Alphabetical order (case-insensitive)
   5. Local date/time (earliest first)
   6. Asset ID (lexicographic)
   ```

1. **First Asset Becomes Parent**:

   ```go
   parent := group[0]
   children := group[1:]
   ```

## Performance Characteristics

### Time Complexity

- **Fetching Assets**: O(n) where n = total assets
- **Grouping**: O(n × m) where m = number of criteria
- **Sorting Groups**: O(k × g log g) where k = number of groups, g = avg group size
- **Creating Stacks**: O(k) API calls
- **Overall**: O(n × m + k × g log g)

### Space Complexity

- **Assets**: O(n) - all assets stored in memory
- **Groups**: O(n) - assets distributed across groups
- **Stacks Map**: O(s) where s = number of existing stacks
- **Overall**: O(n)

### Bottlenecks

1. **Network I/O**: Fetching large asset lists from API
1. **Regex Evaluation**: Complex patterns on every asset
1. **JSON Marshaling**: Large payloads for stack operations
1. **Memory**: Large libraries (100k+ assets) can consume 1-2GB

### Optimization Strategies

- Use simple criteria (Legacy mode) for large libraries
- Increase time deltas to reduce group count
- Optimize regex patterns (anchors, no wildcards)
- Filter assets with WITH_ARCHIVED/WITH_DELETED
- Process in batches for very large libraries

## Logging Architecture

### Log Levels

```go
trace   // Very detailed (e.g., HTTP request/response bodies)
debug   // Detailed (e.g., parent selection decisions)
info    // Standard (e.g., stack created, assets processed)
warn    // Warnings (e.g., retries, unexpected conditions)
error   // Errors (e.g., API failures, invalid config)
```

### Structured Logging

Using logrus for structured logs:

```go
logger.WithFields(logrus.Fields{
    "assetID": asset.ID,
    "filename": asset.OriginalFileName,
    "stackID": stack.ID,
}).Info("Stack created")
```

### Log Formats

**Text Format** (human-readable):

```
level=info msg="Stack created" assetID=abc-123 filename=IMG_1234.jpg
```

**JSON Format** (machine-parseable):

```json
{
  "level": "info",
  "msg": "Stack created",
  "assetID": "abc-123",
  "filename": "IMG_1234.jpg",
  "time": "2025-11-12T10:30:00Z"
}
```

### Dual Logging

When LOG_FILE is set:

```go
if logFile != "" {
    file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
    if err == nil {
        logger.SetOutput(io.MultiWriter(os.Stdout, file))
    } else {
        // Fallback to stdout only
        logger.Warn("Could not open log file, using stdout only")
    }
}
```

## Testing Architecture

### Test Structure

```
pkg/
├─ stacker/
│  ├─ stacker.go          # Implementation
│  ├─ stacker_test.go     # Unit tests
│  └─ stacker_integration_test.go  # Integration tests
│
└─ immich/
   ├─ client.go           # API client
   └─ client_test.go      # Mock API tests
```

### Test Categories

1. **Unit Tests**: Test individual functions in isolation
1. **Integration Tests**: Test component interactions
1. **Mock Tests**: Test API client with mock HTTP server

### Testing Best Practices

- Use table-driven tests for multiple scenarios
- Mock external dependencies (API, filesystem)
- Test edge cases (empty groups, single-asset stacks)
- Verify error handling paths
- Check log output for correct messages

## Security Considerations

### API Key Handling

- Never log API keys (sanitize in logs)
- Store keys in environment variables, not files
- Support multiple keys for multi-user scenarios
- Validate key format before use

### Input Validation

- Validate all user inputs (criteria, env vars)
- Sanitize regex patterns to prevent ReDoS
- Check for SQL injection in any database queries
- Validate file paths for log files

### Network Security

- Use HTTPS for API calls (validate TLS)
- Set reasonable timeouts to prevent DoS
- Implement rate limiting respect
- Handle redirects securely

## Future Architecture Considerations

### Potential Improvements

1. **Incremental Processing**: Track processed assets to skip on subsequent runs
1. **Parallel API Calls**: Concurrent fetching/updating with proper throttling
1. **Persistent Cache**: Cache asset metadata to reduce API calls
1. **Batch Optimization**: Group stack operations into larger batches
1. **Streaming Processing**: Process assets in streaming fashion for very large libraries

### Scalability Limits

Current architecture scales to:

- **Assets**: ~200k (limited by memory)
- **Stacks**: ~50k (limited by API response size)
- **Users**: Unlimited (sequential processing)
- **API Calls**: Respects rate limits with exponential backoff

### Extension Points

Areas designed for extension:

- **New Criteria Types**: Add to criteria.go
- **Custom Comparison Logic**: Extend grouping algorithm
- **Additional Commands**: Add to cmd/ directory
- **Alternative APIs**: Implement new client interface
