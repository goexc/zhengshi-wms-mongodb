import {ConfigEnv, defineConfig, loadEnv, UserConfigExport} from "vite";
import vue from "@vitejs/plugin-vue";
import path from 'path'
import {createSvgIconsPlugin} from 'vite-plugin-svg-icons'
import vueJsx from '@vitejs/plugin-vue-jsx'

// https://vitejs.dev/config/
export default defineConfig(({mode}: ConfigEnv) => {
  //获取不同环境下，对应的变量
  let env = loadEnv(mode, process.cwd())

  console.log('开发环境：', mode)
  return {
    plugins: [
      vue(),
      vueJsx(),
      createSvgIconsPlugin({
        // Specify the icon folder to be cached
        iconDirs: [path.resolve(process.cwd(), 'src/assets/icons')],
        // Specify symbolId format
        symbolId: 'icon-[dir]-[name]',
      }),
    ],
    resolve: {
      alias: {
        "@": path.resolve("./src") // 相对路径别名配置，使用 @ 代替 src
        // '@': fileURLToPath(new URL('./src', import.meta.url)),
        // "@": path.resolve(__dirname, "./src")
        // "@": path.resolve(__dirname, "src")
      }
    },
    css: {
      preprocessorOptions: {
        scss: {
          javascriptEnabled: true,
          additionalData: '@import "./src/styles/variable.scss";',
        },
      },
    },
    server: {
      proxy: {
        [env.VITE_BASE_API]: {
          //服务器地址
          target: env.VITE_SERVER,
          //是否跨域
          changeOrigin: true,
          //路径重写
          rewrite: (path) => path.replace(/&\/api/, ''),
        },
      },
    },
    build: {
      chunkSizeWarningLimit:1300,
      rollupOptions: {
        output: {
          manualChunks(id) {
            if (id.includes('node_modules')) {
              return id.toString().split('node_modules/')[1].split('/')[0].toString();
            }
          }
        }
      }
    }
  }
});
