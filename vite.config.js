import { writeFileSync } from "node:fs";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

const outDir = "internal/web/dist";

export default defineConfig({
  plugins: [
    react(),
    {
      // emptyOutDir removes the committed placeholder that lets `go build`
      // embed the folder before the first frontend build; put it back.
      name: "keep-go-embed-placeholder",
      closeBundle() {
        writeFileSync(`${outDir}/.gitkeep`, "");
      },
    },
  ],
  build: {
    // Inside the Go package that embeds it: go:embed cannot reach upward.
    outDir,
    emptyOutDir: true,
  },
  server: {
    // Proxy the API to the Go server; keeps everything same-origin
    // (localhost:5173) so the session cookie is first-party and the CSRF
    // check sees Sec-Fetch-Site: same-origin.
    proxy: {
      "/api": "http://localhost:8080",
      "/swagger": "http://localhost:8080",
    },
  },
});
