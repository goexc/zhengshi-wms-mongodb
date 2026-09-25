import { apiBaseUrl } from "./endpoints";
export const prod = () => {
 const host = apiBaseUrl;
 return { host, baseUrl: host };
};
