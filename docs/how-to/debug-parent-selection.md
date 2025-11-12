# How to Debug Parent Selection Issues

This guide helps you troubleshoot and debug parent selection problems when stacking photos.

## Understanding Parent Selection

Parent selection determines which asset becomes the visible representative of a stack. The selection follows a strict precedence order:

1. **PARENT_FILENAME_PROMOTE list order** (left to right)
2. **PARENT_EXT_PROMOTE list order** (left to right)
3. **Built-in extension rank** (`.jpeg` > `.jpg` > `.png` > others)
4. **Alphabetical order** (case-insensitive)
5. **Local date/time** (earliest first)
6. **Asset ID** (lexicographic order)

## Common Parent Selection Problems

### Problem 1: Wrong File is Stack Parent

**Symptom**: Expected file is not the stack parent

**Debug Steps**:

1. Enable debug logging:

   ```sh
   LOG_LEVEL=debug
   DRY_RUN=true
   ```

2. Check your promotion rules:

   ```sh
   PARENT_FILENAME_PROMOTE=edit,raw,original
   PARENT_EXT_PROMOTE=.jpg,.dng
   ```

3. Review the logs for parent selection details:
   ```
   level=debug msg="Parent candidate" filename=IMG_1234_edited.jpg rank=1
   level=debug msg="Parent candidate" filename=IMG_1234.jpg rank=2
   ```

**Solutions**:

- Verify promotion strings are case-insensitive but must be substrings of the filename
- Check that promoted files actually exist in the stack
- Review precedence order (filename promotion beats extension promotion)

### Problem 2: Parent Changes Between Runs

**Symptom**: Different file becomes parent on each run

**Causes**:

- Non-deterministic file ordering (this should not happen after recent fixes)
- Changing promotion rules
- Assets with identical ranks using tie-breaking rules

**Debug Steps**:

1. Lock down your configuration:

   ```sh
   PARENT_FILENAME_PROMOTE=edit,raw
   PARENT_EXT_PROMOTE=.jpg,.dng
   ```

2. Run with same configuration multiple times:

   ```sh
   DRY_RUN=true
   LOG_LEVEL=debug
   ```

3. Compare parent selections across runs

**Solutions**:

- Ensure consistent promotion rules
- Use more specific promotion substrings
- Add extension promotion for additional tie-breaking

### Problem 3: Sequence Not Ordering Correctly

**Symptom**: Burst photos in wrong order (e.g., 0001, 0003, 0002)

**Debug Steps**:

1. Check your sequence configuration:

   ```sh
   # Wrong: Using generic substrings
   PARENT_FILENAME_PROMOTE=0001,0002,0003

   # Right: Using sequence keyword
   PARENT_FILENAME_PROMOTE=sequence
   ```

2. Verify sequence detection:
   ```sh
   LOG_LEVEL=debug
   DRY_RUN=true
   ```

**Solutions**:

- Use `sequence` keyword instead of comma-separated numbers
- For specific patterns, use `sequence:4` (4-digit numbers) or `sequence:IMG_` (with prefix)
- Avoid numeric substrings that match timestamps

### Problem 4: Edited Files Not Promoted

**Symptom**: RAW or original files become parents instead of edited versions

**Debug Steps**:

1. Verify your promotion configuration:

   ```sh
   PARENT_FILENAME_PROMOTE=edit,edited,final
   PARENT_EXT_PROMOTE=.jpg,.jpeg
   ```

2. Check filename patterns in logs:

   ```sh
   LOG_LEVEL=debug
   ```

3. Verify edited files actually contain the promotion substring:
   ```
   IMG_1234.jpg        # Does NOT contain "edit"
   IMG_1234_edit.jpg   # Contains "edit" ✓
   IMG_1234edited.jpg  # Contains "edit" ✓
   ```

**Solutions**:

- Use multiple promotion strings: `edit,edited,_edit,final`
- Add extension promotion: `.jpg,.jpeg` to prefer JPEGs
- Use empty string for negative matching to promote files WITHOUT certain strings

## Advanced Debugging Techniques

### Technique 1: Isolate a Specific Stack

Test parent selection for a specific group of files:

1. Create minimal test criteria:

   ```sh
   CRITERIA='[{"key":"originalFileName","regex":{"key":"^IMG_1234","index":0}}]'
   DRY_RUN=true
   LOG_LEVEL=debug
   ```

2. Review detailed logs for only this file group

### Technique 2: Test Promotion Rules

Create a test script to verify promotion logic:

