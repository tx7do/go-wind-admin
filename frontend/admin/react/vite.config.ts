import {defineConfig, loadEnv} from 'vite';

import {handleEnv, createProxy, createVitePlugins, buildOptions} from './build';

// https://vitejs.dev/config/
export default defineConfig(({mode}) => {
    const root = process.cwd();
    const env = loadEnv(mode, root);
    const viteEnv = handleEnv(env);
    const {VITE_SERVER_PORT, VITE_PROXY} = viteEnv;

    return {
        base: '/',
        plugins: createVitePlugins(),
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
            ],
        },
        resolve: {
            alias: {
                '@': '/src',
                '#': '/types',
            },
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
