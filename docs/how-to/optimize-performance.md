# How to Optimize Criteria for Performance

This guide helps you optimize stacking criteria for better performance, especially with large photo libraries (50k+ assets).

## Performance Fundamentals

### Key Performance Factors

1. **Criteria Complexity**: More complex criteria = longer processing time
2. **Library Size**: Linear scaling with asset count
3. **Regex Patterns**: Complex regex can slow processing significantly
4. **Time Deltas**: Smaller deltas = more groups = more processing
5. **Expression Nesting**: Deep nesting multiplies evaluation cost

### Performance Targets

| Library Size    | Target Processing Time | Recommended Mode |
| --------------- | ---------------------- | ---------------- |
| < 10k assets    | < 30 seconds           | Any mode         |
| 10k-50k assets  | < 2 minutes            | Legacy or Groups |
| 50k-100k assets | < 5 minutes            | Legacy preferred |
| > 100k assets   | < 10 minutes           | Legacy only      |

## Choosing the Right Grouping Mode

### Mode Comparison

```
Legacy Mode:
- Simplest and fastest
- AND logic only
- Best for large libraries (100k+ assets)
- Lowest memory usage

Groups Mode:
- Moderate complexity
- Multiple strategies with AND/OR per group
- Good for medium libraries (10k-50k assets)
- Moderate memory usage

Expression Mode:
- Most flexible but slowest
- Unlimited nesting with AND/OR/NOT
- Best for small libraries (< 10k assets)
- Highest memory usage
```

### When to Use Each Mode

**Use Legacy Mode When**:

- Library has > 50k assets
- Simple grouping rules suffice
- Performance is critical
- Memory is limited

**Use Groups Mode When**:

- Need multiple grouping strategies
- Library has 10k-50k assets
- Performance is moderately important
- Some flexibility needed

**Use Expression Mode When**:

- Need complex nested logic
- Library has < 10k assets
- Flexibility is more important than speed
- Have sufficient memory available

## Optimizing Time Delta Criteria

### Time Delta Performance

Smaller time deltas create more, smaller groups which increases processing:

```json
// Tight grouping - MORE processing
{"key": "localDateTime", "delta": {"milliseconds": 100}}

// Loose grouping - LESS processing
{"key": "localDateTime", "delta": {"milliseconds": 5000}}
```

### Choosing the Right Delta

| Use Case                   | Recommended Delta | Reasoning                        |
| -------------------------- | ----------------- | -------------------------------- |
| Burst photos (same moment) | 100-500ms         | Captures rapid sequences         |
| HDR/bracketing             | 1000-2000ms       | Multiple exposures               |
| General stacking           | 1000-3000ms       | Balance accuracy and performance |
| Panoramas                  | 2000-5000ms       | Slower shooting process          |
| Time-lapse frames          | 5000-10000ms      | Intentional time gaps            |

### Testing Delta Performance

```sh
# Test with different deltas and measure time
time CRITERIA='[{"key":"localDateTime","delta":{"milliseconds":1000}}]' ./immich-stack

time CRITERIA='[{"key":"localDateTime","delta":{"milliseconds":5000}}]' ./immich-stack
```

## Optimizing Regex Patterns

### Regex Performance Impact

Complex regex patterns are evaluated for every asset in your library.

**Fast Patterns** (prefer these):

```json
// Simple prefix match
{"key": "PXL_", "index": 0}

// Simple digit pattern
{"key": "IMG_\\d{4}", "index": 0}

// Anchored pattern
{"key": "^IMG_", "index": 0}
```

**Slow Patterns** (avoid):

```json
// Multiple wildcards with backtracking
{"key": ".*photo.*\\d+.*edited.*", "index": 0}

// Complex alternation
{"key": "(IMG|DSC|PXL)_\\d{4,6}_.*", "index": 0}

// Nested repetition
{"key": "(.*)+(IMG|DSC)+", "index": 0}
```

### Regex Optimization Techniques

1. **Use Anchors**: `^IMG_` is faster than `IMG_`
2. **Avoid Wildcards**: `IMG_\d{4}` is faster than `IMG_.*`
3. **Use Character Classes**: `[A-Z]` instead of `(A|B|C|...)`
4. **Limit Repetition**: `\d{4}` instead of `\d+`
5. **Avoid Backtracking**: Don't use nested `.*` or `(.+)+`

### Testing Regex Performance

```sh
# Benchmark different regex patterns
time CRITERIA='[{"key":"originalFileName","regex":{"key":"^PXL_","index":0}}]' ./immich-stack

time CRITERIA='[{"key":"originalFileName","regex":{"key":".*PXL.*","index":0}}]' ./immich-stack
```

## Expression Mode Optimization

### Nesting Depth Impact

| Nesting Level | Performance Impact                 | Use Case                    |
| ------------- | ---------------------------------- | --------------------------- |
| 1-2 levels    | Negligible (< 5% overhead)         | Safe for all libraries      |
| 3-4 levels    | Slight increase (5-15% overhead)   | Acceptable for < 50k assets |
| 5+ levels     | Noticeable impact (> 15% overhead) | Only for < 10k assets       |

