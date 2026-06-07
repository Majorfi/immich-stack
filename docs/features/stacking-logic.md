# Stacking Logic

## Grouping Modes

Immich Stack has three grouping modes, from simple to fully expressive.

### 1. Legacy Mode (Default)

Groups by base filename (before extension) and local capture time. All criteria
must match (AND). Configured as a JSON array in the `CRITERIA` environment
variable.

### 2. Advanced Groups Mode

Multiple grouping strategies, each with its own AND/OR operator. Configured as
an object with `"mode": "advanced"` and a `groups` array.

### 3. Advanced Expression Mode

Nested logical expressions with AND, OR, and NOT (no nesting limit). Configured
as an object with `"mode": "advanced"` and an `expression` tree.

## Custom Criteria

Override default grouping behavior with the `--criteria` flag or `CRITERIA` environment variable using any of the three supported formats. See [Custom Criteria](custom-criteria.md) for complete documentation.

## Sorting

The sort layers, in order:

1. Regex-based promotion when a criterion has `promote_index` configured
   (see [Custom Criteria](custom-criteria.md)).
1. `--parent-filename-promote` / `PARENT_FILENAME_PROMOTE`: comma-separated
   substrings, leftmost match wins. This layer also recognizes:
   - An empty entry (e.g. `,edit`) as a negative match for files that contain
     none of the other substrings.
   - The `sequence` keyword: `sequence`, `sequence:4` (four-digit), or
     `sequence:IMG_` (prefix-scoped).
   - Auto-detected numeric sequences, e.g. `0000,0001,0002` extends naturally
     to bursts beyond your listed range.
1. `biggestNumber` keyword (when present in `PARENT_FILENAME_PROMOTE`):
   tie-break by the largest numeric suffix in the filename.
1. `--parent-ext-promote` / `PARENT_EXT_PROMOTE`: comma-separated extensions.
1. Built-in extension rank: `.jpeg` > `.jpg` > `.png` > others.
1. `biggestSize` / `smallestSize` keyword (when present in
   `PARENT_FILENAME_PROMOTE`): tie-break by `exifInfo.fileSizeInByte` (largest
   or smallest wins). Runs after extension rank so a smaller JPG can still beat
   a larger CR2 when `.jpg` is preferred in `PARENT_EXT_PROMOTE`.
1. Alphabetical filename as the final tie-breaker.

## Examples

### Standard Promotion Example

For files: `L1010229.JPG`, `L1010229.edit.jpg`, `L1010229.DNG`

With `PARENT_FILENAME_PROMOTE=edit` and `PARENT_EXT_PROMOTE=.jpg,.dng` in your .env file, or with `--parent-filename-promote=edit` and `--parent-ext-promote=.jpg,.dng`, the order will be:

```
L1010229.edit.jpg
L1010229.JPG
L1010229.DNG
```

### Empty String (Negative Matching) Example

For files that lack EXIF data after editing, you can prioritize unedited files using empty string matching:

For files: `IMG_1234.jpg`, `IMG_1234_edited.jpg`, `IMG_1234_crop.jpg`

With `PARENT_FILENAME_PROMOTE=,_edited,_crop`, the empty string matches files WITHOUT "\_edited" or "\_crop":

```
IMG_1234.jpg          # Promoted first (doesn't contain _edited or _crop)
IMG_1234_edited.jpg   # Second priority
IMG_1234_crop.jpg     # Third priority
```

This is particularly useful when edited JPGs lose their EXIF data and would otherwise appear in the wrong timeline position.

### Burst Photo Sequence Example

For burst photo files: `DSCPDC_0000_BURST20180828114700954.JPG`, `DSCPDC_0001_BURST20180828114700954.JPG`, `DSCPDC_0002_BURST20180828114700954.JPG`, `DSCPDC_0003_BURST20180828114700954_COVER.JPG`

With `PARENT_FILENAME_PROMOTE=0000,0001,0002,0003`, the system automatically detects this as a numeric sequence and orders them correctly:

```
DSCPDC_0000_BURST20180828114700954.JPG
DSCPDC_0001_BURST20180828114700954.JPG
DSCPDC_0002_BURST20180828114700954.JPG
DSCPDC_0003_BURST20180828114700954_COVER.JPG
```

