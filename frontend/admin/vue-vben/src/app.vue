<script lang="ts" setup>
import { computed } from 'vue';

import { useAntdDesignTokens } from '@vben/hooks';
import { preferences, usePreferences } from '@vben/preferences';

import { App, ConfigProvider, theme } from 'ant-design-vue';

import { antdLocale } from '#/locales';
import { TanstackQueryDevtools } from '#/plugins/vue-query';

defineOptions({ name: 'App' });

const isDev = import.meta.env.DEV;

const { isDark } = usePreferences();
const { tokens } = useAntdDesignTokens();

const tokenTheme = computed(() => {
  const algorithm = isDark.value
    ? [theme.darkAlgorithm]
    : [theme.defaultAlgorithm];

  // antd 紧凑模式算法
  if (preferences.app.compact) {
    algorithm.push(theme.compactAlgorithm);
  }

  return {
    algorithm,
    token: tokens,
    components: {
      Drawer: {
        // 抽屉按 §2.3 属 L1 表面（#111827），不是浮层（#1C2128）；
        // 全局 colorBgElevated 取的是 --popover，单独把 Drawer 拨回 --card
        colorBgElevated: tokens.colorBgContainer,
      },
      // §2.3 表头权威档 #1F2937 的落地走文件底部 <style> 的 CSS 规则：antdv 4.2.6 的
      // 组件 token 合并不吃 headerBg/headerColor（react antd v5 键名）也不吃
      // tableHeaderBg/tableHeaderTextColor（内部名，实测两版都静默落空），CSS 直写。
    },
  };
});
</script>

<template>
  <ConfigProvider :locale="antdLocale" :theme="tokenTheme">
    <App>
      <RouterView />
    </App>
    <!-- Vue Query Devtools（仅开发环境） -->
    <TanstackQueryDevtools v-if="isDev" />
  </ConfigProvider>
</template>

<style>
/* §2.3 弹窗域同层（2026-11-08 对齐 react 基准）：antdv Modal 面 = colorBgElevated
   #1C2128，域内原生 Table 的面（全局 = colorBgContainer #111827）在弹窗里成了
   比面深一档的倒置内嵌——随弹窗面同层，仅靠边框区分。
   antdv cssinjs 规则带 :where 前缀零特异度，本条 (0,2,0) 直接压过，无需 !important。
   亮色两态同为白、天然同层，无需规则。 */
html.dark .ant-modal .ant-table {
  background-color: #1c2128;
}

/* §2.3 表头权威档（2026-11-08 对齐 react 基准）：antdv darkAlgorithm 派生的表头底
   实测近黑 rgb(10,10,10)、文字 85% 白，与三端表头权威值（#1F2937 / #F8FAFC）脱节
   ——列表页与弹窗内嵌表一并钉回权威档。antdv 4.2.6 的组件 token 合并不吃
   headerBg/headerColor（react antd v5 键）也不吃 tableHeaderBg/tableHeaderTextColor
   （内部名，两版都静默落空），故走 CSS。亮色走 antd 默认 #FAFAFA，不覆写。 */
html.dark .ant-table-thead > tr > th {
  background-color: #1f2937;
  color: #f8fafc;
}
</style>
