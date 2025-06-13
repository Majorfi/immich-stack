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
EOL

# Run with Docker (using Docker Hub)
docker run -d --name immich-stack --env-file .env -v ./logs:/app/logs majorfi/immich-stack:latest

# Or using GitHub Container Registry
docker run -d --name immich-stack --env-file .env -v ./logs:/app/logs ghcr.io/majorfi/immich-stack:latest
```

## Documentation

For detailed documentation, please visit our [documentation site](https://majorfi.github.io/immich-stack/).

## Features

- **Automatic Stacking:** Groups similar photos into stacks based on filename, date, and custom criteria
- **Smart Burst Photo Handling:** Automatically detects and properly orders burst photo sequences
- **Multi-User Support:** Process multiple users sequentially with comma-separated API keys
- **Configurable Grouping:** Custom grouping logic via environment variables and command-line flags
- **Parent/Child Promotion:** Fine-grained control over stack parent selection with intelligent sequence detection
- **Safe Operations:** Dry-run mode, stack replacement, and reset with confirmation
- **Comprehensive Logging:** Colorful, structured logs for all operations
- **Tested and Modular:** Table-driven tests and clear separation of concerns

## License

MIT
