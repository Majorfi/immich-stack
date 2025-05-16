# Development Guide

This guide helps you set up and contribute to Immich Stack development.

## Prerequisites

- Go 1.21 or later
- Docker and Docker Compose
- Make (optional, for using Makefile)

## Setup

1. Clone the repository

   ```sh
   git clone https://github.com/majorfi/immich-stack.git
   cd immich-stack
   ```

2. Install dependencies

   ```sh
   go mod download
   ```

3. Create development environment
   ```sh
   cp .env.example .env
   ```

## Development Workflow

### Running Tests

```sh
# Run all tests
go test ./...

# Run specific test
go test ./internal/stack

# Run with coverage
go test -cover ./...
```

### Building

```sh
# Build binary
go build -o immich-stack

# Build for specific platform
GOOS=linux GOARCH=amd64 go build -o immich-stack
```

### Docker Development

```sh
# Build development image
docker build -t immich-stack:dev .

# Run development container
docker run -it --rm \
  --name immich-stack-dev \
  --env-file .env \
  -v $(pwd):/app \
  immich-stack:dev
```

## Code Structure

```
.
├── cmd/                    # Command-line interface
│   └── main.go            # Entry point
├── internal/              # Internal packages
│   ├── asset/            # Asset operations
│   ├── grouping/         # Grouping logic
│   └── stack/            # Stack operations
├── pkg/                   # Public packages
│   ├── immich/           # Immich API client
│   ├── stacker/          # Stacking logic
│   └── utils/            # Utility functions
└── docs/                 # Documentation
```

## Coding Standards

### Go Code

1. **Formatting**

   - Use `go fmt`
   - Follow Go standard formatting
   - Use `gofmt` for consistency

2. **Documentation**

   - Document all exported functions
   - Include examples where helpful
   - Follow Go doc conventions

3. **Testing**
   - Write unit tests
   - Use table-driven tests
   - Test edge cases

### Error Handling

1. **Error Types**

   ```go
   type StackError struct {
       Code    string
       Message string
       Err     error
   }
   ```

2. **Error Wrapping**

   ```go
   return nil, fmt.Errorf("failed to create stack: %w", err)
   ```

3. **Error Checking**
   ```go
   if err != nil {
       return nil, err
   }
   ```

## Contributing

1. **Fork and Clone**

   - Fork the repository
   - Clone your fork
   - Create a feature branch

2. **Development**

   - Write code
   - Add tests
   - Update documentation

3. **Testing**

   - Run all tests
   - Check formatting
   - Verify documentation

4. **Pull Request**
   - Push changes
   - Create pull request
   - Wait for review

## Best Practices

1. **Code Quality**

   - Write clean, readable code
   - Follow Go best practices
   - Use meaningful names

2. **Testing**

   - Write comprehensive tests
   - Test edge cases
   - Maintain test coverage

3. **Documentation**

   - Keep docs up to date
   - Add examples
   - Document changes

4. **Performance**
   - Profile code
   - Optimize bottlenecks
   - Consider memory usage

## Release Process

1. **Versioning**

   - Follow semantic versioning
   - Update version in code
   - Tag releases

2. **Building**

   - Build for all platforms
   - Create Docker images
   - Sign releases

3. **Deployment**
   - Push to registries
   - Update documentation
   - Announce release
