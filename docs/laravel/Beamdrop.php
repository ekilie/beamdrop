<?php

declare(strict_types=1);

namespace App\Services;

use RuntimeException;

/**
 * Exception thrown when a Beamdrop API request fails.
 *
 * Carries the HTTP status code and the decoded JSON error body so callers
 * can inspect the structured error response from Beamdrop.
 */
class BeamdropException extends RuntimeException
{
    /** @var array<string, mixed>|null */
    private ?array $body;

    /**
     * @param string                    $message  Human-readable error message.
     * @param int                       $code     HTTP status code (e.g. 404, 429).
     * @param array<string, mixed>|null $body     Decoded JSON error body from the server.
     */
    public function __construct(string $message, int $code = 0, ?array $body = null)
    {
        parent::__construct($message, $code);
        $this->body = $body;
    }

    /**
     * Get the full decoded JSON error body returned by Beamdrop.
     *
     * @return array<string, mixed>|null
     */
    public function getBody(): ?array
    {
        return $this->body;
    }
}

/**
 * Beamdrop — A production-ready PHP client for the Beamdrop S3-compatible API.
 *
 * Handles HMAC-SHA256 request signing, presigned URL generation, bucket and
 * object management. Designed for use as a Laravel singleton service.
 *
 * Usage:
 *   $beamdrop = new Beamdrop(
 *       baseUrl:   'https://files.example.com',
 *       accessKey: 'BDK_abc123',
 *       secretKey: 'sk_secret',
 *   );
 *
 *   $beamdrop->createBucket('avatars');
 *   $beamdrop->putObject('avatars', 'user-1/photo.jpg', file_get_contents($path));
 *   $url = $beamdrop->presignedUrl('avatars', 'user-1/photo.jpg', 3600);
 *
 * @see https://github.com/ekilie/beamdrop
 */
class Beamdrop
{
    private string $baseUrl;
    private string $accessKey;
    private string $secretKey;

    /** cURL connection timeout in seconds. */
    private int $connectTimeout;

    /** cURL total request timeout in seconds. */
    private int $timeout;

    /**
     * @param string $baseUrl        Base URL of the Beamdrop server (no trailing slash).
     * @param string $accessKey      API access key ID (starts with BDK_).
     * @param string $secretKey      API secret key (starts with sk_).
     * @param int    $connectTimeout Connection timeout in seconds (default: 10).
     * @param int    $timeout        Total request timeout in seconds (default: 120).
     */
    public function __construct(
        string $baseUrl,
        string $accessKey,
        string $secretKey,
        int $connectTimeout = 10,
        int $timeout = 120,
    ) {
        $this->baseUrl = rtrim($baseUrl, '/');
        $this->accessKey = $accessKey;
        $this->secretKey = $secretKey;
        $this->connectTimeout = $connectTimeout;
        $this->timeout = $timeout;
    }

    // ──────────────────────────────────────────────
    //  Bucket operations
    // ──────────────────────────────────────────────

    /**
     * Create a new bucket.
     *
     * Bucket names must be 3–63 lowercase alphanumeric characters, hyphens, or dots.
     * Must start and end with a letter or number. Cannot look like an IP address.
     *
     * @param  string $name  Bucket name (e.g. "avatars", "invoices").
     * @return array{bucket: string, created: string, location: string}
     *
     * @throws BeamdropException 409 if the bucket already exists.
     */
    public function createBucket(string $name): array
    {
        return $this->request('PUT', "/api/v1/buckets/{$name}");
    }

    /**
     * Delete an empty bucket.
     *
     * The bucket must contain no objects. Delete all objects first.
     *
     * @param  string $name  Bucket name.
     * @return true
     *
     * @throws BeamdropException 404 if bucket not found, 409 if not empty.
     */
    public function deleteBucket(string $name): true
    {
        $this->request('DELETE', "/api/v1/buckets/{$name}");

        return true;
    }

