import { Elysia, t } from "elysia";
import { beamdrop, BeamdropException, config } from "./beamdrop";

// ensure the default bucket exists before starting
await beamdrop.createBucketIfNotExists(config.bucket);
console.log(`bucket "${config.bucket}" ready`);

const app = new Elysia()

  // ── Error handler ──────────────────────────────────────────────────
  .onError(({ error, set }) => {
    if (error instanceof BeamdropException) {
      set.status = error.status || 502;
      return { error: error.message, ...(error.body && { details: error.body }) };
    }
    console.error(error);
    set.status = 500;
    return { error: "internal server error" };
  })

  // ── Buckets ────────────────────────────────────────────────────────
  .group("/buckets", (app) =>
    app
      // List all buckets
      .get("/", () => beamdrop.listBuckets())

      // Create a bucket (idempotent)
      .post(
        "/:name",
        async ({ params, set }) => {
          const result = await beamdrop.createBucketIfNotExists(params.name);
          set.status = "exists" in result ? 200 : 201;
          return result;
        },
        { params: t.Object({ name: t.String() }) },
      )

      // Delete a bucket
      .delete(
        "/:name",
        async ({ params, set }) => {
          await beamdrop.deleteBucket(params.name);
          set.status = 204;
        },
        { params: t.Object({ name: t.String() }) },
      ),
  )

  // ── Files ──────────────────────────────────────────────────────────
  .group("/files", (app) =>
    app
      // List files in the default bucket
      .get("/", async ({ query }) => {
        const result = await beamdrop.listObjects(
          config.bucket,
          query.prefix || "",
          "/",
        );
        return {
          files: result.contents ?? [],
          folders: result.commonPrefixes ?? [],
          isTruncated: result.isTruncated,
        };
      }, {
        query: t.Object({ prefix: t.Optional(t.String()) }),
      })

      // Upload a file
      .post(
        "/*",
        async ({ params, body, set }) => {
          const key = params["*"];
          if (!key) {
            set.status = 400;
            return { error: "missing key in path" };
          }

          let content: BodyInit;
          let fileName = key;

          if (body.file instanceof File) {
            content = await body.file.arrayBuffer();
            fileName = key || body.file.name;
          } else {
            set.status = 400;
            return { error: "no file provided" };
          }

          const result = await beamdrop.putObject(config.bucket, fileName, content);
          set.status = 201;
          return result;
        },
        { body: t.Object({ file: t.File() }) },
      )

      // Download a file
      .get("/*", async ({ params, set }) => {
        const key = params["*"];
        if (!key) {
          set.status = 400;
          return { error: "missing key" };
        }

        const meta = await beamdrop.headObject(config.bucket, key);
        const obj = await beamdrop.getObject(config.bucket, key);

        set.headers["content-type"] = meta.content_type;
        set.headers["etag"] = meta.etag;
        return obj.body;
      })

      // Delete a file
      .delete("/*", async ({ params, set }) => {
        const key = params["*"];
        if (!key) {
          set.status = 400;
          return { error: "missing key" };
        }
        await beamdrop.deleteObject(config.bucket, key);
        set.status = 204;
      }),
  )

  // ── Presigned URLs ─────────────────────────────────────────────────
  .group("/presign", (app) =>
    app
      // Generate a client-side presigned URL
      .post(
        "/",
        async ({ body, set }) => {
          const url = await beamdrop.presignedUrl(
            config.bucket,
            body.key,
            body.expiresIn,
          );
          return { url, expiresIn: body.expiresIn };
        },
        {
          body: t.Object({
            key: t.String(),
            expiresIn: t.Number({ default: 3600 }),
          }),
        },
      )

      // Create a server-side pretty presigned URL
      .post(
        "/pretty",
        async ({ body, set }) => {
          const result = await beamdrop.createPrettyPresignedUrl(
            config.bucket,
            body.key,
            body.expiresIn ?? null,
            body.maxDownloads ?? null,
          );
          set.status = 201;
          return result;
        },
        {
          body: t.Object({
            key: t.String(),
            expiresIn: t.Optional(t.Number()),
            maxDownloads: t.Optional(t.Number()),
          }),
        },
      )

      // List all pretty presigned URLs
      .get("/pretty", () => beamdrop.listPrettyPresignedUrls())

      // Revoke a pretty presigned URL
      .delete(
        "/pretty/:token",
        async ({ params, set }) => {
          await beamdrop.revokePrettyPresignedUrl(params.token);
          set.status = 204;
        },
        { params: t.Object({ token: t.String() }) },
      ),
  )

  .listen(config.port);

console.log(`elysia server listening on http://localhost:${app.server?.port}`);
console.log(`beamdrop backend: ${config.baseUrl}`);
