import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { BeamdropClient, BeamdropAPIError } from "../client.js";

export function registerBucketTools(server: McpServer, client: BeamdropClient) {
  server.tool(
    "list_buckets",
    "List all storage buckets on the Beamdrop server",
    {},
    async () => {
      try {
        const result = await client.requestJSON<{
          buckets: Array<{ name: string; created_at: string }>;
          count: number;
        }>("GET", "/api/v1/buckets");
        return { content: [{ type: "text" as const, text: JSON.stringify(result, null, 2) }] };
      } catch (e) {
        return errorResult(e);
      }
    },
  );

  server.tool(
    "create_bucket",
    "Create a new storage bucket. Use idempotent=true to avoid errors if it already exists.",
    {
      name: z.string().describe("Bucket name (3-63 chars, lowercase alphanumeric + hyphens/dots)"),
      idempotent: z.boolean().optional().default(true).describe("If true, don't error if bucket already exists"),
    },
    async ({ name, idempotent }) => {
      try {
        const query = idempotent ? "?createIfNotExists=true" : "";
        const result = await client.requestJSON<{
          bucket: string;
          created: boolean;
          exists?: boolean;
          location: string;
        }>("PUT", `/api/v1/buckets/${encodeURIComponent(name)}${query}`);
        return { content: [{ type: "text" as const, text: JSON.stringify(result, null, 2) }] };
      } catch (e) {
        return errorResult(e);
      }
    },
  );

  server.tool(
    "delete_bucket",
    "Delete an empty storage bucket. All objects must be deleted first.",
    {
      name: z.string().describe("Bucket name to delete"),
    },
    async ({ name }) => {
      try {
        await client.requestNoContent("DELETE", `/api/v1/buckets/${encodeURIComponent(name)}`);
        return { content: [{ type: "text" as const, text: `Bucket "${name}" deleted successfully.` }] };
      } catch (e) {
        return errorResult(e);
      }
    },
  );

  server.tool(
    "bucket_exists",
    "Check whether a storage bucket exists",
    {
      name: z.string().describe("Bucket name to check"),
    },
    async ({ name }) => {
      try {
        await client.headRequest("HEAD", `/api/v1/buckets/${encodeURIComponent(name)}`);
        return { content: [{ type: "text" as const, text: `Bucket "${name}" exists.` }] };
      } catch (e) {
        if (e instanceof BeamdropAPIError && e.statusCode === 404) {
          return { content: [{ type: "text" as const, text: `Bucket "${name}" does not exist.` }] };
        }
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