    /**
     * List all buckets.
     *
     * @return array{buckets: array<int, array{name: string, createdAt: string}>, count: int}
     */
    public function listBuckets(): array
    {
        return $this->request('GET', '/api/v1/buckets');
    }

    /**
     * Check whether a bucket exists.
     *
     * Uses a HEAD request — no response body is transferred.
     *
     * @param  string $name  Bucket name.
     * @return bool
     */
    public function bucketExists(string $name): bool
    {
        try {
            $this->request('HEAD', "/api/v1/buckets/{$name}");

            return true;
        } catch (BeamdropException $e) {
            if ($e->getCode() === 404) {
                return false;
            }
            throw $e;
        }
    }

    // ──────────────────────────────────────────────
    //  Object operations
    // ──────────────────────────────────────────────

    /**
     * Upload a file (raw bytes).
     *
     * The bucket must exist before uploading. The key may contain forward
     * slashes to simulate a directory structure (e.g. "user-1/avatar.jpg").
     *
     * @param  string $bucket  Bucket name.
     * @param  string $key     Object key (relative path inside the bucket).
     * @param  string $body    Raw file contents.
     * @return array{bucket: string, key: string, etag: string, size: int, url: string}
     *
     * @throws BeamdropException 404 bucket not found, 423 object locked, 429 rate limited.
     */
    public function putObject(string $bucket, string $key, string $body): array
    {
        return $this->request('PUT', "/api/v1/buckets/{$bucket}/{$key}", $body);
    }

    /**
     * Download a file.
     *
     * Returns the raw body, content type, length, ETag, and last-modified date.
     *
     * @param  string $bucket  Bucket name.
     * @param  string $key     Object key.
     * @return array{body: string, content_type: string, content_length: int, etag: string, last_modified: string}
     *
     * @throws BeamdropException 404 if bucket or object not found.
     */
    public function getObject(string $bucket, string $key): array
    {
        return $this->rawRequest('GET', "/api/v1/buckets/{$bucket}/{$key}");
    }

    /**
     * Delete a file.
     *
     * @param  string $bucket  Bucket name.
     * @param  string $key     Object key.
     * @return true
     *
     * @throws BeamdropException 404 if not found, 423 if locked.
     */
    public function deleteObject(string $bucket, string $key): true
    {
        $this->request('DELETE', "/api/v1/buckets/{$bucket}/{$key}");

        return true;
    }

    /**
     * Get object metadata without downloading the body.
     *
     * @param  string $bucket  Bucket name.
     * @param  string $key     Object key.
     * @return array{content_type: string, content_length: int, etag: string, last_modified: string}
     *
     * @throws BeamdropException 404 if not found.
     */
    public function headObject(string $bucket, string $key): array
    {
        return $this->rawRequest('HEAD', "/api/v1/buckets/{$bucket}/{$key}");
    }

    /**
     * Check whether an object exists.
     *
     * @param  string $bucket  Bucket name.
     * @param  string $key     Object key.
     * @return bool
     */
    public function objectExists(string $bucket, string $key): bool
    {
        try {
            $this->headObject($bucket, $key);

            return true;
        } catch (BeamdropException $e) {
            if ($e->getCode() === 404) {
                return false;
            }
            throw $e;
        }
    }

    /**
     * List objects in a bucket with optional prefix/delimiter filtering.
     *
     * @param  string      $bucket     Bucket name.
     * @param  string|null $prefix     Only return keys starting with this prefix.
     * @param  string|null $delimiter  Group keys by this character (usually "/") to simulate folders.
     * @param  int         $maxKeys    Maximum number of results (1–1000, default 1000).
     * @return array{bucket: string, prefix: string, delimiter: string, maxKeys: int, isTruncated: bool, contents: array, commonPrefixes: array}
     */
    public function listObjects(
        string $bucket,
        ?string $prefix = null,
        ?string $delimiter = null,
        int $maxKeys = 1000,
    ): array {
        $query = [];

        if ($prefix !== null) {
            $query['prefix'] = $prefix;
        }

        if ($delimiter !== null) {
            $query['delimiter'] = $delimiter;
        }

        if ($maxKeys !== 1000) {
            $query['max-keys'] = $maxKeys;
        }

        $path = "/api/v1/buckets/{$bucket}";

        if (!empty($query)) {
            $path .= '?' . http_build_query($query);
        }

        return $this->request('GET', $path);
    }

