# Installation

## Prerequisites

- [Go](https://golang.org/doc/install) (version 1.21 or later)
- [Git](https://git-scm.com/downloads)

## From Source

1. Clone the repository:

   ```sh
   git clone https://github.com/majorfi/immich-stack.git
   cd immich-stack
   ```

1. Build the binary:

   ```sh
   go build -o immich-stack ./cmd/...
   ```

1. Move the binary to your PATH (optional):

   ```sh
   sudo mv immich-stack /usr/local/bin/
   ```

## Using Pre-built Binaries

1. Download the latest release from the [Releases page](https://github.com/majorfi/immich-stack/releases)
1. Extract the archive
1. Move the binary to your PATH (optional)

## Docker Installation

1. Clone the repository:

   ```sh
   git clone https://github.com/majorfi/immich-stack.git
   cd immich-stack
   ```

1. Create a `.env` file from the example:

   ```sh
   cp .env.example .env
   ```

1. Edit the `.env` file with your Immich credentials and preferences:

   ```sh
   # Required
   API_KEY=your_immich_api_key
   API_URL=http://your_immich_server:3001/api

   # Optional - Default values shown
   DRY_RUN=false
   RESET_STACKS=false
   REPLACE_STACKS=false
   PARENT_FILENAME_PROMOTE=edit
   PARENT_EXT_PROMOTE=.jpg,.dng
   WITH_ARCHIVED=false
   WITH_DELETED=false

   # Run mode settings
   RUN_MODE=once  # Options: once, cron
   CRON_INTERVAL=86400  # in seconds, only used if RUN_MODE=cron
   ```

1. Start the service:

   ```sh
   docker compose up -d
   ```

1. To run in cron mode, set `RUN_MODE=cron` in your `.env` file and restart:

   ```sh
   docker compose down
   docker compose up -d
   ```

1. To view logs:

   ```sh
   docker compose logs -f
   ```

1. To stop the service:

   ```sh
   docker compose down
   ```
