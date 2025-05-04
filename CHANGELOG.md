# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Changed

- Replaced `ENABLE_CRON` with `RUN_MODE` environment variable for better control over execution modes
- Updated Docker setup to handle both one-time runs and cron mode more elegantly
- Improved documentation for run modes and restart policies
- Refactored main.go to support both "once" and "cron" run modes with proper interval handling
- Split stacker logic into separate functions for better code organization and reusability
- Simplified CLI by making the run command the default behavior

### Added

- New run modes: "once" (default) and "cron"
- New restart policy options: "no" (default), "unless-stopped", "always"
- Cron interval configuration via `CRON_INTERVAL` environment variable or `--cron-interval` flag
- Proper logging for run mode and interval information

### Added

- GitHub Actions workflow for automated releases
- GoReleaser configuration for building multi-platform binaries
- Pre-built binary distribution support for Linux, macOS, and Windows
- Support for both AMD64 and ARM64 architectures
- Docker support with multi-stage build
- Docker Compose configuration with optional cron mode via ENABLE_CRON
- Environment variable configuration for Docker
- Docker installation instructions in README
- Example environment file (.env.example)
- Integration guide for Immich Docker Compose

### Changed

- Updated README with installation and running instructions
- Added pre-built binary installation option
- Simplified Docker setup to use a single service with optional cron mode
- Made Docker setup more flexible for both standalone and Immich integration
- Updated Docker commands to use newer `docker compose` syntax
- Improved Docker container with proper shell script handling
- Updated entrypoint script to properly pass command line arguments

### Fixed

- Removed redundant `SortStack` call in `main.go` since stacks are already sorted by `StackBy`
- Fixed Docker container shell script execution issues
- Fixed command line argument passing in Docker container
- Fixed permission issues in Dockerfile by creating entrypoint script before user switch
