import type { Menu, MenuOptions } from "@/api/acl/menu/types.ts";

// Fallbacks for older menus whose title is blank or still a route identifier.
// Keep configured Chinese titles and route names/permissions intact.
const pageTitles: Record<string, string> = {
  "/home": "首页",
  "/login": "登录",
  "/acl": "权限管理",
  "/acl/menu": "菜单管理",
  "/acl/api": "接口管理",
  "/acl/department": "部门管理",
  "/acl/role": "角色管理",
  "/acl/user": "用户管理",
  "/business_partner": "合作伙伴",
  "/business_partner/customer": "客户管理",
  "/business_partner/supplier": "供应商管理",
  "/business_partner/supplier/flow": "供应商流水",
  "/business_partner/supplier/order": "采购订单管理",
  "/business_partner/carrier": "承运商管理",
  "/finance": "客户财务",
  "/finance/acceptance": "客户承兑",
  "/finance/invoice": "销售开票记录",
  "/material": "物料管理",
  "/material/category": "物料分类",
  "/material/list": "物料管理",
  "/material/quote": "客户新增物料报价",
  "/warehouse": "仓储管理",
  "/warehouse/warehouse": "仓库管理",
  "/warehouse/zone": "库区管理",
  "/warehouse/rack": "货架管理",
  "/warehouse/bin": "货位管理",
  "/inventory": "库存管理",
  "/inventory/record": "库存记录",
  "/inbound": "入库管理",
  "/inbound/receipt": "入库单管理",
  "/inbound/report": "入库报表",
  "/outbound": "出库管理",
  "/outbound/receipt": "出库单管理",
  "/outbound/order": "出库订单管理",
  "/outbound/fast": "快速出库",
  "/outbound/report": "出库报表",
  "/image": "素材管理",
  "/product": "产品管理",
  "/product/sku": "商品规格管理（SKU）",
  "/product/spu": "标准产品管理（SPU）",
  "/screen": "数据大屏",
  "/403": "无权访问",
  "/404": "页面不存在",
  "/500": "服务器错误",
};

interface NavigationItem {
  path?: string;
  name?: unknown;
  component?: unknown;
  meta?: { title?: unknown } | Record<string, unknown>;
}

const textValue = (value: unknown): string => typeof value === "string" ? value.trim() : "";
const hasChinese = (value: string) => /[\u3400-\u9fff]/.test(value);
const navigationPath = (value: unknown) => {
  const path = textValue(value).replace(/^\/?src\/views\//, "").replace(/\.vue$/, "").replace(/\/index2?$/, "");
  return path ? `/${path.replace(/^\/+|\/+$/g, "")}` : "";
};

/** One display rule for the sidebar, breadcrumbs and browser title. */
export function getNavigationTitle(item: NavigationItem): string {
  const title = textValue(item.meta?.title);
  const name = textValue(item.name);
  if (hasChinese(title)) return title;
  if (hasChinese(name)) return name;
  return pageTitles[navigationPath(item.path)] || pageTitles[navigationPath(item.component)] || title || name;
}

/** Normalize only the authorized menu tree; finance remains a sibling group. */
export function prepareNavigationMenus(menuList: Menu[]): Menu[] {
  const result = filterRetiredSupplierMenus(menuList);
  const prepare = (items: Menu[]): Menu[] => {
    const sorted = items.map((item) => ({
      ...item,
      meta: { ...item.meta, title: getNavigationTitle(item) },
      ...(item.children ? { children: prepare(item.children) } : {}),
    })).sort((a, b) => a.sort_id - b.sort_id);
    const isGroup = (item: Menu, path: string) => !item.meta.hidden &&
      (navigationPath(item.path) === path || item.meta.perms === path.slice(1));
    const financeIndex = sorted.findIndex((item) => isGroup(item, "/finance"));
    if (financeIndex >= 0 && sorted.some((item) => isGroup(item, "/business_partner"))) {
      const [finance] = sorted.splice(financeIndex, 1);
      const partnerIndex = sorted.findIndex((item) => isGroup(item, "/business_partner"));
      sorted.splice(partnerIndex + 1, 0, finance);
    }
    return sorted;
  };
  return prepare(result);
}

/** 删除旧供应商管理入口，兼容数据库中尚未清理的动态菜单。 */
export function filterRetiredSupplierMenus(menuList: Menu[]): Menu[] {
  const isRetiredPath = (value: string) => /^\/supplier(?:\/|$)/.test(value.trim());
  const isRetiredComponent = (value: string) => /^\/?supplier(?:\/|$)/.test(value.trim());

  return menuList
    .filter((item) => !isRetiredPath(item.path) && !isRetiredComponent(item.component))
    .map((item) => ({
      ...item,
      ...(item.children ? { children: filterRetiredSupplierMenus(item.children) } : {}),
    }));
}

/**
 * @description 使用递归扁平化菜单，方便添加动态路由
 * @param {Array} menuList 菜单列表
 * @returns {Array}
 */
export function getFlatMenuList(menuList: MenuOptions[]): MenuOptions[] {
  const newMenuList: MenuOptions[] = JSON.parse(JSON.stringify(menuList));
  return newMenuList.flatMap((item) => [
    item,
    ...(item.children ? getFlatMenuList(item.children) : []),
  ]);
}
