# How to Migrate Between Criteria Configurations

This guide helps you safely migrate from one stacking criteria configuration to another without losing data or creating inconsistent stacks.

## Why Migrate Criteria

You might need to migrate criteria when:

- Changing from simple to advanced grouping
- Adjusting time delta values
- Switching from Legacy to Groups or Expression mode
- Refining parent selection rules
- Consolidating multiple criteria into one
- Fixing incorrect stacking logic

## Migration Safety Principles

### Core Safety Rules

1. **Always use dry-run first**: Preview changes before applying
1. **Backup your database**: Immich database before major migrations
1. **Test with small subset**: Verify logic on small sample first
1. **Document old configuration**: Save current criteria before changing
1. **Monitor first run**: Watch logs closely for unexpected behavior

### Migration Strategies

```
Strategy 1: Clean Slate (safest, most disruptive)
└─ Delete all stacks → Apply new criteria

Strategy 2: Incremental Replace (safer, less disruptive)
└─ Keep existing stacks → Replace only conflicting ones

Strategy 3: Additive (safest, least disruptive)
└─ Keep existing stacks → Add only new stacks
```

## Pre-Migration Checklist

Before starting any migration:

- [ ] Document current criteria configuration
- [ ] Backup Immich database
- [ ] Test new criteria with `DRY_RUN=true`
- [ ] Review dry-run logs for expected behavior
- [ ] Identify which stacks will be affected
- [ ] Choose migration strategy
- [ ] Plan rollback procedure
- [ ] Schedule migration during low-usage period

## Migration Strategy 1: Clean Slate

### When to Use

- Major criteria changes
- Current stacks are mostly incorrect
- Starting fresh is simpler than incremental updates
- Small library where recreating stacks is fast

### Process

1. **Save current configuration**:

   ```sh
   # Save to file
   echo "CRITERIA='$CRITERIA'" > old-criteria.env
   echo "PARENT_FILENAME_PROMOTE=$PARENT_FILENAME_PROMOTE" >> old-criteria.env
   ```

1. **Test new configuration with dry-run**:

   ```sh
   CRITERIA='[{"key":"originalFileName","split":{"delimiters":["."],"index":0}},{"key":"localDateTime","delta":{"milliseconds":2000}}]'
   DRY_RUN=true
   LOG_LEVEL=debug
   ./immich-stack
   ```

1. **Review dry-run output**:

   - Check number of stacks that would be created
   - Verify parent selection looks correct
   - Ensure grouping logic is as expected

1. **Execute clean slate migration**:

   ```sh
   RUN_MODE=once
   RESET_STACKS=true
   CONFIRM_RESET_STACK="I acknowledge all my current stacks will be deleted and new one will be created"
   CRITERIA='[new criteria here]'
   PARENT_FILENAME_PROMOTE=new,promotion,rules
   ./immich-stack
   ```

1. **Verify results**:

   ```sh
   LOG_LEVEL=info
   ./immich-stack
   ```

### Example: Migrating from Legacy to Expression Mode

**Old Configuration**:

```sh
CRITERIA='[{"key":"originalFileName","split":{"delimiters":["."],"index":0}}]'
```

**New Configuration**:

```sh
CRITERIA='{"mode":"advanced","expression":{"operator":"AND","children":[{"criteria":{"key":"originalFileName","split":{"delimiters":["."],"index":0}}},{"criteria":{"key":"localDateTime","delta":{"milliseconds":1000}}}]}}'
```

**Migration**:

```sh
# Step 1: Dry-run with new criteria
DRY_RUN=true \
CRITERIA='{"mode":"advanced","expression":{"operator":"AND","children":[{"criteria":{"key":"originalFileName","split":{"delimiters":["."],"index":0}}},{"criteria":{"key":"localDateTime","delta":{"milliseconds":1000}}}]}}' \
./immich-stack

# Step 2: Review output, then execute
RUN_MODE=once \
RESET_STACKS=true \
CONFIRM_RESET_STACK="I acknowledge all my current stacks will be deleted and new one will be created" \
CRITERIA='{"mode":"advanced","expression":{"operator":"AND","children":[{"criteria":{"key":"originalFileName","split":{"delimiters":["."],"index":0}}},{"criteria":{"key":"localDateTime","delta":{"milliseconds":1000}}}]}}' \
./immich-stack
```

