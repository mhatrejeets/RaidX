import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      "/api": "http://localhost:3000",
      "/login": "http://localhost:3000",
      "/refresh": "http://localhost:3000",
      "/logout": "http://localhost:3000",
      "/logout-all": "http://localhost:3000"
    }
  }
});