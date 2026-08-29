import { defineConfig } from "vite";
import { handleDevIpc } from "./dev-ipc";

const host = process.env.TAURI_DEV_HOST;

export default defineConfig({
  clearScreen: false,
  server: {
    // 1420 常落在 Windows Hyper-V 排除段（1400–1499），改用 Vite 默认端口。
    port: 5173,
    strictPort: true,
    host: host || "127.0.0.1",
    watch: { ignored: ["**/src-tauri/**"] },
  },
  envPrefix: ["VITE_", "TAURI_ENV_*"],
  plugins: [
    {
      name: "xallor-dev-ipc",
      configureServer(server) {
        server.middlewares.use((req, res, next) => {
          void handleDevIpc(req, res)
            .then((hit) => {
              if (!hit) next();
            })
            .catch(next);
        });
      },
    },
  ],
});
