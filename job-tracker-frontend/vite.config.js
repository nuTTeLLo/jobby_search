import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tidewave from "tidewave/vite-plugin";

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tidewave()],
  server: {
    allowedHosts: [
      "localhost",
      "job-tracker-backend",
      "sprite-studio.tailee323f.ts.net",
    ],
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
    },
  },
});
