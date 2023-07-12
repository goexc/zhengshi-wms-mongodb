//仓库、库区、货架、货位全局组件的小仓库
import { defineStore } from "pinia";
import { reqTrademarkList } from "@/api/product/trademark";
import { reqWarehouseZoneList } from "@/api/warehouse_zone";
import { reqWarehouseRackList } from "@/api/warehouse_rack";
import { reqWarehouseBinList } from "@/api/warehouse_bin";
import { Warehouse } from "@/api/product/trademark/types.ts";
import {
  WarehouseZone,
  WarehouseZoneListRequest,
} from "@/api/warehouse_zone/types.ts";
import {
  WarehouseBin,
  WarehouseBinListRequest,
} from "@/api/warehouse_bin/types.ts";
import {
  WarehouseRack,
  WarehouseRackListRequest,
} from "@/api/warehouse_rack/types.ts";

const useCategoryStore = defineStore("Category", {
  state: () => {
    return {
      warehouses: <Warehouse[]>[], //仓库列表
      warehouse_id: <string>"", //被选中的仓库id
      warehouse_zones: <WarehouseZone[]>[], //库区列表
      warehouse_zone_id: <string>"", //被选中的库区id
      warehouse_racks: <WarehouseRack[]>[], //货架列表
      warehouse_rack_id: <string>"", //被选中的货架id
      warehouse_bins: <WarehouseBin[]>[], //货位列表
    };
  },
  actions: {
    //请求仓库列表
    async getWarehouseList() {
      const res = await reqTrademarkList();
      if (res.code === 200) {
        this.warehouses = res.data.list;
      }
    },
    //请求库区列表
    async getWarehouseZoneList(req: WarehouseZoneListRequest) {
      this.warehouse_zone_id = "";
      const res = await reqWarehouseZoneList(req);
      if (res.code === 200) {
        this.warehouse_zones = res.data.list;
      }
    },
    //请求货架列表
    async getWarehouseRackList(req: WarehouseRackListRequest) {
      this.warehouse_rack_id = "";
      const res = await reqWarehouseRackList(req);
      if (res.code === 200) {
        this.warehouse_racks = res.data.list;
      }
    },
    //请求货位列表
    async getWarehouseBinList(req: WarehouseBinListRequest) {
      const res = await reqWarehouseBinList(req);
      if (res.code === 200) {
        this.warehouse_bins = res.data.list;
      }
    },
  },
  //
  getters: {
    //获取选中仓库的名称
    getWarehouseName: (state) => {
      return (
        state.warehouses.find((one) => {
          return one.id === state.warehouse_id;
        })?.name || ""
      );
    },
    //获取选中库区的名称
    getWarehouseZoneName: (state) => {
      return (
        state.warehouse_zones.find((one) => {
          return one.id === state.warehouse_zone_id;
        })?.name || ""
      );
    },
    //获取选中货架的名称
    getWarehouseRackName: (state) => {
      return (
        state.warehouse_racks.find((one) => {
          return one.id === state.warehouse_rack_id;
        })?.name || ""
      );
    },
  },
  persist: {
    enabled: true,
    strategies: [
      {
        key: "p_category",
        storage: localStorage,
        // 可以选择哪些进入local存储，这样就不用全部都进去存储了
        // 默认是全部进去存储
        // paths: ['token']
      },
    ],
  },
});

export default useCategoryStore;
