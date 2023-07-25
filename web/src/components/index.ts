import { App } from "vue";
//对外暴露插件对象

import SvgIcon from "@/components/SvgIcon/index.vue";
import Pagination from "@/components/Pagination/index.vue";
import PermsButton from "@/components/PermsButton/index.vue";
import PermsLink from "@/components/PermsLink/index.vue";
import ImageUpload from "@/components/ImageUpload/index.vue";
import WarehousePageItem from "@/components/Warehouse/WarehousePageItem.vue";
import WarehouseListItem from "@/components/Warehouse/WarehouseListItem.vue";
import ZonePageItem from "@/components/WarehouseZone/ZonePageItem.vue";
import ZoneListItem from "@/components/WarehouseZone/ZoneListItem.vue";
import RackListItem from "@/components/WarehouseRack/RackListItem.vue";
import RackPageItem from "@/components/WarehouseRack/RackPageItem.vue";
import MaterialCategoryListItem from "@/components/MaterialCategory/MaterialCategoryListItem.vue";

const allGlobalComponent: any = {
  SvgIcon,
  Pagination,
  PermsButton,
  PermsLink,
  ImageUpload,
  WarehousePageItem,
  WarehouseListItem,
  ZonePageItem,
  ZoneListItem,
  RackPageItem,
  RackListItem,
  MaterialCategoryListItem,
};
export default {
  //install方法
  install(app: App) {
    Object.keys(allGlobalComponent).forEach((key) => {
      //注册为全局组件
      app.component(key, allGlobalComponent[key]);
    });
  },
};
