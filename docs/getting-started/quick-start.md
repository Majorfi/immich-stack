# Quick Start

## Basic Usage

1. Create a `.env` file:

```bash
cat > .env << EOL
API_KEY=your_immich_api_key
API_URL=http://immich-server:2283/api
RUN_MODE=cron
CRON_INTERVAL=60
# Optional: Enable file logging for persistent logs
# LOG_FILE=/app/logs/immich-stack.log
EOL
```

2. Run with Docker (using Docker Hub):

```bash
docker run -d --name immich-stack --env-file .env -v ./logs:/app/logs majorfi/immich-stack:latest
```

Or using GitHub Container Registry:

```bash
docker run -d --name immich-stack --env-file .env -v ./logs:/app/logs ghcr.io/majorfi/immich-stack:latest
```

## Running Locally

1. Create a `.env` file in your working directory with your Immich credentials:

```sh
API_KEY=your_immich_api_key
API_URL=http://your_immich_server:3001/api
```

2. Run the stacker:

```sh
# Using the binary
./immich-stack

# Or if installed in PATH
immich-stack
```

## Available Commands

### Create Stacks (Default)

```sh
# Run the main stacking operation
immich-stack
# Or explicitly:
immich-stack stack
```

### Find Duplicates

```sh
# Identify duplicate assets in your library
immich-stack duplicates
```

### Fix Trash Consistency

```sh
# Move related assets to trash when their companions are trashed
immich-stack fix-trash --dry-run  # Preview first
immich-stack fix-trash             # Execute
```

3. Optional: Configure additional options via environment variables or flags:

```sh
# Example with flags
./immich-stack --dry-run --parent-filename-promote=edit --parent-ext-promote=.jpg,.dng --with-archived --with-deleted

# Or using environment variables
export DRY_RUN=true
export PARENT_FILENAME_PROMOTE=edit
export PARENT_EXT_PROMOTE=.jpg,.dng
export WITH_ARCHIVED=true
export WITH_DELETED=true
./immich-stack
```

## Burst Photo Example

For burst photos from cameras like Sony, Canon, etc., you can use the flexible `sequence` keyword or numeric sequences:

### Using the Sequence Keyword (Recommended)

```sh
# Order any burst photos by their numeric sequence
export PARENT_FILENAME_PROMOTE=sequence

# For Sony burst photos with COVER priority
export PARENT_FILENAME_PROMOTE=COVER,sequence

# For Canon burst photos with specific 4-digit format
export PARENT_FILENAME_PROMOTE=sequence:4

# For files with specific prefix
export PARENT_FILENAME_PROMOTE=sequence:IMG_

./immich-stack
```

### Using Numeric Sequences (Legacy)

```sh
# For Sony burst photos (DSCPDC_0000_BURST..., DSCPDC_0001_BURST..., etc.)
export PARENT_FILENAME_PROMOTE=0000,0001,0002,0003

# For Canon burst photos (IMG_0001, IMG_0002, etc.)
export PARENT_FILENAME_PROMOTE=IMG_0001,IMG_0002,IMG_0003

# The system automatically detects sequences and orders photos correctly
# Even files beyond your list (e.g., 0999) will be sorted properly
./immich-stack
```
