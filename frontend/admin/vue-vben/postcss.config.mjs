// 原 internal/tailwind-config/src/postcss.config.ts 内联（去除微服务包）。
// tailwindcss 不显式传 config，自动读取根 tailwind.config.ts。
export default {
  plugins: {
    ...(process.env.NODE_ENV === 'production' ? { cssnano: {} } : {}),
    autoprefixer: {},
    // 修复 element-plus 和 ant-design-vue 的样式和 tailwindcss 冲突问题
    'postcss-antd-fixes': { prefixes: ['ant', 'el'] },
    'postcss-import': {},
    'postcss-preset-env': {},
    tailwindcss: {},
    'tailwindcss/nesting': {},
  },
};
