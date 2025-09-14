# Docker Integration

## Quick Start

Run Immich Stack using Docker:

```bash
# Create a .env file
cat > .env << EOL
API_KEY=your_immich_api_key
API_URL=http://immich-server:2283/api
RUN_MODE=cron
CRON_INTERVAL=60
EOL

# Run with Docker Hub
docker run -d --name immich-stack --env-file .env -v ./logs:/app/logs majorfi/immich-stack:latest

# Or using GitHub Container Registry
docker run -d --name immich-stack --env-file .env -v ./logs:/app/logs ghcr.io/majorfi/immich-stack:latest
```

## Image Sources

Immich Stack is available from two container registries:

1. **Docker Hub** (recommended for Portainer):

   ```bash
   docker pull majorfi/immich-stack:latest
   ```

2. **GitHub Container Registry**:
   ```bash
   docker pull ghcr.io/majorfi/immich-stack:latest
   ```

## Container Configuration

### Environment Variables

All configuration is done through environment variables. See [Environment Variables](../api-reference/environment-variables.md) for details.

### Volumes

The container uses one volume:

- `/app/logs`: For storing log files (only used when `LOG_FILE` is set)
  ```bash
  -v ./logs:/app/logs
  ```

**Note**: The `/app/logs` directory will remain empty unless you set the `LOG_FILE` environment variable. Without it, logs only appear in `docker logs`.

### Network

When running with Immich, use the same Docker network:

```bash
--network immich_default
```

## Building Locally

Build the Docker image locally:

```bash
# Clone the repository
git clone https://github.com/majorfi/immich-stack.git
cd immich-stack

# Build the image
docker build -t immich-stack .

# Run the container
docker run -d \
  --name immich-stack \
  --env-file .env \
  -v ./logs:/app/logs \
  immich-stack
```

## Container Management

### View Logs

```bash
# View logs
docker logs immich-stack

# Follow logs
docker logs -f immich-stack

# View last 100 lines
docker logs --tail 100 immich-stack
```

### File Logging

To enable persistent file logging:

1. Add `LOG_FILE` to your `.env`:

   ```bash
   LOG_FILE=/app/logs/immich-stack.log
   ```

2. Mount the logs volume:

   ```bash
   -v ./logs:/app/logs
   ```

3. Logs will be written to both:
   - Container stdout (viewable with `docker logs`)
   - The file `./logs/immich-stack.log` on your host

### Log Configuration

Control log verbosity and format:

```bash
# Debug logging
LOG_LEVEL=debug

# JSON format for structured logging
LOG_FORMAT=json

# Enable file logging
LOG_FILE=/app/logs/immich-stack.log
```

### Stop Container

```bash
docker stop immich-stack
```

### Remove Container

```bash
docker rm immich-stack
```

### Update Container

```bash
# Pull new image
docker pull majorfi/immich-stack:latest

# Stop and remove old container
docker stop immich-stack
docker rm immich-stack

# Run new container
docker run -d \
  --name immich-stack \
  --env-file .env \
  -v ./logs:/app/logs \
  majorfi/immich-stack:latest
```

## Best Practices

1. **Version Pinning:**

   - Use specific versions in production
   - Test new versions before updating

2. **Resource Limits:**

   - Set memory limits for large libraries
   - Monitor container resource usage

3. **Backup:**

   - Backup your `.env` file
   - Consider backing up logs

4. **Security:**
   - Use Docker secrets for sensitive data
   - Restrict container capabilities
   - Use non-root user
