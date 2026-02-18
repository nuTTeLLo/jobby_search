import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    allowedHosts: [
      "localhost",
      "job-tracker-backend",
      "sprite-studio.tailee323f.ts.net",
    ],
  },
});
