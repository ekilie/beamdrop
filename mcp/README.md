# Beamdrop MCP Server

A [Model Context Protocol](https://modelcontextprotocol.io/) server that exposes Beamdrop's S3-compatible file storage API as tools for AI agents.

## Tools

| Tool                   | Description                                  |
| ---------------------- | -------------------------------------------- |
| `list_buckets`         | List all storage buckets                     |
| `create_bucket`        | Create a bucket (idempotent by default)      |
| `delete_bucket`        | Delete an empty bucket                       |
| `bucket_exists`        | Check if a bucket exists                     |
| `list_objects`         | List objects with prefix/delimiter filtering |
| `put_object`           | Upload content to a bucket                   |
| `get_object`           | Download an object                           |
| `head_object`          | Get object metadata                          |
| `delete_object`        | Delete an object                             |
| `create_presigned_url` | Create a shareable download URL              |
| `list_presigned_urls`  | List all presigned URLs                      |
| `get_presigned_url`    | Get presigned URL details                    |
| `delete_presigned_url` | Revoke a presigned URL                       |
| `list_api_keys`        | List API keys                                |
| `create_api_key`       | Create a new API key                         |
| `delete_api_key`       | Delete an API key                            |

## Prerequisites

1. A running Beamdrop instance with API auth enabled:

   ```bash
   beamdrop -dir /path/to/share -api-auth
   ```

2. An API key — create one:
   ```bash
   curl -X POST http://localhost:7777/api/v1/keys
   ```
   Save the `access_key_id` and `secret_key` from the response.

## Installation

```bash
cd mcp
npm install
npm run build
```

## Configuration

Set these environment variables:

| Variable                 | Required | Description                                         |
| ------------------------ | -------- | --------------------------------------------------- |
| `BEAMDROP_BASE_URL`      | Yes      | Beamdrop server URL (e.g., `http://localhost:7777`) |
| `BEAMDROP_ACCESS_KEY_ID` | Yes      | API access key (format: `BDK_*`)                    |
| `BEAMDROP_SECRET_KEY`    | Yes      | API secret key (format: `sk_*`)                     |

## Claude Desktop

Add to `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "beamdrop": {
      "command": "node",
      "args": ["/absolute/path/to/beamdrop/mcp/dist/index.js"],
      "env": {
        "BEAMDROP_BASE_URL": "http://localhost:7777",
        "BEAMDROP_ACCESS_KEY_ID": "BDK_your_key",
        "BEAMDROP_SECRET_KEY": "sk_your_secret"
      }
    }
  }
}
```

## VS Code

Add to your `.vscode/mcp.json`:

```json
{
  "servers": {
    "beamdrop": {
      "command": "node",
      "args": ["${workspaceFolder}/mcp/dist/index.js"],
      "env": {
        "BEAMDROP_BASE_URL": "http://localhost:7777",
        "BEAMDROP_ACCESS_KEY_ID": "BDK_your_key",
        "BEAMDROP_SECRET_KEY": "sk_your_secret"
      }
    }
  }
}
```

## Usage Examples

Once connected, you can ask your AI agent things like:

- "Create a bucket called 'project-assets'"
- "Upload this code to beamdrop in the 'backups' bucket"
- "List all files in the 'documents' bucket"
- "Generate a download link for reports/quarterly.pdf that expires in 24 hours"
- "Delete all files in the 'temp' bucket with the prefix 'old/'"

## Development

```bash
npm run dev    # Watch mode (recompiles on changes)
npm run build  # Production build
npm start      # Run the server
```
