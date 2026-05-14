import { useState } from "react";
import {
  BookOpen,
  Copy,
  Check,
  ChevronDown,
  ChevronRight,
  Terminal,
  Shield,
  Link,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { toast } from "@/hooks/use-toast";

const methodColors: Record<string, string> = {
  GET: "bg-green-500/10 text-green-600 border-green-500/20",
  PUT: "bg-blue-500/10 text-blue-600 border-blue-500/20",
  POST: "bg-yellow-500/10 text-yellow-600 border-yellow-500/20",
  DELETE: "bg-red-500/10 text-red-600 border-red-500/20",
  HEAD: "bg-purple-500/10 text-purple-600 border-purple-500/20",
};

interface Endpoint {
  method: string;
  path: string;
  title: string;
  description: string;
  headers?: string;
  request?: string;
  response: string;
  status: string;
}

const bucketEndpoints: Endpoint[] = [
  {
    method: "GET",
    path: "/api/v1/buckets",
    title: "List Buckets",
    description: "Returns all buckets on the server.",
    status: "200 OK",
    response: `{
  "buckets": [
    { "name": "uploads", "createdAt": "2026-01-15T08:30:00Z" },
    { "name": "backups", "createdAt": "2026-02-01T12:00:00Z" }
  ],
  "count": 2
}`,
  },
  {
    method: "PUT",
    path: "/api/v1/buckets/{bucket}",
    title: "Create Bucket",
    description: "Creates a new storage bucket. Bucket names must be lowercase alphanumeric with hyphens.",
    status: "201 Created",
    response: `{
  "bucket": "my-bucket",
  "created": "2026-05-14T10:00:00Z",
  "location": "/api/v1/buckets/my-bucket"
}`,
  },
  {
    method: "HEAD",
    path: "/api/v1/buckets/{bucket}",
    title: "Check Bucket",
    description: "Check if a bucket exists. Returns 200 if it exists, 404 if not. No response body.",
    status: "200 OK / 404 Not Found",
    response: "(No body — check status code)",
  },
  {
    method: "DELETE",
    path: "/api/v1/buckets/{bucket}",
    title: "Delete Bucket",
    description: "Deletes an empty bucket. Fails with 409 if the bucket contains objects.",
    status: "204 No Content",
    response: "(No body)",
  },
];

const objectEndpoints: Endpoint[] = [
  {
    method: "PUT",
    path: "/api/v1/buckets/{bucket}/{key}",
    title: "Upload Object",
    description: "Upload a file. Send the raw file bytes in the request body. The key can include path separators (e.g. photos/2026/img.jpg).",
    headers: "Content-Type: application/octet-stream",
    status: "200 OK",
    response: `{
  "bucket": "my-bucket",
  "key": "photos/img.jpg",
  "etag": "d41d8cd98f00b204e9800998ecf8427e",
  "size": 102400,
  "url": "/api/v1/buckets/my-bucket/photos/img.jpg"
}`,
  },
  {
    method: "GET",
    path: "/api/v1/buckets/{bucket}/{key}",
    title: "Download Object",
    description: "Downloads the file. Returns the raw binary content with appropriate Content-Type and Content-Disposition headers.",
    status: "200 OK",
    response: "(Raw file bytes)",
  },
  {
    method: "HEAD",
    path: "/api/v1/buckets/{bucket}/{key}",
    title: "Object Metadata",
    description: "Returns object metadata (size, content type, last modified) in response headers without downloading the file.",
    status: "200 OK",
    response: `(No body — metadata in headers)
Content-Type: image/jpeg
Content-Length: 102400
Last-Modified: Wed, 14 May 2026 10:00:00 GMT
ETag: "d41d8cd98f00b204e9800998ecf8427e"`,
  },
  {
    method: "GET",
    path: "/api/v1/buckets/{bucket}?prefix=...&delimiter=/",
    title: "List Objects",
    description: "List objects in a bucket. Use prefix to filter by path and delimiter=/ to get folder-like grouping.",
    status: "200 OK",
    response: `{
  "bucket": "my-bucket",
  "prefix": "photos/",
  "delimiter": "/",
  "maxKeys": 1000,
  "isTruncated": false,
  "contents": [
    {
      "key": "photos/img.jpg",
      "size": 102400,
      "lastModified": "2026-05-14T10:00:00Z"
    }
  ],
  "commonPrefixes": ["photos/thumbnails/"]
}`,
  },
  {
    method: "DELETE",
    path: "/api/v1/buckets/{bucket}/{key}",
    title: "Delete Object",
    description: "Permanently deletes an object from the bucket.",
    status: "204 No Content",
    response: "(No body)",
  },
];

const presignEndpoints: Endpoint[] = [
  {
    method: "POST",
    path: "/api/v1/presign",
    title: "Create Presigned URL",
    description: "Generate a short, shareable download URL with optional expiry and download limit. The returned URL does not require authentication.",
    headers: "Content-Type: application/json",
    request: `{
  "bucket": "my-bucket",
  "key": "photos/img.jpg",
  "expiresIn": "24h",
  "maxDownloads": 10
}`,
    status: "201 Created",
    response: `{
  "url": "/dl/abc123def456",
  "token": "abc123def456",
  "expiresAt": "2026-05-15T10:00:00Z",
  "maxDownloads": 10
}`,
  },
  {
    method: "GET",
    path: "/api/v1/presign",
    title: "List Presigned URLs",
    description: "List all active presigned URLs with their download counts and expiry status.",
    status: "200 OK",
    response: `{
  "urls": [
    {
      "token": "abc123def456",
      "bucket": "my-bucket",
      "key": "photos/img.jpg",
      "downloads": 3,
      "maxDownloads": 10,
      "expiresAt": "2026-05-15T10:00:00Z",
      "createdAt": "2026-05-14T10:00:00Z"
    }
  ]
}`,
  },
  {
    method: "DELETE",
    path: "/api/v1/presign/{token}",
    title: "Revoke Presigned URL",
    description: "Revoke a presigned URL so it can no longer be used for downloads.",
    status: "204 No Content",
    response: "(No body)",
  },
];

const keyEndpoints: Endpoint[] = [
  {
    method: "POST",
    path: "/api/v1/keys",
    title: "Create API Key",
    description: "Create a new API key. The secret key is returned only once — save it immediately.",
    headers: "Content-Type: application/json",
    request: `{
  "name": "my-app",
  "permissions": "readwrite",
  "bucketScope": "",
  "expiresIn": "90d"
}`,
    status: "201 Created",
    response: `{
  "accessKeyId": "BDK_a1b2c3d4e5f6g7h8",
  "secretKey": "sk_9h8g7f6e5d4c3b2a1...",
  "name": "my-app",
  "permissions": "readwrite",
  "expiresAt": "2026-08-12T10:00:00Z"
}`,
  },
  {
    method: "GET",
    path: "/api/v1/keys",
    title: "List API Keys",
    description: "List all API keys. Secret keys are never returned after creation.",
    status: "200 OK",
    response: `{
  "keys": [
    {
      "id": 1,
      "name": "my-app",
      "accessKeyId": "BDK_a1b2c3d4e5f6g7h8",
      "permissions": "readwrite",
      "createdAt": "2026-05-14T10:00:00Z",
      "lastUsedAt": "2026-05-14T12:30:00Z",
      "disabled": false
    }
  ]
}`,
  },
  {
    method: "DELETE",
    path: "/api/v1/keys?accessKeyId={id}",
    title: "Delete API Key",
    description: "Permanently delete an API key. Any presigned URLs generated with this key's HMAC will stop working.",
    status: "200 OK",
    response: `{ "message": "API key deleted" }`,
  },
];

function CodeBlock({ code, id, onCopy, copiedId }: { code: string; id: string; onCopy: (text: string, id: string) => void; copiedId: string | null }) {
  return (
    <div className="relative">
      <Button
        variant="ghost"
        size="sm"
        className="absolute top-2 right-2 h-6 w-6 p-0"
        onClick={() => onCopy(code, id)}
      >
        {copiedId === id ? (
          <Check className="w-3 h-3 text-green-500" />
        ) : (
          <Copy className="w-3 h-3" />
        )}
      </Button>
      <pre className="p-3 bg-muted rounded-lg text-xs font-mono overflow-x-auto whitespace-pre-wrap pr-10">
        {code}
      </pre>
    </div>
  );
}

function EndpointSection({ title, endpoints, copiedId, onCopy, expandedId, onToggle, baseUrl }: {
  title: string;
  endpoints: Endpoint[];
  copiedId: string | null;
  onCopy: (text: string, id: string) => void;
  expandedId: string | null;
  onToggle: (id: string) => void;
  baseUrl: string;
}) {
  return (
    <div className="space-y-3">
      <h3 className="text-sm font-bold font-mono uppercase tracking-wide text-muted-foreground">
        {title}
      </h3>
      {endpoints.map((ep, i) => {
        const id = `${title}-${ep.method}-${ep.path}-${i}`;
        const isOpen = expandedId === id;
        const curlCmd = buildCurl(ep, baseUrl);

        return (
          <Collapsible key={id} open={isOpen} onOpenChange={() => onToggle(id)}>
            <Card className="overflow-hidden">
              <CollapsibleTrigger className="w-full">
                <div className="flex items-center justify-between p-3 hover:bg-muted/50 transition-colors">
                  <div className="flex items-center gap-3">
                    <Badge variant="outline" className={`font-mono text-xs min-w-[52px] ${methodColors[ep.method]}`}>
                      {ep.method}
                    </Badge>
                    <code className="text-sm font-mono text-left">{ep.path}</code>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-xs text-muted-foreground font-mono hidden sm:block">{ep.title}</span>
                    {isOpen ? <ChevronDown className="w-4 h-4 text-muted-foreground" /> : <ChevronRight className="w-4 h-4 text-muted-foreground" />}
                  </div>
                </div>
              </CollapsibleTrigger>
              <CollapsibleContent>
                <div className="px-3 pb-3 space-y-3 border-t pt-3">
                  <p className="text-sm text-muted-foreground">{ep.description}</p>

                  {ep.headers && (
                    <div>
                      <span className="text-xs font-mono uppercase text-muted-foreground">Headers</span>
                      <pre className="p-2 bg-muted rounded text-xs font-mono mt-1">{ep.headers}</pre>
                    </div>
                  )}

                  {ep.request && (
                    <div>
                      <span className="text-xs font-mono uppercase text-muted-foreground">Request Body</span>
                      <CodeBlock code={ep.request} id={`req-${id}`} onCopy={onCopy} copiedId={copiedId} />
                    </div>
                  )}

                  <div>
                    <span className="text-xs font-mono uppercase text-muted-foreground">Example Request</span>
                    <CodeBlock code={curlCmd} id={`curl-${id}`} onCopy={onCopy} copiedId={copiedId} />
                  </div>

                  <div className="flex items-center gap-2">
                    <span className="text-xs font-mono uppercase text-muted-foreground">Status</span>
                    <Badge variant="outline" className="font-mono text-xs">{ep.status}</Badge>
                  </div>

                  <div>
                    <span className="text-xs font-mono uppercase text-muted-foreground">Response</span>
                    <pre className="p-3 bg-muted rounded-lg text-xs font-mono overflow-x-auto whitespace-pre-wrap mt-1">{ep.response}</pre>
                  </div>
                </div>
              </CollapsibleContent>
            </Card>
          </Collapsible>
        );
      })}
    </div>
  );
}

function buildCurl(ep: Endpoint, baseUrl: string): string {
  const url = `${baseUrl}${ep.path}`;
  const parts = [`curl -X ${ep.method}`];

  if (ep.headers) {
    ep.headers.split("\n").forEach((h) => {
      parts.push(`  -H "${h.trim()}"`);
    });
  }

  // Add auth header placeholder for authenticated endpoints
  if (!ep.path.startsWith("/dl/")) {
    parts.push(`  -H "Authorization: Bearer <ACCESS_KEY_ID>:<SIGNATURE>"`);
    parts.push(`  -H "X-Beamdrop-Date: <ISO_8601_TIMESTAMP>"`);
  }

  if (ep.request) {
    parts.push(`  -d '${ep.request.replace(/\n/g, "").replace(/\s+/g, " ")}'`);
  }

  if (ep.method === "PUT" && ep.path.includes("{key}")) {
    parts.push(`  --data-binary @yourfile.txt`);
  }

  parts.push(`  "${url}"`);
  return parts.join(" \\\n");
}

export default function ApiDocsPage() {
  const [copiedId, setCopiedId] = useState<string | null>(null);
  const [expandedId, setExpandedId] = useState<string | null>(null);

  const baseUrl = typeof window !== "undefined" ? window.location.origin : "http://localhost:7777";

  const handleCopy = async (text: string, id: string) => {
    try {
      await navigator.clipboard.writeText(text);
      setCopiedId(id);
      setTimeout(() => setCopiedId(null), 2000);
    } catch {
      toast({ title: "Error", description: "Failed to copy", variant: "destructive" });
    }
  };

  const handleToggle = (id: string) => {
    setExpandedId(expandedId === id ? null : id);
  };

  return (
    <div className="p-6 space-y-6 animate-fade-in max-w-4xl">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold font-mono uppercase tracking-wide flex items-center gap-2">
          <BookOpen className="w-6 h-6" />
          API Documentation
        </h1>
        <p className="text-sm text-muted-foreground font-mono mt-1">
          S3-compatible REST API — all examples use curl (adapt to any HTTP client)
        </p>
      </div>

      <Tabs defaultValue="endpoints" className="w-full">
        <TabsList className="grid w-full grid-cols-3">
          <TabsTrigger value="endpoints" className="font-mono text-xs uppercase">Endpoints</TabsTrigger>
          <TabsTrigger value="quickstart" className="font-mono text-xs uppercase">Quick Start</TabsTrigger>
          <TabsTrigger value="auth" className="font-mono text-xs uppercase">Authentication</TabsTrigger>
        </TabsList>

        {/* ── Endpoints ── */}
        <TabsContent value="endpoints" className="space-y-8 mt-6">
          <EndpointSection title="Buckets" endpoints={bucketEndpoints} copiedId={copiedId} onCopy={handleCopy} expandedId={expandedId} onToggle={handleToggle} baseUrl={baseUrl} />
          <EndpointSection title="Objects" endpoints={objectEndpoints} copiedId={copiedId} onCopy={handleCopy} expandedId={expandedId} onToggle={handleToggle} baseUrl={baseUrl} />
          <EndpointSection title="Presigned URLs" endpoints={presignEndpoints} copiedId={copiedId} onCopy={handleCopy} expandedId={expandedId} onToggle={handleToggle} baseUrl={baseUrl} />
          <EndpointSection title="API Key Management" endpoints={keyEndpoints} copiedId={copiedId} onCopy={handleCopy} expandedId={expandedId} onToggle={handleToggle} baseUrl={baseUrl} />
        </TabsContent>

        {/* ── Quick Start ── */}
        <TabsContent value="quickstart" className="space-y-4 mt-6">
          <Card>
            <CardHeader>
              <CardTitle className="font-mono uppercase text-sm flex items-center gap-2">
                <Terminal className="w-4 h-4" />
                Quick Start (curl)
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <p className="text-sm text-muted-foreground mb-2">1. Create a bucket</p>
                <CodeBlock code={`curl -X PUT "${baseUrl}/api/v1/buckets/my-files" \\
  -H "Authorization: Bearer <KEY>:<SIG>" \\
  -H "X-Beamdrop-Date: <TIMESTAMP>"`} id="qs-1" onCopy={handleCopy} copiedId={copiedId} />
              </div>
              <div>
                <p className="text-sm text-muted-foreground mb-2">2. Upload a file</p>
                <CodeBlock code={`curl -X PUT "${baseUrl}/api/v1/buckets/my-files/photo.jpg" \\
  -H "Authorization: Bearer <KEY>:<SIG>" \\
  -H "X-Beamdrop-Date: <TIMESTAMP>" \\
  --data-binary @photo.jpg`} id="qs-2" onCopy={handleCopy} copiedId={copiedId} />
              </div>
              <div>
                <p className="text-sm text-muted-foreground mb-2">3. List objects</p>
                <CodeBlock code={`curl "${baseUrl}/api/v1/buckets/my-files" \\
  -H "Authorization: Bearer <KEY>:<SIG>" \\
  -H "X-Beamdrop-Date: <TIMESTAMP>"`} id="qs-3" onCopy={handleCopy} copiedId={copiedId} />
              </div>
              <div>
                <p className="text-sm text-muted-foreground mb-2">4. Download a file</p>
                <CodeBlock code={`curl "${baseUrl}/api/v1/buckets/my-files/photo.jpg" \\
  -H "Authorization: Bearer <KEY>:<SIG>" \\
  -H "X-Beamdrop-Date: <TIMESTAMP>" \\
  -o photo.jpg`} id="qs-4" onCopy={handleCopy} copiedId={copiedId} />
              </div>
              <div>
                <p className="text-sm text-muted-foreground mb-2">5. Create a presigned download link (no auth needed to download)</p>
                <CodeBlock code={`curl -X POST "${baseUrl}/api/v1/presign" \\
  -H "Authorization: Bearer <KEY>:<SIG>" \\
  -H "X-Beamdrop-Date: <TIMESTAMP>" \\
  -H "Content-Type: application/json" \\
  -d '{"bucket":"my-files","key":"photo.jpg","expiresIn":"24h"}'`} id="qs-5" onCopy={handleCopy} copiedId={copiedId} />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="font-mono uppercase text-sm flex items-center gap-2">
                <Link className="w-4 h-4" />
                Base URL
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground mb-2">
                All API endpoints are relative to your BeamDrop server:
              </p>
              <code className="px-3 py-2 bg-muted rounded font-mono text-sm block">{baseUrl}</code>
            </CardContent>
          </Card>
        </TabsContent>

        {/* ── Authentication ── */}
        <TabsContent value="auth" className="space-y-4 mt-6">
          <Card>
            <CardHeader>
              <CardTitle className="font-mono uppercase text-sm flex items-center gap-2">
                <Shield className="w-4 h-4" />
                HMAC-SHA256 Authentication
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <p className="text-sm text-muted-foreground">
                Every API request must include two headers for authentication:
              </p>

              <div className="space-y-2">
                <div className="p-3 bg-muted rounded-lg">
                  <code className="text-xs font-mono">
                    Authorization: Bearer {"<ACCESS_KEY_ID>"}:{"<SIGNATURE>"}
                  </code>
                </div>
                <div className="p-3 bg-muted rounded-lg">
                  <code className="text-xs font-mono">
                    X-Beamdrop-Date: {"<ISO_8601_TIMESTAMP>"}
                  </code>
                </div>
              </div>

              <div className="space-y-2">
                <h4 className="text-sm font-bold font-mono">How to compute the signature</h4>
                <ol className="text-sm text-muted-foreground space-y-2 list-decimal list-inside">
                  <li>Build the string to sign: <code className="bg-muted px-1 rounded text-xs">METHOD\nPATH\nTIMESTAMP</code></li>
                  <li>Compute HMAC-SHA256 of that string using your <strong>secret key</strong></li>
                  <li>Hex-encode the result — that's your signature</li>
                </ol>
              </div>

              <div>
                <span className="text-xs font-mono uppercase text-muted-foreground">Example — signing a GET request</span>
                <CodeBlock code={`# String to sign:
#   GET\n/api/v1/buckets\n2026-05-14T10:00:00Z

# Pseudocode (any language):
string_to_sign = "GET\\n/api/v1/buckets\\n2026-05-14T10:00:00Z"
signature = hex(hmac_sha256(secret_key, string_to_sign))

# Then send:
curl "${baseUrl}/api/v1/buckets" \\
  -H "Authorization: Bearer BDK_abc123:$signature" \\
  -H "X-Beamdrop-Date: 2026-05-14T10:00:00Z"`} id="auth-example" onCopy={handleCopy} copiedId={copiedId} />
              </div>

              <div className="p-3 border rounded-lg bg-yellow-500/5 border-yellow-500/20">
                <p className="text-sm text-muted-foreground">
                  <strong>Clock skew:</strong> The timestamp must be within <strong>15 minutes</strong> of the server time. Use UTC ISO 8601 format.
                </p>
              </div>

              <div className="p-3 border rounded-lg bg-blue-500/5 border-blue-500/20">
                <p className="text-sm text-muted-foreground">
                  <strong>Tip:</strong> When API auth is disabled (<code className="bg-muted px-1 rounded text-xs">-api-auth=false</code>), the Authorization header is not required and all S3 API endpoints are open.
                </p>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}
