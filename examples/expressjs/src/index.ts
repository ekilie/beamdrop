import express from "express";
import multer from "multer";
import { beamdrop, BeamdropException, config } from "./beamdrop.js";

const app = express();
const upload = multer({ storage: multer.memoryStorage(), limits: { fileSize: 50 * 1024 * 1024 } });

app.use(express.json());

// ── Buckets ──────────────────────────────────────────────────────────

// List all buckets
app.get("/buckets", async (_req, res) => {
  const result = await beamdrop.listBuckets();
  res.json(result);
});

// Create a bucket
app.post("/buckets/:name", async (req, res) => {
  const result = await beamdrop.createBucketIfNotExists(req.params.name);
  res.status("exists" in result ? 200 : 201).json(result);
});

// Delete a bucket
app.delete("/buckets/:name", async (req, res) => {
  await beamdrop.deleteBucket(req.params.name);
  res.status(204).end();
});

// ── Objects ──────────────────────────────────────────────────────────

// List objects in the default bucket
app.get("/files", async (req, res) => {
  const prefix = (req.query["prefix"] as string) || "";
  const result = await beamdrop.listObjects(config.bucket, prefix, "/");
  res.json({
    files: result.contents ?? [],
    folders: result.commonPrefixes ?? [],
    isTruncated: result.isTruncated,
  });
});

// Upload a file to the default bucket
app.post("/files/*key", upload.single("file"), async (req, res) => {
  if (!req.file) {
    res.status(400).json({ error: "no file provided" });
    return;
  }
  const key = req.params["key"] || req.file.originalname;
  const result = await beamdrop.putObject(config.bucket, key, req.file.buffer);
  res.status(201).json(result);
});

// Download a file from the default bucket
app.get("/files/*key", async (req, res) => {
  const key = req.params["key"];
  if (!key) {
    res.status(400).json({ error: "missing key" });
    return;
  }

  const meta = await beamdrop.headObject(config.bucket, key);
  const obj = await beamdrop.getObject(config.bucket, key);

  res.set("Content-Type", meta.content_type);
  res.set("ETag", meta.etag);
  res.send(obj.body);
});

// Delete a file from the default bucket
app.delete("/files/*key", async (req, res) => {
  const key = req.params["key"];
  if (!key) {
    res.status(400).json({ error: "missing key" });
    return;
  }
  await beamdrop.deleteObject(config.bucket, key);
  res.status(204).end();
});

// ── Presigned URLs ───────────────────────────────────────────────────

// Generate a client-side presigned download URL
app.post("/presign", async (req, res) => {
  const { key, expiresIn = 3600 } = req.body;
  if (!key) {
    res.status(400).json({ error: "missing key" });
    return;
  }
  const url = await beamdrop.presignedUrl(config.bucket, key, expiresIn);
  res.json({ url, expiresIn });
});

// Create a server-side pretty presigned URL
app.post("/presign/pretty", async (req, res) => {
  const { key, expiresIn, maxDownloads } = req.body;
  if (!key) {
    res.status(400).json({ error: "missing key" });
    return;
  }
  const result = await beamdrop.createPrettyPresignedUrl(
    config.bucket,
    key,
    expiresIn ?? null,
    maxDownloads ?? null,
  );
  res.status(201).json(result);
});

// List all server-side presigned URLs
app.get("/presign/pretty", async (_req, res) => {
  const result = await beamdrop.listPrettyPresignedUrls();
  res.json(result);
});

// Revoke a server-side presigned URL
app.delete("/presign/pretty/:token", async (req, res) => {
  await beamdrop.revokePrettyPresignedUrl(req.params.token);
  res.status(204).end();
});

// ── Error handler ────────────────────────────────────────────────────

app.use((err: unknown, _req: express.Request, res: express.Response, _next: express.NextFunction) => {
  if (err instanceof BeamdropException) {
    res.status(err.status || 502).json({
      error: err.message,
      ...(err.body && { details: err.body }),
    });
    return;
  }
  console.error(err);
  res.status(500).json({ error: "internal server error" });
});

// ── Start ────────────────────────────────────────────────────────────

async function main() {
  // ensure the default bucket exists on startup
  await beamdrop.createBucketIfNotExists(config.bucket);
  console.log(`bucket "${config.bucket}" ready`);

  app.listen(config.port, () => {
    console.log(`express server listening on http://localhost:${config.port}`);
    console.log(`beamdrop backend: ${config.baseUrl}`);
  });
}

main().catch((err) => {
  console.error("failed to start:", err);
  process.exit(1);
});
