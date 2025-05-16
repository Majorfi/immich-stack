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

- `/app/logs`: For storing log files
  ```bash
  -v ./logs:/app/logs
  ```

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
