import vue from "@vitejs/plugin-vue";
import { type ConfigEnv, type UserConfig, loadEnv, defineConfig, type PluginOption } from "vite";

import AutoImport from "unplugin-auto-import/vite";
import Components from "unplugin-vue-components/vite";
import { ElementPlusResolver } from "unplugin-vue-components/resolvers";

import { mockDevServerPlugin } from "vite-plugin-mock-dev-server";

import tailwindcss from "@tailwindcss/vite";
import pkg from "./package.json" with { type: "json" };

// 平台的名称、版本、运行所需的 node 版本、依赖、构建时间的类型提示
const __APP_INFO__ = {
  pkg: {
    name: pkg.name,
    version: pkg.version,
    engines: pkg.engines,
    dependencies: pkg.dependencies,
    devDependencies: pkg.devDependencies,
  },
  buildTimestamp: Date.now(),
};

// Vite配置  https://cn.vitejs.dev/config
export default defineConfig(({ mode }: ConfigEnv): UserConfig => {
  const env = loadEnv(mode, process.cwd());
  const isProduction = mode === "production";

  return {
    resolve: {
      // Vite 8 新特性：自动读取 tsconfig.json 中的 paths 别名
      tsconfigPaths: true,
    },
    css: {
      preprocessorOptions: {
        // 定义全局 SCSS 变量
        scss: {
          additionalData: `@use "@/styles/_variables.scss" as *;`,
        },
      },
    },
    server: {
      host: "0.0.0.0",
      port: +env.VITE_APP_PORT,
      open: true,
      proxy: {
        [env.VITE_APP_BASE_API]: {
          changeOrigin: true,
          target: env.VITE_APP_API_URL,
          rewrite: (path: string) => path.replace(new RegExp("^" + env.VITE_APP_BASE_API), ""),
        },
      },
    },
    plugins: [
      vue(),
      ...(env.VITE_MOCK_DEV_SERVER === "true" ? [mockDevServerPlugin()] : []),
      tailwindcss(),
      // API 自动导入
      AutoImport({
        // 导入 Vue 函数，如：ref, reactive, toRef 等
        imports: ["vue", "@vueuse/core", "pinia", "vue-router", "vue-i18n"],
        resolvers: [
          // 导入 Element Plus函数，如：ElMessage, ElMessageBox 等
          ElementPlusResolver(),
        ],
        eslintrc: {
          enabled: false,
          filepath: "./.eslintrc-auto-import.json",
          globalsPropValue: true,
        },
        vueTemplate: true,
        // 导入函数类型声明文件路径 (false:关闭自动生成)
        dts: false,
        // dts: "src/types/auto-imports.d.ts",
      }),
      // 组件自动导入
      Components({
        resolvers: [
          // 导入 Element Plus 组件
          ElementPlusResolver(),
        ],
        // 指定自定义组件位置(默认:src/components)
        dirs: ["src/components", "src/**/components"],
        // 导入组件类型声明文件路径 (false:关闭自动生成)
        dts: false,
        //dts: "src/types/components.d.ts",
      }),
    ] as PluginOption[],
    // 预加载项目必需的组件
    optimizeDeps: {
      include: [
        "vue",
        "vue-router",
        "element-plus",
        "pinia",
        "axios",
        "@vueuse/core",
        "codemirror-editor-vue3",
        "exceljs",
        "path-to-regexp",
        "echarts/core",
        "echarts/renderers",
        "echarts/charts",
        "echarts/components",
        "vue-i18n",
        "nprogress",
        "sortablejs",
        "qs",
        "path-browserify",
        "@element-plus/icons-vue",
        "element-plus/es",
        "element-plus/es/locale/lang/en",
        "element-plus/es/locale/lang/zh-cn",
        "element-plus/es/components/alert/style/css",
        "element-plus/es/components/avatar/style/css",
        "element-plus/es/components/backtop/style/css",
        "element-plus/es/components/badge/style/css",
        "element-plus/es/components/base/style/css",
        "element-plus/es/components/breadcrumb-item/style/css",
        "element-plus/es/components/breadcrumb/style/css",
        "element-plus/es/components/button/style/css",
        "element-plus/es/components/card/style/css",
        "element-plus/es/components/cascader/style/css",
        "element-plus/es/components/checkbox-group/style/css",
        "element-plus/es/components/checkbox/style/css",
        "element-plus/es/components/col/style/css",
        "element-plus/es/components/color-picker/style/css",
        "element-plus/es/components/config-provider/style/css",
        "element-plus/es/components/date-picker/style/css",
        "element-plus/es/components/descriptions-item/style/css",
        "element-plus/es/components/descriptions/style/css",
        "element-plus/es/components/dialog/style/css",
        "element-plus/es/components/divider/style/css",
        "element-plus/es/components/drawer/style/css",
        "element-plus/es/components/dropdown-item/style/css",
        "element-plus/es/components/dropdown-menu/style/css",
        "element-plus/es/components/dropdown/style/css",
        "element-plus/es/components/empty/style/css",
        "element-plus/es/components/form-item/style/css",
        "element-plus/es/components/form/style/css",
        "element-plus/es/components/icon/style/css",
        "element-plus/es/components/image-viewer/style/css",
        "element-plus/es/components/image/style/css",
        "element-plus/es/components/input-number/style/css",
        "element-plus/es/components/input-tag/style/css",
        "element-plus/es/components/input/style/css",
        "element-plus/es/components/link/style/css",
        "element-plus/es/components/loading/style/css",
        "element-plus/es/components/menu-item/style/css",
        "element-plus/es/components/menu/style/css",
        "element-plus/es/components/message-box/style/css",
        "element-plus/es/components/message/style/css",
        "element-plus/es/components/notification/style/css",
        "element-plus/es/components/option/style/css",
        "element-plus/es/components/pagination/style/css",
        "element-plus/es/components/popover/style/css",
        "element-plus/es/components/progress/style/css",
        "element-plus/es/components/radio-button/style/css",
        "element-plus/es/components/radio-group/style/css",
        "element-plus/es/components/radio/style/css",
        "element-plus/es/components/row/style/css",
        "element-plus/es/components/scrollbar/style/css",
        "element-plus/es/components/select/style/css",
        "element-plus/es/components/skeleton-item/style/css",
        "element-plus/es/components/skeleton/style/css",
        "element-plus/es/components/step/style/css",
        "element-plus/es/components/steps/style/css",
        "element-plus/es/components/sub-menu/style/css",
        "element-plus/es/components/switch/style/css",
        "element-plus/es/components/tab-pane/style/css",
        "element-plus/es/components/table-column/style/css",
        "element-plus/es/components/table/style/css",
        "element-plus/es/components/tabs/style/css",
        "element-plus/es/components/tag/style/css",
        "element-plus/es/components/text/style/css",
        "element-plus/es/components/time-picker/style/css",
        "element-plus/es/components/time-select/style/css",
        "element-plus/es/components/timeline-item/style/css",
        "element-plus/es/components/timeline/style/css",
        "element-plus/es/components/tooltip/style/css",
        "element-plus/es/components/tree-select/style/css",
        "element-plus/es/components/tree/style/css",
        "element-plus/es/components/upload/style/css",
        "element-plus/es/components/watermark/style/css",
        "element-plus/es/components/checkbox-button/style/css",
        "element-plus/es/components/space/style/css",

      ],
    },
    // 构建配置
    build: {
      chunkSizeWarningLimit: 2000, // 消除打包大小超过500kb警告
      reportCompressedSize: false,
      minify: isProduction ? "esbuild" : false, // 使用 esbuild 压缩，比 terser 快 20-100 倍
      target: "esnext",
      rollupOptions: {
        output: {
          // 手动分块：将 Vue 相关包分离到独立 chunk，避免 Rolldown 产生循环依赖
          manualChunks(id) {
            if (id.includes("node_modules")) {
              // Vue 核心 & 生态
              if (
                id.includes("/vue/") ||
                id.includes("\\vue\\") ||
                id.includes("/@vue/") ||
                id.includes("\\@vue\\") ||
                id.includes("/vue-router/") ||
                id.includes("\\vue-router\\") ||
                id.includes("/pinia/") ||
                id.includes("\\pinia\\") ||
                id.includes("/vue-i18n/") ||
                id.includes("\\vue-i18n\\") ||
                id.includes("/@vueuse/") ||
                id.includes("\\@vueuse\\")
              ) {
                return "vue-vendor";
              }
            }
          },
          // manualChunks: {
          //   "vue-i18n": ["vue-i18n"],
          // },
          // 用于从入口点创建的块的打包输出格式[name]表示文件名,[hash]表示该文件内容hash值
          entryFileNames: "js/[name].[hash].js",
          // 用于命名代码拆分时创建的共享块的输出命名
          chunkFileNames: "js/[name].[hash].js",
          // 用于输出静态资源的命名，[ext]表示文件扩展名
          assetFileNames: (assetInfo: any) => {
            // Vite 8 / Rolldown: 添加空值保护
            if (!assetInfo.name) {
              return "assets/[name].[hash][extname]";
            }
            const info = assetInfo.name.split(".");
            let extType = info[info.length - 1];
            // console.log('文件信息', assetInfo.name)
            if (/\.(mp4|webm|ogg|mp3|wav|flac|aac)(\?.*)?$/i.test(assetInfo.name)) {
              extType = "media";
            } else if (/\.(png|jpe?g|gif|svg)(\?.*)?$/.test(assetInfo.name)) {
              extType = "img";
            } else if (/\.(woff2?|eot|ttf|otf)(\?.*)?$/i.test(assetInfo.name)) {
              extType = "fonts";
            }
            return `${extType}/[name].[hash].[ext]`;
          },
        },
      },
    },
    define: {
      __APP_INFO__: JSON.stringify(__APP_INFO__),
      // esbuild 会将 console.log / console.debug / debugger 视为无副作用并移除
      ...(isProduction
        ? {
            "console.log": "(() => {})",
            "console.debug": "(() => {})",
            "console.info": "(() => {})",
          }
        : {}),
    },
  };
});
