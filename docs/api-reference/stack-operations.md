# Stack Operations

The stack operations are implemented in `internal/stack/stack.go`.

## Stack Structure

```go
type Stack struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    CreatedAt time.Time `json:"createdAt"`
    UpdatedAt time.Time `json:"updatedAt"`
    Assets    []Asset   `json:"assets"`
}
```

## Available Operations

### Create Stack

```go
func CreateStack(ctx context.Context, client *immich.Client, assets []Asset) (*Stack, error)
```

Creates a new stack with the given assets.

**Parameters:**

- `ctx`: Context for the operation
- `client`: Immich API client
- `assets`: Array of assets to include in the stack

**Returns:**

- `*Stack`: Created stack
- `error`: Any error that occurred

### Get Stack

```go
func GetStack(ctx context.Context, client *immich.Client, stackID string) (*Stack, error)
```

Retrieves a stack by its ID.

**Parameters:**

- `ctx`: Context for the operation
- `client`: Immich API client
- `stackID`: ID of the stack to retrieve

**Returns:**

- `*Stack`: Retrieved stack
- `error`: Any error that occurred

### Update Stack

```go
func UpdateStack(ctx context.Context, client *immich.Client, stack *Stack) error
```

Updates an existing stack.

**Parameters:**

- `ctx`: Context for the operation
- `client`: Immich API client
- `stack`: Stack to update

**Returns:**

- `error`: Any error that occurred

### Delete Stack

```go
func DeleteStack(ctx context.Context, client *immich.Client, stackID string) error
```

Deletes a stack by its ID.

**Parameters:**

- `ctx`: Context for the operation
- `client`: Immich API client
- `stackID`: ID of the stack to delete

**Returns:**

- `error`: Any error that occurred

### List Stacks

```go
func ListStacks(ctx context.Context, client *immich.Client) ([]Stack, error)
```

Lists all stacks.

**Parameters:**

- `ctx`: Context for the operation
- `client`: Immich API client

**Returns:**

- `[]Stack`: Array of stacks
- `error`: Any error that occurred

## Error Handling

All operations handle the following error cases:

- Invalid stack ID
- Stack not found
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

3. **Asset Management**

   - Validate assets before operations
   - Handle missing assets gracefully
   - Maintain asset order

4. **Stack Naming**
   - Use descriptive names
   - Include relevant metadata
   - Follow consistent naming patterns

## Example Usage

```go
// Create a new stack
stack, err := CreateStack(ctx, client, assets)
if err != nil {
    log.Printf("Error creating stack: %v", err)
    return
}

// Update stack assets
stack.Assets = append(stack.Assets, newAsset)
err = UpdateStack(ctx, client, stack)
if err != nil {
    log.Printf("Error updating stack: %v", err)
    return
}

// Delete stack
err = DeleteStack(ctx, client, stack.ID)
if err != nil {
    log.Printf("Error deleting stack: %v", err)
    return
}
```
