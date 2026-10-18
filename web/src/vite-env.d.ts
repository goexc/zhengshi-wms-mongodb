/// <reference types="vite/client" />
// This virtual module only registers the SVG sprite through import side effects.
declare module "virtual:svg-icons-register";
declare module "*.vue" {
  import type { DefineComponent } from "vue";

  const vueComponent: DefineComponent<any, any, any>;

  export default vueComponent;
}
