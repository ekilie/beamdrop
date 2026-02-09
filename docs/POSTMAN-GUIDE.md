# Beamdrop API - Postman Collection

Comprehensive Postman collection for testing the Beamdrop S3-compatible file storage API.

## Quick Start

### 1. Import Files

1. Open Postman
2. Click **Import** (top-left)
3. Import both files:
   - `beamdrop-api.postman_collection.json` - The collection
   - `beamdrop-api.postman_environment.json` - Environment variables

### 2. Configure Environment

1. Click the environment dropdown (top-right)
2. Select **Beamdrop API Environment**
3. Click the eye icon to view/edit variables
4. Update `base_url` if your server isn't at `http://localhost:8080`

### 3. Start Testing

1. **First, create an API key:**
   - Go to `1. API Key Management` → `Create API Key`
   - Click **Send**
   - The access key and secret are automatically saved to environment variables

2. **Create a bucket:**
   - Go to `2. Bucket Operations` → `Create Bucket`
   - Click **Send**

3. **Upload a file:**
   - Go to `3. Object Operations` → `Upload Object (Text)`
   - Click **Send**

4. **Continue exploring!**

## Collection Structure

```
├── 1. API Key Management
│   ├── Create API Key      # Creates key, saves credentials
│   ├── List API Keys       # View all keys
│   └── Delete API Key      # Remove a key
│
├── 2. Bucket Operations
│   ├── Create Bucket       # Create storage bucket
│   ├── List Buckets        # View all buckets
│   ├── Check Bucket Exists # HEAD request
│   └── Delete Bucket       # Remove empty bucket
│
├── 3. Object Operations
│   ├── Upload Object (Text)    # Upload text content
│   ├── Upload Object (JSON)    # Upload JSON file
│   ├── Upload Object (Binary)  # Upload any file
│   ├── Download Object         # Get file content
│   ├── Download Object (Range) # Partial download
│   ├── Get Object Metadata     # HEAD request
│   ├── List Objects            # List bucket contents
│   ├── List Objects (Prefix)   # Filter by prefix
│   ├── List Objects (Delimiter)# Directory-like listing
│   └── Delete Object           # Remove file
│
├── 4. Complete Flow Tests
│   ├── Flow 1: Basic CRUD      # Full create/read/update/delete
│   └── Flow 2: Nested Objects  # Test prefixes/directories
│
└── 5. Error Cases
    ├── Get Non-Existent Bucket
    ├── Get Non-Existent Object
    ├── Create Bucket Invalid Name
    └── Delete Non-Empty Bucket
```

## Authentication

The collection uses **HMAC-SHA256 signature authentication**. A pre-request script automatically handles signature generation for all requests.

### How It Works

1. For each request, the script generates:
   ```
   StringToSign = METHOD + "\n" + PATH + "\n" + TIMESTAMP
   Signature = HMAC-SHA256(StringToSign, SecretKey)
   ```

2. Headers are added automatically:
   ```
   Authorization: Bearer BDK_xxx:base64signature
   X-Beamdrop-Date: 2026-02-09T12:00:00Z
   ```

### Manual Signature Generation

If you need to generate signatures outside Postman:

```javascript
const crypto = require('crypto');

function generateSignature(secretKey, method, path, timestamp) {
    const stringToSign = `${method}\n${path}\n${timestamp}`;
    const signature = crypto
        .createHmac('sha256', secretKey)
        .update(stringToSign)
        .digest('base64');
    return signature;
}

// Example
const timestamp = new Date().toISOString();
const signature = generateSignature(
    'sk_your_secret_key',
    'GET',
    '/api/v1/buckets/my-bucket/file.txt',
    timestamp
);
```

## Running Flow Tests

The "Complete Flow Tests" folder contains end-to-end test scenarios:

### Run All Tests (Collection Runner)

1. Click the **Run** button next to the collection name
2. Select the folder to run (e.g., "Flow 1: Basic CRUD")
3. Click **Run Beamdrop S3-Compatible API**
4. Watch tests execute in sequence

### Flow 1: Basic CRUD

Tests the full lifecycle:
1. Create a bucket
2. Upload a file
3. Verify file in listing
4. Download the file
5. Delete the file
6. Delete the bucket

### Flow 2: Nested Objects

Tests prefix/directory functionality:
1. Create bucket
2. Upload to `images/photo.jpg`
3. Upload to `documents/report.pdf`
4. List with delimiter (shows directories)
5. List with prefix (filter by folder)
6. Cleanup

## Environment Variables

| Variable | Description | Set By |
|----------|-------------|--------|
| `base_url` | Server URL | Manual |
| `access_key_id` | API access key | Create API Key request |
| `secret_key` | API secret key | Create API Key request |
| `bucket_name` | Default test bucket | Manual |
| `object_key` | Default test file | Manual |
| `key_id` | Created key ID | Create API Key request |

## Tips

### Testing with cURL

You can export any request as cURL:
1. Right-click the request
2. Select **Copy as cURL**
3. Paste in terminal

### Uploading Binary Files

1. Go to `Upload Object (Binary)`
2. In the Body tab, click **Select File**
3. Choose any file from your computer
4. Update the URL path to match your filename

### Checking Server Logs

Run beamdrop with verbose logging to debug issues:
```bash
./beamdrop -share /path/to/dir -api-auth
```

## Troubleshooting

### "Invalid signature" error
- Check that your system clock is accurate
- Regenerate API key if secret was lost
- Ensure environment variables are set correctly

### "Bucket not found" error
- Create the bucket first using `Create Bucket` request
- Check the bucket_name environment variable

### "Unauthorized" error
- Create an API key first
- Make sure the environment is selected in Postman
- Check that access_key_id and secret_key are set

## Support

For issues with the API, check the server logs or open an issue on the repository.
