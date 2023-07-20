import { defineStore } from "pinia";
import { UserState } from "@/store/interface";
import piniaPersistConfig from "@/config/piniaPersist";
import { Account } from "@/api/user/types.ts";

export const useUserStore = defineStore({
  id: "zs-user",
  state: (): UserState => ({
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
