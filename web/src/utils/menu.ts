import type { Menu, MenuOptions } from "@/api/acl/menu/types.ts";

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
