import { defineStore } from "pinia";
import piniaPersistConfig from "@/config/piniaPersist";
import { Account } from "@/api/user/types.ts";

export const useUserStore = defineStore({
  id: "zs-user",
  state: (): { account: Account; token: string } => ({
    token: "",
    account: <Account>{},
  }),
  getters: {},
  actions: {
    // Set Token
    setToken(token: string) {
      this.token = token;
    },
    // Set setUserInfo
    setAccount(account: Account) {
      this.account = account;
    },
  },
  persist: piniaPersistConfig("zs-user"),
});