## Migration Strategy 2: Incremental Replace

### When to Use

- Minor criteria adjustments
- Most existing stacks are correct
- Only specific stacks need updating
- Large library where full reset is time-consuming

### Process

1. **Save current configuration**:

   ```sh
   echo "CRITERIA='$CRITERIA'" > old-criteria.env
   ```

1. **Update criteria and enable replace mode**:

   ```sh
   REPLACE_STACKS=true
   CRITERIA='[updated criteria]'
   DRY_RUN=true
   ./immich-stack
   ```

1. **Review which stacks will be replaced**:

   - Logs will show "Deleted Stack ... - replacing child stack with new one"
   - Count how many stacks will be affected

1. **Execute replacement**:

   ```sh
   REPLACE_STACKS=true
   CRITERIA='[updated criteria]'
   DRY_RUN=false
   ./immich-stack
   ```

1. **Monitor logs for unexpected replacements**

### Example: Adjusting Time Delta

**Old Configuration** (too tight):

```sh
CRITERIA='[{"key":"localDateTime","delta":{"milliseconds":500}}]'
```

**New Configuration** (more reasonable):

```sh
CRITERIA='[{"key":"localDateTime","delta":{"milliseconds":2000}}]'
```

**Migration**:

```sh
# Dry-run first
REPLACE_STACKS=true \
DRY_RUN=true \
CRITERIA='[{"key":"localDateTime","delta":{"milliseconds":2000}}]' \
./immich-stack

# Execute after verifying
REPLACE_STACKS=true \
DRY_RUN=false \
CRITERIA='[{"key":"localDateTime","delta":{"milliseconds":2000}}]' \
./immich-stack
```

## Migration Strategy 3: Additive

### When to Use

- Adding new grouping rules
- Existing stacks should remain untouched
- Only creating new stacks for previously ungrouped assets
- Safest option when existing stacks are correct

### Process

1. **Disable stack replacement**:

   ```sh
   REPLACE_STACKS=false
   ```

1. **Add new criteria**:

   ```sh
   CRITERIA='[new criteria including previous logic]'
   DRY_RUN=true
   ./immich-stack
   ```

1. **Verify only new stacks are created**:

   - Check logs for "Stack created" (not "replacing")
   - Verify existing stacks are skipped

1. **Execute addition**:

   ```sh
   REPLACE_STACKS=false
   CRITERIA='[new criteria]'
   ./immich-stack
   ```

### Example: Adding Filename Grouping to Time-Based Stacks

**Old Configuration**:

```sh
CRITERIA='[{"key":"localDateTime","delta":{"milliseconds":1000}}]'
```

**New Configuration** (adds filename grouping):

```sh
CRITERIA='[{"key":"originalFileName","split":{"delimiters":["."],"index":0}},{"key":"localDateTime","delta":{"milliseconds":1000}}]'
```

**Migration**:

```sh
# Additive migration - keep existing stacks
REPLACE_STACKS=false \
CRITERIA='[{"key":"originalFileName","split":{"delimiters":["."],"index":0}},{"key":"localDateTime","delta":{"milliseconds":1000}}]' \
./immich-stack
```

## Advanced Migration Scenarios

### Scenario 1: Migrating Parent Selection Rules

When changing parent selection rules without changing grouping:

```sh
# Old promotion rules
PARENT_FILENAME_PROMOTE=raw,original

# New promotion rules (prefer edited files)
PARENT_FILENAME_PROMOTE=edited,final,raw,original

# Migration
REPLACE_STACKS=true  # Must replace to change parents
PARENT_FILENAME_PROMOTE=edited,final,raw,original
./immich-stack
```

**Note**: Changing promotion rules requires `REPLACE_STACKS=true` because parent relationships must be updated.

### Scenario 2: Splitting One Large Criteria into Multiple

**Old** (single criteria):

```sh
CRITERIA='[{"key":"originalFileName","regex":{"key":".*","index":0}}]'
```

