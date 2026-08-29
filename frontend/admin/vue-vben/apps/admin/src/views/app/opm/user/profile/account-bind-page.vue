<script lang="ts" setup>
import { computed } from 'vue';

import { Page } from '@vben/common-ui';
import { $t } from '@vben/locales';
import { IconifyIcon } from '@vben/icons';

import { List } from 'ant-design-vue';

import { useGetUserProfile } from '#/api';

const ListItem = List.Item;
const ListItemMeta = List.Item.Meta;

const emit = defineEmits<{ switchTab: [key: string] }>();

// 第三方账号（GitHub/微信/微博等）绑定条目已移除：
// 后端社交登录未实现，BindContact/VerifyContact 换绑接口亦为占位实现。
// 邮箱/手机展示当前账号的真实绑定状态，修改入口跳转基本设置
// （该页经 UpdateUser 直接更新联系方式，为现行产品行为）。
const { data: user } = useGetUserProfile();

// 图标颜色统一使用主题感知前景色（亮/暗模式自动切换）
const ICON_COLOR = 'hsl(var(--foreground))';

const maskEmail = (email: string) => {
  const at = email.lastIndexOf('@');
  if (at <= 0) return email;
  return `${email.slice(0, Math.min(2, at))}***${email.slice(at)}`;
};

const maskPhone = (phone: string) => {
  if (phone.length < 7) return phone;
  return `${phone.slice(0, 3)}****${phone.slice(-4)}`;
};

interface BindItem {
  key: string;
  title: string;
  description: string;
  avatar: string;
}

const bindList = computed<BindItem[]>(() => {
  const email = user.value?.email ?? '';
  const mobile = user.value?.mobile ?? '';

  return [
    {
      key: 'email',
      title: $t('page.user.profile.bind.email'),
      description: email
        ? $t('page.user.profile.bind.bound', { value: maskEmail(email) })
        : $t('page.user.profile.bind.unbound'),
      avatar: 'ri:mail-fill',
    },
    {
      key: 'phone',
      title: $t('page.user.profile.bind.phone'),
      description: mobile
        ? $t('page.user.profile.bind.bound', { value: maskPhone(mobile) })
        : $t('page.user.profile.bind.unbound'),
      avatar: 'ri:smartphone-fill',
    },
  ];
});
</script>

<template>
  <Page :title="$t('page.user.profile.tab.accountBind')">
    <List>
      <template v-for="item in bindList" :key="item.key">
        <ListItem>
          <ListItemMeta>
            <template #avatar>
              <IconifyIcon
                class="avatar"
                :icon="item.avatar"
                :color="ICON_COLOR"
              />
            </template>
            <template #title>
              {{ item.title }}
              <a-button
                type="link"
                size="small"
                class="extra"
                @click="emit('switchTab', '1')"
              >
                {{ $t('page.user.profile.bind.modify') }}
              </a-button>
            </template>
            <template #description>
              <div>{{ item.description }}</div>
            </template>
          </ListItemMeta>
        </ListItem>
      </template>
    </List>
  </Page>
</template>

<style lang="less" scoped>
.avatar {
  font-size: 40px !important;
}

.extra {
  float: right;
  margin-top: 10px;
  margin-right: 30px;
  cursor: pointer;
}
</style>
