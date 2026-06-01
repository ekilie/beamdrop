import { useState, useEffect } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Copy, Check, ExternalLink, Cpu, Wrench, Shield, Zap } from "lucide-react";

interface MCPInfo {
  protocol: string;
  transport: string;
  version: string;
  name: string;
  description: string;
  methods: string[];
}

export default function McpPage() {
  const [info, setInfo] = useState<MCPInfo | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [copiedField, setCopiedField] = useState<string | null>(null);

  const mcpUrl = `${window.location.origin}/mcp`;

  useEffect(() => {
    fetch("/mcp")
      .then((res) => {
        if (!res.ok) throw new Error(`Failed: ${res.status}`);
        return res.json();
      })
      .then(setInfo)
      .catch((e) => setError(e.message));
  }, []);

  const copyToClipboard = (text: string, field: string) => {
    navigator.clipboard.writeText(text);
    setCopiedField(field);
    setTimeout(() => setCopiedField(null), 2000);
  };

  const CopyButton = ({ text, field }: { text: string; field: string }) => (
    <Button
      variant="ghost"
      size="sm"
      onClick={() => copyToClipboard(text, field)}
      className="h-7 w-7 p-0"
    >
      {copiedField === field ? (
        <Check className="h-3.5 w-3.5 text-green-500" />
      ) : (
        <Copy className="h-3.5 w-3.5" />
      )}
    </Button>
  );

  const claudeConfig = JSON.stringify(
    {
      mcpServers: {
        beamdrop: {
          url: mcpUrl,
          headers: {
            Authorization: "Bearer BDK_your_key:signature",
            "X-Beamdrop-Date": "ISO-8601-timestamp",
          },
        },
      },
    },
    null,
    2
  );

  const tools = [
    { name: "list_buckets", desc: "List all storage buckets" },
    { name: "create_bucket", desc: "Create a new bucket" },
    { name: "delete_bucket", desc: "Delete an empty bucket" },
    { name: "bucket_exists", desc: "Check if a bucket exists" },
    { name: "list_objects", desc: "List objects with prefix/delimiter" },
    { name: "put_object", desc: "Upload content to a bucket" },
    { name: "get_object", desc: "Download an object" },
    { name: "head_object", desc: "Get object metadata" },
    { name: "delete_object", desc: "Delete an object" },
    { name: "create_presigned_url", desc: "Create a shareable download URL" },
    { name: "list_presigned_urls", desc: "List all presigned URLs" },
    { name: "get_presigned_url", desc: "Get presigned URL details" },
    { name: "delete_presigned_url", desc: "Revoke a presigned URL" },
    { name: "list_api_keys", desc: "List API keys (no secrets)" },
    { name: "create_api_key", desc: "Create a new API key" },
    { name: "delete_api_key", desc: "Delete an API key" },
  ];

  return (
    <div className="flex flex-col gap-6 p-6 max-w-5xl">
      <div>
        <h1 className="text-2xl font-bold font-mono uppercase tracking-tight">
          MCP Server
        </h1>
        <p className="text-sm text-muted-foreground mt-1">
          Model Context Protocol — connect AI agents directly to this Beamdrop
          instance
        </p>
      </div>

      {/* Status */}
      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <CardTitle className="text-sm font-mono uppercase tracking-wide flex items-center gap-2">
              <Cpu className="h-4 w-4" />
              Endpoint Status
            </CardTitle>
            {info && (
              <Badge
                variant="secondary"
                className="bg-green-500/10 text-green-500 border-green-500/30"
              >
                ACTIVE
              </Badge>
            )}
            {error && (
              <Badge variant="destructive">
                ERROR
              </Badge>
            )}
          </div>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="flex items-center justify-between p-3 rounded-lg bg-muted/50">
            <div>
              <p className="text-xs text-muted-foreground font-mono">
                ENDPOINT URL
              </p>
              <p className="text-sm font-mono mt-0.5">{mcpUrl}</p>
            </div>
            <CopyButton text={mcpUrl} field="url" />
          </div>

          {info && (
            <div className="grid grid-cols-3 gap-3">
              <div className="p-3 rounded-lg bg-muted/50">
                <p className="text-xs text-muted-foreground font-mono">
                  PROTOCOL
                </p>
                <p className="text-sm font-mono mt-0.5">
                  MCP {info.version}
                </p>
              </div>
              <div className="p-3 rounded-lg bg-muted/50">
                <p className="text-xs text-muted-foreground font-mono">
                  TRANSPORT
                </p>
                <p className="text-sm font-mono mt-0.5">
                  {info.transport}
                </p>
              </div>
              <div className="p-3 rounded-lg bg-muted/50">
                <p className="text-xs text-muted-foreground font-mono">
                  TOOLS
                </p>
                <p className="text-sm font-mono mt-0.5">
                  {tools.length} available
                </p>
              </div>
            </div>
          )}

          {error && (
            <p className="text-sm text-red-500 font-mono">Error: {error}</p>
          )}
        </CardContent>
      </Card>

      {/* Security */}
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-sm font-mono uppercase tracking-wide flex items-center gap-2">
            <Shield className="h-4 w-4" />
            Authentication
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-2 text-sm">
            <p className="text-muted-foreground">
              POST requests require <strong>HMAC-SHA256 API key authentication</strong> — the same auth used by the S3 API.
              GET requests are public (returns server info only).
            </p>
            <div className="flex items-center gap-2 mt-3">
              <Badge variant="outline" className="font-mono text-xs">
                Authorization: Bearer BDK_...:signature
              </Badge>
              <Badge variant="outline" className="font-mono text-xs">
                X-Beamdrop-Date: ISO-8601
              </Badge>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Configuration */}
      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <CardTitle className="text-sm font-mono uppercase tracking-wide flex items-center gap-2">
              <Zap className="h-4 w-4" />
              Claude / Copilot Configuration
            </CardTitle>
            <CopyButton text={claudeConfig} field="config" />
          </div>
        </CardHeader>
        <CardContent>
          <pre className="p-4 rounded-lg bg-muted/50 text-xs font-mono overflow-x-auto whitespace-pre">
            {claudeConfig}
          </pre>
          <p className="text-xs text-muted-foreground mt-3">
            Add this to your <code className="text-foreground">claude_desktop_config.json</code> or MCP client settings.
            Replace the Authorization header with your actual API key and HMAC signature.
          </p>
        </CardContent>
      </Card>

      {/* Tools */}
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-sm font-mono uppercase tracking-wide flex items-center gap-2">
            <Wrench className="h-4 w-4" />
            Available Tools ({tools.length})
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
            {tools.map((tool) => (
              <div
                key={tool.name}
                className="flex items-start gap-3 p-3 rounded-lg bg-muted/30 hover:bg-muted/50 transition-colors"
              >
                <div className="min-w-0">
                  <p className="text-sm font-mono font-medium truncate">
                    {tool.name}
                  </p>
                  <p className="text-xs text-muted-foreground mt-0.5">
                    {tool.desc}
                  </p>
                </div>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* Links */}
      <div className="flex flex-wrap gap-3">
        <a
          href="/llms.txt"
          className="inline-flex items-center gap-1.5 text-sm font-mono text-primary hover:underline"
        >
          <ExternalLink className="h-3.5 w-3.5" />
          llms.txt
        </a>
        <a
          href="/api-docs"
          className="inline-flex items-center gap-1.5 text-sm font-mono text-muted-foreground hover:text-foreground"
        >
          <ExternalLink className="h-3.5 w-3.5" />
          API Docs
        </a>
      </div>
    </div>
  );
}