```sh
#!/bin/bash

# Test different promotion configurations
declare -a configs=(
  "edit,raw"
  "raw,edit"
  "sequence,edit"
  ",edit,raw"  # Empty string for negative matching
)

for config in "${configs[@]}"; do
  echo "Testing: PARENT_FILENAME_PROMOTE=$config"
  PARENT_FILENAME_PROMOTE="$config" \
  DRY_RUN=true \
  LOG_LEVEL=info \
  ./immich-stack | grep "Parent"
  echo "---"
done
```

### Technique 3: Compare Expected vs Actual

1. Document your expected parent for each stack:

   ```
   Expected: IMG_1234_edited.jpg (rank 1: contains "edited")
   Actual: IMG_1234.jpg (rank 3: alphabetical)
   ```

2. Trace through precedence rules to identify where expectations diverge

### Technique 4: Use Dry-Run with Verbose Logging

Combine dry-run mode with debug logging to see all parent selection decisions:

```sh
DRY_RUN=true
LOG_LEVEL=debug
LOG_FORMAT=json  # For easier parsing
./immich-stack > debug-output.log 2>&1
```

Then analyze the log:

```sh
# Find all parent selection events
grep "Parent candidate" debug-output.log

# Count parent selections by filename pattern
grep "Parent candidate" debug-output.log | awk '{print $5}' | sort | uniq -c
```

## Testing Parent Selection Rules

### Test Case 1: Basic Promotion

**Setup**:

```
Files:
- IMG_1234.jpg
- IMG_1234_edited.jpg
- IMG_1234.dng

Config:
PARENT_FILENAME_PROMOTE=edited
```

**Expected**: IMG_1234_edited.jpg is parent

**Verification**:

```sh
DRY_RUN=true LOG_LEVEL=debug ./immich-stack | grep "1234"
```

### Test Case 2: Extension Precedence

**Setup**:

```
Files:
- IMG_5678.jpg
- IMG_5678.jpeg
- IMG_5678.png

Config:
PARENT_FILENAME_PROMOTE=""
PARENT_EXT_PROMOTE=.jpg,.jpeg
```

**Expected**: IMG_5678.jpeg is parent (built-in rank: jpeg > jpg > png)

### Test Case 3: Sequence Ordering

**Setup**:

```
Files:
- BURST_0001.jpg
- BURST_0003.jpg
- BURST_0002.jpg

Config:
PARENT_FILENAME_PROMOTE=sequence
```

**Expected**: Order: 0001, 0002, 0003

## Common Edge Cases

### Unicode Filenames

```sh
# Works correctly - case-insensitive substring matching
PARENT_FILENAME_PROMOTE=編集,★favorites,café
```

### Empty String Matching

```sh
# Promote files that DON'T contain "_edited" or "_crop"
PARENT_FILENAME_PROMOTE=,_edited,_crop

# Result:
# IMG_1234.jpg          → Promoted (no _edited, no _crop)
# IMG_1234_edited.jpg   → Not promoted (contains _edited)
```

### Multiple Sequence Keywords

```sh
# Only first "sequence" is used, rest ignored
PARENT_FILENAME_PROMOTE=COVER,sequence,edited,sequence:4
# Result: COVER files first, then sequences, then "edited", then others
```

## Troubleshooting Checklist

When debugging parent selection:

- [ ] Verify promotion strings are substrings (not regex patterns)
- [ ] Check case-insensitive matching is working
- [ ] Confirm promoted files actually exist in stack
- [ ] Review precedence order (filename > extension > built-in > alpha)
- [ ] Test with dry-run mode first
- [ ] Enable debug logging for detailed output
- [ ] Check for typos in promotion configuration
- [ ] Verify environment variables are loaded correctly
- [ ] Compare results across multiple runs for consistency

## Best Practices

1. **Start Simple**: Test with basic promotion rules first
2. **Use Dry-Run**: Always test with `DRY_RUN=true` before production
3. **Enable Debug Logs**: Use `LOG_LEVEL=debug` for detailed insights
4. **Document Expected Behavior**: Write down what you expect before running
5. **Test Incrementally**: Add one promotion rule at a time
6. **Use Sequence Keyword**: Prefer `sequence` over comma-separated numbers
7. **Verify Configuration Loading**: Check that env vars or CLI flags are applied

## Getting Help

If you're still having parent selection issues:

1. Collect debug logs with `LOG_LEVEL=debug`
2. Document your configuration (CRITERIA, PARENT_FILENAME_PROMOTE, etc.)
3. Provide example filenames and expected vs actual parents
4. Include relevant log excerpts showing parent selection
5. Open an issue on GitHub with this information