The sequence detection works even with numbers beyond your promote list. For example, if you have `PARENT_FILENAME_PROMOTE=0000,0001,0002,0003` but your files include `DSCPDC_0999_BURST...`, it will be sorted at position 999 automatically.

### Sequence Keyword Examples

The `sequence` keyword handles sequential filenames with a few variants:

```sh
# Order any numeric sequence (1, 2, 10, 100, etc.)
PARENT_FILENAME_PROMOTE=sequence

# Order only 4-digit sequences (0001, 0002, 0010, 0100, etc.)
PARENT_FILENAME_PROMOTE=sequence:4

# Order sequences with specific prefix
PARENT_FILENAME_PROMOTE=sequence:IMG_

# Mix with other promote values - COVER files first, then sequence
PARENT_FILENAME_PROMOTE=COVER,sequence

# Multiple criteria - edited files first, then sequences
PARENT_FILENAME_PROMOTE=edit,hdr,sequence
```

### Supported Sequence Patterns (Legacy Method)

The system can detect various sequence patterns when using comma-separated numbers:

- Pure numbers: `0000,0001,0002,0003`
- Prefixed numbers: `img1,img2,img3` or `IMG_0001,IMG_0002,IMG_0003`
- Suffixed numbers: `1a,2a,3a` or `001_final,002_final,003_final`
- Complex patterns: `photo_001_v2,photo_002_v2,photo_003_v2`

**Note:** The `sequence` keyword is more flexible and recommended over listing individual sequence numbers.

## Stacking Process

1. Fetch all stacks and assets from Immich.
1. Pick the grouping mode based on `CRITERIA`:
   - Legacy: apply AND logic to the criteria array.
   - Groups: process each group with its configured AND/OR operator.
   - Expression: recursively evaluate the nested logical tree.
1. Group assets into stacks using the selected mode and criteria.
1. Sort each stack to determine parent and children using the promotion rules.
1. Apply changes via the Immich API (create, update, or delete stacks).
1. Log all actions. In dry-run mode no API writes happen.

## Safe Operations

The stacker includes several safety features:

- `--dry-run` or `DRY_RUN=true` simulates the run without writing anything back
  to Immich.
- `--replace-stacks` or `REPLACE_STACKS=true` rebuilds existing stacks when the
  criteria pick a different membership.
- `--reset-stacks` or `RESET_STACKS=true` deletes all stacks before
  re-stacking. Requires `RUN_MODE=once` and explicit confirmation via
  `CONFIRM_RESET_STACK`.

## Parent Selection Edge Cases

### Multiple Promotion Rules

When multiple promotion rules apply to different files in a group, the selection follows this strict precedence order:

1. `PARENT_FILENAME_PROMOTE` list order (left to right)
1. `PARENT_EXT_PROMOTE` list order (left to right)
1. Built-in extension rank (`.jpeg` > `.jpg` > `.png` > others)
1. Alphabetical order (case-insensitive)

**Example with multiple matches**:

Files: `IMG_1234_edited.jpg`, `IMG_1234_raw.dng`, `IMG_1234_edited.dng`

With `PARENT_FILENAME_PROMOTE=edited,raw` and `PARENT_EXT_PROMOTE=.jpg,.dng`:

```
IMG_1234_edited.jpg  # Wins: "edited" matches first in filename list, .jpg matches first in ext list
IMG_1234_edited.dng  # Second: "edited" matches first in filename list, .dng second in ext list
IMG_1234_raw.dng     # Third: "raw" is second in filename list
```

### Sequence Keyword Behavior

When multiple `sequence` keywords appear in the promote list, only the **first occurrence** is used. Additional sequence keywords are ignored.

```sh
# Only the first "sequence" is processed
PARENT_FILENAME_PROMOTE=COVER,sequence,edited,sequence:4
# Result: COVER files first, then all sequences, then "edited", then everything else
```

To use specific sequence patterns, place them strategically:

```sh
# Correct: specific pattern first, general sequence later
PARENT_FILENAME_PROMOTE=sequence:IMG_,COVER,sequence

# This orders IMG_ sequences first, then COVER files, then other sequences
```

### Mixed Case Filenames

Filename matching is **case-insensitive** for promotion rules:

```sh
PARENT_FILENAME_PROMOTE=EDIT,HDR
```

This will match: `edit`, `Edit`, `EDIT`, `eDiT`, `hdr`, `HDR`, etc.

