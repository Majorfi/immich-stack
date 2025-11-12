# API Operations Reference

All API operations are implemented in `pkg/immich/client.go`. The client provides a high-level interface to the Immich API with built-in retry logic, error handling, and dry-run support.

## Client Structure

The `Client` struct handles all API interactions:

```go
type Client struct {
    client                  *http.Client
    apiURL                  string
    apiKey                  string
    resetStacks             bool
    replaceStacks           bool
    dryRun                  bool
    withArchived            bool
    withDeleted             bool
    removeSingleAssetStacks bool
    logger                  *logrus.Logger
}
```

## Client Configuration

### Creating a Client

```go
client := immich.NewClient(
    apiURL,                    // Base URL of Immich API
    apiKey,                    // API key for authentication
    resetStacks,               // Delete all existing stacks
    replaceStacks,             // Replace stacks for new groups
    dryRun,                    // Simulate without making changes
    withArchived,              // Include archived assets
    withDeleted,               // Include deleted assets
    removeSingleAssetStacks,   // Remove single-asset stacks
    logger,                    // Logger instance
)
```

### Client Settings

- **Timeout**: 600 seconds for all requests
- **Retry Logic**: Up to 3 retries with 500ms base delay
- **Connection Pool**: 100 max idle connections
- **Idle Timeout**: 90 seconds

## Stack Operations

### FetchAllStacks

Retrieves all existing stacks from Immich.

```go
func (c *Client) FetchAllStacks() (map[string]utils.TStack, error)
```

**Returns**:

- `map[string]utils.TStack`: Map of stack IDs to stack objects
- `error`: Any error that occurred

**Usage**:

```go
stacks, err := client.FetchAllStacks()
if err != nil {
    log.Fatalf("Error fetching stacks: %v", err)
}
```

### ModifyStack

Creates or updates a stack with the given asset IDs. The first asset in the array becomes the stack parent.

```go
func (c *Client) ModifyStack(assetIDs []string) error
```

**Parameters**:

- `assetIDs`: Array of asset IDs (first is parent, rest are children)

**Returns**:

- `error`: Any error that occurred

**Behavior**:

- Respects `dryRun` flag (no-op if enabled)
- Automatically retries on failure
- Logs debug message on success

**Usage**:

```go
assetIDs := []string{parentID, child1ID, child2ID}
err := client.ModifyStack(assetIDs)
if err != nil {
    log.Errorf("Error modifying stack: %v", err)
}
```

### DeleteStack

Deletes a stack by its ID.

```go
func (c *Client) DeleteStack(stackID string, reason string) error
```

**Parameters**:

- `stackID`: ID of the stack to delete
- `reason`: Reason constant for logging (e.g., `utils.REASON_REPLACE_CHILD_STACK_WITH_NEW_ONE`)

**Returns**:

- `error`: Any error that occurred

**Behavior**:

- Respects `dryRun` flag (no-op if enabled)
- Logs the deletion reason
- Automatically retries on failure

**Usage**:

```go
err := client.DeleteStack(stackID, utils.REASON_REPLACE_CHILD_STACK_WITH_NEW_ONE)
if err != nil {
    log.Errorf("Error deleting stack: %v", err)
}
```

## Asset Operations

### FetchAssets

Fetches all assets from Immich with pagination support.

```go
func (c *Client) FetchAssets(size int, stacksMap map[string]utils.TStack) ([]utils.TAsset, error)
```

**Parameters**:

- `size`: Page size for pagination (e.g., 1000)
- `stacksMap`: Map of existing stacks to associate with assets

**Returns**:

- `[]utils.TAsset`: Array of all assets
- `error`: Any error that occurred

**Behavior**:

- Fetches assets in pages until all are retrieved
- Filters based on `withArchived` and `withDeleted` flags
- Associates assets with their stacks from stacksMap

**Usage**:

```go
assets, err := client.FetchAssets(1000, stacksMap)
if err != nil {
    log.Fatalf("Error fetching assets: %v", err)
}
```

### ListDuplicates

Identifies and lists duplicate assets based on filename and timestamp.

```go
func (c *Client) ListDuplicates(allAssets []utils.TAsset) error
```

**Parameters**:

- `allAssets`: Array of all assets to check for duplicates

**Returns**:

- `error`: Any error that occurred

**Behavior**:

- Groups assets by original filename and local datetime
- Logs duplicate groups with details
- Does not modify any assets

**Usage**:

```go
err := client.ListDuplicates(assets)
if err != nil {
    log.Errorf("Error listing duplicates: %v", err)
}
```

