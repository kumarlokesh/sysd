# S3 Clone - Minimal Implementation

A minimal implementation of an S3-like object storage service written in Go. This project demonstrates the core functionality of Amazon S3, including bucket and object operations.

## Features

- **Bucket Operations**
  - Create buckets
  - List all buckets
  - Delete empty buckets

- **Object Operations**
  - Upload objects
  - Download objects
  - List objects in a bucket
  - Delete objects

## Architecture

The application follows a clean architecture with the following components:

1. **API Layer**: Handles HTTP requests and responses
2. **Service Layer**: Implements business logic
3. **Storage Layer**: Handles data persistence
   - In-memory storage (default)
   - Filesystem storage (optional)

## API Endpoints

### Bucket Operations

- `GET /` - List all buckets
- `PUT /{bucket}` - Create a new bucket
- `DELETE /{bucket}` - Delete an empty bucket
- `GET /{bucket}/` - List objects in a bucket

### Object Operations

- `PUT /{bucket}/{key}` - Upload an object
- `GET /{bucket}/{key}` - Download an object
- `DELETE /{bucket}/{key}` - Delete an object

## Getting Started

### Prerequisites

- Go 1.16 or higher

### Running the Server

1. Clone the repository
2. Build and run the server:

   ```bash
   go run cmd/server/main.go
   ```

### Using the API

### Create a bucket

```bash
curl -X PUT http://localhost:8080/test-bucket
```

#### List all buckets

```bash
curl http://localhost:8080/
```

#### Upload an object

```bash
echo 'Hello, World!' > test.txt
curl -X PUT -T test.txt http://localhost:8080/test-bucket/test.txt
```

#### List objects in a bucket

```bash
curl http://localhost:8080/test-bucket
```

#### Download an object

```bash
curl http://localhost:8080/test-bucket/test.txt
```

#### Delete an object

```bash
curl -X DELETE http://localhost:8080/test-bucket/test.txt
```

#### Delete a bucket

```bash
curl -X DELETE http://localhost:8080/test-bucket
```

## Storage Backends

### In-Memory Storage (Default)

- Data is stored in memory
- Data is lost when the server restarts

### Filesystem Storage

- Data is persisted to disk
- Configure with `--storage=filesystem --data-dir=/path/to/data`

## Configuration

```text
Usage of ./s3-clone:
  -addr string
        Server address (default ":8080")
  -data-dir string
        Data directory for filesystem storage (default "./data")
  -debug
        Enable debug logging
  -storage string
        Storage backend to use (memory or filesystem) (default "memory")
```

## Testing

Run the test suite:

```bash
go test -v ./...
```

## Future Improvements

- Implement authentication and authorization
- Add support for object metadata
- Implement multipart uploads
- Add support for object versioning
- Implement bucket policies and ACLs
- Add support for CORS
- Implement server-side encryption
- Add support for object lifecycle policies

## License

MIT
