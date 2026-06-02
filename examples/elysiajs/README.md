# Beamdrop Elysia.js Example

A REST API server using [Elysia](https://elysiajs.com) (Bun-native web framework) and the [`beamdrop`](https://www.npmjs.com/package/beamdrop) npm package as the storage backend.

## Features

- File upload, download, list, and delete via REST endpoints
- Bucket management (create, list, delete)
- Client-side presigned URL generation
- Server-side pretty presigned URL management
- Elysia type-safe schema validation with TypeBox
- BeamdropException error handling

## Prerequisites

1. Install [Bun](https://bun.sh).
2. Start Beamdrop with API auth enabled:

```bash
beamdrop -dir ./storage/app -api-auth
```

3. Create an API key through the Beamdrop server.
4. Copy `.env.example` to `.env` and fill in your credentials.

## Setup

```bash
bun install
cp .env.example .env
# edit .env with your credentials
```

## Run

```bash
bun run start
# or with auto-reload
bun run dev
```

## API Endpoints

### Buckets

| Method   | Path             | Description                  |
| -------- | ---------------- | ---------------------------- |
| `GET`    | `/buckets`       | List all buckets             |
| `POST`   | `/buckets/:name` | Create a bucket (idempotent) |
| `DELETE` | `/buckets/:name` | Delete an empty bucket       |

### Files

| Method   | Path                      | Description                              |
| -------- | ------------------------- | ---------------------------------------- |
| `GET`    | `/files?prefix=photos/`   | List files (with optional prefix)        |
| `POST`   | `/files/path/to/file.txt` | Upload a file (multipart, field: `file`) |
| `GET`    | `/files/path/to/file.txt` | Download a file                          |
| `DELETE` | `/files/path/to/file.txt` | Delete a file                            |

### Presigned URLs

| Method   | Path                     | Description                               |
| -------- | ------------------------ | ----------------------------------------- |
| `POST`   | `/presign`               | Generate a client-side presigned URL      |
| `POST`   | `/presign/pretty`        | Create a server-side pretty presigned URL |
| `GET`    | `/presign/pretty`        | List all pretty presigned URLs            |
| `DELETE` | `/presign/pretty/:token` | Revoke a pretty presigned URL             |

## Usage Examples

```bash
# Upload a file
curl -F file=@photo.jpg http://localhost:3000/files/photos/photo.jpg

# List files
curl http://localhost:3000/files?prefix=photos/

# Download a file
curl http://localhost:3000/files/photos/photo.jpg -o photo.jpg

# Generate a presigned URL (1 hour)
curl -X POST http://localhost:3000/presign \
  -H "Content-Type: application/json" \
  -d '{"key": "photos/photo.jpg", "expiresIn": 3600}'

# Create a pretty presigned URL (7 days, max 100 downloads)
curl -X POST http://localhost:3000/presign/pretty \
  -H "Content-Type: application/json" \
  -d '{"key": "photos/photo.jpg", "expiresIn": 604800, "maxDownloads": 100}'

# Delete a file
curl -X DELETE http://localhost:3000/files/photos/photo.jpg
```