### Example: Optimizing Deep Nesting

**Before** (5 levels deep):

```json
{
  "mode": "advanced",
  "expression": {
    "operator": "AND",
    "children": [
      {
        "operator": "OR",
        "children": [
          {
            "operator": "AND",
            "children": [
              {
                "operator": "OR",
                "children": [
                  {
                    "operator": "AND",
                    "children": [
                      {
                        "criteria": {
                          "key": "originalFileName",
                          "regex": { "key": "PXL_", "index": 0 }
                        }
                      },
                      {
                        "criteria": {
                          "key": "localDateTime",
                          "delta": { "milliseconds": 1000 }
                        }
                      }
                    ]
                  }
                ]
              }
            ]
          }
        ]
      }
    ]
  }
}
```

**After** (2 levels deep - same logic, flattened):

```json
{
  "mode": "advanced",
  "expression": {
    "operator": "AND",
    "children": [
      {
        "criteria": {
          "key": "originalFileName",
          "regex": { "key": "PXL_", "index": 0 }
        }
      },
      {
        "criteria": {
          "key": "localDateTime",
          "delta": { "milliseconds": 1000 }
        }
      }
    ]
  }
}
```

**Result**: 60% faster processing for same logic.

### Simplification Strategies

1. **Flatten Nested ANDs**: `AND(AND(A, B), C)` → `AND(A, B, C)`
2. **Flatten Nested ORs**: `OR(OR(A, B), C)` → `OR(A, B, C)`
3. **Remove Redundant Operators**: Single-child operators can be eliminated
4. **Use Groups Mode**: If nesting > 3 levels, consider Groups mode instead

## Memory Optimization

### Memory Usage Scaling

Memory usage depends on:

1. **Asset count**: ~1KB per asset in memory
2. **Criteria complexity**: Expression trees consume additional memory
3. **Stack size**: Larger stacks increase memory overhead

### Expected Memory Usage

| Library Size | Legacy Mode | Groups Mode     | Expression Mode |
| ------------ | ----------- | --------------- | --------------- |
| 10k assets   | < 100MB     | < 150MB         | < 200MB         |
| 50k assets   | 100-500MB   | 500-750MB       | 750MB-1GB       |
| 100k assets  | 500MB-1GB   | 1-1.5GB         | 1.5-2GB         |
| 200k assets  | 1-2GB       | Not recommended | Not recommended |

### Memory Optimization Techniques

1. **Use Filters**:

   ```sh
   WITH_ARCHIVED=false  # Exclude archived assets
   WITH_DELETED=false   # Exclude deleted assets
   ```

2. **Process in Batches**: For very large libraries, process subsets:

   ```sh
   # Process only recent photos
   CRITERIA='[{"key":"localDateTime","delta":{"milliseconds":1000}},{"key":"originalFileName","regex":{"key":"^2025","index":0}}]'
   ```

3. **Use Simpler Criteria**: Legacy mode uses less memory than Expression mode

4. **Increase Swap**: For systems with limited RAM

## Benchmarking Your Configuration

### Performance Testing Script

```sh
#!/bin/bash

echo "Testing performance with different configurations..."

# Test 1: Legacy mode with simple criteria
echo "Test 1: Legacy mode"
time CRITERIA='[{"key":"originalFileName","split":{"delimiters":["."],"index":0}}]' \
  DRY_RUN=true \
  LOG_LEVEL=warn \
  ./immich-stack

# Test 2: Groups mode
echo "Test 2: Groups mode"
time CRITERIA='{"mode":"advanced","groups":[{"operator":"AND","criteria":[{"key":"originalFileName","split":{"delimiters":["."],"index":0}}]}]}' \
  DRY_RUN=true \
  LOG_LEVEL=warn \
  ./immich-stack

# Test 3: Expression mode
echo "Test 3: Expression mode"
time CRITERIA='{"mode":"advanced","expression":{"operator":"AND","children":[{"criteria":{"key":"originalFileName","split":{"delimiters":["."],"index":0}}}]}}' \
  DRY_RUN=true \
  LOG_LEVEL=warn \
  ./immich-stack
```

### Analyzing Results

Compare execution times and choose the fastest configuration that meets your needs.

```
Test 1: Legacy mode      - 45 seconds
Test 2: Groups mode      - 62 seconds
Test 3: Expression mode  - 78 seconds

Recommendation: Use Legacy mode (42% faster than Expression)
```

## Cron Mode Performance Tuning

### Interval Sizing

Choose CRON_INTERVAL based on processing time:

**Formula**: `CRON_INTERVAL = (processing_time × 2) + buffer`

Example:

```sh
# Processing takes 10 minutes
# Interval = (10 × 2) + 10 = 30 minutes minimum
CRON_INTERVAL=1800  # 30 minutes
```

### Preventing Overlap

Monitor logs for timing warnings:

```
⚠️ Warning: Processing took 5400s, which exceeds interval of 3600s
```

**Solution**: Increase CRON_INTERVAL or optimize criteria to reduce processing time.

### Resource Limits

Set Docker resource limits to prevent runaway usage:

