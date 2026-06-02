import { Beamdrop, BeamdropException } from "beamdrop";

function requiredEnv(key: string): string {
  const value = process.env[key]?.trim();
  if (!value) {
    console.error(`missing required environment variable ${key}`);
    process.exit(1);
  }
  return value;
}

export const config = {
  baseUrl: requiredEnv("BEAMDROP_BASE_URL"),
  accessKey: requiredEnv("BEAMDROP_ACCESS_KEY_ID"),
  secretKey: requiredEnv("BEAMDROP_SECRET_KEY"),
  bucket: process.env["BEAMDROP_BUCKET"]?.trim() || "beamdrop-elysia-example",
  port: parseInt(process.env["PORT"] || "3000", 10),
};

export const beamdrop = new Beamdrop({
  baseUrl: config.baseUrl,
  accessKey: config.accessKey,
  secretKey: config.secretKey,
});

export { BeamdropException };
