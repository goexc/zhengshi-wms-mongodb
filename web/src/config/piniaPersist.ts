// import { PersistedStateOptions } from "pinia-plugin-persistedstate";
import { PersistOptions } from "pinia-plugin-persist";

/**
 * @description pinia 持久化参数配置
 * @param {String} key 存储到持久化的 name
 * @param {Array} paths 需要持久化的 state name
 * @return persist
 * */
const piniaPersistConfig = (key: string, paths?: string[]) => {
  const persist: PersistOptions = {
    enabled: true,
    strategies: [
      {
        key: key,
        storage: localStorage,
        // 可以选择哪些进入local存储，这样就不用全部都进去存储了
        // 默认是全部进去存储
        paths: paths,
      },
    ],
  };
  return persist;
};

export default piniaPersistConfig;
