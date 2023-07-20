import { MenuOptions } from "@/api/acl/menu/types.ts";

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
