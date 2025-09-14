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

For detailed documentation, please visit our [documentation site](https://majorfi.github.io/immich-stack/).

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

Identifies trashed assets and moves their related stack members to trash for consistency.

## Features

- **Automatic Stacking:** Groups similar photos into stacks based on filename, date, and custom criteria
- **Smart Burst Photo Handling:** Automatically detects and properly orders burst photo sequences
- **Duplicate Detection:** Find and list duplicate assets based on filename and timestamp
- **Stack-Aware Trash Management:** Fix incomplete trash operations by moving related stack members to trash
- **Multi-User Support:** Process multiple users sequentially with comma-separated API keys
- **Configurable Grouping:** Custom grouping logic via environment variables and command-line flags
- **Parent/Child Promotion:** Fine-grained control over stack parent selection with intelligent sequence detection and regex-based promotion
- **Safe Operations:** Dry-run mode, stack replacement, and reset with confirmation
- **Comprehensive Logging:** Colorful, structured logs for all operations
- **Tested and Modular:** Table-driven tests and clear separation of concerns

## License

MIT