**New** (multiple specific criteria):

```sh
CRITERIA='{"mode":"advanced","groups":[{"operator":"AND","criteria":[{"key":"originalFileName","regex":{"key":"^PXL_","index":0}},{"key":"localDateTime","delta":{"milliseconds":1000}}]},{"operator":"AND","criteria":[{"key":"originalFileName","regex":{"key":"^IMG_","index":0}},{"key":"localDateTime","delta":{"milliseconds":1000}}]}]}'
```

**Migration**:

```sh
# Clean slate recommended for major restructuring
RUN_MODE=once
RESET_STACKS=true
CONFIRM_RESET_STACK="I acknowledge all my current stacks will be deleted and new one will be created"
CRITERIA='[new groups configuration]'
./immich-stack
```

### Scenario 3: Merging Multiple Criteria

**Old** (multiple runs with different criteria):

```sh
# Run 1: Group by filename
CRITERIA='[{"key":"originalFileName","split":{"delimiters":["."],"index":0}}]'

# Run 2: Group by time
CRITERIA='[{"key":"localDateTime","delta":{"milliseconds":1000}}]'
```

**New** (unified criteria):

```sh
CRITERIA='[{"key":"originalFileName","split":{"delimiters":["."],"index":0}},{"key":"localDateTime","delta":{"milliseconds":1000}}]'
```

**Migration**:

```sh
# Clean slate to unify into single logic
RUN_MODE=once
RESET_STACKS=true
CONFIRM_RESET_STACK="I acknowledge all my current stacks will be deleted and new one will be created"
CRITERIA='[{"key":"originalFileName","split":{"delimiters":["."],"index":0}},{"key":"localDateTime","delta":{"milliseconds":1000}}]'
./immich-stack
```

## Testing Migrations

### Test Framework

Create a test script to validate migration:

```sh
#!/bin/bash

echo "Migration Test Script"

# Step 1: Save current state
STACK_COUNT_BEFORE=$(curl -s $API_URL/stacks -H "x-api-key: $API_KEY" | jq 'length')
echo "Current stack count: $STACK_COUNT_BEFORE"

# Step 2: Dry-run with new criteria
echo "Running dry-run..."
DRY_RUN=true \
CRITERIA='[new criteria]' \
./immich-stack > migration-test.log

# Step 3: Analyze dry-run results
EXPECTED_STACKS=$(grep "formed" migration-test.log | awk '{print $6}')
echo "Expected new stacks: $EXPECTED_STACKS"

# Step 4: Confirm before executing
read -p "Proceed with migration? (yes/no) " -n 3 -r
echo
if [[ $REPLY =~ ^yes$ ]]; then
  echo "Executing migration..."
  DRY_RUN=false \
  CRITERIA='[new criteria]' \
  ./immich-stack

  # Verify
  STACK_COUNT_AFTER=$(curl -s $API_URL/stacks -H "x-api-key: $API_KEY" | jq 'length')
  echo "Stack count after: $STACK_COUNT_AFTER"
else
  echo "Migration cancelled"
fi
```

### Validation Checklist

After migration, verify:

- [ ] Stack count matches expectations
- [ ] Parent assets are correct for sample stacks
- [ ] No unexpected stack deletions
- [ ] Logs show expected behavior
- [ ] UI shows stacks correctly grouped
- [ ] No assets lost or duplicated

## Rollback Procedures

### Immediate Rollback

If migration fails or produces unexpected results:

1. **Stop current process**:

   ```sh
   docker stop immich-stack
   ```

1. **Restore database backup**:

   ```sh
   # Restore Immich database from backup
   # (Process varies by setup - consult Immich docs)
   ```

1. **Revert to old configuration**:

   ```sh
   source old-criteria.env
   ./immich-stack
   ```

### Partial Rollback

If only some stacks are problematic:

1. **Identify affected assets**:

   - Use Immich UI to find incorrectly stacked photos

1. **Manually unstack in Immich UI**

1. **Re-run with corrected criteria**:

   ```sh
   REPLACE_STACKS=true
   CRITERIA='[corrected criteria]'
   ./immich-stack
   ```

## Migration Best Practices

