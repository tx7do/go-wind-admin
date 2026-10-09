<script lang="ts" setup>
// 对齐 react/ele：登录等认证页左上角不放品牌信息（logo/appName 不传，
// AuthPageLayout 内 v-if="logo || appName" 的品牌块即不渲染），仅保留右上角工具栏。
import { AuthPageLayout } from '@vben/layouts';

import { $t } from '#/locales';
</script>

<template>
  <AuthPageLayout
    :page-description="$t('authentication.pageDesc')"
    :page-title="$t('authentication.pageTitle')"
  >
    <!-- 自定义工具栏 -->
    <!-- <template #toolbar></template> -->
  </AuthPageLayout>
</template>
