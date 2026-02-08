# Code Standards

## Formatting

Use gofmt. Run before committing:

```bash
go fmt ./...
```

## Linting

Use golangci-lint:

```bash
golangci-lint run
```

## Code Organization

### Project Layout

```
/cmd          - Main applications
/internal     - Private code
/pkg          - Public libraries
```

### Imports

Group imports: standard library, external, internal.

```go
import (
    "context"
    "fmt"
    
    "github.com/gin-gonic/gin"
    
    "myproject/internal/db"
)
```

## Error Handling

Always handle errors. Never ignore them.

```go
// ❌ Bad
result, _ := doSomething()

// ✅ Good
result, err := doSomething()
if err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}
```

### Error Wrapping

Wrap errors with context:

```go
if err != nil {
    return fmt.Errorf("creating user %s: %w", name, err)
}
```

## Functions

- Keep functions focused
- Return early on errors
- Document exported functions

```go
// CreateUser creates a new user with the given name.
// Returns an error if the name is empty or already taken.
func CreateUser(ctx context.Context, name string) (*User, error) {
    if name == "" {
        return nil, errors.New("name is required")
    }
    // ...
}
```

## Testing

Use table-driven tests:

```go
func TestAdd(t *testing.T) {
    tests := []struct {
        name     string
        a, b     int
        expected int
    }{
        {"positive", 1, 2, 3},
        {"negative", -1, -2, -3},
        {"zero", 0, 0, 0},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := Add(tt.a, tt.b)
            if result != tt.expected {
                t.Errorf("got %d, want %d", result, tt.expected)
            }
        })
    }
}
```

## Comments

- Document all exported functions, types, and constants
- Start comments with the name of the thing being documented

```go
// User represents a registered user in the system.
type User struct {
    ID   int
    Name string
}
```
