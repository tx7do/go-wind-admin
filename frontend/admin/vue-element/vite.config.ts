import vue from "@vitejs/plugin-vue";
import { type ConfigEnv, type UserConfig, loadEnv, defineConfig, type PluginOption } from "vite";

import AutoImport from "unplugin-auto-import/vite";
import Components from "unplugin-vue-components/vite";
import { ElementPlusResolver } from "unplugin-vue-components/resolvers";

import { mockDevServerPlugin } from "vite-plugin-mock-dev-server";

import archiver from "archiver";
import tailwindcss from "@tailwindcss/vite";
import pkg from "./package.json" with { type: "json" };
import { createWriteStream, existsSync, readdirSync } from "node:fs";
import { createHash } from "node:crypto";
import { join, resolve } from "node:path";

/**
 * 枚举 element-plus 全部组件样式模块。ElementPlusResolver 按需生成的
 * "element-plus/es/components/{name}/style/css" 导入只会在页面首次用到
 * 该组件时才被发现，每发现一个就触发一次 vite 依赖再优化 + 整页强刷
 * ——dev 白屏竞态的主要来源。启动时全部预纳入 optimizeDeps，令其无从晚发现。
 */
function elementPlusStyleDeps(): string[] {
  const base = resolve(process.cwd(), "node_modules/element-plus/es/components");
  if (!existsSync(base)) return [];
  return readdirSync(base)
    .filter((name) => existsSync(resolve(base, name, "style", "css.mjs")))
    .map((name) => `element-plus/es/components/${name}/style/css`);
}

/**
 * 生产构建期向 index.html 注入 CSP meta。
 * dev 下 vite 注入的 HMR 内联脚本会被 script-src 'self' 拦截，故仅 build 注入。
 * vue-element 仓库内无独立部署 header 层，meta CSP 是规范支持、部署无关的途径。
 * 限制：meta 不支持 frame-ancestors 等，点击劫持防护仍需部署侧 header。
 */
/**
 * 对 index.html 内联脚本逐个计算 sha256，以 'sha256-...' 追加进 script-src：
 * 内容任一字符变动哈希即失效，外来注入的内联脚本无法借道通过。
 */
function collectInlineScriptHashes(html: string): string[] {
  const hashes: string[] = [];
  const scriptPattern = /<script(?![^>]*\bsrc=)[^>]*>([\s\S]*?)<\/script>/gi;
  let match: RegExpExecArray | null;
  while ((match = scriptPattern.exec(html)) !== null) {
    const content = match[1];
    if (content.trim().length === 0) continue;
    const digest = createHash("sha256").update(content, "utf8").digest("base64");
    hashes.push(`'sha256-${digest}'`);
  }
  return hashes;
}

function cspMetaPlugin(): PluginOption {
  let isBuild = false;
  // vite 对 transformIndexHtml.handler 返回 undefined 的联合类型展开存在版本间
  // 类型摩擦，手写插件的形状是稳定的，直接断言为 PluginOption
  return {
    name: "gowind-csp-meta",
    configResolved(config: any) {
      isBuild = config.command === "build";
    },
    transformIndexHtml: {
      order: "post" as const,
      handler(html: string) {
        if (!isBuild) return;
        const scriptSrc = ["script-src 'self'", ...collectInlineScriptHashes(html)].join(" ");
        return {
          tags: [
            {
              tag: "meta",
              attrs: {
                "http-equiv": "Content-Security-Policy",
                content: `${scriptSrc}; base-uri 'self'; object-src 'none'`,
              },
              injectTo: "head-prepend" as const,
            },
          ],
        };
      },
    },
  } as PluginOption;
}

/**
 * 构建完成后将 dist 目录打包为 dist.zip，便于交付部署
 * （对齐 vue-vben 的构建产物行为，见 internal/vite-config/src/plugins/archiver.ts）
 * apply: "build" 确保 dev server 不加载本插件，仅生产构建生效
 */
function archiverPlugin(): PluginOption {
  return {
    apply: "build",
    closeBundle: {
      handler() {
        setTimeout(async () => {
          const zipOutputPath = join(process.cwd(), "dist.zip");
          try {
            await zipFolder("dist", zipOutputPath);
            console.log(`Folder has been zipped to: ${zipOutputPath}`);
          } catch (error) {
            console.error("Error zipping folder:", error);
          }
        }, 0);
      },
      order: "post",
    },
    enforce: "post",
    name: "vite:archiver",
  };
}

/**
 * 流式压缩指定目录为 zip 文件
 */
async function zipFolder(folderPath: string, outputPath: string): Promise<void> {
  return new Promise((resolve, reject) => {
    const output = createWriteStream(outputPath);
    const archive = archiver("zip", {
      zlib: { level: 9 }, // 最高压缩率
    });

    output.on("close", () => {
      console.log(`ZIP file created: ${outputPath} (${archive.pointer()} total bytes)`);
      resolve();
    });

    archive.on("error", (err) => {
      reject(err);
    });

    archive.pipe(output);

    // 以流方式压缩目录，减少内存占用
    archive.directory(folderPath, false);

    archive.finalize();
  });
}

