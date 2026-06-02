import { Beamdrop, BeamdropException } from "beamdrop";

// ── helpers ──────────────────────────────────────────────────────────

function requiredEnv(key: string): string {
  const value = process.env[key]?.trim();
  if (!value) {
    console.error(`missing required environment variable ${key}`);
    process.exit(1);
  }
  return value;
}

function optionalEnv(key: string, fallback: string): string {
  return process.env[key]?.trim() || fallback;
}

function logStep(name: string) {
  console.log(`\n== ${name} ==`);
}

// ── config ───────────────────────────────────────────────────────────

const config = {
  baseUrl: requiredEnv("BEAMDROP_BASE_URL"),
  accessKey: requiredEnv("BEAMDROP_ACCESS_KEY_ID"),
  secretKey: requiredEnv("BEAMDROP_SECRET_KEY"),
  bucket: optionalEnv("BEAMDROP_BUCKET", "beamdrop-bun-example"),
  cleanup: optionalEnv("BEAMDROP_CLEANUP", "false").toLowerCase() === "true",
};

// ── main ─────────────────────────────────────────────────────────────

const client = new Beamdrop({
  baseUrl: config.baseUrl,
  accessKey: config.accessKey,
  secretKey: config.secretKey,
});

try {
  // 1. check if bucket exists
  logStep("checking bucket");
  const bucketExists = await client.bucketExists(config.bucket);
  console.log(`bucket "${config.bucket}" exists before setup: ${bucketExists}`);

  // 2. ensure bucket exists (idempotent)
  logStep("ensuring bucket exists");
  const bucketResult = await client.createBucketIfNotExists(config.bucket);
  if ("exists" in bucketResult) {
    console.log(`bucket "${bucketResult.bucket}" already existed`);
  } else {
    console.log(`bucket "${bucketResult.bucket}" created at ${bucketResult.created}`);
  }

  // 3. upload objects
  logStep("uploading objects");
  const firstUpload = await client.putObject(
    config.bucket,
    "demo/hello.txt",
    "hello beamdrop from the Bun.js client example\n",
  );
  const secondUpload = await client.putObject(
    config.bucket,
    "demo/notes/streamed.txt",
    "this object was uploaded from Bun.js\n",
  );
  console.log(`uploaded: ${firstUpload.key} (${firstUpload.size} bytes)`);
  console.log(`uploaded: ${secondUpload.key} (${secondUpload.size} bytes)`);

  // 4. list buckets
  logStep("listing buckets");
  const { buckets } = await client.listBuckets();
  for (const bucket of buckets) {
    console.log(`bucket: ${bucket.name} created=${bucket.createdAt}`);
  }

  // 5. list objects with prefix
  logStep("listing objects with prefix");
  const objects = await client.listObjects(config.bucket, "demo/", "/", 100);
  for (const obj of objects.contents) {
    console.log(`object: ${obj.key} size=${obj.size} etag=${obj.etag}`);
  }
  if (objects.commonPrefixes.length > 0) {
    console.log(`common prefixes: ${objects.commonPrefixes.join(", ")}`);
  }

  // 6. read object metadata
  logStep("reading object metadata");
  const metadata = await client.headObject(config.bucket, "demo/hello.txt");
  console.log(
    `metadata: contentType=${metadata.content_type} contentLength=${metadata.content_length} etag=${metadata.etag}`,
  );

  // 7. download object
  logStep("downloading object");
  const downloaded = await client.getObject(config.bucket, "demo/hello.txt");
  console.log(`downloaded body: "${downloaded.body.trim()}"`);

  // 8. check object existence
  logStep("checking object existence");
  const objectExists = await client.objectExists(config.bucket, "demo/notes/streamed.txt");
  console.log(`object exists: ${objectExists}`);

  // 9. generate a client-side presigned URL (no server round-trip)
  logStep("creating client-side presigned URL");
  const clientPresignedUrl = await client.presignedUrl(
    config.bucket,
    "demo/hello.txt",
    15 * 60, // 15 minutes
  );
  console.log(`client-side presigned URL: ${clientPresignedUrl}`);

  // 10. create a server-side pretty presigned URL
  logStep("creating server-side presigned URL");
  const serverPresignedUrl = await client.createPrettyPresignedUrl(
    config.bucket,
    "demo/hello.txt",
    900, // 15 minutes
  );
  console.log(`server-side presigned URL: ${serverPresignedUrl.url}`);

  // 11. list server-side presigned URLs
  logStep("listing server-side presigned URLs");
  const { urls } = await client.listPrettyPresignedUrls();
  for (const url of urls) {
    console.log(`presigned token=${url.token} method=${url.method} key=${url.key}`);
  }

  // 12. cleanup (optional)
  if (config.cleanup) {
    logStep("cleanup");
    await client.revokePrettyPresignedUrl(serverPresignedUrl.token);
    await client.deleteObject(config.bucket, "demo/notes/streamed.txt");
    await client.deleteObject(config.bucket, "demo/hello.txt");
    console.log("deleted demo objects and presigned URL");
  }

  console.log("\nexample completed successfully");
} catch (err) {
  if (err instanceof BeamdropException) {
    console.error(`beamdrop error (${err.status}): ${err.message}`);
    if (err.body) console.error("details:", err.body);
  } else {
    throw err;
  }
  process.exit(1);
}