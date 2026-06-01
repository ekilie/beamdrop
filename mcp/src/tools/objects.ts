import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { BeamdropClient, BeamdropAPIError } from "../client.js";

export function registerObjectTools(server: McpServer, client: BeamdropClient) {
  server.tool(
    "list_objects",
    "List objects in a bucket with optional prefix and delimiter filtering. Use delimiter='/' for directory-like listing.",
    {
      bucket: z.string().describe("Bucket name"),
      prefix: z.string().optional().default("").describe("Filter objects by key prefix (e.g., 'folder/')"),
      delimiter: z.string().optional().default("").describe("Group objects by delimiter (use '/' for directory listing)"),
      maxKeys: z.number().optional().default(1000).describe("Maximum number of results to return"),
    },
    async ({ bucket, prefix, delimiter, maxKeys }) => {
      try {
        const params = new URLSearchParams({ list: "true" });
        if (prefix) params.set("prefix", prefix);
        if (delimiter) params.set("delimiter", delimiter);
        if (maxKeys !== 1000) params.set("max-keys", String(maxKeys));

        const result = await client.requestJSON<{
          bucket: string;
          prefix: string;
          delimiter: string;
          max_keys: number;
          is_truncated: boolean;
          contents: Array<{
            key: string;
            size: number;
            last_modified: string;
            etag: string;
            content_type: string;
          }>;
          common_prefixes: string[];
        }>("GET", `/api/v1/buckets/${encodeURIComponent(bucket)}?${params.toString()}`);
        return { content: [{ type: "text" as const, text: JSON.stringify(result, null, 2) }] };
      } catch (e) {
        return errorResult(e);
      }
    },
  );

  server.tool(
    "put_object",
    "Upload content to a bucket as an object. For text content, provide it directly. For binary, provide base64-encoded content.",
    {
      bucket: z.string().describe("Bucket name"),
      key: z.string().describe("Object key/path (e.g., 'folder/file.txt')"),
      content: z.string().describe("File content (text or base64-encoded binary)"),
      isBase64: z.boolean().optional().default(false).describe("Set to true if content is base64-encoded binary data"),
    },
    async ({ bucket, key, content, isBase64 }) => {
      try {
        const body = isBase64 ? Buffer.from(content, "base64") : content;
        const path = `/api/v1/buckets/${encodeURIComponent(bucket)}/${key}`;
        const response = await client.request("PUT", path, typeof body === "string" ? body : body);
        const result = await response.json();
        return { content: [{ type: "text" as const, text: JSON.stringify(result, null, 2) }] };
      } catch (e) {
        return errorResult(e);
      }
    },
  );

  server.tool(
    "get_object",
    "Download an object from a bucket. Returns text content directly, or base64-encoded content for binary files.",
    {
      bucket: z.string().describe("Bucket name"),
      key: z.string().describe("Object key/path"),
    },
    async ({ bucket, key }) => {
      try {
        const path = `/api/v1/buckets/${encodeURIComponent(bucket)}/${key}`;
        const response = await client.request("GET", path);
        const contentType = response.headers.get("content-type") ?? "application/octet-stream";
        const isText = contentType.startsWith("text/") || contentType.includes("json") || contentType.includes("xml");

        if (isText) {
          const text = await response.text();
          return {
            content: [{ type: "text" as const, text }],
          };
        }

        const buffer = Buffer.from(await response.arrayBuffer());
        return {
          content: [{
            type: "text" as const,
            text: `[Binary file: ${contentType}, ${buffer.length} bytes]\nBase64: ${buffer.toString("base64")}`,
          }],
        };
      } catch (e) {
        return errorResult(e);
      }
    },
  );

  server.tool(
    "head_object",
    "Get metadata about an object without downloading the body. Returns content type, size, ETag, and last modified.",
    {
      bucket: z.string().describe("Bucket name"),
      key: z.string().describe("Object key/path"),
    },
    async ({ bucket, key }) => {
      try {
        const path = `/api/v1/buckets/${encodeURIComponent(bucket)}/${key}`;
        const headers = await client.headRequest("HEAD", path);
        const metadata = {
          contentType: headers.get("content-type"),
          contentLength: headers.get("content-length"),
          etag: headers.get("etag"),
          lastModified: headers.get("last-modified"),
        };
        return { content: [{ type: "text" as const, text: JSON.stringify(metadata, null, 2) }] };
      } catch (e) {
        return errorResult(e);
      }
    },
  );

  server.tool(
    "delete_object",
    "Delete an object from a bucket",
    {
      bucket: z.string().describe("Bucket name"),
      key: z.string().describe("Object key/path to delete"),
    },
    async ({ bucket, key }) => {
      try {
        const path = `/api/v1/buckets/${encodeURIComponent(bucket)}/${key}`;
        await client.requestNoContent("DELETE", path);
        return { content: [{ type: "text" as const, text: `Object "${key}" deleted from bucket "${bucket}".` }] };
      } catch (e) {
        return errorResult(e);
      }
    },
  );
}

function errorResult(e: unknown) {
  const msg = e instanceof BeamdropAPIError
    ? `Error ${e.statusCode} (${e.code}): ${e.message}${e.retryable ? ` [retryable in ${e.retryAfter}s]` : ""}`
    : `Error: ${e instanceof Error ? e.message : String(e)}`;
  return { content: [{ type: "text" as const, text: msg }], isError: true };
}
