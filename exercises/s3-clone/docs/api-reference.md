# API Reference

This document provides detailed information about the S3 Clone API endpoints, request/response formats, and examples.

## Base URL

All API endpoints are relative to the base URL:

```text
http://<host>:<port>/
```

Default: `http://localhost:8080/`

## Authentication

> **Note**: The current implementation does not require authentication. In a production environment, you should implement proper authentication.

## Common Headers

| Header | Description | Required |
|--------|-------------|----------|
| `Content-Type` | Should be `application/json` for JSON responses | No |
| `Accept` | Should be `application/json` for JSON responses | No |

## Common Response Codes

| Status Code | Description |
|-------------|-------------|
| 200 OK | Request was successful |
| 201 Created | Resource was created successfully |
| 204 No Content | Request was successful, no content to return |
| 400 Bad Request | Invalid request format or parameters |
| 404 Not Found | Requested resource was not found |
| 409 Conflict | Resource already exists or conflict in state |
| 500 Internal Server Error | Server encountered an error |

## Bucket Operations

### List All Buckets

Returns a list of all buckets.

```http
GET /
```

**Example Request:**

```bash
curl http://localhost:8080/
```

**Example Response (200 OK):**

```json
{
  "buckets": ["bucket1", "bucket2"]
}
```

### Create Bucket

Creates a new bucket.

```http
PUT /{bucket}
```

**Path Parameters:**

- `bucket` (string, required): Name of the bucket to create

**Example Request:**

```bash
curl -X PUT http://localhost:8080/my-bucket
```

**Example Response (200 OK):**

```json
{
  "message": "bucket created"
}
```

### Delete Bucket

Deletes an empty bucket.

```http
DELETE /{bucket}
```

**Path Parameters:**

- `bucket` (string, required): Name of the bucket to delete

**Example Request:**

```bash
curl -X DELETE http://localhost:8080/my-bucket
```

**Response (204 No Content):**

```text
```

### List Objects in Bucket

Lists all objects in the specified bucket.

```http
GET /{bucket}
```

**Path Parameters:**

- `bucket` (string, required): Name of the bucket

**Query Parameters:**

- `prefix` (string, optional): Limits the response to keys that begin with the specified prefix
- `delimiter` (string, optional): A delimiter is a character that groups keys
- `max-keys` (integer, optional): Maximum number of keys to return (default: 1000)

**Example Request:**

```bash
curl "http://localhost:8080/my-bucket/?prefix=photos/"
```

**Example Response (200 OK):**

```json
{
  "bucket": "my-bucket",
  "prefix": "photos/",
  "maxKeys": 1000,
  "isTruncated": false,
  "contents": [
    {
      "key": "file1.txt",
      "size": 1024,
      "lastModified": "2023-01-01T00:00:00Z"
    }
  ]
}
```

## Object Operations

### Upload Object

Uploads an object to the specified bucket.

```http
PUT /{bucket}/{key}
```

**Path Parameters:**

- `bucket` (string, required): Name of the bucket
- `key` (string, required): Object key (path)

**Headers:**

- `Content-Type`: MIME type of the content (optional)
- `X-Amz-Meta-*`: User-defined metadata (optional)

**Request Body:**

The raw bytes of the object to upload

**Example Request:**

```bash
curl -X PUT -T ./photo.jpg \
  -H "Content-Type: image/jpeg" \
  -H "X-Amz-Meta-Uploaded-By: user123" \
  http://localhost:8080/my-bucket/photos/vacation.jpg
```

**Example Response (200 OK):**

```json
{
  "bucket": "my-bucket",
  "key": "photos/vacation.jpg"
}
```

### Download Object

Downloads an object from the specified bucket.

```http
GET /{bucket}/{key}
```

**Path Parameters:**

- `bucket` (string, required): Name of the bucket
- `key` (string, required): Object key (path)

**Example Request:**

```bash
curl -o downloaded.jpg http://localhost:8080/my-bucket/photos/vacation.jpg
```

**Response Headers:**

- `Content-Type`: MIME type of the object
- `Content-Length`: Size of the object in bytes
- `Last-Modified`: Timestamp of when the object was last modified
- `X-Amz-Meta-*`: User-defined metadata

**Response (200 OK):**

```text
[Binary content of the object]
```

### Delete Object

Deletes an object from the specified bucket.

```http
DELETE /{bucket}/{key}
```

**Path Parameters:**

- `bucket` (string, required): Name of the bucket
- `key` (string, required): Object key to delete

**Example Request:**

```bash
curl -X DELETE http://localhost:8080/my-bucket/photos/vacation.jpg
```

**Response (204 No Content):**

```text
```

## Error Responses

All error responses follow this format:

```json
{
  "error": "Error message describing the issue"
}
```

**Example Error Response (404 Not Found):**

```json
{
  "error": "object not found"
}
```

## Rate Limiting

> **Note**: The current implementation does not enforce rate limiting. In a production environment, you should implement rate limiting to prevent abuse.

## Best Practices

1. Always check the response status code
2. Handle errors gracefully
3. Use appropriate content types for uploads
4. Clean up unused resources
5. Implement proper error handling in your client code
