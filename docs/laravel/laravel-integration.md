# Beamdrop Laravel Integration Guide

A complete guide to using Beamdrop as your file storage backend in a Laravel application. Covers setup, authentication, the PHP service class, and real-world usage patterns for profile avatars, images, PDFs, invoices, and general file storage.

---

## Table of Contents

1. [Prerequisites](#1-prerequisites)
2. [Beamdrop Server Setup](#2-beamdrop-server-setup)
3. [Create an API Key](#3-create-an-api-key)
4. [Laravel Configuration](#4-laravel-configuration)
5. [Install the Service Class](#5-install-the-service-class)
6. [Register the Service Provider](#6-register-the-service-provider)
7. [Usage Examples](#7-usage-examples)
   - [Profile Avatars](#71-profile-avatars)
   - [Invoice PDFs](#72-invoice-pdfs)
   - [General File Uploads](#73-general-file-uploads)
   - [Listing Files in a Bucket](#74-listing-files-in-a-bucket)
   - [Presigned URLs (Temporary Links)](#75-presigned-urls-temporary-links)
   - [Deleting Files](#76-deleting-files)
8. [Error Handling](#8-error-handling)
9. [Security Best Practices](#9-security-best-practices)
10. [API Reference (Quick Cheat Sheet)](#10-api-reference)
11. [Troubleshooting](#11-troubleshooting)

---

<a name="1-prerequisites"></a>

## 1. Prerequisites

| Requirement     | Version                                |
| --------------- | -------------------------------------- |
| PHP             | 8.1+                                   |
| Laravel         | 10.x / 11.x / 12.x                     |
| Beamdrop server | Running and accessible over HTTP/HTTPS |
| `ext-curl`      | Enabled (ships with most PHP installs) |

No external Composer packages are required the service class uses PHP's native `cURL` functions.

---

<a name="2-beamdrop-server-setup"></a>

## 2. Beamdrop Server Setup

Start Beamdrop with API authentication enabled:

```bash
# Minimal   auth enabled, default port 8090
./beamdrop -dir /srv/beamdrop-data -api-auth

# Production   with rate limiting and password protection
./beamdrop \
  -dir /srv/beamdrop-data \
  -port 8090 \
  -api-auth \
  -rate-limit 100 \
  -password "your-ui-password" \
  -log-level info
```

> **Important:** The `-api-auth` flag activates HMAC-SHA256 signature verification on all `/api/v1/*` endpoints. Without it, anyone with network access can read/write files.

---

<a name="3-create-an-api-key"></a>

## 3. Create an API Key

Use curl (or the web UI) to generate a key pair:

```bash
curl -X POST http://your-server:8090/api/v1/keys \
  -H "Content-Type: application/json" \
  -d '{"name": "laravel-production"}'
```

Response:

```json
{
  "id": 1,
  "name": "laravel-production",
  "accessKeyId": "BDK_a1b2c3d4e5f6g7h8",
  "secretKey": "sk_1234567890abcdef1234567890abcdef12345678",
  "warning": "Save the secret key now. It cannot be retrieved later."
}
```

**Save both `accessKeyId` and `secretKey`** the secret is shown only once.

---

<a name="4-laravel-configuration"></a>

## 4. Laravel Configuration

Add these values to your `.env` file:

```dotenv
BEAMDROP_URL=https://files.yoursite.com
BEAMDROP_ACCESS_KEY=BDK_a1b2c3d4e5f6g7h8
BEAMDROP_SECRET_KEY=sk_1234567890abcdef1234567890abcdef12345678
```

Add the matching config entries in `config/services.php`:

```php
// config/services.php

return [
    // ... other services

    'beamdrop' => [
        'url'        => env('BEAMDROP_URL', 'http://localhost:8090'),
        'access_key' => env('BEAMDROP_ACCESS_KEY'),
        'secret_key' => env('BEAMDROP_SECRET_KEY'),
    ],
];
```

---

<a name="5-install-the-service-class"></a>

## 5. Install the Service Class

Copy the `Beamdrop.php` file (provided below in this repo, or shown at the bottom of this guide) into your Laravel project:

```
app/
└── Services/
    └── Beamdrop.php
```

Then register it as a singleton in a service provider.

---

<a name="6-register-the-service-provider"></a>

## 6. Register the Service Provider

In `app/Providers/AppServiceProvider.php` (or a dedicated provider):

```php
use App\Services\Beamdrop;

public function register(): void
{
    $this->app->singleton(Beamdrop::class, function ($app) {
        return new Beamdrop(
            baseUrl:   config('services.beamdrop.url'),
            accessKey: config('services.beamdrop.access_key'),
            secretKey: config('services.beamdrop.secret_key'),
        );
    });
}
```

Now you can inject or resolve `Beamdrop` anywhere:

```php
// Via injection
public function store(Request $request, Beamdrop $beamdrop) { ... }

// Via the container
$beamdrop = app(Beamdrop::class);
```

---

<a name="7-usage-examples"></a>

## 7. Usage Examples

### 7.1 Profile Avatars

**Controller:**

```php
use App\Services\Beamdrop;
use Illuminate\Http\Request;

class ProfileController extends Controller
{
    public function updateAvatar(Request $request, Beamdrop $beamdrop)
    {
        $request->validate([
            'avatar' => 'required|image|mimes:jpg,jpeg,png,webp|max:2048', // 2 MB
        ]);

        $user = $request->user();
        $file = $request->file('avatar');

        // Delete old avatar if it exists
        if ($user->avatar_key) {
            $beamdrop->deleteObject('avatars', $user->avatar_key);
        }

        // Upload new avatar   key: "user-42/avatar.jpg"
        $key = "user-{$user->id}/avatar." . $file->getClientOriginalExtension();

        $result = $beamdrop->putObject(
            bucket: 'avatars',
            key:    $key,
            body:   file_get_contents($file->getRealPath()),
        );

        // Save the key to the database
        $user->update(['avatar_key' => $result['key']]);

        return back()->with('success', 'Avatar updated.');
    }
}
```

**Displaying the avatar in a Blade template:**

```blade
@php
    $beamdrop = app(\App\Services\Beamdrop::class);
@endphp

{{-- Generate a 1-hour presigned URL --}}
<img
    src="{{ $beamdrop->presignedUrl('avatars', $user->avatar_key, 3600) }}"
    alt="{{ $user->name }}"
    class="rounded-full w-16 h-16"
>
```

Or serve it through your own controller for extra control:

```php
// Route: GET /avatar/{user}
public function showAvatar(User $user, Beamdrop $beamdrop)
{
    $response = $beamdrop->getObject('avatars', $user->avatar_key);

    return response($response['body'])
        ->header('Content-Type', $response['content_type'])
        ->header('Cache-Control', 'public, max-age=86400');
}
```

---

### 7.2 Invoice PDFs

```php
class InvoiceController extends Controller
{
    public function store(Invoice $invoice, Beamdrop $beamdrop)
    {
        // Generate the PDF (using barryvdh/laravel-dompdf, Spatie, etc.)
        $pdf = \PDF::loadView('invoices.template', compact('invoice'));

        $key = "company-{$invoice->company_id}/{$invoice->created_at->format('Y/m')}/INV-{$invoice->number}.pdf";

        $result = $beamdrop->putObject(
            bucket: 'invoices',
            key:    $key,
            body:   $pdf->output(),
        );

        $invoice->update([
            'storage_key' => $result['key'],
            'storage_etag' => $result['etag'],
        ]);

        return $result;
    }

    public function download(Invoice $invoice, Beamdrop $beamdrop)
    {
        $response = $beamdrop->getObject('invoices', $invoice->storage_key);

        return response($response['body'])
            ->header('Content-Type', 'application/pdf')
            ->header('Content-Disposition', "attachment; filename=\"INV-{$invoice->number}.pdf\"");
    }
}
```

---

### 7.3 General File Uploads

```php
public function upload(Request $request, Beamdrop $beamdrop)
{
    $request->validate([
        'file' => 'required|file|max:51200', // 50 MB
    ]);

    $file = $request->file('file');
    $key  = 'uploads/' . now()->format('Y/m/d') . '/' . Str::uuid() . '.' . $file->getClientOriginalExtension();

    $result = $beamdrop->putObject(
        bucket: 'documents',
        key:    $key,
        body:   file_get_contents($file->getRealPath()),
    );

    return response()->json($result, 201);
}
```

---

### 7.4 Listing Files in a Bucket

```php
// List all invoices for a company, paginated into "folders"
$result = $beamdrop->listObjects(
    bucket:    'invoices',
    prefix:    'company-5/2026/',
    delimiter: '/',
    maxKeys:   100,
);

// $result['contents']         array of file objects (key, size, etag, lastModified)
// $result['commonPrefixes']   virtual "subfolders" like ["company-5/2026/01/", "company-5/2026/02/"]
```

---

### 7.5 Presigned URLs (Temporary Links)

Beamdrop supports **two methods** for creating presigned URLs. You can use either or both depending on your needs:

|                            | Method 1: Client-Side (HMAC)                       | Method 2: Server-Side (Pretty)             |
| -------------------------- | -------------------------------------------------- | ------------------------------------------ |
| **Function**               | `presignedUrl()`                                   | `createPrettyPresignedUrl()`               |
| **URL shape**              | `/api/v1/buckets/…?token=…&expires=…&access_key=…` | `/dl/a1b2c3d4e5f6…`                        |
| **Server round-trip**      | None computed locally                              | POST to `/api/v1/presign`                  |
| **Max download limit**     | No                                                 | Yes                                        |
| **Individually revocable** | No                                                 | Yes (`revokePrettyPresignedUrl()`)         |
| **Download tracking**      | No                                                 | Yes (server counts downloads)              |
| **Requires**               | Nothing extra                                      | Server-side presigned URL registry feature |

#### Method 1: Client-Side HMAC Presigned URL

Generate a time-limited link computed entirely on the client side no server call needed:

```php
$url = $beamdrop->presignedUrl(
    bucket:    'invoices',
    key:       'company-5/2026/01/INV-1042.pdf',
    expiresIn: 3600, // 1 hour in seconds
);

// → https://files.yoursite.com/api/v1/buckets/invoices/company-5/2026/01/INV-1042.pdf
//     ?token=ABC123...&expires=2026-02-23T15:00:00Z&access_key=BDK_a1b2c3d4e5f6g7h8
```

#### Method 2: Server-Side Pretty Presigned URL

Create a short, clean `/dl/{token}` URL via the server-side registry. Supports download limits and individual revocation:

```php
// Create a pretty presigned URL (1-hour expiry, max 10 downloads)
$result = $beamdrop->createPrettyPresignedUrl(
    bucket:       'invoices',
    key:          'company-5/2026/01/INV-1042.pdf',
    expiresIn:    3600,    // 1 hour (null = no expiry)
    maxDownloads: 10,      // optional limit (null = unlimited)
);

// $result['url']   → "https://files.yoursite.com/dl/a1b2c3d4e5f6789a..."
// $result['token'] → "a1b2c3d4e5f6789a..."

// Revoke a pretty presigned URL immediately
$beamdrop->revokePrettyPresignedUrl($result['token']);

// List all active pretty presigned URLs
$all = $beamdrop->listPrettyPresignedUrls();
// $all['urls']    array of presigned URL records
// $all['count']   total count
```

#### When to Use Which

- **Use `presignedUrl()`** for quick, ephemeral links where you don't need revocation or download tracking. No server round-trip.
- **Use `createPrettyPresignedUrl()`** for user-facing links (emails, client portals, invoices) where you want clean URLs, download limits, or the ability to revoke individual links.

Both methods can be used together in the same application.

#### How It Works Under the Hood (Client-Side Method)

The `presignedUrl()` method builds a self-contained URL with three query parameters:

| Parameter    | Value                                            | Purpose                                                          |
| ------------ | ------------------------------------------------ | ---------------------------------------------------------------- |
| `token`      | `Base64URL(HMAC-SHA256(secret, message))`        | Proves the URL was generated by someone who knows the secret key |
| `expires`    | ISO 8601 timestamp (e.g. `2026-02-24T15:00:00Z`) | When the link stops working                                      |
| `access_key` | Your `BDK_xxx` access key ID                     | Tells the server which secret to verify the token against        |

The signed message is: `METHOD\nBUCKET\nKEY\nUNIX_TIMESTAMP` changing any part invalidates the token.

#### How It Works Under the Hood (Server-Side Method)

The `createPrettyPresignedUrl()` method calls `POST /api/v1/presign` which:

1. Generates a random 16-byte hex token
2. Stores it in the database with the bucket, key, expiry, and download limit
3. Returns a short `/dl/{token}` URL

When someone visits `/dl/{token}`, the server looks up the token, checks expiry and download limits, increments the counter, and streams the file.

#### Suggested Expiry Times

| Use case                                     | `expiresIn` value                                  |
| -------------------------------------------- | -------------------------------------------------- |
| One-time download (email/chat)               | `3600` (1 hour)                                    |
| Embedded in a web page (avatars, thumbnails) | `86400`–`604800` (1–7 days)                        |
| Client portal / invoice link                 | `604800`–`2592000` (7–30 days)                     |
| Semi-permanent assets                        | `315360000` (10 years works but see warning below) |

> **Warning (client-side method):** Client-side presigned URLs **always expire** there is no "permanent" option. If you delete, disable, or rotate the API key, **all** client-side presigned URLs generated with it become invalid immediately.

> **Warning (server-side method):** Pretty presigned URLs are stored in the database. If you delete the database, all pretty URLs are lost. However, they are **not** tied to the API key rotating your API key does NOT invalidate pretty URLs.

#### Revoking URLs

| Method                 | Client-Side HMAC    | Server-Side Pretty                            |
| ---------------------- | ------------------- | --------------------------------------------- |
| Revoke a single URL    | Not possible        | `$beamdrop->revokePrettyPresignedUrl($token)` |
| Delete the object      | URL returns 404     | URL returns 404                               |
| Disable/rotate API key | All HMAC URLs break | Pretty URLs still work                        |
| Wait for expiry        | Resolves itself     | Resolves itself                               |

---

### 7.6 Deleting Files

```php
$beamdrop->deleteObject('avatars', 'user-42/avatar.jpg');
```

Returns `true` on success. Throws `BeamdropException` on failure.

---

<a name="8-error-handling"></a>

## 8. Error Handling

The service class throws `App\Services\BeamdropException` on any non-2xx HTTP response. Every exception carries the HTTP status code and the structured error body from Beamdrop.

```php
use App\Services\BeamdropException;

try {
    $beamdrop->putObject('avatars', 'user-42/pic.jpg', $data);
} catch (BeamdropException $e) {
    // $e->getCode()      HTTP status (404, 409, 423, 429, 500, …)
    // $e->getMessage()   Human-readable error message
    // $e->getBody()      Full decoded JSON error body

    if ($e->getCode() === 429) {
        // Rate limited   back off and retry
        $retryAfter = $e->getBody()['error']['retryAfter'] ?? 10;
        sleep($retryAfter);
    }

    if ($e->getCode() === 423) {
        // File is locked (another upload in progress)   retry shortly
    }

    Log::error('Beamdrop error', [
        'status'  => $e->getCode(),
        'message' => $e->getMessage(),
    ]);
}
```

---

<a name="9-security-best-practices"></a>

## 9. Security Best Practices

1. **Always enable `-api-auth`** on the Beamdrop server in production.
2. **Store keys in `.env`** never commit `BEAMDROP_SECRET_KEY` to version control.
3. **Use HTTPS** in production (put Caddy/Nginx in front of Beamdrop or use the bundled Caddyfile).
4. **Use presigned URLs** to serve files to end users instead of proxying through your PHP app keeps bandwidth off your Laravel server.
5. **Validate uploads** in Laravel before sending to Beamdrop (mime type, size, extension).
6. **Scope keys per environment** create separate API keys for staging and production.
7. **Use structured key paths** like `user-{id}/avatars/file.jpg` to keep files organized and avoid collisions.
8. **Set key expiry** on API keys if your security policy requires rotation:
   ```bash
   curl -X POST http://server:8090/api/v1/keys \
     -H "Content-Type: application/json" \
     -d '{"name": "laravel-prod", "expiresIn": 7776000}' # 90 days
   ```

---

<a name="10-api-reference"></a>

## 10. API Reference Quick Cheat Sheet

All methods available on the `Beamdrop` service class:

| Method                                                       | Description                                               | Returns                                                               |
| ------------------------------------------------------------ | --------------------------------------------------------- | --------------------------------------------------------------------- |
| `createBucket(string $name)`                                 | Create a new bucket (directory)                           | `['bucket', 'created', 'location']`                                   |
| `createBucketIfNotExists(string $name)`                      | Create bucket if it doesn't exist (idempotent)            | `['bucket', 'location', ...]`                                         |
| `deleteBucket(string $name)`                                 | Delete an empty bucket                                    | `true`                                                                |
| `listBuckets()`                                              | List all buckets                                          | `['buckets' => [...], 'count' => n]`                                  |
| `bucketExists(string $name)`                                 | Check if bucket exists                                    | `bool`                                                                |
| `putObject(string $bucket, string $key, string $body)`       | Upload a file (raw bytes)                                 | `['bucket', 'key', 'etag', 'size', 'url']`                            |
| `getObject(string $bucket, string $key)`                     | Download a file                                           | `['body', 'content_type', 'content_length', 'etag', 'last_modified']` |
| `deleteObject(string $bucket, string $key)`                  | Delete a file                                             | `true`                                                                |
| `headObject(string $bucket, string $key)`                    | Get file metadata without downloading                     | `['content_type', 'content_length', 'etag', 'last_modified']`         |
| `objectExists(string $bucket, string $key)`                  | Check if a file exists                                    | `bool`                                                                |
| `listObjects(string $bucket, ...)`                           | List objects with prefix/delimiter filtering              | `['contents', 'commonPrefixes', ...]`                                 |
| `presignedUrl(string $bucket, string $key, int $expiresIn)`  | Generate a client-side HMAC presigned URL                 | `string` (full URL)                                                   |
| `createPrettyPresignedUrl(string $bucket, string $key, ...)` | Create a server-side pretty presigned URL (`/dl/{token}`) | `['token', 'url', 'bucket', 'key', ...]`                              |
| `revokePrettyPresignedUrl(string $token)`                    | Revoke a server-side pretty presigned URL                 | `true`                                                                |
| `listPrettyPresignedUrls()`                                  | List all server-side pretty presigned URLs                | `['urls' => [...], 'count' => n]`                                     |

---

<a name="11-troubleshooting"></a>

## 11. Troubleshooting

| Problem                            | Cause                                      | Fix                                                                  |
| ---------------------------------- | ------------------------------------------ | -------------------------------------------------------------------- |
| `401 Missing Authorization header` | Auth is enabled but no key configured      | Set `BEAMDROP_ACCESS_KEY` and `BEAMDROP_SECRET_KEY` in `.env`        |
| `401 Request timestamp is too old` | Clock skew > 15 minutes                    | Sync server clocks with NTP                                          |
| `403 Invalid signature`            | Wrong secret key or encoding issue         | Regenerate the API key; confirm `.env` has no trailing whitespace    |
| `404 Bucket not found`             | Bucket doesn't exist                       | Call `$beamdrop->createBucket('my-bucket')` first                    |
| `409 Bucket already exists`        | Creating a bucket that exists              | Use `$beamdrop->createBucketIfNotExists()` to avoid this             |
| `423 Object locked`                | Another upload to the same key in progress | Retry after a short delay                                            |
| `429 Too Many Requests`            | Rate limit hit                             | Respect `Retry-After` header; increase `-rate-limit` on server       |
| `cURL error`                       | Beamdrop server unreachable                | Check `BEAMDROP_URL`, firewall rules, and that the server is running |

---

## Bucket Organization Recommendations

Here's a suggested bucket structure for a typical Laravel app:

```
buckets/
├── avatars/                     ← Profile pictures
│   └── user-{id}/
│       └── avatar.{ext}
├── documents/                   ← General uploads
│   └── {date}/
│       └── {uuid}.{ext}
├── invoices/                    ← PDF invoices
│   └── company-{id}/
│       └── {year}/{month}/
│           └── INV-{number}.pdf
└── exports/                     ← CSV/Excel exports
    └── {date}/
        └── {type}-{uuid}.csv
```

Create all buckets on deploy (idempotent no error if bucket already exists):

```php
// In a seeder or deploy script
$beamdrop = app(Beamdrop::class);

foreach (['avatars', 'documents', 'invoices', 'exports'] as $bucket) {
    $beamdrop->createBucketIfNotExists($bucket);
}
```

> **Tip:** `createBucketIfNotExists()` returns 201 if newly created, 200 if it already existed. No need to catch 409 errors.
