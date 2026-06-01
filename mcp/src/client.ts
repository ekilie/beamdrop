import { createHmac } from "node:crypto";

export interface BeamdropConfig {
  baseURL: string;
  accessKeyId: string;
  secretKey: string;
}

export interface APIError {
  statusCode: number;
  error: string;
  category: string;
  message: string;
  retryable: boolean;
  retryAfter: number;
}

export class BeamdropAPIError extends Error {
  constructor(
    public statusCode: number,
    public code: string,
    public category: string,
    public retryable: boolean,
    public retryAfter: number,
  ) {
    super(`Beamdrop API error ${statusCode}: ${code}`);
    this.name = "BeamdropAPIError";
  }
}

export class BeamdropClient {
  private baseURL: string;
  private accessKeyId: string;
  private secretKey: string;

  constructor(config: BeamdropConfig) {
    this.baseURL = config.baseURL.replace(/\/+$/, "");
    this.accessKeyId = config.accessKeyId;
    this.secretKey = config.secretKey;
  }

  private sign(method: string, path: string): { authorization: string; timestamp: string } {
    const timestamp = new Date().toISOString().replace(/\.\d{3}Z$/, "Z");
    const stringToSign = `${method}\n${path}\n${timestamp}`;
    const signature = createHmac("sha256", this.secretKey)
      .update(stringToSign)
      .digest("base64");
    return {
      authorization: `Bearer ${this.accessKeyId}:${signature}`,
      timestamp,
    };
  }

  async request(method: string, path: string, body?: string | Buffer): Promise<Response> {
    const { authorization, timestamp } = this.sign(method, path);
    const url = `${this.baseURL}${path}`;

    const headers: Record<string, string> = {
      Authorization: authorization,
      "X-Beamdrop-Date": timestamp,
    };

    if (body && typeof body === "string") {
      headers["Content-Type"] = "application/octet-stream";
    }

    const response = await fetch(url, {
      method,
      headers,
      body: body ?? undefined,
    });

    if (!response.ok) {
      let errorBody: APIError | undefined;
      try {
        errorBody = (await response.json()) as APIError;
      } catch {
        // response wasn't JSON
      }
      throw new BeamdropAPIError(
        response.status,
        errorBody?.error ?? `HTTP_${response.status}`,
        errorBody?.category ?? "UNKNOWN",
        errorBody?.retryable ?? false,
        errorBody?.retryAfter ?? 0,
      );
    }

    return response;
  }

  async requestJSON<T>(method: string, path: string, body?: object): Promise<T> {
    const { authorization, timestamp } = this.sign(method, path);
    const url = `${this.baseURL}${path}`;

    const headers: Record<string, string> = {
      Authorization: authorization,
      "X-Beamdrop-Date": timestamp,
    };

    let bodyStr: string | undefined;
    if (body) {
      headers["Content-Type"] = "application/json";
      bodyStr = JSON.stringify(body);
    }

    const response = await fetch(url, {
      method,
      headers,
      body: bodyStr,
    });

    if (!response.ok) {
      let errorBody: APIError | undefined;
      try {
        errorBody = (await response.json()) as APIError;
      } catch {
        // not JSON
      }
      throw new BeamdropAPIError(
        response.status,
        errorBody?.error ?? `HTTP_${response.status}`,
        errorBody?.category ?? "UNKNOWN",
        errorBody?.retryable ?? false,
        errorBody?.retryAfter ?? 0,
      );
    }

    // Handle 204 No Content
    if (response.status === 204) {
      return undefined as T;
    }

    return (await response.json()) as T;
  }

  async requestNoContent(method: string, path: string): Promise<void> {
    const { authorization, timestamp } = this.sign(method, path);
    const url = `${this.baseURL}${path}`;

    const response = await fetch(url, {
      method,
      headers: {
        Authorization: authorization,
        "X-Beamdrop-Date": timestamp,
      },
    });

    if (!response.ok) {
      let errorBody: APIError | undefined;
      try {
        errorBody = (await response.json()) as APIError;
      } catch {
        // not JSON
      }
      throw new BeamdropAPIError(
        response.status,
        errorBody?.error ?? `HTTP_${response.status}`,
        errorBody?.category ?? "UNKNOWN",
        errorBody?.retryable ?? false,
        errorBody?.retryAfter ?? 0,
      );
    }
  }

  async headRequest(method: string, path: string): Promise<Headers> {
    const { authorization, timestamp } = this.sign(method, path);
    const url = `${this.baseURL}${path}`;

    const response = await fetch(url, {
      method,
      headers: {
        Authorization: authorization,
        "X-Beamdrop-Date": timestamp,
      },
    });

    if (!response.ok) {
      throw new BeamdropAPIError(
        response.status,
        `HTTP_${response.status}`,
        response.status === 404 ? "NOT_FOUND" : "UNKNOWN",
        false,
        0,
      );
    }

    return response.headers;
  }
}
