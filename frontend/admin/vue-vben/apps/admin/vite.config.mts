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
