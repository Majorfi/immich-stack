# Immich Stack

Immich Stack is a Go CLI tool and library for automatically grouping ("stacking") similar photos in the [Immich](https://github.com/immich-app/immich) photo management system. It provides configurable, robust, and extensible logic for grouping, sorting, and managing photo stacks via the Immich API.
This project is heavily inspired by [immich-auto-stack](github.com/tenekev/immich-auto-stack).

---

## Features

- **Automatic Stacking:** Groups similar photos into stacks based on filename, date, and custom criteria.
- **Configurable Grouping:** Supports custom grouping logic via environment variables and command-line flags.
- **Parent/Child Promotion:** Fine-grained control over which files are promoted as stack parents (by substring or extension).
- **CLI Tool:** Command-line interface for batch processing and automation.
- **Safe Operations:** Supports dry-run mode, stack replacement, and reset with user confirmation.
- **Comprehensive Logging:** Colorful, structured logs for all operations.
- **Tested and Modular:** Table-driven tests, modular helpers, and clear separation of concerns.

---

## Directory Structure

```
immich-auto-stack/
├── cmd/                # CLI entrypoint (main.go)
├── pkg/
│   ├── stacker/        # Stacking logic, types, and tests
│   ├── immich/         # Immich API client and integration
│   └── utils/          # Utility helpers and logging
```

---

## CLI Usage

The main entrypoint is `cmd/main.go`, which provides a Cobra-based CLI:

```sh
go run ./cmd/main.go run --api-key <API_KEY> --api-url <API_URL> [flags]
```

### Flags and Environment Variables

| Flag                        | Env Var                   | Description                                  |
| --------------------------- | ------------------------- | -------------------------------------------- |
| `--api-key`                 | `API_KEY`                 | Immich API key                               |
| `--api-url`                 | `API_URL`                 | Immich API base URL                          |
| `--reset-stacks`            | `RESET_STACKS`            | Delete all existing stacks before processing |
| `--replace-stacks`          | `REPLACE_STACKS`          | Replace stacks for new groups                |
| `--dry-run`                 | `DRY_RUN`                 | Simulate actions without making changes      |
| `--criteria`                | `CRITERIA`                | Custom grouping criteria                     |
| `--parent-filename-promote` | `PARENT_FILENAME_PROMOTE` | Substrings to promote as parent filenames    |
| `--parent-ext-promote`      | `PARENT_EXT_PROMOTE`      | Extensions to promote as parent files        |

- Flags take precedence over environment variables.
- If `--reset-stacks` is set, user confirmation is required.

---

## Stacking Logic

### Grouping

- **Default Criteria:** Groups by base filename (before extension) and local capture time.
- **Custom Criteria:** Override with the `--criteria` flag or `CRITERIA` environment variable.

### Sorting

- **Parent Promotion:** Use `--parent-filename-promote` or `PARENT_FILENAME_PROMOTE` (comma-separated substrings) to promote files as stack parents.
- **Extension Promotion:** Use `--parent-ext-promote` or `PARENT_EXT_PROMOTE` (comma-separated extensions) to further prioritize.
- **Extension Rank:** Built-in priority: `.jpeg` > `.jpg` > `.png` > others.
- **Alphabetical:** Final tiebreaker.

### Example

For files: `L1010229.JPG`, `L1010229.edit.jpg`, `L1010229.DNG`
With `PARENT_FILENAME_PROMOTE=edit` and `PARENT_EXT_PROMOTE=.jpg,.dng` in your .env file, or with `--parent-filename-promote=edit` and `--parent-ext-promote=.jpg,.dng`, the order will be:

```
L1010229.edit.jpg
L1010229.JPG
L1010229.DNG
```

---

## Library Structure

### pkg/stacker

- **StackBy:** Groups assets into stacks and sorts them based on promotion rules.
- **SortStack:** Sorts assets in a stack by promotion and extension rules.
- **Types:** `Asset`, `Stack`, `Criteria`, etc.

### pkg/immich

- **Client:** Handles all Immich API interactions (fetch, modify, delete stacks/assets).
- **FetchAllStacks:** Retrieves all stacks, with reset and cleanup logic.
- **FetchAssets:** Retrieves all assets, paginated.
- **ModifyStack/DeleteStack:** Stack management.
- **ListDuplicates:** Finds and logs duplicate assets.

### pkg/utils

- **helper.go:** Array comparison, string cleaning.
- **logs.go:** Colorful, structured logging helpers (info, error, debug, pretty-print).

---

## Example Workflow

1. **Fetch all stacks and assets** from Immich.
2. **Group assets** into stacks using criteria.
3. **Sort each stack** to determine the parent and children.
4. **Apply changes** via the Immich API (create, update, or delete stacks as needed).
5. **Log all actions** and optionally run in dry-run mode for safety.

---

## Testing

- Table-driven tests for all major logic in `pkg/stacker/stacker_test.go` and `pkg/immich/client_test.go`.
- Run with:
  ```sh
  go test ./pkg/...
  ```

---

## Extending

- **Custom Grouping:** Edit or override criteria via command-line flags or environment variables.
- **Custom Promotion:** Set `--parent-filename-promote` and/or `--parent-ext-promote` for your workflow.
- **API Integration:** Extend `pkg/immich/client.go` for new Immich endpoints.

---

## Contributing

- Follow the code style and comment conventions (see code for examples).
- Add tests for new features.
- Document all exported functions and types.

---

## License

MIT
