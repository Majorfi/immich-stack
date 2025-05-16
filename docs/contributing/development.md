# Development Guide

## Directory Structure

```
immich-auto-stack/
├── cmd/                # CLI entrypoint (main.go)
├── pkg/
│   ├── stacker/        # Stacking logic, types, and tests
│   ├── immich/         # Immich API client and integration
│   └── utils/          # Utility helpers and logging
```

## Building Locally

```bash
# Build locally
docker build -t immich-stack .

# Run locally
docker run -d \
  --name immich-stack \
  --env-file .env \
  -v ./logs:/app/logs \
  immich-stack
```

## Code Style

- Follow the code style and comment conventions (see code for examples)
- Add tests for new features
- Document all exported functions and types

## Extending

- **Custom Grouping:** Edit or override criteria via command-line flags or environment variables
- **Custom Promotion:** Set `--parent-filename-promote` and/or `--parent-ext-promote` for your workflow
- **API Integration:** Extend `pkg/immich/client.go` for new Immich endpoints

## Library Structure

### pkg/stacker

- **StackBy:** Groups assets into stacks and sorts them based on promotion rules
- **SortStack:** Sorts assets in a stack by promotion and extension rules
- **Types:** `Asset`, `Stack`, `Criteria`, etc.

### pkg/immich

- **Client:** Handles all Immich API interactions (fetch, modify, delete stacks/assets)
- **FetchAllStacks:** Retrieves all stacks, with reset and cleanup logic
- **FetchAssets:** Retrieves all assets, paginated
- **ModifyStack/DeleteStack:** Stack management
- **ListDuplicates:** Finds and logs duplicate assets

### pkg/utils

- **helper.go:** Array comparison, string cleaning
- **logs.go:** Colorful, structured logging helpers (info, error, debug, pretty-print)
