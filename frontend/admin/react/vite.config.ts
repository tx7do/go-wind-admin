import {defineConfig, loadEnv} from 'vite';
import {realpathSync} from 'node:fs';
import {resolve} from 'node:path';

import {handleEnv, createProxy, createVitePlugins, buildOptions} from './build';

// https://vitejs.dev/config/
export default defineConfig(({mode}) => {
    const root = process.cwd();
    const env = loadEnv(mode, root);
    const viteEnv = handleEnv(env);
    const {VITE_SERVER_PORT, VITE_PROXY} = viteEnv;

    // @go-wind/admin-react 以 link 形态安装时真实目录在本仓之外，dev server 默认
    // 拒绝服务根外文件；把链接目标加进 fs.allow（未 link 安装时忽略）。
    let linkedAdminReactRoot: string[] = [];
    try {
        linkedAdminReactRoot = [realpathSync(resolve(root, 'node_modules/@go-wind/admin-react'))];
    } catch {
        // 非 link 安装（正式 npm 依赖）无需额外放行
    }

    return {
        base: '/',
        plugins: createVitePlugins(),
        // react 宿主里有真实 Vue 运行时：BI 设计器（@go-wind/bi-react 内部以
        // createApp 桥接 @go-wind/bi-vue 的 Vue canvas）与 json-editor-vue。
        // Vue 的 esm-bundler 构建要求 bundler 注入这三个特性开关（否则 dev 警告、
        // prod 无法按开关 tree-shake）。OPTIONS_API 保守开——json-editor-vue 走
        // Options API；devtools 与 hydration 调试细节仅 prod 生效，按需关掉。
        define: {
            __VUE_OPTIONS_API__: 'true',
            __VUE_PROD_DEVTOOLS__: 'false',
            __VUE_PROD_HYDRATION_MISMATCH_DETAILS__: 'false',
        },
        optimizeDeps: {
            // editor-react 产物含 monaco 的 `?worker` 导入，依赖优化器处理不了，必须走源码链路。
            // 被 exclude 的子树其传递依赖同样不预构建，其中两类是裸 CJS、浏览器解析不了具名导出：
            //   - @tiptap/react 从 use-sync-external-store/shim 具名导入 useSyncExternalStore
            //   - lowlight 深引 highlight.js/lib/*（es/*.js 只是转发到 lib/*.js 的 CJS 实现）
            // 显式纳入预构建即可，预构建会把这些 CJS 转成 ESM 并补齐 default/具名导出。
            exclude: ['@go-wind/editor-react'],
            include: [
                'lowlight',
                'highlight.js',
                'highlight.js/lib/core',
                'use-sync-external-store/shim',
                'use-sync-external-store/shim/index.js',
                'use-sync-external-store/shim/with-selector',
                'use-sync-external-store/shim/with-selector.js',
                // @tiptap/react（editor-react exclude 子树的传递依赖，保持 raw）会 default-import
                // 'react-dom'。机制层收编进 @go-wind/admin-react 包后，app 源码不再裸导 react-dom，
                // 优化器不再为它建入口 chunk，raw CJS 直接 serve 即缺 default 导出——显式收编。
                'react-dom',
            ],
        },
        resolve: {
            alias: {
                '@': '/src',
                '#': '/types',
            },
            // 注意：不要给 @go-wind/admin-react 的 peers 加 resolve.dedupe。
            // link 源码形态消费时 dedupe 曾用于防单例库双实例；但走 npm 依赖后
            // pnpm lockfile 已保证 peer 单实例，而 dedupe 会把这些裸依赖的解析
            // 绕过依赖优化器、直接 serve 原始 CJS（react-dom/index.js 无命名
            // default 导出，页面直接白屏卡加载页）。
        },
        css: {
            preprocessorOptions: {
                less: {
                    javascriptEnabled: true,
                    charset: false,
                },
            },
        },
        server: {
            open: true,
            port: VITE_SERVER_PORT,
            fs: {
                allow: [root, ...linkedAdminReactRoot],
            },
            // 跨域处理
            proxy: createProxy(VITE_PROXY),
            // 减少文件监听开销
            watch: {
                ignored: ['**/node_modules/**', '**/.git/**', '**/dist/**'],
                usePolling: false,
            },
            // HMR 优化，减少开发环境切换卡顿
            hmr: {
                overlay: true,
            },
            // 开发态安全响应头。X-Frame-Options/HSTS/CSP 仅在生产 nginx 生效——
            // DENY 会阻断 vue-devtools 等开发期同源 iframe，HSTS/CSP 依赖 HTTPS。
            headers: {
                'X-Content-Type-Options': 'nosniff',
                'Referrer-Policy': 'strict-origin-when-cross-origin',
            },
        },
        build: buildOptions(),
    };
});
