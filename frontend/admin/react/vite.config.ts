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
