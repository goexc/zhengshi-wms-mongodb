import { Perms} from "@/api/acl/menu/types.ts";

export interface PermsResponse {
  code: number;
  msg: string;
  data: Perms;
}