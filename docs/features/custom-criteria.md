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

### Combined Criteria

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
  },
  {
    "key": "fileCreatedAt",
    "delta": {
      "milliseconds": 5000
    }
  }
]
```

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
