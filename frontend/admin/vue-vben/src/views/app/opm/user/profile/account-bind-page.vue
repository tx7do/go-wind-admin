<script lang="ts" setup>

import { computed, reactive, ref } from 'vue';

import { Page } from '@vben/common-ui';
import { $t } from '@vben/locales';
import { IconifyIcon } from '@vben/icons';

import { List, message } from 'ant-design-vue';

import { apiClient, useGetUserProfile } from '#/api';
import { queryClient } from '#/plugins/vue-query';

const ListItem = List.Item;
const ListItemMeta = List.Item.Meta;

const emit = defineEmits<{ switchTab: [key: string] }>();

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

// ===== 绑定邮箱（发送验证码 → 校验并写入 EMAIL 凭证）=====
const bindOpen = ref(false);
const bindForm = reactive({ email: '', code: '' });
const codeSent = ref(false);
const sending = ref(false);
const verifying = ref(false);

function openBind() {
  bindForm.email = user.value?.email ?? '';
  bindForm.code = '';
  codeSent.value = false;
  bindOpen.value = true;
}

async function handleSendCode() {
  const email = bindForm.email.trim();
  if (!email) {
    message.error($t('page.user.profile.bind.newEmailRequired'));
    return;
  }
  sending.value = true;
  try {
    await apiClient.userProfileService.BindContact({ email: { email } });
    codeSent.value = true;
    message.success($t('page.user.profile.bind.vcodeSent'));
  } catch (error: any) {
    message.error(error?.message || $t('page.user.profile.bind.vcodeSendFailed'));
  } finally {
    sending.value = false;
  }
}

async function handleVerify() {
  const email = bindForm.email.trim();
  if (!email || !bindForm.code) {
    message.error($t('page.user.profile.bind.vcodeRequired'));
    return;
  }
  verifying.value = true;
  try {
    await apiClient.userProfileService.VerifyContact({
      email: { email, code: bindForm.code },
    });
    message.success($t('page.user.profile.bind.bindSuccess'));
    bindOpen.value = false;
    // 刷新用户信息以更新绑定状态展示
    queryClient.invalidateQueries({ queryKey: ['getMe'] });
  } catch (error: any) {
    message.error(error?.message || $t('page.user.profile.bind.bindFailed'));
  } finally {
    verifying.value = false;
  }
}

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
                v-if="item.key === 'email'"
                type="link"
                size="small"
                class="extra"
                @click="openBind"
              >
                {{ user?.email ? $t('page.user.profile.bind.rebind') : $t('page.user.profile.bind.bindNow') }}
              </a-button>
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

    <!-- 绑定邮箱对话框：发送验证码 → 校验并写入 EMAIL 凭证 -->
    <a-modal
      v-model:open="bindOpen"
      :title="$t('page.user.profile.bind.bindEmailTitle')"
      :confirm-loading="verifying"
      destroy-on-close
      @ok="handleVerify"
    >
      <a-form layout="vertical" class="pt-2">
        <a-form-item :label="$t('page.user.profile.bind.newEmail')" required>
          <a-input
            v-model:value="bindForm.email"
            :placeholder="$t('page.user.profile.bind.newEmailPlaceholder')"
          />
        </a-form-item>
        <a-form-item :label="$t('page.user.profile.bind.vcode')" required>
          <div class="flex items-center gap-2">
            <a-input
              v-model:value="bindForm.code"
              :placeholder="$t('page.user.profile.bind.vcodePlaceholder')"
            />
            <a-button :disabled="codeSent" :loading="sending" @click="handleSendCode">
              {{ codeSent ? $t('page.user.profile.bind.codeSent') : $t('page.user.profile.bind.sendVcode') }}
            </a-button>
          </div>
        </a-form-item>
      </a-form>
    </a-modal>
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
