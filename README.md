# Immich Front Back

Automatically stacks front/back photos (or original + enhanced versions) in Immich.

## Overview

Groups photo variants based on filename suffixes using **OR logic**: assets are stacked if they share the same base filename (regardless of when they were uploaded) OR were captured within 1000ms of each other.

- `FILENAME.jpg` — Original (becomes stack parent/thumbnail)
- `FILENAME_a.jpg` — Enhanced version or front scan (stacked)
- `FILENAME_b.jpg` — Back of photo/document (stacked)
- `FILENAME_c.jpg` … `FILENAME_z.jpg` — Additional variants (stacked)

This means `FILENAME.jpg` and `FILENAME_a.jpg` will always group together even if they were scanned hours or days apart.

## Quick Start

```bash
docker run -d \
  --name immich-front-back \
  -e API_KEY=your_immich_api_key \
  -e API_URL=http://immich:2283/api \
  ghcr.io/sd-leighericksen/immich-front-back:latest
```

## Configuration

| Variable | Default | Description |
|---|---|---|
| `API_KEY` | — | Required. Comma-separated list for multiple users |
| `API_URL` | — | Required. e.g. `http://immich:2283/api` |
| `RUN_MODE` | `cron` | `once` or `cron` |
| `CRON_INTERVAL` | `3600` | Seconds between runs |
| `DRY_RUN` | `false` | Preview changes without applying |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `RESET_STACKS` | `false` | Delete all existing stacks before running |
| `REPLACE_STACKS` | `true` | Replace stacks when a group changes |
| `WITH_ARCHIVED` | `false` | Include archived assets |
| `WITH_DELETED` | `false` | Include trashed assets |
| `PARENT_FILENAME_PROMOTE` | `,a,b` | Suffix priority for stack cover (bare name first) |
| `PARENT_EXT_PROMOTE` | `.jpg,.png,.jpeg,.heic,.dng` | Extension priority for stack cover |
| `CRITERIA` | — | Override stacking criteria (JSON, see below) |

## Examples

### Document Scanning
```
document_001.jpg     → Stack parent (visible in library)
document_001_a.jpg   → Enhanced scan (stacked)
document_001_b.jpg   → Back of document (stacked)
```

### Photo Enhancement Workflow
```
IMG_1234.jpg    → Original (stack parent)
IMG_1234_a.jpg  → Topaz AI enhanced (stacked)
IMG_1234_b.jpg  → Alternate edit (stacked)
```

## Docker Compose

```yaml
services:
  immich-front-back:
    image: ghcr.io/sd-leighericksen/immich-front-back:latest
    environment:
      - API_KEY=${IMMICH_API_KEY}
      - API_URL=http://immich:2283/api
      - RUN_MODE=cron
      - CRON_INTERVAL=3600
    restart: unless-stopped
```

For multiple Immich users, provide a comma-separated list of API keys:

```yaml
environment:
  - API_KEY=key_for_user1,key_for_user2
```

Note: set the value directly in the `environment:` block (not in a `.env` file) to avoid comma truncation.

## Custom Criteria

Override the default stacking logic with a JSON criteria expression:

```bash
# AND logic: same base filename AND within 1 second
CRITERIA='{"mode":"advanced","expression":{"operator":"AND","children":[{"criteria":{"key":"originalFileName","regex":{"key":"^(.+?)(?:_[a-z])?\\.","index":1}}},{"criteria":{"key":"localDateTime","delta":{"milliseconds":1000}}}]}}'

# Legacy array format (AND logic)
CRITERIA='[{"key":"originalFileName","regex":{"key":"^(.+?)(?:_[a-z])?\\.", "index":1}},{"key":"localDateTime","delta":{"milliseconds":1000}}]'
```

## CLI Usage

```bash
immich-front-back [flags]
immich-front-back duplicates
immich-front-back fix-trash
```

Flags mirror the environment variables above (e.g. `--dry-run`, `--api-key`, `--log-level`).

## License

MIT (forked from https://github.com/Majorfi/immich-stack)
