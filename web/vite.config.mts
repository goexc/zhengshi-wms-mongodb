import {type ConfigEnv, defineConfig, loadEnv} from "vite";
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
          additionalData: '@use "@/styles/variable.scss" as *;\n',
        },
      },
    },
    server: {
      port: 8000,
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
      // Preserve the browsers supported before the Vite 8 default target change.
      target: ['es2020', 'edge88', 'firefox78', 'chrome87', 'safari14'],
      chunkSizeWarningLimit:1300,
      rolldownOptions: {
        output: {
          codeSplitting: {
            groups: [{
              debugName: 'vendor-packages',
              name(id) {
                const normalizedId = id.replace(/\\/g, '/');
                if (normalizedId.includes('/node_modules/exceljs/')) {
                  return 'vendor-excel';
                }
                // Keep shared runtime helpers outside the lazy ExcelJS chunk.
                if (normalizedId.includes('/node_modules/') || normalizedId.includes('rolldown/runtime')) {
                  return 'vendor';
                }
                return null;
              },
            }],
          }
        }
      }
    }
  }
});
