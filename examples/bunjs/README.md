# Beamdrop Bun.js Client Example

This example shows how to use the [`beamdrop`](https://www.npmjs.com/package/beamdrop) npm package with [Bun](https://bun.sh) for the main storage flows:

- Create or reuse a bucket
- Upload objects with `putObject`
- List buckets and objects (with prefix/delimiter filtering)
- Read metadata with `headObject`
- Download objects with `getObject`
- Check existence with `bucketExists` and `objectExists`
- Generate a client-side presigned URL (no server round-trip)
- Create, list, and optionally revoke server-side pretty presigned URLs
- Error handling with `BeamdropException`

## Prerequisites

1. Start Beamdrop with API auth enabled:

```bash
beamdrop -dir ./storage/app -api-auth
```

2. Create an API key through the Beamdrop server.
3. Copy `.env.example` to `.env` and fill in your credentials.

## Setup

```bash
bun install
cp .env.example .env
# edit .env with your credentials
```

## Run

```bash
bun run start
```

## Configuration

| Variable | Required | Default | Description |
|---|---|---|---|
| `BEAMDROP_BASE_URL` | Yes | — | Server URL (e.g. `http://localhost:7777`) |
| `BEAMDROP_ACCESS_KEY_ID` | Yes | — | API access key (starts with `BDK_`) |
| `BEAMDROP_SECRET_KEY` | Yes | — | API secret key (starts with `sk_`) |
| `BEAMDROP_BUCKET` | No | `beamdrop-bun-example` | Bucket name to use |
| `BEAMDROP_CLEANUP` | No | `false` | Set to `true` to delete demo objects on exit |

## What It Does

The example uses one bucket and writes two objects under the `demo/` prefix. It prints uploaded object details, fetches metadata, downloads one object, creates both client-side and server-side presigned URLs, and lists the server-side presigned URL registry.

If you set `BEAMDROP_CLEANUP=true`, it removes the created presigned URL and the demo objects before exiting.
