import {Api} from "@/api/acl/api/types.ts";

export function sortTree(tree:Api[]) {
  if (!tree || !Array.isArray(tree)) {
    return tree;
  }

  const sortedTree = [...tree]; // 创建一个副本进行排序

  // 对当前层级的子数组进行排序
  sortedTree.sort((a, b) => {
    // 根据需要的排序逻辑进行比较
    return a.sort_id - b.sort_id
  });

  // 对当前层级的子数组进行递归排序
  for (const node of sortedTree) {
    if (node.children && Array.isArray(node.children)) {
      node.children = sortTree(node.children);
    }
  }

  return sortedTree;
}