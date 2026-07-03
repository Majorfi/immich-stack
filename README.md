# Immich Stack

Automatically groups similar photos into stacks within the Immich photo management system.

## Quick Start

```bash
# Create a .env file
cat > .env << EOL
API_KEY=your_immich_api_key
API_URL=http://immich-server:2283/api
RUN_MODE=cron
CRON_INTERVAL=60
# Optional: Enable file logging for persistent logs
# LOG_FILE=/app/logs/immich-stack.log
EOL

# Run with Docker (using Docker Hub)
docker run -d --name immich-stack --env-file .env -v ./logs:/app/logs majorfi/immich-stack:latest

# Or using GitHub Container Registry
docker run -d --name immich-stack --env-file .env -v ./logs:/app/logs ghcr.io/majorfi/immich-stack:latest

# View logs
docker logs -f immich-stack

# If LOG_FILE is set, logs are also saved to ./logs/immich-stack.log
```

## Documentation

For detailed documentation, please visit the repo's [wiki](https://github.com/Majorfi/immich-stack/wiki) and our [documentation site](https://majorfi.github.io/immich-stack/)

## Commands

Immich Stack provides multiple commands for different operations:

### Main Stacking Command

```bash
immich-stack
```

The default command that processes and creates stacks based on your criteria.

### Find Duplicates

```bash
immich-stack duplicates
```

Scans your library and reports duplicate assets based on filename and timestamp.

### Fix Trash Issues

```bash
immich-stack fix-trash
```

Identifies trashed assets and moves their related stack members to trash for consistency. With `--trash-orphaned-raws`, it also removes RAW files left without a developed companion (JPG, HEIC, ...).

## Features

- Groups similar photos into stacks based on filename, date, and custom criteria.
- Detects burst sequences via the `sequence` keyword (Sony's `DSCPDC_0001_BURST`,
  Canon's `IMG_0001`, and similar patterns).
- Lists duplicates by filename and capture time.
- Cleans up trash operations: moves stack members of trashed assets to trash too, and
  optionally RAW files left without a developed companion.
- Runs against multiple users in one go (comma-separated API keys).
- Lets you pick the stack parent by extension, filename pattern, regex, or size.
- Dry-run mode, stack replacement, and a confirmation-gated reset.
- Colorized, structured logs at configurable verbosity.

## License

MIT