### FetchTrashedAssets

Retrieves all assets in the trash.

```go
func (c *Client) FetchTrashedAssets(size int) ([]utils.TAsset, error)
```

**Parameters**:

- `size`: Page size for pagination

**Returns**:

- `[]utils.TAsset`: Array of trashed assets
- `error`: Any error that occurred

**Usage**:

```go
trashedAssets, err := client.FetchTrashedAssets(1000)
if err != nil {
    log.Errorf("Error fetching trashed assets: %v", err)
}
```

### TrashAssets

Moves assets to the trash.

```go
func (c *Client) TrashAssets(assetIDs []string) error
```

**Parameters**:

- `assetIDs`: Array of asset IDs to trash

**Returns**:

- `error`: Any error that occurred

**Behavior**:

- Respects `dryRun` flag
- Processes in batches
- Automatically retries on failure

**Usage**:

```go
err := client.TrashAssets([]string{assetID1, assetID2})
if err != nil {
    log.Errorf("Error trashing assets: %v", err)
}
```

## Album Operations

### FetchAlbums

Retrieves all albums.

```go
func (c *Client) FetchAlbums() ([]utils.TAlbum, error)
```

**Returns**:

- `[]utils.TAlbum`: Array of all albums
- `error`: Any error that occurred

**Usage**:

```go
albums, err := client.FetchAlbums()
if err != nil {
    log.Errorf("Error fetching albums: %v", err)
}
```

### FetchAlbumAssets

Retrieves all assets in a specific album.

```go
func (c *Client) FetchAlbumAssets(albumID string) ([]utils.TAsset, error)
```

**Parameters**:

- `albumID`: ID of the album

**Returns**:

- `[]utils.TAsset`: Array of assets in the album
- `error`: Any error that occurred

**Usage**:

```go
assets, err := client.FetchAlbumAssets(albumID)
if err != nil {
    log.Errorf("Error fetching album assets: %v", err)
}
```

### CreateAlbum

Creates a new album.

```go
func (c *Client) CreateAlbum(name, description string) (*utils.TAlbum, error)
```

**Parameters**:

- `name`: Album name
- `description`: Album description

**Returns**:

- `*utils.TAlbum`: Created album
- `error`: Any error that occurred

**Behavior**:

- Respects `dryRun` flag
- Returns mock album in dry-run mode

**Usage**:

```go
album, err := client.CreateAlbum("Vacation 2024", "Summer vacation photos")
if err != nil {
    log.Errorf("Error creating album: %v", err)
}
```

### AddAssetsToAlbum

Adds assets to an existing album.

```go
func (c *Client) AddAssetsToAlbum(albumID string, assetIDs []string) error
```

**Parameters**:

- `albumID`: ID of the album
- `assetIDs`: Array of asset IDs to add

**Returns**:

- `error`: Any error that occurred

**Behavior**:

- Respects `dryRun` flag
- Automatically retries on failure

**Usage**:

```go
err := client.AddAssetsToAlbum(albumID, []string{assetID1, assetID2})
if err != nil {
    log.Errorf("Error adding assets to album: %v", err)
}
```

### RemoveAssetsFromAlbum

Removes assets from an album.

```go
func (c *Client) RemoveAssetsFromAlbum(albumID string, assetIDs []string) error
```

**Parameters**:

- `albumID`: ID of the album
- `assetIDs`: Array of asset IDs to remove

**Returns**:

- `error`: Any error that occurred

**Behavior**:

- Respects `dryRun` flag
- Automatically retries on failure

**Usage**:

```go
err := client.RemoveAssetsFromAlbum(albumID, []string{assetID1})
if err != nil {
    log.Errorf("Error removing assets from album: %v", err)
}
```

### UpdateAlbum

Updates album properties.

```go
func (c *Client) UpdateAlbum(albumID string, updates map[string]interface{}) error
```

**Parameters**:

- `albumID`: ID of the album to update
- `updates`: Map of properties to update (e.g., `{"albumName": "New Name"}`)

**Returns**:

- `error`: Any error that occurred

**Behavior**:

- Respects `dryRun` flag
- Automatically retries on failure

**Usage**:

```go
updates := map[string]interface{}{
    "albumName": "Vacation 2024 - Updated",
    "description": "Updated description",
}
err := client.UpdateAlbum(albumID, updates)
if err != nil {
    log.Errorf("Error updating album: %v", err)
}
```

## User Operations

### GetCurrentUser

Retrieves information about the authenticated user.

```go
func (c *Client) GetCurrentUser() (utils.TUserResponse, error)
```