    // ──────────────────────────────────────────────
    //  Presigned URLs
    // ──────────────────────────────────────────────

    /**
     * Generate a time-limited presigned URL for downloading a file.
     *
     * The URL can be shared with anyone — no API key required to access it.
     * The link expires after $expiresIn seconds.
     *
     * @param  string $bucket     Bucket name.
     * @param  string $key        Object key.
     * @param  int    $expiresIn  Seconds until the URL expires (e.g. 3600 = 1 hour).
     * @param  string $method     HTTP method the URL is valid for (default: "GET").
     * @return string  The full presigned URL.
     */
    public function presignedUrl(
        string $bucket,
        string $key,
        int $expiresIn,
        string $method = 'GET',
    ): string {
        $expiresAt = time() + $expiresIn;
        $expiresAtDt = gmdate('Y-m-d\TH:i:s\Z', $expiresAt);

        // Token = Base64URL(HMAC-SHA256("METHOD\nBUCKET\nKEY\nUNIX_TIMESTAMP", secretKey))
        $message = implode("\n", [$method, $bucket, $key, (string) $expiresAt]);
        $token = rtrim(strtr(base64_encode(
            hash_hmac('sha256', $message, $this->secretKey, binary: true)
        ), '+/', '-_'), '=');

        $path = "/api/v1/buckets/{$bucket}/{$key}";
        $query = http_build_query([
            'token' => $token,
            'expires' => $expiresAtDt,
            'access_key' => $this->accessKey,
        ]);

        return "{$this->baseUrl}{$path}?{$query}";
    }

    // ──────────────────────────────────────────────
    //  Internal: HTTP transport & signing
    // ──────────────────────────────────────────────

    /**
     * Sign and send an API request that expects a JSON response.
     *
     * @param  string      $method  HTTP method (GET, PUT, POST, DELETE, HEAD).
     * @param  string      $path    Request path including query string.
     * @param  string|null $body    Raw request body (for PUT/POST).
     * @return array<string, mixed>  Decoded JSON response.
     *
     * @throws BeamdropException on any non-2xx response.
     */
    private function request(string $method, string $path, ?string $body = null): array
    {
        [$statusCode, $responseBody, $responseHeaders] = $this->sendRequest($method, $path, $body);

        // No-content responses (204)
        if ($statusCode === 204) {
            return [];
        }

        $decoded = json_decode($responseBody, true);

        // Error responses
        if ($statusCode < 200 || $statusCode >= 300) {
            $message = $decoded['error']['message']
                ?? $decoded['message']
                ?? "Beamdrop request failed with status {$statusCode}";

            throw new BeamdropException($message, $statusCode, $decoded);
        }

        return $decoded ?? [];
    }

    /**
     * Sign and send an API request that returns raw content (file downloads, HEAD).
     *
     * @param  string $method  HTTP method.
     * @param  string $path    Request path.
     * @return array{body?: string, content_type: string, content_length: int, etag: string, last_modified: string}
     *
     * @throws BeamdropException on any non-2xx response.
     */
    private function rawRequest(string $method, string $path): array
    {
        [$statusCode, $responseBody, $responseHeaders] = $this->sendRequest($method, $path);

        if ($statusCode < 200 || $statusCode >= 300) {
            $decoded = json_decode($responseBody, true);
            $message = $decoded['error']['message']
                ?? "Beamdrop request failed with status {$statusCode}";

            throw new BeamdropException($message, $statusCode, $decoded);
        }

        $result = [
            'content_type' => $responseHeaders['content-type'] ?? 'application/octet-stream',
            'content_length' => (int) ($responseHeaders['content-length'] ?? 0),
            'etag' => trim($responseHeaders['etag'] ?? '', '"'),
            'last_modified' => $responseHeaders['last-modified'] ?? '',
        ];

        if ($method !== 'HEAD') {
            $result['body'] = $responseBody;
        }

        return $result;
    }