// Vite配置  https://cn.vitejs.dev/config
export default defineConfig(({ mode }: ConfigEnv): UserConfig => {
  const env = loadEnv(mode, process.cwd());
  const isProduction = mode === "production";

  return {
    resolve: {
      // Vite 8 新特性：自动读取 tsconfig.json 中的 paths 别名
      tsconfigPaths: true,
      // SCSS 预处理器不走 tsconfigPaths——显式补 @ 别名
      alias: {
        "@": resolve(__dirname, "src"),
      },
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
          // 后端按 /admin/v1 前缀注册路由（剥前缀会全量 404），前缀原样透传
        },
      },
      // 开发态安全响应头。X-Frame-Options/HSTS/CSP 仅在生产 nginx 生效——
      // DENY 会阻断 vue-devtools 等开发期同源 iframe，HSTS/CSP 依赖 HTTPS。
      headers: {
        "X-Content-Type-Options": "nosniff",
        "Referrer-Policy": "strict-origin-when-cross-origin",
      },
    },
    plugins: [
      vue(),
      ...(env.VITE_MOCK_DEV_SERVER === "true" ? [mockDevServerPlugin()] : []),
      tailwindcss(),
      // 生产构建期注入 CSP meta（内部 isBuild 守卫，dev 空操作）
      cspMetaPlugin(),
      // 构建完成后将 dist 打包为 dist.zip（对齐 vue-vben 构建产物行为）
      archiverPlugin(),
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
      }),
    ] as PluginOption[],
    // 预加载项目必需的组件（精简列表，Vite 8 + ElementPlusResolver 自动处理组件样式）
    optimizeDeps: {
      // 库产物含 monaco ?worker 导入，依赖优化器处理不了——排除走源码链路
      exclude: ["@go-wind/editor-vue"],
      include: [
        // 被 exclude 的子树其传递依赖同样不预构建，其中几类是裸 CJS，浏览器解析不了具名导出：
        //   - lowlight 深引 highlight.js/lib/*（es/*.js 只是转发到 lib/*.js 的 CJS 实现）
        // 显式纳入预构建，让优化器把它们转成 ESM 并补齐 default/具名导出
        "lowlight",
        "highlight.js",
        "highlight.js/lib/core",
        // json-editor-vue 的 ESM 产物 import json-source-map（纯 CJS、无 default 导出）
        "json-editor-vue",
        "json-source-map",
        "vue",
        "vue-router",
        "element-plus",
        "pinia",
        "axios",
        "@vueuse/core",
        "vue-i18n",
        "nprogress",
        "qs",
        "path-browserify",
        "path-to-regexp",
        "@element-plus/icons-vue",
        "element-plus/es",
        "element-plus/es/locale/lang/en",
        "element-plus/es/locale/lang/zh-cn",
        // 懒加载路由页才引入的重依赖：不预打包会在会话中途触发依赖再优化，
        // 使已加载 chunk 失效（dev 白屏竞态的主要来源），全部提前纳入
        "@tanstack/vue-query",
        "echarts",
        "echarts/core",
        "vxe-table",
        // element-plus 组件样式全量预纳入（见 elementPlusStyleDeps 注释）
        ...elementPlusStyleDeps(),
      ],
    },
    // 构建配置
    build: {
      chunkSizeWarningLimit: 2000,
      reportCompressedSize: false,
      minify: isProduction ? "esbuild" : false,
      target: "esnext",
      cssCodeSplit: true,
      rollupOptions: {
        output: {
          // 手动分块策略：将大型第三方库分离到独立 chunk，优化缓存和加载
          manualChunks(id) {
            if (!id.includes("node_modules")) return;

            // Monaco Editor — 体积巨大，单独拆分
            if (id.includes("monaco-editor")) return "monaco-editor";

            // ECharts — 图表库
            if (id.includes("echarts") || id.includes("zrender")) return "echarts";

            // Element Plus — UI 组件库
            if (id.includes("element-plus") || id.includes("@element-plus")) return "element-plus";

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

            // VxeTable — 表格组件
            if (id.includes("vxe-table") || id.includes("vxe-pc-ui")) return "vxe-table";

            // Tiptap 富文本编辑器
            if (id.includes("@tiptap") || id.includes("tiptap")) return "tiptap";

            // 工具库
            if (id.includes("lodash") || id.includes("dayjs") || id.includes("axios"))
              return "utils-vendor";
          },
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
    // 生产环境精简 __APP_INFO__：不打包完整 dependencies/devDependencies，减少包体积
    define: {
      __APP_INFO__: JSON.stringify({
        pkg: {
          name: pkg.name,
          version: pkg.version,
          engines: pkg.engines,
        },
        buildTimestamp: Date.now(),
      }),
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
