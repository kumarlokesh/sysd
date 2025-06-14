# System Design Exercises

A collection of system design exercises implemented in Go. Each exercise is a self-contained project demonstrating the design and implementation of various distributed systems concepts.

## Structure

- `exercises/` - Contains individual system design exercises
  - `exercise-name/` - Each exercise is in its own directory
    - `cmd/` - Main application packages
    - `internal/` - Private application code
    - `pkg/` - Public packages that can be imported by other exercises
    - `test/` - Integration and end-to-end tests
    - `go.mod` - Go module definition
    - `README.md` - Exercise documentation
- `pkg/` - Shared packages across exercises
- `docs/` - Additional documentation
- `.github/` - GitHub workflows and templates
- `Makefile` - Common build and test commands

## Getting Started

1. Clone the repository
2. Navigate to an exercise directory
3. Run `go test ./...` to run tests
4. Check the exercise's README for specific instructions

## Adding a New Exercise

1. Create a new directory under `exercises/`
2. Initialize a new Go module: `go mod init github.com/yourusername/sysd/exercises/exercise-name`
3. Follow the structure outlined above
4. Add comprehensive tests
5. Document your design in the README.md

## Contributing

Contributions are welcome! Please ensure your code is well-tested and documented.
