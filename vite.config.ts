import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    // Proxy API + auth calls to `wrangler dev`; keeps everything same-origin
    // (localhost:5173) so Better Auth cookies are first-party.
    proxy: { "/api": "http://localhost:8787" },
  },
});
