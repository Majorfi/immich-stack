# Immich Stack

Immich Stack is a Go CLI tool and library for automatically grouping ("stacking") similar photos in the [Immich](https://github.com/immich-app/immich) photo management system. It provides configurable, robust, and extensible logic for grouping, sorting, and managing photo stacks via the Immich API.

## Features

- **Automatic Stacking:** Groups similar photos into stacks based on filename, date, and custom criteria
- **Multi-User Support:** Process multiple users sequentially with comma-separated API keys
- **Configurable Grouping:** Custom grouping logic via environment variables and command-line flags
- **Parent/Child Promotion:** Fine-grained control over stack parent selection
- **Safe Operations:** Dry-run mode, stack replacement, and reset with confirmation
- **Comprehensive Logging:** Colorful, structured logs for all operations
- **Tested and Modular:** Table-driven tests and clear separation of concerns

## Quick Links

- [Installation](getting-started/installation.md)
- [Quick Start](getting-started/quick-start.md)
- [Configuration](getting-started/configuration.md)
- [Stacking Logic](features/stacking-logic.md)
- [API Reference](api-reference/environment-variables.md)

## License

MIT
