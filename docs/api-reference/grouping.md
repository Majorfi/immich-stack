# Grouping Operations

The grouping operations are implemented in `internal/grouping/grouping.go`.

## Group Structure

```go
type Group struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    Assets    []Asset   `json:"assets"`
    CreatedAt time.Time `json:"createdAt"`
    UpdatedAt time.Time `json:"updatedAt"`
}
```

## Available Operations

### Group Assets

```go
func GroupAssets(ctx context.Context, client *immich.Client, assets []Asset, criteria []Criterion) ([]Group, error)
```

Groups assets based on specified criteria.

**Parameters:**

- `ctx`: Context for the operation
- `client`: Immich API client
- `assets`: Array of assets to group
- `criteria`: Array of grouping criteria

**Returns:**

- `[]Group`: Array of groups
- `error`: Any error that occurred

### Get Group

```go
func GetGroup(ctx context.Context, client *immich.Client, groupID string) (*Group, error)
```

Retrieves a group by ID.

**Parameters:**

- `ctx`: Context for the operation
- `client`: Immich API client
- `groupID`: ID of the group to retrieve

**Returns:**

- `*Group`: Retrieved group
- `error`: Any error that occurred

### Update Group

```go
func UpdateGroup(ctx context.Context, client *immich.Client, group *Group) error
```

Updates an existing group.

**Parameters:**

- `ctx`: Context for the operation
- `client`: Immich API client
- `group`: Group to update

**Returns:**

- `error`: Any error that occurred

### Delete Group

```go
func DeleteGroup(ctx context.Context, client *immich.Client, groupID string) error
```

Deletes a group by ID.

**Parameters:**

- `ctx`: Context for the operation
- `client`: Immich API client
- `groupID`: ID of the group to delete

**Returns:**

- `error`: Any error that occurred

## Grouping Criteria

```go
type Criterion struct {
    Key    string      `json:"key"`
    Split  *SplitConfig `json:"split,omitempty"`
    Delta  *DeltaConfig `json:"delta,omitempty"`
}

type SplitConfig struct {
    Delimiters []string `json:"delimiters"`
    Index      int      `json:"index"`
}

type DeltaConfig struct {
    Milliseconds int64 `json:"milliseconds"`
}
```

## Error Handling

All operations handle the following error cases:

- Invalid group ID
- Group not found
- API errors
- Network errors
- Invalid group data
- Invalid criteria

## Best Practices

1. **Error Handling**

   - Always check returned errors
   - Use appropriate error handling strategies
   - Log errors for debugging

2. **Context Usage**

   - Pass context through all operations
   - Use context for cancellation
   - Set appropriate timeouts

3. **Group Management**

   - Validate groups before operations
   - Handle missing groups gracefully
   - Maintain group consistency

4. **Criteria Usage**
   - Use appropriate criteria
   - Handle edge cases
   - Consider performance implications

## Example Usage

```go
// Group assets with criteria
criteria := []Criterion{
    {
        Key: "originalFileName",
        Split: &SplitConfig{
            Delimiters: []string{"~", "."},
            Index:      0,
        },
    },
    {
        Key: "localDateTime",
        Delta: &DeltaConfig{
            Milliseconds: 1000,
        },
    },
}
groups, err := GroupAssets(ctx, client, assets, criteria)
if err != nil {
    log.Printf("Error grouping assets: %v", err)
    return
}

// Get single group
group, err := GetGroup(ctx, client, "group-id")
if err != nil {
    log.Printf("Error getting group: %v", err)
    return
}

// Update group
group.Name = "New Name"
err = UpdateGroup(ctx, client, group)
if err != nil {
    log.Printf("Error updating group: %v", err)
    return
}

// Delete group
err = DeleteGroup(ctx, client, "group-id")
if err != nil {
    log.Printf("Error deleting group: %v", err)
    return
}
```
