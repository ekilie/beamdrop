#!/usr/bin/env node

import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { BeamdropClient } from "./client.js";
import { registerBucketTools } from "./tools/buckets.js";
import { registerObjectTools } from "./tools/objects.js";
import { registerPresignedTools } from "./tools/presigned.js";
import { registerKeyTools } from "./tools/keys.js";

const baseURL = process.env.BEAMDROP_BASE_URL;
const accessKeyId = process.env.BEAMDROP_ACCESS_KEY_ID;
const secretKey = process.env.BEAMDROP_SECRET_KEY;

if (!baseURL) {
  console.error("BEAMDROP_BASE_URL environment variable is required");
  process.exit(1);
}
if (!accessKeyId || !secretKey) {
  console.error("BEAMDROP_ACCESS_KEY_ID and BEAMDROP_SECRET_KEY environment variables are required");
  process.exit(1);
}

const client = new BeamdropClient({ baseURL, accessKeyId, secretKey });

const server = new McpServer({
  name: "beamdrop",
  version: "0.1.0",
});

registerBucketTools(server, client);
registerObjectTools(server, client);
registerPresignedTools(server, client);
registerKeyTools(server, client);

const transport = new StdioServerTransport();
await server.connect(transport);
