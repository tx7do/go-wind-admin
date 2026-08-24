import { defineConfig } from '@vben/vite-config';

export default defineConfig(async () => {
  return {
    application: {},
    vite: {
      server: {
        proxy: {
          '/api': {
            changeOrigin: true,
            rewrite: (path) => path.replace(/^\/api/, ''),
            // mock代理目标地址
            target: 'http://localhost:5320/api',
            ws: true,
          },
        },
        // 开发态安全响应头。X-Frame-Options/HSTS/CSP 仅在生产 nginx 生效——
        // DENY 会阻断 vue-devtools 等开发期同源 iframe，HSTS/CSP 依赖 HTTPS。
        headers: {
          'X-Content-Type-Options': 'nosniff',
          'Referrer-Policy': 'strict-origin-when-cross-origin',
        },
      },
      build: {
        rollupOptions: {
          external: (id: string) => {
            // vue-query-devtools v6 引入 jiti，导致生产构建失败
            // devtools 通过动态 import 加载，生产环境不需要打包
            return (
              id.includes('@tanstack/vue-query-devtools') ||
              id.includes('jiti')
            );
          },
        },
      },
    },
  };
});
