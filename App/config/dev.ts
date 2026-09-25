import { apiBaseUrl } from "./endpoints";
export const dev = () => {
 const host = apiBaseUrl;
 let baseUrl = host;
 // #ifdef H5
 baseUrl = "/dev";
 // #endif
 return { host, baseUrl };
};