**Returns**:

- `utils.TUserResponse`: User information
- `error`: Any error that occurred

**Usage**:

```go
user, err := client.GetCurrentUser()
if err != nil {
    log.Errorf("Error fetching user: %v", err)
}
log.Infof("User: %s (%s)", user.Name, user.Email)
```

## Error Handling

All operations implement consistent error handling:

### Retry Logic

- **Maximum retries**: 3 attempts
- **Base delay**: 500ms
- **Backoff**: Exponential (500ms, 1s, 2s)
- **Automatic retry**: Network errors, 5xx responses, 429 rate limit

### Error Types

```go
// Request errors
fmt.Errorf("error creating request: %w", err)
fmt.Errorf("error marshaling request body: %w", err)

// Response errors
fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
fmt.Errorf("error reading response body: %w", err)
fmt.Errorf("error decoding response: %w", err)
```

### Dry-Run Behavior

When `dryRun` is enabled:

- All write operations (create, update, delete) return success without making changes
- Read operations work normally
- Logging indicates dry-run mode

## Best Practices

### Error Handling

```go
assets, err := client.FetchAssets(1000, stacksMap)
if err != nil {
    logger.Fatalf("Critical error: %v", err)
}
```

### Batch Operations

```go
// Process in batches for large datasets
const batchSize = 1000
for i := 0; i < len(assetIDs); i += batchSize {
    end := i + batchSize
    if end > len(assetIDs) {
        end = len(assetIDs)
    }
    batch := assetIDs[i:end]
    err := client.TrashAssets(batch)
    if err != nil {
        logger.Errorf("Batch failed: %v", err)
    }
}
```

### Dry-Run Testing

```go
// Test operations without making changes
client := immich.NewClient(
    apiURL, apiKey,
    false,  // resetStacks
    true,   // replaceStacks
    true,   // dryRun - ENABLE FOR TESTING
    false,  // withArchived
    false,  // withDeleted
    false,  // removeSingleAssetStacks
    logger,
)
```

### Multi-User Support

```go
// Process multiple users sequentially
apiKeys := strings.Split(os.Getenv("API_KEYS"), ",")
for _, key := range apiKeys {
    client := immich.NewClient(apiURL, key, ...)
    user, err := client.GetCurrentUser()
    if err != nil {
        logger.Errorf("Failed for key %s: %v", key, err)
        continue
    }
    logger.Infof("Processing user: %s", user.Name)
    // ... perform operations
}
```

## Type Definitions

Key types from `pkg/utils/types.go`:

```go
type TAsset struct {
    ID               string
    OriginalFileName string
    LocalDateTime    time.Time
    OriginalPath     string
    Stack            *TStack
    IsArchived       bool
    IsTrashed        bool
    // ... other fields
}

type TStack struct {
    ID             string
    PrimaryAssetID string
    Assets         []TAsset
}

type TAlbum struct {
    ID          string
    AlbumName   string
    Description string
    AssetCount  int
    // ... other fields
}

type TUserResponse struct {
    ID    string
    Email string
    Name  string
}
```

## Common Patterns

### Complete Stack Workflow

```go
// 1. Fetch existing stacks
stacks, err := client.FetchAllStacks()
if err != nil {
    log.Fatalf("Error: %v", err)
}

// 2. Fetch all assets
assets, err := client.FetchAssets(1000, stacks)
if err != nil {
    log.Fatalf("Error: %v", err)
}

// 3. Group assets into new stacks
groups := stacker.StackBy(assets, criteria, ...)

// 4. Delete old conflicting stacks
for _, group := range groups {
    if needsReplacement(group) {
        err := client.DeleteStack(oldStackID, utils.REASON_REPLACE)
        if err != nil {
            log.Errorf("Delete failed: %v", err)
        }
    }
}

// 5. Create/update stacks
for _, group := range groups {
    assetIDs := extractIDs(group)
    err := client.ModifyStack(assetIDs)
    if err != nil {
        log.Errorf("Modify failed: %v", err)
    }
}
```

### Album Management

```go
// Create album
album, err := client.CreateAlbum("Best Photos", "Top selections")
if err != nil {
    log.Fatalf("Error: %v", err)
}

// Add assets
assetIDs := []string{id1, id2, id3}
err = client.AddAssetsToAlbum(album.ID, assetIDs)
if err != nil {
    log.Errorf("Error: %v", err)
}

// Update metadata
updates := map[string]interface{}{
    "description": "Updated: Top 50 photos",
}
err = client.UpdateAlbum(album.ID, updates)
```
