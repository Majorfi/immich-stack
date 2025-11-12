# Stacking Logic

## Grouping Modes

Immich Stack supports three grouping modes with increasing complexity and power:

### 1. Legacy Mode (Default)

- **Default Criteria:** Groups by base filename (before extension) and local capture time
- **Logic:** Simple AND operation - all criteria must match
- **Configuration:** Array format in `CRITERIA` environment variable

### 2. Advanced Groups Mode

- **Multiple Strategies:** Support for multiple grouping approaches
- **Logic:** Configurable AND/OR operations per group
- **Configuration:** Object format with `"mode": "advanced"` and `groups` array

### 3. Advanced Expression Mode

- **Maximum Flexibility:** Unlimited nested logical expressions
- **Logic:** Full support for AND, OR, and NOT operations with unlimited nesting
- **Configuration:** Object format with `"mode": "advanced"` and `expression` tree

## Custom Criteria

Override default grouping behavior with the `--criteria` flag or `CRITERIA` environment variable using any of the three supported formats. See [Custom Criteria](custom-criteria.md) for complete documentation.

## Sorting

- **Parent Promotion:** Use `--parent-filename-promote` or `PARENT_FILENAME_PROMOTE` (comma-separated substrings) to promote files as stack parents
- **Empty String for Negative Matching:** Use an empty string in the promote list to prioritize files that DON'T contain any of the other substrings (e.g., `,edit` promotes unedited files first)
- **Sequence Keyword:** Use the `sequence` keyword for flexible sequential file handling (e.g., `sequence`, `sequence:4`, `sequence:IMG_`)
- **Sequence Detection:** Automatically detects numeric sequences in promote lists (e.g., `0000,0001,0002`) and uses intelligent matching for burst photos
- **Extension Promotion:** Use `--parent-ext-promote` or `PARENT_EXT_PROMOTE` (comma-separated extensions) to further prioritize
- **Extension Rank:** Built-in priority: `.jpeg` > `.jpg` > `.png` > others
- **Alphabetical:** Final tiebreaker

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

The `sequence` keyword provides powerful and flexible sequence handling:

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

1. **Fetch all stacks and assets** from Immich
2. **Determine grouping mode** based on `CRITERIA` configuration:
   - **Legacy Mode:** Apply simple AND logic to array of criteria
   - **Groups Mode:** Process each criteria group with configured AND/OR logic
   - **Expression Mode:** Recursively evaluate nested logical expressions
3. **Group assets** into stacks using the selected mode and criteria
4. **Sort each stack** to determine the parent and children using promotion rules
5. **Apply changes** via the Immich API (create, update, or delete stacks as needed)
6. **Log all actions** and optionally run in dry-run mode for safety

## Safe Operations

The stacker includes several safety features:

- **Dry Run Mode:** Use `--dry-run` or `DRY_RUN=true` to simulate actions without making changes
- **Stack Replacement:** Use `--replace-stacks` or `REPLACE_STACKS=true` to replace existing stacks
- **Stack Reset:** Use `--reset-stacks` or `RESET_STACKS=true` with confirmation to delete all stacks (requires `RUN_MODE=once`)
- **Confirmation Required:** Stack reset requires explicit confirmation via `CONFIRM_RESET_STACK`

## Parent Selection Edge Cases

### Multiple Promotion Rules

When multiple promotion rules apply to different files in a group, the selection follows this strict precedence order:

1. **PARENT_FILENAME_PROMOTE list order** (left to right)
2. **PARENT_EXT_PROMOTE list order** (left to right)
3. **Built-in extension rank** (`.jpeg` > `.jpg` > `.png` > others)
4. **Alphabetical order** (case-insensitive)

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

- **Unicode characters** are supported in filename matching
- **Special regex characters** in promote strings are treated as literals (no regex escaping needed)
- **Whitespace** in filenames is preserved and matched exactly

**Example**:

```sh
PARENT_FILENAME_PROMOTE=編集,★favorites,café
```

This will match files containing these exact Unicode strings.

### Tie-Breaking Logic

When two files have equal rank after all promotion rules, the final tie-breaker is:

1. **Original filename** (alphabetically, case-insensitive)
2. If filenames are identical: **Local date/time** (earliest first)
3. If both are identical: **Asset ID** (lexicographic order)

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

1. **Asset count**: ~1KB per asset in memory
2. **Criteria complexity**: Expression trees consume additional memory per evaluation
3. **Stack size**: Larger stacks (more assets per group) increase memory overhead

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

The stacker provides comprehensive logging:

- Colorful, structured logs for all operations
- Clear indication of actions taken
- Error reporting with context
- Progress updates for long-running operations
