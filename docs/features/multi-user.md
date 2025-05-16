# Multi-User Support

Immich Stack supports processing multiple users' photos by accepting multiple API keys. This is useful for:

- Family accounts
- Shared photo libraries
- Multiple user management

## Configuration

To use multiple API keys, separate them with commas in the `API_KEY` environment variable:

```sh
API_KEY=key1,key2,key3
```

Or when using Docker:

```yaml
environment:
  - API_KEY=key1,key2,key3
```

## Processing Flow

1. The stacker will process each user sequentially
2. Each user's name and email are logged before processing
3. Stacks are created and managed separately for each user
4. Logs clearly indicate which user is being processed

## Example

```sh
# .env file
API_KEY=abc123,def456,ghi789
API_URL=http://immich-server:2283/api
```

When running, you'll see logs like:

```
Processing user: John Doe (john@example.com)
Found 1000 assets
Created 50 stacks
...

Processing user: Jane Doe (jane@example.com)
Found 800 assets
Created 40 stacks
...

Processing user: Bob Smith (bob@example.com)
Found 1200 assets
Created 60 stacks
...
```

## Best Practices

1. **API Key Management:**

   - Keep API keys secure
   - Rotate keys periodically
   - Use different keys for different users

2. **Resource Usage:**

   - Consider running during off-peak hours
   - Monitor system resources
   - Adjust cron interval based on library size

3. **Error Handling:**
   - If one user fails, others will still be processed
   - Check logs for any user-specific issues
   - Retry failed users if needed
