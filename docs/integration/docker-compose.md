# Docker Compose Integration

## Basic Configuration

```yaml
version: "3.8"

services:
  immich-stack:
    container_name: immich_stack
    # Use Docker Hub image (recommended for Portainer)
    image: majorfi/immich-stack:main # or :latest after next release
    # Or use GitHub Container Registry
    # image: ghcr.io/majorfi/immich-stack:latest
    environment:
      - API_KEY=${API_KEY} # Can be a single key or comma-separated for multiple users
      - API_URL=${API_URL:-http://immich-server:2283/api}
      - DRY_RUN=${DRY_RUN:-false}
      - RESET_STACKS=${RESET_STACKS:-false}
      - CONFIRM_RESET_STACK=${CONFIRM_RESET_STACK}
      # Note: RESET_STACKS requires RUN_MODE=once; it will error in cron mode
      - REPLACE_STACKS=${REPLACE_STACKS:-false}
      - PARENT_FILENAME_PROMOTE=${PARENT_FILENAME_PROMOTE:-edit}
      - PARENT_EXT_PROMOTE=${PARENT_EXT_PROMOTE:-.jpg,.dng}
      - WITH_ARCHIVED=${WITH_ARCHIVED:-false}
      - WITH_DELETED=${WITH_DELETED:-false}
      - RUN_MODE=${RUN_MODE:-once}
      - CRON_INTERVAL=${CRON_INTERVAL:-86400}
    volumes:
      - ./logs:/app/logs
    restart: on-failure
```

## Integration with Immich Docker Compose

To integrate with an existing Immich installation:

1. Copy the `immich-stack` service from our `docker-compose.yml` into your Immich's `docker-compose.yml`

2. Add these environment variables to your Immich's `.env` file (you can also add the optional ones):

   ```sh
   # Immich Stack settings
   API_KEY=your_immich_api_key
   API_URL=http://immich-server:2283/api  # Use internal Docker network
   RUN_MODE=once  # Options: once, cron
   CRON_INTERVAL=86400  # in seconds, only used if RUN_MODE=cron
   ```

3. Add the service dependency in Immich's `docker-compose.yml`:

   ```yaml
   immich-stack:
     container_name: immich_stack
     image: ghcr.io/majorfi/immich-stack:latest
     environment:
       - API_KEY=${API_KEY}
     - API_URL=${API_URL:-http://immich-server:2283/api}
     - DRY_RUN=${DRY_RUN:-false}
     - RESET_STACKS=${RESET_STACKS:-false}
     - CONFIRM_RESET_STACK=${CONFIRM_RESET_STACK}
      # Note: RESET_STACKS requires RUN_MODE=once; it will error in cron mode
     - REPLACE_STACKS=${REPLACE_STACKS:-false}
     - PARENT_FILENAME_PROMOTE=${PARENT_FILENAME_PROMOTE:-edit}
     - PARENT_EXT_PROMOTE=${PARENT_EXT_PROMOTE:-.jpg,.dng}
     - WITH_ARCHIVED=${WITH_ARCHIVED:-false}
     - WITH_DELETED=${WITH_DELETED:-false}
     - RUN_MODE=${RUN_MODE:-once}
       - CRON_INTERVAL=${CRON_INTERVAL:-86400}
     volumes:
       - ./logs:/app/logs
     restart: on-failure
     depends_on:
       immich-server:
         condition: service_healthy
   ```

4. Restart your Immich stack:
   ```sh
   docker compose down
   docker compose up -d
   ```
