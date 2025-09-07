# Custom Criteria

Immich Stack allows you to define custom criteria for grouping photos using a JSON configuration. This gives you fine-grained control over how photos are grouped into stacks.

## Criteria Formats

The `CRITERIA` environment variable supports three formats with increasing complexity and power:

### 1. Legacy Array Format (Simple)

Basic format where ALL criteria must match (AND logic):

```json
[
  {
    "key": "originalFileName",
    "split": {
      "delimiters": ["~", "."],
      "index": 0
    }
  },
  {
    "key": "localDateTime",
    "delta": {
      "milliseconds": 1000
    }
  }
]
```

### 2. Advanced Groups Format (Medium Complexity)

Supports multiple grouping strategies with configurable AND/OR logic per group:

```json
{
  "mode": "advanced",
  "groups": [
    {
      "operator": "AND",
      "criteria": [
        { "key": "originalFileName", "regex": { "key": "PXL_", "index": 0 } },
        { "key": "localDateTime", "delta": { "milliseconds": 1000 } }
      ]
    }
  ]
}
```

### 3. Advanced Expression Format (Maximum Power)

Supports unlimited nested logical expressions with AND, OR, and NOT operations:

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
            "criteria": {
              "key": "originalFileName",
              "regex": { "key": "PXL_", "index": 0 }
            }
          },
          {
            "criteria": {
              "key": "originalPath",
              "split": { "delimiters": ["/"], "index": 2 }
            }
          }
        ]
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

## Available Keys

You can use any of these keys in your criteria:

| Key                | Description                    |
| ------------------ | ------------------------------ |
| `originalFileName` | Original filename of the asset |
| `originalPath`     | Original path of the asset     |
| `localDateTime`    | Local capture time             |
| `fileCreatedAt`    | File creation time             |
| `fileModifiedAt`   | File modification time         |
| `updatedAt`        | Last update time               |

## Split Configuration

The `split` configuration allows you to extract parts of string values using delimiters:

```json
{
  "key": "originalFileName",
  "split": {
    "delimiters": ["~", "."], // Array of delimiters to split on
    "index": 0 // Which part to use (0-based)
  }
}
```

For example, with a file named `IMG_1234~edit.jpg`:

1. Split on `~` and `.` gives `["IMG_1234", "edit", "jpg"]`
2. Using `index: 0` selects `"IMG_1234"`

For paths, you can split by directory separators:

```json
{
  "key": "originalPath",
  "split": {
    "delimiters": ["/"],
    "index": 2
  }
}
```

For a path like `photos/2023/vacation/IMG_001.jpg`:

1. Split on `/` gives `["photos", "2023", "vacation", "IMG_001.jpg"]`
2. Using `index: 2` selects `"vacation"`

