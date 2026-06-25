import type { R2Bucket } from "./env";

// Storage backend for resume PDFs. Routes depend only on this interface, so a
// future "bring your own cloud" backend (e.g. Google Drive via drive.file scope)
// can be slotted in without touching route code.
export interface Storage {
  put(userId: string, bytes: ArrayBuffer | Uint8Array): Promise<string>;
  get(key: string): Promise<Uint8Array | null>;
  delete(key: string): Promise<void>;
}

export function r2Storage(bucket: R2Bucket): Storage {
  return {
    async put(userId, bytes) {
      const key = `${userId}/${crypto.randomUUID()}.pdf`;
      await bucket.put(key, bytes);
      return key;
    },
    async get(key) {
      const obj = await bucket.get(key);
      if (!obj) return null;
      return new Uint8Array(await obj.arrayBuffer());
    },
    async delete(key) {
      await bucket.delete(key);
    },
  };
}
