import { useState } from "react";
import {
    BookOpen,
    Copy,
    Check,
    ChevronDown,
    ChevronRight,
    Terminal,
    Code,
    Database,
    Upload,
    Download,
    Trash2,
    FolderPlus,
    List,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
    Collapsible,
    CollapsibleContent,
    CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { toast } from "@/hooks/use-toast";

interface ApiDocsDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
}

const endpoints = [
    {
        method: "GET",
        path: "/api/v1/buckets",
        title: "List Buckets",
        description: "Get a list of all buckets",
        icon: List,
        curl: `curl -X GET "http://localhost:8080/api/v1/buckets"`,
        response: `{
  "buckets": [
    {
      "name": "my-bucket",
      "createdAt": "2026-02-09T12:00:00Z"
    }
  ],
  "count": 1
}`,
    },
    {
        method: "PUT",
        path: "/api/v1/buckets/{bucket}",
        title: "Create Bucket",
        description: "Create a new bucket",
        icon: FolderPlus,
        curl: `curl -X PUT "http://localhost:8080/api/v1/buckets/my-bucket"`,
        response: `{
  "bucket": "my-bucket",
  "created": "2026-02-09T12:00:00Z",
  "location": "/api/v1/buckets/my-bucket"
}`,
    },
    {
        method: "DELETE",
        path: "/api/v1/buckets/{bucket}",
        title: "Delete Bucket",
        description: "Delete an empty bucket",
        icon: Trash2,
        curl: `curl -X DELETE "http://localhost:8080/api/v1/buckets/my-bucket"`,
        response: `(204 No Content)`,
    },
    {
        method: "PUT",
        path: "/api/v1/buckets/{bucket}/{key}",
        title: "Upload Object",
        description: "Upload a file to a bucket",
        icon: Upload,
        curl: `curl -X PUT \\
  -H "Content-Type: application/octet-stream" \\
  --data-binary @file.txt \\
  "http://localhost:8080/api/v1/buckets/my-bucket/path/to/file.txt"`,
        response: `{
  "bucket": "my-bucket",
  "key": "path/to/file.txt",
  "etag": "d41d8cd98f00b204e9800998ecf8427e",
  "size": 1024,
  "url": "/api/v1/buckets/my-bucket/path/to/file.txt"
}`,
    },
    {
        method: "GET",
        path: "/api/v1/buckets/{bucket}/{key}",
        title: "Download Object",
        description: "Download a file from a bucket",
        icon: Download,
        curl: `curl -X GET "http://localhost:8080/api/v1/buckets/my-bucket/path/to/file.txt" -o file.txt`,
        response: `(File binary content)`,
    },
    {
        method: "GET",
        path: "/api/v1/buckets/{bucket}?prefix=...",
        title: "List Objects",
        description: "List objects in a bucket with optional prefix filter",
        icon: Database,
        curl: `curl -X GET "http://localhost:8080/api/v1/buckets/my-bucket?prefix=images/&delimiter=/"`,
        response: `{
  "bucket": "my-bucket",
  "prefix": "images/",
  "delimiter": "/",
  "maxKeys": 1000,
  "isTruncated": false,
  "contents": [
    {
      "key": "images/photo.jpg",
      "size": 102400,
      "lastModified": "2026-02-09T12:00:00Z"
    }
  ],
  "commonPrefixes": ["images/thumbnails/"]
}`,
    },
    {
        method: "DELETE",
        path: "/api/v1/buckets/{bucket}/{key}",
        title: "Delete Object",
        description: "Delete a file from a bucket",
        icon: Trash2,
        curl: `curl -X DELETE "http://localhost:8080/api/v1/buckets/my-bucket/path/to/file.txt"`,
        response: `(204 No Content)`,
    },
];

const methodColors: Record<string, string> = {
    GET: "bg-green-500/10 text-green-600 border-green-500/20",
    PUT: "bg-blue-500/10 text-blue-600 border-blue-500/20",
    POST: "bg-yellow-500/10 text-yellow-600 border-yellow-500/20",
    DELETE: "bg-red-500/10 text-red-600 border-red-500/20",
    HEAD: "bg-purple-500/10 text-purple-600 border-purple-500/20",
};

