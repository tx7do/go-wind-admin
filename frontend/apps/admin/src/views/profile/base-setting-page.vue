<script lang="ts" setup>
import type { identityservicev1_User as User } from '#/generated/api/admin/service/v1';

import { ref } from 'vue';

import { Page } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { Col, notification, Row } from 'ant-design-vue';

import { useVbenForm } from '#/adapter/form';
import { genderList, useUserProfileStore } from '#/stores';

const userProfileStore = useUserProfileStore();

const data = ref<null | User>();

const [BaseForm, baseFormApi] = useVbenForm({
  showDefaultActions: false,
  commonConfig: {
    componentProps: {
      class: 'w-full',
    },
  },
  schema: [
    {
      fieldName: 'nickname',
      component: 'Input',
      label: $t('page.user.table.nickname'),
    },
    {
      fieldName: 'realname',
      component: 'Input',
      label: $t('page.user.table.realname'),
    },
    {
      fieldName: 'email',
      component: 'Input',
      label: $t('page.user.table.email'),
    },
    {
      fieldName: 'mobile',
      component: 'Input',
      label: $t('page.user.table.mobile'),
    },
    {
      fieldName: 'telephone',
      component: 'Input',
      label: $t('page.user.table.telephone'),
    },
    {
      fieldName: 'gender',
      component: 'Select',
      label: $t('page.user.table.gender'),
      componentProps: {
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        allowClear: true,
        showSearch: true,
        options: genderList,
        placeholder: $t('ui.placeholder.select'),
      },
    },
    {
      fieldName: 'region',
      component: 'Input',
      label: $t('page.user.table.region'),
    },
    {
      fieldName: 'address',
      component: 'Input',
      label: $t('page.user.table.address'),
    },
    {
      fieldName: 'description',
      component: 'Textarea',
      label: $t('page.user.table.description'),
    },
  ],
});

async function handleSubmit() {
  console.log('submit');

  // 校验输入的数据
  const validate = await baseFormApi.validate();
  if (!validate.valid) {
    return;
  }

  setLoading(true);

  // 获取表单数据
  const values = await baseFormApi.getValues();

  try {
    await userProfileStore.updateUser(values);

    notification.success({
      message: $t('ui.notification.update_success'),
    });
  } catch {
    notification.error({
      message: $t('ui.notification.update_failed'),
    });
  } finally {
    setLoading(false);
  }
}

function setLoading(_loading: boolean) {}

/**
 * 重新加载用户信息
 */
async function reload() {
  data.value = await userProfileStore.getMe();
  await baseFormApi.setValues(data.value || {});
}

reload();
</script>

<template>
  <Page
    :title="$t('page.user.profile.tab.basicSettings')"
    :body-style="{ padding: 0 }"
    class="edge-card"
    style="margin: 0"
  >
    <Row :gutter="24">
      <Col :span="14">
        <BaseForm />
      </Col>
      <Col :span="10">
        <div class="change-avatar">
          <div class="mb-2">{{ $t('page.user.table.avatar') }}</div>
        </div>
      </Col>
    </Row>
    <a-button type="primary" @click="handleSubmit">
      {{ $t('page.user.button.updateUserInfo') }}
    </a-button>
  </Page>
</template>

<style lang="less" scoped>
.change-avatar {
  img {
    display: block;
    margin-bottom: 15px;
    border-radius: 50%;
  }
}

.edge-card {
  .ant-card-body {
    padding: 0 !important;
  }
}
</style>