Note: The `originalPath` splitter automatically normalizes Windows-style backslashes (`\`) to forward slashes (`/`).

## Regex Configuration

The `regex` configuration allows you to extract parts of string values using regular expressions. This provides more powerful pattern matching than simple delimiter splitting:

```json
{
  "key": "originalFileName",
  "regex": {
    "key": "PXL_(\\d{8})_(\\d{9})", // Regular expression pattern
    "index": 1 // Which capture group to use (0 = full match, 1+ = capture groups)
  }
}
```

### Regex with Promotion

Regex can also be used to control the promotion order within a stack. By specifying `promote_index` and `promote_keys`, you can extract a different capture group for promotion:

```json
{
  "key": "originalFileName",
  "regex": {
    "key": "PXL_(\\d{8})_(\\d{9})(_\\w+)?", // Pattern with optional suffix
    "index": 1, // Group by date (capture group 1)
    "promote_index": 3, // Use suffix for promotion (capture group 3)
    "promote_keys": ["_MP", "_edit", "_crop", ""] // Order of promotion (first = highest priority)
  }
}
```

This configuration:

- Groups files by date (capture group 1: `20230503`)
- Promotes files based on suffix (capture group 3: `_MP`, `_edit`, etc.)
- Files with `_MP` suffix become the primary asset
- Files with no suffix (empty string) have lowest priority

For example, with a file named `PXL_20230503_152823814.jpg`:

1. The regex `PXL_(\\d{8})_(\\d{9})` matches and creates capture groups:
   - Index 0 (full match): `"PXL_20230503_152823814"`
   - Index 1 (first group): `"20230503"` (date)
   - Index 2 (second group): `"152823814"` (time)
2. Using `index: 1` selects the date `"20230503"`

### Regex Examples

**Extract date from filename:**

```json
{
  "key": "originalFileName",
  "regex": {
    "key": "IMG_(\\d{8})_\\d{6}",
    "index": 1
  }
}
```

**Extract year from path:**

```json
{
  "key": "originalPath",
  "regex": {
    "key": "photos/(\\d{4})/",
    "index": 1
  }
}
```

**Extract camera model from filename:**

```json
{
  "key": "originalFileName",
  "regex": {
    "key": "(IMG|PXL|DSC)(\\d+)",
    "index": 1
  }
}
```

**Complex path pattern matching:**

```json
{
  "key": "originalPath",
  "regex": {
    "key": "camera_uploads/(\\d{4}-\\d{2}-\\d{2})/DCIM/([^/]+)/",
    "index": 1
  }
}
```

### Regex vs Split

| Feature         | Split                  | Regex                        |
| --------------- | ---------------------- | ---------------------------- |
| **Complexity**  | Simple delimiter-based | Powerful pattern matching    |
| **Use Case**    | Fixed delimiters       | Complex patterns, validation |
| **Performance** | Faster                 | Slightly slower              |
| **Learning**    | Easy                   | Requires regex knowledge     |

Choose **split** for simple cases like separating by `~`, `.`, or `/`.
Choose **regex** for complex patterns like extracting dates, validating formats, or advanced text processing.

## Expression Format Deep Dive

The advanced expression format provides the most powerful grouping capabilities through recursive logical expressions.

### Expression Structure

Each expression node has one of two forms:

**Criteria Node (Leaf):**

```json
{
  "criteria": {
    "key": "originalFileName",
    "regex": { "key": "PXL_", "index": 0 }
  }
}
```

**Operator Node (Branch):**

```json
{
  "operator": "AND",
  "children": [
    // Array of child expressions
  ]
}
```

### Supported Operators

| Operator | Description                   | Children Required |
| -------- | ----------------------------- | ----------------- |
| `AND`    | All children must match       | 1 or more         |
| `OR`     | At least one child must match | 1 or more         |
| `NOT`    | Child must NOT match          | Exactly 1         |

### Expression Examples

**Simple AND condition:**

```json
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
      "criteria": { "key": "localDateTime", "delta": { "milliseconds": 1000 } }
    }
  ]
}
```

**OR condition for multiple camera types:**

```json
{
  "operator": "OR",
  "children": [
    {
      "criteria": {
        "key": "originalFileName",
        "regex": { "key": "PXL_", "index": 0 }
      }
    },
    {
      "criteria": {
        "key": "originalFileName",
        "regex": { "key": "IMG_", "index": 0 }
      }
    },
    {
      "criteria": {
        "key": "originalFileName",
        "regex": { "key": "DSC", "index": 0 }
      }
    }
  ]
}
```

**NOT condition to exclude archived photos:**

```json
{
  "operator": "NOT",
  "children": [{ "criteria": { "key": "isArchived" } }]
}
```

**Complex nested expression:**

```json
{
  "operator": "AND",
  "children": [
    {
      "operator": "OR",
      "children": [
        {
          "criteria": {
            "key": "originalFileName",
            "regex": { "key": "PXL_", "index": 0 }
          }
        },
        {
          "criteria": {
            "key": "originalFileName",
            "regex": { "key": "IMG_", "index": 0 }
          }
        }
      ]
    },
    {
      "operator": "NOT",
      "children": [{ "criteria": { "key": "isArchived" } }]
    },
    {
      "criteria": { "key": "localDateTime", "delta": { "milliseconds": 2000 } }
    }
  ]
}
```

This complex example groups assets that:

1. Have filenames starting with "PXL*" OR "IMG*"
2. AND are NOT archived
3. AND were taken within 2 seconds of each other

## Delta Configuration

The `delta` configuration allows for flexible time matching:

```json
{
  "key": "localDateTime",
  "delta": {
    "milliseconds": 1000 // Time difference to allow (in milliseconds)
  }
}
```

This is useful for:

- Burst photos
- Photos taken in quick succession
- Different time zones
- Camera clock differences

## Examples by Format

### Legacy Array Format Examples

**Basic Filename Grouping:**

```json
[
  {
    "key": "originalFileName",
    "split": {
      "delimiters": ["~", "."],
      "index": 0
    }
  }
]
```

**Regex-Based Date Grouping:**

```json
[
  {
    "key": "originalFileName",
    "regex": {
      "key": "PXL_(\\d{8})_\\d{9}",
      "index": 1
    }
  }
]
```

**Combined Path and Time Criteria:**

```json
[
  {
    "key": "originalPath",
    "split": {
      "delimiters": ["/"],
      "index": 2
    }
  },
  {
    "key": "localDateTime",
    "delta": {
      "milliseconds": 1000
    }
  }
]
```

### Advanced Groups Format Examples

**Multiple Camera Types with OR Logic:**

```json
{
  "mode": "advanced",
  "groups": [
    {
      "operator": "OR",
      "criteria": [
        { "key": "originalFileName", "regex": { "key": "PXL_", "index": 0 } },
        { "key": "originalFileName", "regex": { "key": "IMG_", "index": 0 } },
        { "key": "originalFileName", "regex": { "key": "DSC", "index": 0 } }
      ]
    }
  ]
}
```

**Group by Directory OR Timestamp:**

```json
{
  "mode": "advanced",
  "groups": [
    {
      "operator": "OR",
      "criteria": [
        { "key": "originalPath", "split": { "delimiters": ["/"], "index": 2 } },
        { "key": "localDateTime", "delta": { "milliseconds": 1000 } }
      ]
    }
  ]
}
```

### Advanced Expression Format Examples

**Complex Multi-Camera Setup:**

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
            "criteria": {
              "key": "originalFileName",
              "regex": { "key": "PXL_(\\d{8})", "index": 1 }
            }
          },
          {
            "criteria": {
              "key": "originalFileName",
              "regex": { "key": "IMG_(\\d{8})", "index": 1 }
            }
          }
        ]
      },
      {
        "criteria": {
          "key": "localDateTime",
          "delta": { "milliseconds": 2000 }
        }
      }
    ]
  }
}
```

