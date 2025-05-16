# Environment Variables

| Variable                  | Description                                                                                                                  | Default                         |
| ------------------------- | ---------------------------------------------------------------------------------------------------------------------------- | ------------------------------- |
| `API_KEY`                 | Your Immich API key(s), comma-separated for multiple users                                                                   | (required)                      |
| `API_URL`                 | Immich API URL                                                                                                               | `http://immich-server:2283/api` |
| `RUN_MODE`                | Run mode (`once` or `cron`)                                                                                                  | `once`                          |
| `CRON_INTERVAL`           | Interval in seconds for cron mode                                                                                            | `86400`                         |
| `DRY_RUN`                 | Don't apply changes                                                                                                          | `false`                         |
| `RESET_STACKS`            | Delete all existing stacks                                                                                                   | `false`                         |
| `CONFIRM_RESET_STACK`     | Required for RESET_STACKS. Must be set to: 'I acknowledge all my current stacks will be deleted and new one will be created' | (required for RESET_STACKS)     |
| `REPLACE_STACKS`          | Replace stacks for new groups                                                                                                | `false`                         |
| `PARENT_FILENAME_PROMOTE` | Parent filename promote                                                                                                      | `edit`                          |
| `PARENT_EXT_PROMOTE`      | Parent extension promote                                                                                                     | `.jpg,.dng`                     |
| `WITH_ARCHIVED`           | Include archived assets                                                                                                      | `false`                         |
| `WITH_DELETED`            | Include deleted assets                                                                                                       | `false`                         |
| `CRITERIA`                | JSON array of custom criteria for grouping photos (see [Custom Criteria](features/custom-criteria.md))                       | See Default Configuration       |

## Default Configuration

### Default Criteria

By default, Immich Stack groups photos based on two criteria:

1. Original filename (before extension)
   - Splits the filename on "~" and "." delimiters
   - Uses the first part (index 0) for grouping
2. Local capture time (localDateTime)
   - By default, no delta is applied (exact time matching)
   - Can be configured with a delta for flexible time matching

### Time Delta Feature

The delta feature allows for flexible time matching when grouping photos. It's particularly useful when dealing with burst photos or photos taken in quick succession that might have slight time differences.

For example, these two timestamps would normally be considered different:

```
2023-08-24T17:00:15.915Z
2023-08-24T17:00:15.810Z
```

By setting a delta of 1000ms (1 second), both timestamps would be rounded to the nearest second and considered the same for grouping purposes:

```
2023-08-24T17:00:15.000Z
```

Delta can be configured for any time-based field:

- `localDateTime`
- `fileCreatedAt`
- `fileModifiedAt`
- `updatedAt`
