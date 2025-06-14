# Immich Stack

Immich Stack is a Go CLI tool and library for automatically grouping ("stacking") similar photos in the [Immich](https://github.com/immich-app/immich) photo management system. It provides configurable, robust, and extensible logic for grouping, sorting, and managing photo stacks via the Immich API.

## Features

- **Automatic Stacking:** Groups similar photos into stacks based on filename, date, and custom criteria
- **Smart Burst Photo Handling:** Automatically detects and properly orders burst photo sequences with the flexible `sequence` keyword (e.g., Sony's DSCPDC_0001_BURST, Canon's IMG_0001, etc.)
- **Duplicate Detection:** Find and list duplicate assets based on filename and timestamp
- **Stack-Aware Trash Management:** Fix incomplete trash operations by moving related stack members to trash
- **Multi-User Support:** Process multiple users sequentially with comma-separated API keys
- **Configurable Grouping:** Custom grouping logic via environment variables and command-line flags
- **Parent/Child Promotion:** Fine-grained control over stack parent selection with intelligent sequence detection and the `sequence` keyword
- **Safe Operations:** Dry-run mode, stack replacement, and reset with confirmation
- **Comprehensive Logging:** Colorful, structured logs with configurable levels and formats
- **Tested and Modular:** Table-driven tests and clear separation of concerns

## Quick Links

### Getting Started

- [Installation](getting-started/installation.md)
- [Quick Start](getting-started/quick-start.md)
- [Configuration](getting-started/configuration.md)

### Commands

- [Stack Command](api-reference/cli-usage.md#stack-command-flags) - Main stacking functionality
- [Duplicates Command](commands/duplicates.md) - Find duplicate assets
- [Fix-Trash Command](commands/fix-trash.md) - Fix incomplete trash operations

### Features & Reference

- [Stacking Logic](features/stacking-logic.md)
- [CLI Usage](api-reference/cli-usage.md)
- [API Reference](api-reference/environment-variables.md)

## License

MIT
