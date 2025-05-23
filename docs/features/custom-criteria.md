# Custom Criteria

Immich Stack allows you to define custom criteria for grouping photos using a JSON configuration. This gives you fine-grained control over how photos are grouped into stacks.

## Basic Structure

The `CRITERIA` environment variable accepts a JSON array of criteria objects. Each criterion has:

- `key`: The field to group by
- Optional configuration (split, regex, delta, etc.)

Example:

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

## Examples

### Basic Filename Grouping

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

### Regex-Based Date Grouping

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

This groups all Pixel camera photos taken on the same date.

### Time-Based Grouping

```json
[
  {
    "key": "localDateTime",
    "delta": {
      "milliseconds": 5000
    }
  }
]
```

### Directory-Based Grouping

```json
[
  {
    "key": "originalPath",
    "split": {
      "delimiters": ["/"],
      "index": 2
    }
  }
]
```

This will group photos by their directory name (e.g., all photos in the "vacation" directory will be grouped together).

### Advanced: Regex Path and Filename Combination

```json
[
  {
    "key": "originalFileName",
    "regex": {
      "key": "PXL_(\\d{8})_\\d{9}",
      "index": 1
    }
  },
  {
    "key": "originalPath",
    "regex": {
      "key": "photos/\\d{4}/([^/]+)/",
      "index": 1
    }
  }
]
```

This groups photos by both the date extracted from the filename AND the folder name from the path.

### Combined Path and Time Criteria

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

This will group photos that are both in the same directory and taken within 1 second of each other.

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

4. **Testing:**
   - Use `DRY_RUN=true` to test configurations
   - Check logs for grouping results
   - Adjust criteria based on results

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