1. **Always Test First**: Use `DRY_RUN=true` before any migration
1. **Backup Before Changing**: Database backups are essential
1. **Start Small**: Test criteria on small subset first
1. **Monitor Closely**: Watch logs during first real run
1. **Validate Results**: Check sample stacks in UI after migration
1. **Document Changes**: Keep log of what changed and why
1. **Schedule Wisely**: Migrate during low-usage periods
1. **Have Rollback Plan**: Know how to revert if needed

## Common Migration Mistakes

### Mistake 1: Not Testing with Dry-Run

**Problem**: Executing migration without preview

**Solution**: Always run with `DRY_RUN=true` first

### Mistake 2: Forgetting REPLACE_STACKS

**Problem**: New criteria doesn't affect existing stacks

**Solution**:

```sh
REPLACE_STACKS=true  # Required to update existing stacks
```

### Mistake 3: Wrong Migration Strategy

**Problem**: Using additive when clean slate is needed

**Solution**: Choose strategy based on extent of changes (minor = incremental, major = clean slate)

### Mistake 4: No Rollback Plan

**Problem**: Can't undo migration if it fails

**Solution**: Backup database and save old configuration before migrating

### Mistake 5: Incomplete Criteria Conversion

**Problem**: Missing part of old logic in new criteria

**Solution**: Document old criteria completely, verify new criteria includes all required logic

## Troubleshooting Failed Migrations

### Issue: Stacks Not Created as Expected

**Debug**:

```sh
LOG_LEVEL=debug
DRY_RUN=true
./immich-stack > debug-migration.log
```

**Check**:

- Criteria syntax is valid JSON
- All required fields present
- Regex patterns are correct
- Time deltas are reasonable

### Issue: Too Many Stacks Replaced

**Debug**:

- Review REPLACE_STACKS setting
- Check if criteria changed more than expected
- Verify parent promotion rules

**Solution**:

- Use more conservative criteria
- Start with REPLACE_STACKS=false
- Incrementally add replacing logic

### Issue: Performance Degraded After Migration

**Debug**:

- New criteria may be more complex
- Expression nesting may be too deep
- Regex patterns may be inefficient

**Solution**: See [How to Optimize Criteria for Performance](optimize-performance.md)

## Migration Templates

### Template 1: Simple Time Delta Adjustment

```sh
# Save old config
OLD_DELTA=1000
NEW_DELTA=2000

# Test
DRY_RUN=true \
CRITERIA="[{\"key\":\"localDateTime\",\"delta\":{\"milliseconds\":$NEW_DELTA}}]" \
./immich-stack

# Execute
REPLACE_STACKS=true \
CRITERIA="[{\"key\":\"localDateTime\",\"delta\":{\"milliseconds\":$NEW_DELTA}}]" \
./immich-stack
```

### Template 2: Adding New Grouping Criteria

```sh
# Old: filename only
# New: filename + time

DRY_RUN=true \
CRITERIA='[{"key":"originalFileName","split":{"delimiters":["."],"index":0}},{"key":"localDateTime","delta":{"milliseconds":1000}}]' \
./immich-stack

# Execute as clean slate
RUN_MODE=once \
RESET_STACKS=true \
CONFIRM_RESET_STACK="I acknowledge all my current stacks will be deleted and new one will be created" \
CRITERIA='[{"key":"originalFileName","split":{"delimiters":["."],"index":0}},{"key":"localDateTime","delta":{"milliseconds":1000}}]' \
./immich-stack
```

### Template 3: Parent Promotion Rule Changes

```sh
# Test new promotion rules
DRY_RUN=true \
REPLACE_STACKS=true \
PARENT_FILENAME_PROMOTE=new,rules,here \
./immich-stack

# Execute
REPLACE_STACKS=true \
PARENT_FILENAME_PROMOTE=new,rules,here \
./immich-stack
```

## Getting Help with Migrations

If you encounter issues during migration:

1. Collect full logs with `LOG_LEVEL=debug`
1. Document old and new configurations
1. Describe expected vs actual behavior
1. Check if dry-run showed different results
1. Verify database state via Immich API or UI
1. Open issue on GitHub with details
