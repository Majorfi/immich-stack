# Quick Start

## Basic Usage

1. Create a `.env` file:

```bash
cat > .env << EOL
API_KEY=your_immich_api_key
API_URL=http://immich-server:2283/api
RUN_MODE=cron
CRON_INTERVAL=60
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
