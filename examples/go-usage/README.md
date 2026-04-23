# Beamdrop Go Client Example

This example project shows how to use the in-repo Beamdrop Go client for the main storage flows:

- create or reuse a bucket
- upload objects with both `PutObject` and `PutObjectReader`
- list buckets and objects
- read metadata with `HeadObject`
- download objects with `GetObject`
- check existence with `BucketExists` and `ObjectExists`
- generate a client-side presigned URL
- create, list, fetch, and optionally delete server-side presigned URLs

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
- `BEAMDROP_CLEANUP=true` deletes the demo objects and the server-side presigned URL at the end

## Run

From the repository root:

```bash
go run ./examples/go-usage
```

## What It Does

The example uses one bucket and writes two objects under the `demo/` prefix. It prints the uploaded object details, fetches metadata, downloads one object, creates both kinds of presigned URLs, and lists the server-side presigned URL registry.

If you set `BEAMDROP_CLEANUP=true`, it removes the created presigned URL and the demo objects before exiting.

## Notes

- The example intentionally uses environment variables instead of hard-coded credentials.
- The client-side presigned URL is generated locally using your API secret.
- The server-side presigned URL is created through Beamdrop's `/api/v1/presign` endpoint.
