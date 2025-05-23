# Custom Criteria

Immich Stack allows you to define custom criteria for grouping photos using a JSON configuration. This gives you fine-grained control over how photos are grouped into stacks.

## Basic Structure

The `CRITERIA` environment variable accepts a JSON array of criteria objects. Each criterion has:

- `key`: The field to group by
- Optional configuration (split, delta, etc.)

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

The `split` configuration allows you to extract parts of string values:

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

3. **Testing:**
   - Use `DRY_RUN=true` to test configurations
   - Check logs for grouping results
   - Adjust criteria based on results
