/// <reference types="vite/client" />
declare module "*.vue" {
  import type { DefineComponent } from "vue";

  const vueComponent: DefineComponent<any, any, any>;

  export default vueComponent;
}