This groups photos from Pixel or iPhone cameras that were taken on the same date AND within 2 seconds of each other.

**Exclude Archived Photos from Grouping:**

```json
{
  "mode": "advanced",
  "expression": {
    "operator": "AND",
    "children": [
      {
        "criteria": {
          "key": "originalFileName",
          "split": { "delimiters": ["~", "."], "index": 0 }
        }
      },
      {
        "operator": "NOT",
        "children": [{ "criteria": { "key": "isArchived" } }]
      }
    ]
  }
}
```

**Advanced Professional Workflow:**

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
                  "key": "originalPath",
                  "regex": { "key": "/RAW/", "index": 0 }
                }
              },
              {
                "criteria": {
                  "key": "originalFileName",
                  "regex": { "key": "\\.(CR3|NEF|ARW)$", "index": 0 }
                }
              }
            ]
          },
          {
            "operator": "AND",
            "children": [
              {
                "criteria": {
                  "key": "originalPath",
                  "regex": { "key": "/JPEG/", "index": 0 }
                }
              },
              {
                "criteria": {
                  "key": "originalFileName",
                  "regex": { "key": "\\.jpe?g$", "index": 0 }
                }
              }
            ]
          }
        ]
      },
      {
        "criteria": {
          "key": "localDateTime",
          "delta": { "milliseconds": 5000 }
        }
      },
      {
        "operator": "NOT",
        "children": [{ "criteria": { "key": "isTrashed" } }]
      }
    ]
  }
}
```

This complex professional workflow:

1. Groups either (RAW files in /RAW/ folder) OR (JPEG files in /JPEG/ folder)
2. AND taken within 5 seconds
3. AND NOT in trash

## Advanced Grouping Behavior

### Expression-Based Grouping

Advanced mode with expressions performs both **filtering** and **grouping** based on the leaf criteria values that actually match for each asset:

1. **Filter phase**: Only assets that match the expression are considered for stacking
2. **Grouping phase**: Matching assets are grouped by the specific criteria values that contributed to their match
3. **Sorting phase**: Each group is sorted using the same promotion/delimiter rules as legacy mode

**Key differences from legacy mode:**

- **Regex criteria**: Use the matched portion as the grouping key (e.g., `PXL_` instead of full filename)
- **OR branches**: Only values from the first matching branch are included in the grouping key
- **NOT operations**: Contribute no values to grouping keys (used purely for filtering)

> **Note:** In OR expressions, only the first matching branch contributes to the grouping key. Branch order matters—criteria are evaluated in the order they appear in the expression.

**Example - Multiple stacks from one expression:**

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
            "criteria": {
              "key": "originalFileName",
              "regex": { "key": "^PXL_", "index": 0 }
            }
          },
          {
            "criteria": {
              "key": "originalFileName",
              "regex": { "key": "^IMG_", "index": 0 }
            }
          }
        ]
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

This creates separate stacks for:

- All PXL photos taken within the same time window: `originalFileName=PXL_|localDateTime=2023-01-01T12:00:00.000000000Z`
- All IMG photos taken within the same time window: `originalFileName=IMG_|localDateTime=2023-01-01T12:00:00.000000000Z`

### OR Groups Union Semantics

In groups-based advanced mode, OR groups use "union" semantics instead of "exact match" semantics:

- **Legacy behavior**: Assets must share identical matching criteria to be grouped
- **Advanced behavior**: Assets are grouped if they share ANY matching criteria from OR groups

This creates connected components where assets that share any criteria keys are linked together.

**Example:**

```json
{
  "mode": "advanced",
  "groups": [
    {
      "operator": "OR",
      "criteria": [
        { "key": "originalPath", "split": { "delimiters": ["/"], "index": 2 } },
        { "key": "localDateTime", "delta": { "milliseconds": 1000 } }
      ]
    }
  ]
}
```

Assets that share either the same folder OR the same time window will be connected and grouped together, even if they don't share both criteria.

### BiggestNumber Support in Advanced Mode

For `biggestNumber` sorting to work in advanced mode, you must specify `delimiters` in the `originalFileName.split.delimiters` configuration:

```json
{
  "mode": "advanced",
  "expression": {
    "criteria": {
      "key": "originalFileName",
      "split": { "delimiters": ["~", "."], "index": 0 }
    }
  }
}
```

Without delimiters specified, `biggestNumber` sorting falls back to alphabetical ordering.

## Best Practices

1. **Start Simple:**

   - Begin with basic filename grouping
   - Add time-based criteria if needed
   - Test with small sets first

2. **Delta Values:**

   - Use smaller deltas for burst photos (1000ms)
   - Use larger deltas for time zone differences (3600000ms = 1 hour)
   - Consider your camera's burst mode settings

3. **Regex Considerations:**

   - Escape special characters properly (`\\d` for digits, `\\.` for literal dots)
   - Test your regex patterns with sample filenames first
   - Use online regex testers to validate patterns
   - Remember that index 0 is the full match, capture groups start at index 1

4. **Boolean Criteria (Advanced Mode):**

   - Boolean criteria (`isArchived`, `isFavorite`, `isTrashed`, etc.) are filter-only
   - They don't contribute values to grouping keys—used purely for inclusion/exclusion
   - Use them to filter assets before applying other grouping criteria

5. **Testing:**
   - Use `DRY_RUN=true` to test configurations
   - Check logs for grouping results
   - Adjust criteria based on results

## Common Gotchas

> **⚠️ Important Behaviors to Remember:**
>
> - **OR branch order matters**: Only the first matching OR branch contributes to grouping keys
> - **Boolean criteria are filter-only**: `isArchived`, `isFavorite`, etc. don't contribute grouping values
> - **biggestNumber in advanced mode**: Requires `filename.split.delimiters` to be specified in the expression/criteria

## Common Regex Patterns

Here are some useful regex patterns for common filename formats:

```json
// Google Pixel photos: PXL_20230503_152823814.jpg
{
  "key": "originalFileName",
  "regex": {
    "key": "PXL_(\\d{8})_(\\d{9})",
    "index": 1  // Extract date: 20230503
  }
}

