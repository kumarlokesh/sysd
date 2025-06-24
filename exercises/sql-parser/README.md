# SQL Parser Exercise

A simplified SQL parser implementation in Go, focusing on understanding PostgreSQL's query processing internals.

## Overview

This exercise implements a basic SQL parser that can parse a subset of PostgreSQL's SQL syntax. The parser converts SQL text into an Abstract Syntax Tree (AST) that can be further processed or analyzed.

## Features

- Lexical analysis (tokenization)
- Syntax analysis (parsing)
- AST generation
- Support for basic SELECT queries including:
  - Column selection
  - Table references
  - WHERE clauses
  - Basic expressions and comparisons

## Project Structure

```
sql-parser/
├── cmd/              # Main application packages
│   └── sql-parser/   # Command-line interface
├── internal/         # Private application code
│   ├── ast/         # Abstract Syntax Tree nodes
│   ├── lexer/       # Lexical analysis (tokenization)
│   └── parser/     # Syntax analysis and parsing
├── test/            # Test files
├── go.mod          # Go module definition
└── README.md       # This file
```

## Getting Started

1. Build the parser:

   ```bash
   go build -o bin/sql-parser ./cmd/sql-parser
   ```

2. Run the parser:

   ```bash
   ./bin/sql-parser "SELECT id, name FROM users WHERE age > 18"
   ```

## Example

```go
// Parse a simple SQL query
query := "SELECT id, name FROM users WHERE age > 18"
stmt, err := parser.Parse(query)
if err != nil {
    log.Fatal(err)
}

// Print the parsed statement
fmt.Printf("%+v\n", stmt)
```

## Testing

Run the test suite:

```bash
go test ./...
```

## Implementation Status

- [x] Complete lexer implementation with tokenization
- [x] Parser with recursive descent and Pratt parsing for expressions
- [x] Full SELECT query support including:
  - [x] Column selection (including wildcard *)
  - [x] Table references
  - [x] WHERE clauses with expressions
  - [x] Operator precedence handling
- [x] Comprehensive test coverage

## Example Queries

```sql
-- Simple select
SELECT * FROM users;

-- Select with condition
SELECT id, name FROM users WHERE age > 18;

-- Complex conditions
SELECT * FROM products WHERE price < 100 AND in_stock = true;
```

## Limitations

- Only supports SELECT statements
- No support for JOINs, subqueries, or aggregations
- Limited set of SQL operators and functions
- Basic error handling

## Future Enhancements

- [ ] Support for JOINs, GROUP BY, and other SQL clauses
- [ ] Improved error messages and recovery
- [ ] Support for prepared statements
- [ ] Query optimization
- [ ] Semantic analysis and validation

## Implementation Details

The parser is implemented using a two-phase approach:

1. **Lexical Analysis**:
   - Converts SQL text into tokens
   - Handles whitespace, keywords, identifiers, literals, and operators

2. **Syntax Analysis**:
   - Uses recursive descent for statement parsing
   - Implements Pratt parsing for expressions with proper operator precedence
   - Builds an Abstract Syntax Tree (AST) representing the query

### Error Handling

- Reports syntax errors with position information
- Attempts to recover from some errors to report multiple issues
- Provides meaningful error messages for common mistakes

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Linting

```bash
golangci-lint run
```
