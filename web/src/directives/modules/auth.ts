/**
 * v-auth
 * 按钮权限指令
 */
import type { Directive, DirectiveBinding } from "vue";
import {useAuthStore} from "@/store/modules/auth.ts";

const auth: Directive = {
  mounted(el: HTMLElement, binding: DirectiveBinding) {
    const { value } = binding;
    const authStore = useAuthStore();
    // const currentPageRoles = authStore.authButtonListGet[authStore.routeName] ?? [];
    const currentPageRoles = authStore.authButtonListGet ?? [];
    if (value instanceof Array && value.length) {
      const hasPermission = value.every(item => currentPageRoles.map(item=>item.perms).includes(item));
      if (!hasPermission) el.remove();
    } else {
      if (!currentPageRoles.map(item=>item.perms).includes(value)) el.remove();
    }
  }
};

export default auth;