// iPhone photos: IMG_20230503_152823.jpg
{
  "key": "originalFileName",
  "regex": {
    "key": "IMG_(\\d{8})_(\\d{6})",
    "index": 1  // Extract date: 20230503
  }
}

// Canon photos: DSC01234.jpg
{
  "key": "originalFileName",
  "regex": {
    "key": "(DSC)(\\d+)",
    "index": 2  // Extract number: 01234
  }
}

// Date-time from path: photos/2023-05-03/
{
  "key": "originalPath",
  "regex": {
    "key": "photos/(\\d{4}-\\d{2}-\\d{2})/",
    "index": 1  // Extract date: 2023-05-03
  }
}
```

## Complete Example: Regex Promotion for Pixel Photos

Imagine you have Google Pixel photos with different processing suffixes:

```
photos/
├── PXL_20230503_152823814.jpg        # Original
├── PXL_20230503_152823814_MP.jpg     # Motion Photo
├── PXL_20230503_152823814_edit.jpg   # Edited version
├── PXL_20230503_152823814_crop.jpg   # Cropped version
├── PXL_20230504_091234567.jpg        # Different photo
└── PXL_20230504_091234567_MP.jpg     # Its Motion Photo
```

You want to:

1. Group photos by date and time
2. Prioritize Motion Photos (\_MP) as primary assets
3. Then edited versions, then cropped, then originals

**Configuration:**

```json
[
  {
    "key": "originalFileName",
    "regex": {
      "key": "(PXL_\\d{8}_\\d{9})(_\\w+)?\\.(jpg|JPG)",
      "index": 1, // Group by base filename
      "promote_index": 2, // Use suffix for promotion
      "promote_keys": ["_MP", "_edit", "_crop", ""]
    }
  }
]
```

**Result:**

- Stack 1: Primary: `PXL_20230503_152823814_MP.jpg`, Others: `_edit`, `_crop`, original
- Stack 2: Primary: `PXL_20230504_091234567_MP.jpg`, Others: original

## Complete Example: Multi-Camera Setup

Imagine you have photos from multiple cameras with different naming conventions, all organized in date-based folders:

```
photos/
├── 2023-05-03/
│   ├── PXL_20230503_152823814.jpg       # Google Pixel
│   ├── PXL_20230503_152823814.dng       # Pixel RAW
│   ├── IMG_20230503_152830.jpg          # iPhone
│   ├── IMG_20230503_152830.heic         # iPhone RAW
│   └── DSC01234.jpg                     # Canon
└── 2023-05-04/
    ├── PXL_20230504_091234567.jpg
    └── IMG_20230504_091240.jpg
```

You want to:

1. Group Pixel photos (JPG + DNG) by date
2. Group iPhone photos (JPG + HEIC) by date
3. Group photos within the same date folder

**Configuration:**

```json
[
  {
    "key": "originalFileName",
    "regex": {
      "key": "(PXL|IMG)_(\\d{8})_\\d+",
      "index": 2
    }
  },
  {
    "key": "originalPath",
    "regex": {
      "key": "photos/(\\d{4}-\\d{2}-\\d{2})/",
      "index": 1
    }
  }
]
```

**Result:**

- `PXL_20230503_152823814.jpg` and `PXL_20230503_152823814.dng` → grouped by date "20230503" and folder "2023-05-03"
- `IMG_20230503_152830.jpg` and `IMG_20230503_152830.heic` → grouped by date "20230503" and folder "2023-05-03"
- Photos from different dates remain separate even if taken at similar times

This approach gives you precise control over grouping logic while handling multiple camera formats automatically.
