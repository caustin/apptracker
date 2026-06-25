// Minimal structural type for the R2 binding we use, to avoid pulling the full
// @cloudflare/workers-types globals into this DOM-typed repo. The real R2Bucket
// satisfies this at runtime; `wrangler types` can generate richer types later.
export interface R2ObjectBody {
  arrayBuffer(): Promise<ArrayBuffer>;
}
export interface R2Bucket {
  put(
    key: string,
    value: ArrayBuffer | Uint8Array | ReadableStream | string,
  ): Promise<unknown>;
  get(key: string): Promise<R2ObjectBody | null>;
  delete(key: string): Promise<void>;
}

// Static-assets fetcher binding (serves the built React app + SPA fallback).
export interface Fetcher {
  fetch(request: Request): Promise<Response>;
}

// Worker bindings (wrangler.jsonc vars/secrets + R2 bucket + assets).
export interface Env {
  DATABASE_URL: string;
  BETTER_AUTH_SECRET: string;
  BETTER_AUTH_URL: string;
  GOOGLE_CLIENT_ID?: string;
  GOOGLE_CLIENT_SECRET?: string;
  RESUMES: R2Bucket;
  ASSETS: Fetcher;
}