    /**
     * Low-level cURL transport. Signs the request with HMAC-SHA256.
     *
     * Signature scheme:
     *   StringToSign = METHOD + "\n" + PATH + "\n" + TIMESTAMP
     *   Signature    = Base64(HMAC-SHA256(StringToSign, SecretKey))
     *   Header       = Authorization: Bearer ACCESS_KEY:SIGNATURE
     *
     * @param  string      $method  HTTP method.
     * @param  string      $path    Path with optional query string (e.g. "/api/v1/buckets/photos").
     * @param  string|null $body    Request body.
     * @return array{0: int, 1: string, 2: array<string, string>}  [statusCode, body, headers]
     *
     * @throws BeamdropException if cURL fails entirely (network error).
     */
    private function sendRequest(string $method, string $path, ?string $body = null): array
    {
        // ── Build the signed headers ──

        // For signing, use only the path portion (strip query string)
        $signPath = parse_url($path, PHP_URL_PATH) ?: $path;
        $timestamp = gmdate('Y-m-d\TH:i:s\Z');

        $stringToSign = implode("\n", [$method, $signPath, $timestamp]);
        $signature = base64_encode(
            hash_hmac('sha256', $stringToSign, $this->secretKey, binary: true)
        );

        $headers = [
            "Authorization: Bearer {$this->accessKey}:{$signature}",
            "X-Beamdrop-Date: {$timestamp}",
        ];

        if ($body !== null) {
            $headers[] = 'Content-Type: application/octet-stream';
            $headers[] = 'Content-Length: ' . strlen($body);
        }

        // ── Execute the cURL request ──

        $url = $this->baseUrl . $path;
        $ch = curl_init();

        curl_setopt_array($ch, [
            CURLOPT_URL => $url,
            CURLOPT_CUSTOMREQUEST => $method,
            CURLOPT_HTTPHEADER => $headers,
            CURLOPT_RETURNTRANSFER => true,
            CURLOPT_CONNECTTIMEOUT => $this->connectTimeout,
            CURLOPT_TIMEOUT => $this->timeout,
            CURLOPT_FOLLOWLOCATION => true,
            CURLOPT_MAXREDIRS => 3,
        ]);

        if ($body !== null) {
            curl_setopt($ch, CURLOPT_POSTFIELDS, $body);
        }

        if ($method === 'HEAD') {
            curl_setopt($ch, CURLOPT_NOBODY, true);
        }

        // Capture response headers
        $responseHeaders = [];
        curl_setopt($ch, CURLOPT_HEADERFUNCTION, function ($ch, string $header) use (&$responseHeaders): int {
            $len = strlen($header);
            $parts = explode(':', $header, 2);

            if (count($parts) === 2) {
                $name = strtolower(trim($parts[0]));
                $value = trim($parts[1]);
                $responseHeaders[$name] = $value;
            }

            return $len;
        });

        $responseBody = curl_exec($ch);

        if ($responseBody === false) {
            $error = curl_error($ch);
            $errno = curl_errno($ch);
            curl_close($ch);

            throw new BeamdropException(
                "cURL request to Beamdrop failed: [{$errno}] {$error}",
                0
            );
        }

        $statusCode = (int) curl_getinfo($ch, CURLINFO_HTTP_CODE);
        curl_close($ch);

        return [$statusCode, $responseBody, $responseHeaders];
    }
}
