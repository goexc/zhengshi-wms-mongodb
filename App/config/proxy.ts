import { apiBaseUrl } from "./endpoints";
const target = process.env["VITE_WMS_API_URL"] ?? apiBaseUrl;
export const proxy = { dev: { target, changeOrigin: true, rewrite: (path: string) => path.replace(/^\/dev/, "") } };
export const value = "dev";
