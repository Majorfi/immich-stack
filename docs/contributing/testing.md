# Testing Guide

## Running Tests

Run all tests with:

```sh
go test ./pkg/...
```

## Test Structure

The project uses table-driven tests for all major logic in:

- `pkg/stacker/stacker_test.go`
- `pkg/immich/client_test.go`

## Test Coverage

To check test coverage:

```sh
go test -cover ./pkg/...
```

For a detailed coverage report:

```sh
go test -coverprofile=coverage.out ./pkg/...
go tool cover -html=coverage.out
```

## Writing Tests

When writing new tests:

1. Use table-driven tests for similar test cases
2. Test both success and failure scenarios
3. Include edge cases
4. Mock external dependencies
5. Use descriptive test names

Example test structure:

```go
func TestStackBy(t *testing.T) {
    tests := []struct {
        name     string
        input    []Asset
        expected []Stack
    }{
        {
            name: "basic stacking",
            input: []Asset{
                // test data
            },
            expected: []Stack{
                // expected results
            },
        },
        // more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := StackBy(tt.input)
            // assertions
        })
    }
}
```
