import { fileURLToPath } from 'node:url';

import { defineConfig } from '@vben/vite-config';

// 框架层源码位于 monorepo 根 framework/（非 workspace 包），按原包名做前缀别名，
// 业务代码与框架内部互引的 '@vben/*'、'@vben-core/*' 导入保持原样零改动。
const fw = (p: string) => fileURLToPath(new URL(`./framework/${p}`, import.meta.url));

const frameworkAliases: Array<{ find: string; replacement: string }> = [
  // @vben-core（原 packages/@core）
  { find: '@vben-core/shared', replacement: fw('base/shared') },
  { find: '@vben-core/typings', replacement: fw('base/typings') },
  { find: '@vben-core/icons', replacement: fw('base/icons') },
  { find: '@vben-core/design', replacement: fw('base/design') },
  { find: '@vben-core/composables', replacement: fw('core/composables') },
  { find: '@vben-core/preferences', replacement: fw('core/preferences') },
  { find: '@vben-core/shadcn-ui', replacement: fw('ui-kit/shadcn-ui') },
  { find: '@vben-core/form-ui', replacement: fw('ui-kit/form-ui') },
  { find: '@vben-core/popup-ui', replacement: fw('ui-kit/popup-ui') },
  { find: '@vben-core/menu-ui', replacement: fw('ui-kit/menu-ui') },
  { find: '@vben-core/layout-ui', replacement: fw('ui-kit/layout-ui') },
  { find: '@vben-core/tabs-ui', replacement: fw('ui-kit/tabs-ui') },
  // @vben effects（原 packages/effects）
  { find: '@vben/access', replacement: fw('effects/access') },
  { find: '@vben/common-ui', replacement: fw('effects/common-ui') },
  { find: '@vben/hooks', replacement: fw('effects/hooks') },
  { find: '@vben/layouts', replacement: fw('effects/layouts') },
  { find: '@vben/plugins', replacement: fw('effects/plugins') },
  { find: '@vben/request', replacement: fw('effects/request') },
  // @vben 根级（原 packages/*）
  { find: '@vben/constants', replacement: fw('constants') },
  { find: '@vben/icons', replacement: fw('icons') },
  { find: '@vben/locales', replacement: fw('locales') },
  { find: '@vben/preferences', replacement: fw('preferences') },
  { find: '@vben/stores', replacement: fw('stores') },
  { find: '@vben/styles/antd', replacement: fw('styles/antd/index.css') },
  { find: '@vben/styles/ele', replacement: fw('styles/ele/index.css') },
  { find: '@vben/styles', replacement: fw('styles') },
  { find: '@vben/types', replacement: fw('types') },
  { find: '@vben/utils', replacement: fw('utils') },
];

export default defineConfig(async () => {
  return {
    application: {},
    vite: {
      resolve: {
        alias: frameworkAliases,
      },
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
