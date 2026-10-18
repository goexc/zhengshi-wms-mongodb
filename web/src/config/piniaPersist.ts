import type { PersistenceOptions } from "pinia-plugin-persistedstate";

/**
 * @description pinia 持久化参数配置
 * @param {String} key 存储到持久化的 name
 * @param {Array} paths 需要持久化的 state name
 * @return persist
 * */
const piniaPersistConfig = (
  key: string,
  paths?: string[],
): PersistenceOptions => {
  return {
    // 保留原 localStorage key 和 JSON 格式，升级后可直接恢复已有会话和设置。
    key,
    storage: localStorage,
    serializer: {
      serialize: JSON.stringify,
      deserialize: JSON.parse,
    },
    // 未指定时保存全部状态；空数组只保存空对象。
    pick: paths,
  };
};

export default piniaPersistConfig;
