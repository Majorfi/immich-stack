# Immich Stack

Immich Stack is a Go CLI (and library) that groups similar photos into stacks
in [Immich](https://github.com/immich-app/immich). Grouping criteria, sort
order, and parent selection are configurable via env vars or CLI flags.

## Features

- Groups similar photos into stacks based on filename, date, and custom criteria.
- Detects burst sequences via the `sequence` keyword (Sony's `DSCPDC_0001_BURST`,
  Canon's `IMG_0001`, and similar patterns).
- Lists duplicates by filename and capture time.
- Cleans up trash operations: moves stack members of trashed assets to trash too.
- Runs against multiple users in one go (comma-separated API keys).
- Lets you pick the stack parent by extension, filename pattern, regex, or size.
- Dry-run mode, stack replacement, and a confirmation-gated reset.
- Colorized, structured logs at configurable verbosity.

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
