import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { BeamdropClient, BeamdropAPIError } from "../client.js";

export function registerPresignedTools(server: McpServer, client: BeamdropClient) {
  server.tool(
    "create_presigned_url",
    "Create a server-side presigned URL for downloading a file. The URL is shareable and requires no authentication. Supports optional expiry and download limits.",
    {
      bucket: z.string().describe("Bucket name"),
      key: z.string().describe("Object key/path"),
      method: z.string().optional().default("GET").describe("HTTP method (GET for download)"),
      expiresIn: z.number().optional().describe("Expiry in seconds (e.g., 3600 for 1 hour). Omit for no expiry."),
      maxDownloads: z.number().optional().describe("Maximum number of downloads allowed. Omit for unlimited."),
    },
    async ({ bucket, key, method, expiresIn, maxDownloads }) => {
      try {
        const body: Record<string, unknown> = { bucket, key, method };
        if (expiresIn !== undefined) body.expires_in = expiresIn;
        if (maxDownloads !== undefined) body.max_downloads = maxDownloads;

        const result = await client.requestJSON<{
          id: number;
          token: string;
          url: string;
          bucket: string;
          key: string;
          method: string;
          expires_at: string;
          max_downloads: number | null;
          download_count: number;
          created_at: string;
        }>("POST", "/api/v1/presign", body);
        return { content: [{ type: "text" as const, text: JSON.stringify(result, null, 2) }] };
      } catch (e) {
        return errorResult(e);
      }
    },
  );

  server.tool(
    "list_presigned_urls",
    "List all server-side presigned URLs",
    {},
    async () => {
      try {
        const result = await client.requestJSON<{
          urls: Array<{
            id: number;
            token: string;
            url: string;
            bucket: string;
            key: string;
            method: string;
            expires_at: string;
            max_downloads: number | null;
            download_count: number;
            created_at: string;
          }>;
          count: number;
        }>("GET", "/api/v1/presign");
        return { content: [{ type: "text" as const, text: JSON.stringify(result, null, 2) }] };
      } catch (e) {
        return errorResult(e);
      }
    },
  );

  server.tool(
    "get_presigned_url",
    "Get details about a specific presigned URL by token",
    {
      token: z.string().describe("Presigned URL token"),
    },
    async ({ token }) => {
      try {
        const result = await client.requestJSON<{
          id: number;
          token: string;
          url: string;
          bucket: string;
          key: string;
          method: string;
          expires_at: string;
          max_downloads: number | null;
          download_count: number;
          created_at: string;
        }>("GET", `/api/v1/presign/${encodeURIComponent(token)}`);
        return { content: [{ type: "text" as const, text: JSON.stringify(result, null, 2) }] };
      } catch (e) {
        return errorResult(e);
      }
    },
  );

  server.tool(
    "delete_presigned_url",
    "Revoke a presigned URL. The URL will immediately return 404.",
    {
      token: z.string().describe("Presigned URL token to revoke"),
    },
    async ({ token }) => {
      try {
        await client.requestNoContent("DELETE", `/api/v1/presign/${encodeURIComponent(token)}`);
        return { content: [{ type: "text" as const, text: `Presigned URL "${token}" revoked successfully.` }] };
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
