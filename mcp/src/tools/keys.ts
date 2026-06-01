import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { BeamdropClient, BeamdropAPIError } from "../client.js";

export function registerKeyTools(server: McpServer, client: BeamdropClient) {
  server.tool(
    "list_api_keys",
    "List all API keys (secrets are not included in the response)",
    {},
    async () => {
      try {
        const result = await client.requestJSON<{
          keys: Array<{
            id: number;
            name: string;
            access_key_id: string;
            bucket_scope: string;
            permissions: string[];
            created_at: string;
          }>;
          count: number;
        }>("GET", "/api/v1/keys");
        return { content: [{ type: "text" as const, text: JSON.stringify(result, null, 2) }] };
      } catch (e) {
        return errorResult(e);
      }
    },
  );

  server.tool(
    "create_api_key",
    "Create a new API key. The secret key is shown only once in the response — save it immediately.",
    {
      name: z.string().optional().describe("Human-readable name for the key"),
      bucketScope: z.string().optional().describe("Restrict key to a specific bucket (omit for all buckets)"),
      permissions: z
        .array(z.string())
        .optional()
        .describe('Allowed actions: "GetObject", "PutObject", "DeleteObject", "ListBucket", or "*" for all'),
    },
    async ({ name, bucketScope, permissions }) => {
      try {
        const body: Record<string, unknown> = {};
        if (name !== undefined) body.name = name;
        if (bucketScope !== undefined) body.bucket_scope = bucketScope;
        if (permissions !== undefined) body.permissions = permissions;

        const result = await client.requestJSON<{
          id: number;
          name: string;
          access_key_id: string;
          secret_key: string;
          bucket_scope: string;
          permissions: string[];
          created_at: string;
        }>("POST", "/api/v1/keys", body);
        return {
          content: [{
            type: "text" as const,
            text: `API key created successfully.\n\n⚠️ Save the secret key — it won't be shown again!\n\n${JSON.stringify(result, null, 2)}`,
          }],
        };
      } catch (e) {
        return errorResult(e);
      }
    },
  );

  server.tool(
    "delete_api_key",
    "Delete an API key permanently. Any requests using this key will immediately fail.",
    {
      id: z.number().describe("API key ID to delete"),
    },
    async ({ id }) => {
      try {
        await client.requestNoContent("DELETE", `/api/v1/keys/${id}`);
        return { content: [{ type: "text" as const, text: `API key ${id} deleted successfully.` }] };
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
