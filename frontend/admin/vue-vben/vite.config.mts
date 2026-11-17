import { fileURLToPath } from 'node:url';

import { defineConfig } from './build-config/vite-config/src';

// 框架层已分层抽包：冻结/低频内核（base/core/ui-kit/根级 8 目录/effects 的
// access+hooks+common-ui）迁入 @go-wind/admin-vben（源码形态包，经 tsconfig paths
// 指向包 src 消费）；业务倒逼修改热区 effects/layouts + effects/plugins 留宿主
// framework/ 目录，按原包名保留前缀别名。
const fw = (p: string) => fileURLToPath(new URL(`./framework/${p}`, import.meta.url));
const nm = (p: string) => fileURLToPath(new URL(`./node_modules/${p}`, import.meta.url));

// vue 系运行时物理映射（正则精确匹配）：@go-wind/admin-vben 源码位于 node_modules
// link 内，其 import 'vue' 等若从包真实位置向上会解析到 go-wind-ui 树的另一份实例。
// 包整体 exclude 出预构建后，此物理映射仅作用于运行时/构建解析（不产生双份预构建
// deps），把 vue 系统一压回宿主唯一副本（vue 双实例会使 reactive 等内部符号互斥）。
const vueRuntimeAliases: Array<{ find: string | RegExp; replacement: string }> = [
  { find: /^vue$/, replacement: nm('vue') },
  { find: /^vue\/(.*)$/, replacement: nm('vue/$1') },
  { find: /^vue-router$/, replacement: nm('vue-router') },
  { find: /^vue-router\//, replacement: nm('vue-router/') },
  { find: /^pinia$/, replacement: nm('pinia') },
  { find: /^pinia\//, replacement: nm('pinia/') },
  { find: /^pinia-plugin-persistedstate/, replacement: nm('pinia-plugin-persistedstate') },
  { find: /^ant-design-vue$/, replacement: nm('ant-design-vue') },
  { find: /^ant-design-vue\//, replacement: nm('ant-design-vue/') },
  { find: /^@vueuse\/core/, replacement: nm('@vueuse/core') },
  { find: /^@vueuse\/integrations/, replacement: nm('@vueuse/integrations') },
  { find: /^@tanstack\/vue-query$/, replacement: nm('@tanstack/vue-query') },
  { find: /^@tanstack\/vue-store/, replacement: nm('@tanstack/vue-store') },
  { find: /^vue-i18n$/, replacement: nm('vue-i18n') },
  { find: /^vue-i18n\//, replacement: nm('vue-i18n/') },
];

const frameworkAliases: Array<{ find: string; replacement: string }> = [
  ...vueRuntimeAliases,
  // 已抽入 @go-wind/admin-vben（源码包）：旧 @vben/* import 重定向到包内子路径
  //（replacement 为包说明符，rolldown 二跳解析 node_modules）
  { find: '@vben-core/shared', replacement: '@go-wind/admin-vben/src/base/shared' },
  { find: '@vben-core/typings', replacement: '@go-wind/admin-vben/src/base/typings' },
  { find: '@vben-core/icons', replacement: '@go-wind/admin-vben/src/base/icons' },
  { find: '@vben-core/design', replacement: '@go-wind/admin-vben/src/base/design' },
  { find: '@vben-core/composables', replacement: '@go-wind/admin-vben/src/core/composables' },
  { find: '@vben-core/preferences', replacement: '@go-wind/admin-vben/src/core/preferences' },
  { find: '@vben-core/shadcn-ui', replacement: '@go-wind/admin-vben/src/ui-kit/shadcn-ui' },
  { find: '@vben-core/form-ui', replacement: '@go-wind/admin-vben/src/ui-kit/form-ui' },
  { find: '@vben-core/popup-ui', replacement: '@go-wind/admin-vben/src/ui-kit/popup-ui' },
  { find: '@vben-core/menu-ui', replacement: '@go-wind/admin-vben/src/ui-kit/menu-ui' },
  { find: '@vben-core/layout-ui', replacement: '@go-wind/admin-vben/src/ui-kit/layout-ui' },
  { find: '@vben-core/tabs-ui', replacement: '@go-wind/admin-vben/src/ui-kit/tabs-ui' },
  { find: '@vben/access', replacement: '@go-wind/admin-vben/src/effects/access' },
  { find: '@vben/common-ui', replacement: '@go-wind/admin-vben/src/effects/common-ui' },
  { find: '@vben/hooks', replacement: '@go-wind/admin-vben/src/effects/hooks' },
  { find: '@vben/constants', replacement: '@go-wind/admin-vben/src/constants' },
  { find: '@vben/icons', replacement: '@go-wind/admin-vben/src/icons' },
  { find: '@vben/locales', replacement: '@go-wind/admin-vben/src/locales' },
  { find: '@vben/preferences', replacement: '@go-wind/admin-vben/src/preferences' },
  { find: '@vben/stores', replacement: '@go-wind/admin-vben/src/stores' },
  { find: '@vben/styles/antd', replacement: '@go-wind/admin-vben/src/styles/antd/index.css' },
  { find: '@vben/styles/ele', replacement: '@go-wind/admin-vben/src/styles/ele/index.css' },
  { find: '@vben/styles', replacement: '@go-wind/admin-vben/src/styles' },
  { find: '@vben/types', replacement: '@go-wind/admin-vben/src/types' },
  { find: '@vben/utils', replacement: '@go-wind/admin-vben/src/utils' },
  // 保留在宿主的高频变异层
  { find: '@vben/layouts', replacement: fw('effects/layouts') },
  { find: '@vben/plugins', replacement: fw('effects/plugins') },
];

export default defineConfig(async () => {
  return {
    application: {},
    vite: {
      resolve: {
        alias: frameworkAliases,
      },
      optimizeDeps: {
        // @go-wind/admin-vben 是源码形态包（node_modules link/registry，ts/vue/css
        // 由宿主管线处理），必须 exclude 出预构建；其 import 的裸 CJS 传递依赖
        // 随之不被预构建、浏览器直取 raw 会缺 default 导出——逐个 include 收编
        //（照 editor-react/react 端同款药方，清单随包依赖演进维护）。
        exclude: [
          '@go-wind/admin-vben',
          '@vben-core/shared',
          '@vben-core/typings',
          '@vben-core/icons',
          '@vben-core/design',
          '@vben-core/composables',
          '@vben-core/preferences',
          '@vben-core/shadcn-ui',
          '@vben-core/form-ui',
          '@vben-core/popup-ui',
          '@vben-core/menu-ui',
          '@vben-core/layout-ui',
          '@vben-core/tabs-ui',
          '@vben/access',
          '@vben/common-ui',
          '@vben/hooks',
          // 这三个曾漏出 exclude：被预构建成 .vite/deps/@vben_locales.js 等，
          // 而包内组件的相对 import 走 .pnpm 源码路径——同一 locales 模块
          // 出现两个 createI18n 实例，包组件侧 composer 全空，登录页按钮/
          // 记住我等文案渲染成裸 key 且永不自愈（locale 切换只重渲染
          // 实例 A 侧文本）。alias 到包内源码的裸说明符必须全部 exclude。
          '@vben/locales',
          '@vben/icons',
          '@vben/constants',
          '@vben/preferences',
          '@vben/stores',
          '@vben/types',
          '@vben/utils',
          '@vben/styles',
        ],
        include: [
          'dayjs/plugin/relativeTime',
          'dayjs/plugin/timezone',
          'dayjs/plugin/utc',
          'dayjs/locale/zh-cn',
          'dayjs/locale/en',
          'json-bigint',
          'vue-json-viewer',
          'lodash.clonedeep',
          'lodash.get',
          'lodash.isequal',
          'lodash.defaults',
          'lodash.memoize',
          'nprogress',
        ],
      },
      server: {
        proxy: {
          // 后端 API 代理（对齐 react 的 VITE_PROXY 定式）：完整路径原样转发，
          // 前缀 /admin 不剥——后端路由本身就挂在 /admin/v1 下
          '/admin': {
            // 后端 refresh token 走 HttpOnly Cookie，changeOrigin 保证 Host 指向后端
            changeOrigin: true,
            target: 'http://localhost:7788',
          },
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
