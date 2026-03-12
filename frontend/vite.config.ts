import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import cesium from "vite-plugin-cesium";

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [
    react(),
    // Handles CesiumJS asset copying and CESIUM_BASE_URL injection.
    // Static assets (terrain, imagery workers) are copied to dist/cesium/.
    cesium(),
  ],
  resolve: {
    alias: {
      // Allows proto imports as: import { Unit } from "@proto/engine/v1/unit_pb"
      "@proto": "/src/proto",
    },
  },
  build: {
    // Wails embeds the dist folder; target modern browsers only.
    target: "esnext",
    // CesiumJS is large (~10MB); chunking keeps the initial load fast.
    rollupOptions: {
      output: {
        manualChunks: {
          // cesium is externalized by vite-plugin-cesium; do not list it here
          react: ["react", "react-dom"],
          zustand: ["zustand"],
          proto: ["@bufbuild/protobuf"],
        },
      },
    },
  },
  // Wails injects the dev server URL into the app; this matches.
  server: {
    port: 34115,
    strictPort: true,
  },
});
