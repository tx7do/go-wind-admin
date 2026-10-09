<script lang="ts" setup>
import { Page } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { List } from 'ant-design-vue';

import MfaManagement from './MfaManagement.vue';

const ListItem = List.Item;
const ListItemMeta = List.Item.Meta;

const emit = defineEmits<{ switchTab: [key: string] }>();

// 密保问题/备用邮箱等演示条目已移除：后端无对应能力。
// 手机/邮箱绑定状态展示在“账号绑定”页；修改密码入口跳转对应 tab。
</script>

<template>
  <Page :title="$t('page.user.profile.tab.securitySettings')">
    <List>
      <ListItem>
        <ListItemMeta>
          <template #title>
            {{ $t('page.user.profile.security.password') }}
            <a-button
              type="link"
              size="small"
              class="extra"
              @click="emit('switchTab', '2')"
            >
              {{ $t('page.user.profile.security.modify') }}
            </a-button>
          </template>
          <template #description>
            <div>{{ $t('page.user.profile.security.passwordDesc') }}</div>
          </template>
        </ListItemMeta>
      </ListItem>

      <!-- MFA（TOTP）绑定/解绑：真实功能组件，绑定后登录需二次验证 -->
      <ListItem>
        <MfaManagement />
      </ListItem>
    </List>
  </Page>
</template>

<style lang="less" scoped>
.extra {
  float: right;
  margin-top: 10px;
  margin-right: 30px;
  font-weight: normal;
  cursor: pointer;
}
</style>
