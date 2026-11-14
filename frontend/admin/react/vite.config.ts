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
            // @go-wind/admin-react 以 link 源码形态消费（未发版前）时，包内对外部库的裸导入
            // 会从包所在目录（go-wind-ui workspace）向上解析，与宿主 node_modules 形成两套
            // 实例——react/antd/zustand/i18next/react-i18next/react-router/react-query 都是
            // 单例敏感库（context/private 字段），双实例直接坏登录与 i18n。dedupe 强制这些
            // 包一律从宿主根解析。发版改走 npm 依赖后本段仍保留无害。
            dedupe: [
                'react',
                'react-dom',
                'antd',
                '@ant-design/icons',
                '@ant-design/pro-components',
                'react-router-dom',
                'react-i18next',
                'i18next',
                '@tanstack/react-query',
                'zustand',
                'axios',
                '@microsoft/fetch-event-source',
                'dayjs',
            ],
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