However, alphabetical tie-breaking is also case-insensitive, so `IMG_123.jpg` and `img_123.jpg` sort together regardless of case.

### Unicode and Special Characters

- Unicode characters are supported in filename matching.
- Special regex characters in promote strings are treated as literals; no
  escaping needed.
- Whitespace in filenames is preserved and matched exactly.

**Example**:

```sh
PARENT_FILENAME_PROMOTE=編集,★favorites,café
```

This will match files containing these exact Unicode strings.

### Tie-Breaking Logic

When two files have equal rank after all promotion rules, the final tie-breaker is:

1. Original filename (alphabetical, case-insensitive)
1. If filenames are identical: local date/time, earliest first
1. If both are identical: asset ID, lexicographic order

This ensures deterministic, reproducible parent selection across multiple runs.

### Empty String Edge Cases

When using empty string (`,`) for negative matching:

```sh
PARENT_FILENAME_PROMOTE=,_edited,_crop
```

A file matches the empty string rule if it contains **none** of the other non-empty substrings in the list:

- `IMG_1234.jpg` → Matches empty string (no `_edited`, no `_crop`)
- `IMG_1234_final.jpg` → Matches empty string (no `_edited`, no `_crop`)
- `IMG_1234_edited.jpg` → Does NOT match empty string (contains `_edited`)
- `IMG_1234_crop.jpg` → Does NOT match empty string (contains `_crop`)
- `IMG_1234_edited_crop.jpg` → Does NOT match empty string (contains both)

## Performance Implications

### Expression Mode Complexity

Expression mode with deep nesting can impact performance:

| Nesting Level | Performance Impact | Recommendation                |
| ------------- | ------------------ | ----------------------------- |
| 1-2 levels    | Negligible         | Safe for all use cases        |
| 3-4 levels    | Slight increase    | Acceptable for most libraries |
| 5+ levels     | Noticeable impact  | Consider simplifying criteria |

**Example of deep nesting**:

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
}
```

This has 3 levels of nesting. Each additional level multiplies the evaluation cost.

### Memory Usage

Memory usage scales with:

1. Asset count: ~1KB per asset in memory.
1. Criteria complexity: expression trees add memory per evaluation.
1. Stack size: larger stacks (more assets per group) raise the overhead.

**Guidelines**:

| Library Size          | Recommended Mode | Expected Memory |
| --------------------- | ---------------- | --------------- |
| < 10,000 assets       | Any mode         | < 100MB         |
| 10,000-50,000 assets  | Legacy or Groups | 100-500MB       |
| 50,000-100,000 assets | Legacy preferred | 500MB-1GB       |
| > 100,000 assets      | Legacy only      | > 1GB           |

For very large libraries (100k+ assets), consider:

- Using simpler criteria (Legacy mode)
- Processing in batches using `WITH_ARCHIVED` and `WITH_DELETED` filters
- Running during off-peak hours

### Regex Performance

Regex-based criteria (`"regex": {"key": "..."}`) are evaluated for each asset. Complex regex patterns can slow processing:

**Fast regex patterns**:

```json
{ "key": "PXL_", "index": 0 }          // Simple prefix match
{ "key": "IMG_\\d{4}", "index": 0 }     // Simple digit pattern
```

**Slow regex patterns**:

```json
{ "key": ".*photo.*\\d+.*edited.*", "index": 0 }  // Multiple wildcards and backtracking
{ "key": "(IMG|DSC|PXL)_\\d{4,6}_.*", "index": 0 } // Complex alternation
```

Optimize regex by:

- Using anchors (`^`, `$`) when possible
- Avoiding unnecessary wildcards (`.*`)
- Using character classes instead of alternation when possible

### Time Delta Performance

Time-based criteria with small deltas create smaller, more numerous groups:

```json
{ "key": "localDateTime", "delta": { "milliseconds": 100 } }   // Very tight grouping, more groups
{ "key": "localDateTime", "delta": { "milliseconds": 5000 } }  // Looser grouping, fewer groups
```

**Recommendation**: Use the largest delta that still achieves your grouping goals. 1000ms (1 second) is a good starting point for most burst photos.

## Logging

The stacker logs:

- Colorized, structured output for every operation.
- The action taken on each stack (created, updated, skipped).
- Errors with the asset or stack context that caused them.
- Progress counters during long runs.