```yaml
services:
  immich-stack:
    deploy:
      resources:
        limits:
          memory: 2G # Adjust based on library size
          cpus: "1.0"
```

## Real-World Optimization Examples

### Example 1: Large Google Photos Library

**Before** (8 minutes):

```json
{
  "mode": "advanced",
  "expression": {
    "operator": "AND",
    "children": [
      {
        "criteria": {
          "key": "originalFileName",
          "regex": { "key": ".*PXL.*", "index": 0 }
        }
      },
      {
        "criteria": { "key": "localDateTime", "delta": { "milliseconds": 500 } }
      }
    ]
  }
}
```

**After** (3 minutes - 62% faster):

```json
[
  { "key": "originalFileName", "regex": { "key": "^PXL_", "index": 0 } },
  { "key": "localDateTime", "delta": { "milliseconds": 1000 } }
]
```

**Changes**:

- Switched to Legacy mode
- Anchored regex pattern (^PXL\_)
- Increased time delta (500ms → 1000ms)

### Example 2: Mixed Camera Library

**Before** (12 minutes):

```json
{
  "mode": "advanced",
  "expression": {
    "operator": "OR",
    "children": [
      {
        "criteria": {
          "key": "originalFileName",
          "regex": { "key": "(IMG|DSC|PXL)_.*", "index": 0 }
        }
      },
      {
        "criteria": {
          "key": "originalFileName",
          "regex": { "key": ".*BURST.*", "index": 0 }
        }
      }
    ]
  }
}
```

**After** (5 minutes - 58% faster):

```json
{
  "mode": "advanced",
  "groups": [
    {
      "operator": "AND",
      "criteria": [
        {
          "key": "originalFileName",
          "split": { "delimiters": ["_", "."], "index": 0 }
        },
        { "key": "localDateTime", "delta": { "milliseconds": 1000 } }
      ]
    }
  ]
}
```

**Changes**:

- Switched to Groups mode
- Removed complex regex patterns
- Used split-based grouping instead

### Example 3: Event Photography

**Before** (20 minutes, 150k assets):

```json
{
  "mode": "advanced",
  "expression": {
    "operator": "AND",
    "children": [
      {
        "criteria": {
          "key": "originalFileName",
          "regex": { "key": ".*\\d{4}.*", "index": 0 }
        }
      },
      {
        "criteria": { "key": "localDateTime", "delta": { "milliseconds": 100 } }
      }
    ]
  }
}
```

**After** (6 minutes - 70% faster):

```json
[
  {
    "key": "originalFileName",
    "split": { "delimiters": ["_", "-", "."], "index": 0 }
  },
  { "key": "localDateTime", "delta": { "milliseconds": 2000 } }
]
```

**Changes**:

- Switched to Legacy mode
- Removed regex (used split instead)
- Increased time delta (100ms → 2000ms)
- Simpler filename grouping

## Performance Monitoring

### Key Metrics to Track

1. **Processing Time**: Total time for stack operations
2. **Assets per Second**: `total_assets / processing_time`
3. **Memory Usage**: Peak memory during processing
4. **Stack Count**: Number of stacks created/modified

### Logging Performance Data

```sh
LOG_LEVEL=info
LOG_FORMAT=json

# Processing logs will include:
# - "🌄 52193 assets fetched"
# - "Legacy criteria stacking formed 4275 stacks from 52193 assets"
# - Execution time in logs
```

### Setting Performance Goals

| Library Size   | Target Rate      | Example                   |
| -------------- | ---------------- | ------------------------- |
| < 10k assets   | > 200 assets/sec | 10k assets in 50 seconds  |
| 10k-50k assets | > 150 assets/sec | 50k assets in 5 minutes   |
| > 50k assets   | > 100 assets/sec | 100k assets in 15 minutes |

## Troubleshooting Slow Performance

### Diagnostic Steps

1. **Enable timing logs**:

   ```sh
   LOG_LEVEL=debug
   LOG_FORMAT=json
   time ./immich-stack
   ```

2. **Check memory usage**:

   ```sh
   docker stats immich-stack
   ```

3. **Profile regex patterns**: Test individual regex patterns with small datasets

4. **Simplify criteria**: Try Legacy mode with basic criteria as baseline

### Common Bottlenecks

1. **Complex Regex**: Switch to split-based grouping
2. **Small Time Deltas**: Increase delta to reduce groups
3. **Deep Expression Nesting**: Flatten or switch to Groups/Legacy mode
4. **Large Library**: Use filters to process subsets
5. **Low Memory**: Increase swap or process in batches

## Best Practices Summary

1. **Start Simple**: Use Legacy mode for initial setup
2. **Benchmark**: Test different configurations and measure
3. **Use Appropriate Delta**: 1000-2000ms works for most use cases
4. **Optimize Regex**: Use anchors and avoid wildcards
5. **Limit Nesting**: Keep expression nesting < 3 levels
6. **Filter Assets**: Exclude archived/deleted when possible
7. **Size Intervals**: Set CRON_INTERVAL > 2× processing time
8. **Monitor Resources**: Track memory and CPU usage
9. **Test Incrementally**: Add complexity only when needed
10. **Document Performance**: Track metrics over time
