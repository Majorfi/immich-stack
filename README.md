# Immich Front Back

Automatically stacks front/back photos (or original + enhanced versions) in Immich.

## Overview

Groups three versions of photos based on filename suffixes:
- `FILENAME.jpg` - Original (becomes stack parent/thumbnail)
- `FILENAME_a.jpg` - Enhanced version (hidden)
- `FILENAME_b.jpg` - Back of photo/document (hidden)

## Quick Start

```bash
docker run -d \
  --name immich-front-back \
  -e API_KEY=your_immich_api_key \
  -e API_URL=http://immich:2283/api \
  ghcr.io/sd-leighericksen/immich-front-back:latest
```

## Configuration

Environment variables:

```bash
API_KEY=your_immich_api_key       # Required
API_URL=http://immich:2283/api    # Required
RUN_MODE=cron                      # 'once' or 'cron' (default: cron)
CRON_INTERVAL=3600                 # Seconds between runs
DRY_RUN=false                      # Set to 'true' to preview only
LOG_LEVEL=info                     # debug, info, warn, error
```

## Examples

### Document Scanning
```
document_001.jpg     → Stack parent (visible)
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

## License

MIT (forked from https://github.com/Majorfi/immich-stack)