export function ApiDocsDialog({ open, onOpenChange }: ApiDocsDialogProps) {
    const [copiedCode, setCopiedCode] = useState<string | null>(null);
    const [expandedEndpoint, setExpandedEndpoint] = useState<string | null>(null);

    const handleCopy = async (text: string, id: string) => {
        try {
            await navigator.clipboard.writeText(text);
            setCopiedCode(id);
            setTimeout(() => setCopiedCode(null), 2000);
        } catch {
            toast({
                title: "Error",
                description: "Failed to copy to clipboard",
                variant: "destructive",
            });
        }
    };

    const baseUrl = typeof window !== "undefined" ? window.location.origin : "http://localhost:8080";

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="sm:max-w-[700px] max-h-[80vh]">
                <DialogHeader>
                    <DialogTitle className="font-mono uppercase flex items-center gap-2">
                        <BookOpen className="w-5 h-5" />
                        API Documentation
                    </DialogTitle>
                    <DialogDescription className="font-mono text-xs">
                        S3-compatible REST API for file storage
                    </DialogDescription>
                </DialogHeader>

                <Tabs defaultValue="endpoints" className="w-full">
                    <TabsList className="grid w-full grid-cols-3">
                        <TabsTrigger value="endpoints" className="font-mono text-xs uppercase">
                            Endpoints
                        </TabsTrigger>
                        <TabsTrigger value="quickstart" className="font-mono text-xs uppercase">
                            Quick Start
                        </TabsTrigger>
                        <TabsTrigger value="auth" className="font-mono text-xs uppercase">
                            Authentication
                        </TabsTrigger>
                    </TabsList>

                    <ScrollArea className="h-[50vh] mt-4">
                        <TabsContent value="endpoints" className="space-y-3 pr-4">
                            {endpoints.map((endpoint, index) => {
                                const Icon = endpoint.icon;
                                const isExpanded = expandedEndpoint === `${endpoint.method}-${endpoint.path}`;

                                return (
                                    <Collapsible
                                        key={index}
                                        open={isExpanded}
                                        onOpenChange={() =>
                                            setExpandedEndpoint(isExpanded ? null : `${endpoint.method}-${endpoint.path}`)
                                        }
                                    >
                                        <Card className="overflow-hidden">
                                            <CollapsibleTrigger className="w-full">
                                                <div className="flex items-center justify-between p-3 hover:bg-muted/50 transition-colors">
                                                    <div className="flex items-center gap-3">
                                                        <Badge
                                                            variant="outline"
                                                            className={`font-mono text-xs ${methodColors[endpoint.method]}`}
                                                        >
                                                            {endpoint.method}
                                                        </Badge>
                                                        <code className="text-sm font-mono">{endpoint.path}</code>
                                                    </div>
                                                    <div className="flex items-center gap-2">
                                                        <span className="text-xs text-muted-foreground font-mono hidden sm:block">
                                                            {endpoint.title}
                                                        </span>
                                                        {isExpanded ? (
                                                            <ChevronDown className="w-4 h-4 text-muted-foreground" />
                                                        ) : (
                                                            <ChevronRight className="w-4 h-4 text-muted-foreground" />
                                                        )}
                                                    </div>
                                                </div>
                                            </CollapsibleTrigger>
                                            <CollapsibleContent>
                                                <div className="px-3 pb-3 space-y-3 border-t">
                                                    <p className="text-sm text-muted-foreground pt-3">
                                                        {endpoint.description}
                                                    </p>

                                                    <div className="space-y-2">
                                                        <div className="flex items-center justify-between">
                                                            <span className="text-xs font-mono uppercase text-muted-foreground">
                                                                cURL Example
                                                            </span>
                                                            <Button
                                                                variant="ghost"
                                                                size="sm"
                                                                className="h-6 px-2"
                                                                onClick={(e) => {
                                                                    e.stopPropagation();
                                                                    handleCopy(endpoint.curl.replace("http://localhost:8080", baseUrl), `curl-${index}`);
                                                                }}
                                                            >
                                                                {copiedCode === `curl-${index}` ? (
                                                                    <Check className="w-3 h-3 text-green-500" />
                                                                ) : (
                                                                    <Copy className="w-3 h-3" />
                                                                )}
                                                            </Button>
                                                        </div>
                                                        <pre className="p-3 bg-muted rounded-lg text-xs font-mono overflow-x-auto">
                                                            {endpoint.curl.replace("http://localhost:8080", baseUrl)}
                                                        </pre>
                                                    </div>

                                                    <div className="space-y-2">
                                                        <span className="text-xs font-mono uppercase text-muted-foreground">
                                                            Response
                                                        </span>
                                                        <pre className="p-3 bg-muted rounded-lg text-xs font-mono overflow-x-auto">
                                                            {endpoint.response}
                                                        </pre>
                                                    </div>
                                                </div>
                                            </CollapsibleContent>
                                        </Card>
                                    </Collapsible>
                                );
                            })}
                        </TabsContent>

                        <TabsContent value="quickstart" className="space-y-4 pr-4">
                            <Card className="p-4 space-y-4">
                                <h3 className="font-mono uppercase text-sm font-bold flex items-center gap-2">
                                    <Terminal className="w-4 h-4" />
                                    Quick Start Guide
                                </h3>

                                <div className="space-y-3">
                                    <div>
                                        <p className="text-sm text-muted-foreground mb-2">
                                            1. Create a bucket
                                        </p>
                                        <pre className="p-3 bg-muted rounded-lg text-xs font-mono">
                                            curl -X PUT "{baseUrl}/api/v1/buckets/my-files"
                                        </pre>
                                    </div>

                                    <div>
                                        <p className="text-sm text-muted-foreground mb-2">
                                            2. Upload a file
                                        </p>
                                        <pre className="p-3 bg-muted rounded-lg text-xs font-mono">
                                            curl -X PUT \{"\n"}  --data-binary @photo.jpg \{"\n"}  "{baseUrl}/api/v1/buckets/my-files/photos/photo.jpg"
                                        </pre>
                                    </div>

                                    <div>
                                        <p className="text-sm text-muted-foreground mb-2">
                                            3. List files in bucket
                                        </p>
                                        <pre className="p-3 bg-muted rounded-lg text-xs font-mono">
                                            curl "{baseUrl}/api/v1/buckets/my-files?prefix=photos/"
                                        </pre>
                                    </div>

                                    <div>
                                        <p className="text-sm text-muted-foreground mb-2">
                                            4. Download a file
                                        </p>
                                        <pre className="p-3 bg-muted rounded-lg text-xs font-mono">
                                            curl "{baseUrl}/api/v1/buckets/my-files/photos/photo.jpg" -o photo.jpg
                                        </pre>
                                    </div>
                                </div>
                            </Card>

                            <Card className="p-4 space-y-4">
                                <h3 className="font-mono uppercase text-sm font-bold flex items-center gap-2">
                                    <Code className="w-4 h-4" />
                                    JavaScript Example
                                </h3>
                                <pre className="p-3 bg-muted rounded-lg text-xs font-mono overflow-x-auto whitespace-pre-wrap">
                                    {`// Upload a file
const formData = new FormData();
formData.append('file', fileInput.files[0]);

await fetch('${baseUrl}/api/v1/buckets/my-bucket/uploads/file.txt', {
  method: 'POST',
  body: formData
});

// List objects
const response = await fetch('${baseUrl}/api/v1/buckets/my-bucket');
const data = await response.json();
console.log(data.contents);`}
                                </pre>
                            </Card>
                        </TabsContent>

                        <TabsContent value="auth" className="space-y-4 pr-4">
                            <Card className="p-4 space-y-4">
                                <h3 className="font-mono uppercase text-sm font-bold">
                                    API Key Authentication
                                </h3>
                                <p className="text-sm text-muted-foreground">
                                    When API authentication is enabled (-api-auth flag), you need to include
                                    your credentials in the request headers.
                                </p>

                                <div className="space-y-2">
                                    <span className="text-xs font-mono uppercase text-muted-foreground">
                                        Authorization Header Format
                                    </span>
                                    <pre className="p-3 bg-muted rounded-lg text-xs font-mono">
                                        Authorization: Bearer {"<access_key_id>"}:{"<signature>"}
                                    </pre>
                                </div>

                                <div className="space-y-2">
                                    <span className="text-xs font-mono uppercase text-muted-foreground">
                                        Required Headers
                                    </span>
                                    <pre className="p-3 bg-muted rounded-lg text-xs font-mono">
                                        {`X-Beamdrop-Date: 2026-02-09T12:00:00Z
Authorization: Bearer BDK_xxx:signature_here`}
                                    </pre>
                                </div>

                                <div className="space-y-2">
                                    <span className="text-xs font-mono uppercase text-muted-foreground">
                                        Signature Generation (Python)
                                    </span>
                                    <pre className="p-3 bg-muted rounded-lg text-xs font-mono overflow-x-auto whitespace-pre-wrap">
                                        {`import hmac
import hashlib
import base64

def generate_signature(secret_key, method, path, timestamp):
    message = f"{method}\\n{path}\\n{timestamp}"
    signature = hmac.new(
        secret_key.encode(),
        message.encode(),
        hashlib.sha256
    ).digest()
    return base64.b64encode(signature).decode()`}
                                    </pre>
                                </div>
                            </Card>

                            <Card className="p-4 bg-blue-500/5 border-blue-500/20">
                                <p className="text-sm text-blue-600 dark:text-blue-400">
                                    <strong>Tip:</strong> For testing and development, you can disable API
                                    authentication by not using the -api-auth flag. All API endpoints will
                                    be publicly accessible within your network.
                                </p>
                            </Card>
                        </TabsContent>
                    </ScrollArea>
                </Tabs>
            </DialogContent>
        </Dialog>
    );
}
