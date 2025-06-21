# Quick Start Guide

This guide will help you quickly set up and start using the S3 clone.

## Prerequisites

- Go 1.16 or higher
- Basic understanding of REST APIs and cURL

## Installation

1. Clone the repository:

   ```bash
   git clone <repository-url>
   cd s3-clone
   ```

2. Build the server:

   ```bash
   go build -o s3-clone cmd/server/main.go
   ```

## Starting the Server

### In-Memory Mode (Default)

```bash
./s3-clone
```

### Filesystem Mode (Persistent Storage)

```bash
mkdir -p /tmp/s3-data
./s3-clone --storage=filesystem --data-dir=/tmp/s3-data
```

## Basic Usage Examples

### 1. Create a Bucket

```bash
curl -X PUT http://localhost:8080/my-bucket
```

### 2. List All Buckets

```bash
curl http://localhost:8080/
```

### 3. Upload a File

```bash
echo "Hello, S3 Clone!" > hello.txt
curl -X PUT -T hello.txt http://localhost:8080/my-bucket/hello.txt
```

### 4. List Objects in a Bucket

```bash
curl http://localhost:8080/my-bucket/
```

### 5. Download a File

```bash
curl http://localhost:8080/my-bucket/hello.txt
```

### 6. Delete a File

```bash
curl -X DELETE http://localhost:8080/my-bucket/hello.txt
```

### 7. Delete a Bucket (must be empty)

```bash
curl -X DELETE http://localhost:8080/my-bucket
```

## Configuration Options

| Flag | Default | Description |
|------|---------|-------------|
| `-addr` | `:8080` | Server address and port |
| `-storage` | `memory` | Storage backend (`memory` or `filesystem`) |
| `-data-dir` | `./data` | Data directory for filesystem storage |
| `-debug` | `false` | Enable debug logging |

## Example: Complete Workflow

```bash
# Start the server with filesystem storage
mkdir -p /tmp/s3-data
./s3-clone --storage=filesystem --data-dir=/tmp/s3-data &

# Create a bucket
curl -X PUT http://localhost:8080/photos

# Upload an image
curl -X PUT -T ~/Pictures/photo.jpg http://localhost:8080/photos/vacation.jpg

# List objects
curl http://localhost:8080/photos/

# Download the image
curl http://localhost:8080/photos/vacation.jpg > downloaded.jpg

# Clean up
curl -X DELETE http://localhost:8080/photos/vacation.jpg
curl -X DELETE http://localhost:8080/photos

# Stop the server
pkill s3-clone
```
