# Asset Operations

The asset operations are implemented in `internal/asset/asset.go`.

## Asset Structure

```go
type Asset struct {
    ID              string    `json:"id"`
    DeviceAssetID   string    `json:"deviceAssetId"`
    OwnerID         string    `json:"ownerId"`
    DeviceID        string    `json:"deviceId"`
    Type            string    `json:"type"`
    OriginalPath    string    `json:"originalPath"`
    OriginalFileName string   `json:"originalFileName"`
    Resized         bool      `json:"resized"`
    FileCreatedAt   time.Time `json:"fileCreatedAt"`
    FileModifiedAt  time.Time `json:"fileModifiedAt"`
    UpdatedAt       time.Time `json:"updatedAt"`
    IsFavorite      bool      `json:"isFavorite"`
    IsArchived      bool      `json:"isArchived"`
    IsReadOnly      bool      `json:"isReadOnly"`
    Duration        string    `json:"duration"`
    ExifInfo        ExifInfo  `json:"exifInfo"`
}
```

## Available Operations

### List Assets

```go
func ListAssets(ctx context.Context, client *immich.Client, options *ListOptions) ([]Asset, error)
```

Lists assets with optional filtering.

**Parameters:**

- `ctx`: Context for the operation
- `client`: Immich API client
- `options`: List options (filters, pagination, etc.)

**Returns:**

- `[]Asset`: Array of assets
- `error`: Any error that occurred

### Get Asset

```go
func GetAsset(ctx context.Context, client *immich.Client, assetID string) (*Asset, error)
```

Retrieves a single asset by ID.

**Parameters:**

- `ctx`: Context for the operation
- `client`: Immich API client
- `assetID`: ID of the asset to retrieve

**Returns:**

- `*Asset`: Retrieved asset
- `error`: Any error that occurred

### Update Asset

```go
func UpdateAsset(ctx context.Context, client *immich.Client, asset *Asset) error
```

Updates an existing asset.

**Parameters:**

- `ctx`: Context for the operation
- `client`: Immich API client
- `asset`: Asset to update

**Returns:**

- `error`: Any error that occurred

### Delete Asset

```go
func DeleteAsset(ctx context.Context, client *immich.Client, assetID string) error
```

Deletes an asset by ID.

**Parameters:**

- `ctx`: Context for the operation
- `client`: Immich API client
- `assetID`: ID of the asset to delete

**Returns:**

- `error`: Any error that occurred

## List Options

```go
type ListOptions struct {
    IsArchived *bool
    IsFavorite *bool
    Skip       int
    Take       int
}
```

## Error Handling

All operations handle the following error cases:

- Invalid asset ID
- Asset not found
- API errors
- Network errors
- Invalid asset data

## Best Practices

1. **Error Handling**

   - Always check returned errors
   - Use appropriate error handling strategies
   - Log errors for debugging

2. **Context Usage**

   - Pass context through all operations
   - Use context for cancellation
   - Set appropriate timeouts

3. **Asset Filtering**

   - Use appropriate filters
   - Handle pagination properly
   - Consider performance implications

4. **Asset Updates**
   - Validate changes before updating
   - Handle conflicts gracefully
   - Maintain data consistency

## Example Usage

```go
// List assets with filters
options := &ListOptions{
    IsArchived: &false,
    IsFavorite: &true,
    Take:       100,
}
assets, err := ListAssets(ctx, client, options)
if err != nil {
    log.Printf("Error listing assets: %v", err)
    return
}

// Get single asset
asset, err := GetAsset(ctx, client, "asset-id")
if err != nil {
    log.Printf("Error getting asset: %v", err)
    return
}

// Update asset
asset.IsFavorite = true
err = UpdateAsset(ctx, client, asset)
if err != nil {
    log.Printf("Error updating asset: %v", err)
    return
}

// Delete asset
err = DeleteAsset(ctx, client, "asset-id")
if err != nil {
    log.Printf("Error deleting asset: %v", err)
    return
}
```
