# Beamdrop Go Client Examples

This directory contains several standalone Go programs that demonstrate how to use the in-repo Beamdrop Go client.

## Prerequisites

1. Start Beamdrop with API auth enabled.
2. Create an API key.
3. Export the environment variables shown in `.env.example`.

Example local startup:

```bash
beamdrop -dir ./storage/app -api-auth
```

## Configuration

Copy the values from `.env.example` into your shell or your preferred env loader.

Required variables:

- `BEAMDROP_BASE_URL`
- `BEAMDROP_ACCESS_KEY_ID`
- `BEAMDROP_SECRET_KEY`

Optional variables:

- `BEAMDROP_BUCKET` defaults to `beamdrop-go-example`
- `BEAMDROP_CLEANUP=true` deletes the demo objects at the end of the main and presigned-download examples

## Examples

### basic-s3 — S3-style CRUD (`go run ./examples/go-usage`)

Covers the core storage flows:

- create or reuse a bucket
- upload objects with both `PutObject` and `PutObjectReader`
- list buckets and objects
- read metadata with `HeadObject`
- download objects with `GetObject`
- check existence with `BucketExists` and `ObjectExists`
- generate a client-side presigned URL
- create, list, fetch, and optionally delete server-side presigned URLs

### presigned-download — Presigned URL download (`go run ./examples/go-usage/presigned-download`)

Shows how to share private files without exposing API keys:

1. Upload a private object to the authenticated client.
2. Create a server-side presigned URL with a download limit (`MaxDownloads`).
3. Download the object using only the presigned URL (plain HTTP client, no auth).
4. Attempt a direct (non-presigned) download to confirm it fails with 401.

### bucket-lifecycle — Bucket lifecycle (`go run ./examples/go-usage/bucket-lifecycle`)

Demonstrates full bucket management with proper error handling:

- strict creation (`CreateBucket`) — fails with 409 if the bucket exists
- idempotent creation (`CreateBucketIfNotExists`)
- listing and existence checks (`ListBuckets`, `BucketExists`)
- deletion rejection when the bucket is non-empty (409)
- object cleanup followed by bucket deletion
- deletion verification

## Notes

- The examples intentionally use environment variables instead of hard-coded credentials.
- The client-side presigned URL is generated locally using your API secret.
- The server-side presigned URL is created through Beamdrop's `/api/v1/presign` endpoint.
