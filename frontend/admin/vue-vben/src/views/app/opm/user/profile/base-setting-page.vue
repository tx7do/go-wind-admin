<script lang="ts" setup>
import type { identityservicev1_User as User } from '#/api';

import { ref } from 'vue';

import { Page } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { Avatar, Col, notification, Row, Upload } from 'ant-design-vue';

import { useVbenForm } from '#/adapter/form';
import {
  genderList,
  getMe,
  useDeleteAvatar,
  useUpdateUserProfile,
  useUploadAvatar,
} from '#/api';

const { mutateAsync: updateUserProfile } = useUpdateUserProfile();
const { mutateAsync: uploadAvatar } = useUploadAvatar();
const { mutateAsync: deleteAvatar } = useDeleteAvatar();

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

  // 校验输入的数据
  const validate = await baseFormApi.validate();
  if (!validate.valid) {
    return;
  }

  setLoading(true);

  // 获取表单数据
  const values = await baseFormApi.getValues();

  try {
    await updateUserProfile({ id: data.value!.id!, values });

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

/**
 * 头像上传前置校验：仅允许图片，且不超过 5MB（后端上限 50MB，此处更严格以避免无谓的 base64 转换开销）
 */
function beforeAvatarUpload(file: File): boolean {
  const isImage = file.type.startsWith('image/');
  const withinSize = file.size / 1024 / 1024 < 5;
  if (!isImage || !withinSize) {
    notification.error({
      message: $t('ui.notification.upload_failed'),
    });
    return false;
  }
  return true;
}

/**
 * 头像上传：将选中文件转为 base64（剥离 dataURL 前缀），调用 UploadAvatar，成功后刷新预览。
 * 后端用 base64.StdEncoding.DecodeString 解码，不识别 dataURL 前缀，故必须传纯 base64。
 */
function handleUploadAvatar(options: any) {
  const { file, onSuccess, onError } = options;

  const reader = new FileReader();
  reader.addEventListener('load', async (e) => {
    try {
      const dataUrl = e.target?.result as string;
      // 剥离 "data:image/...;base64," 前缀，得到纯 base64
      const base64 = dataUrl.replace(/^data:[^;]+;base64,/, '');
      await uploadAvatar({ imageBase64: base64 });
      await reload();
      onSuccess?.({}, file);
      notification.success({
        message: $t('ui.notification.upload_success'),
      });
    } catch (error) {
      try {
        onError?.(error, file);
      } catch {}
      notification.error({
        message: $t('ui.notification.upload_failed'),
      });
    }
  });
  reader.addEventListener('error', () => {
    try {
      onError?.(reader.error, file);
    } catch {}
    notification.error({
      message: $t('ui.notification.upload_failed'),
    });
  });
  reader.readAsDataURL(file);
}

/**
 * 删除头像：调用 DeleteAvatar，成功后刷新预览。
 */
async function handleDeleteAvatar() {
  try {
    await deleteAvatar();
    await reload();
    notification.success({
      message: $t('ui.notification.delete_success'),
    });
  } catch {
    notification.error({
      message: $t('ui.notification.delete_failed'),
    });
  }
}

function setLoading(_loading: boolean) {}

/**
 * 重新加载用户信息
 */
async function reload() {
  data.value = await getMe();
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
        <div class="mb-2">{{ $t('page.user.table.avatar') }}</div>
        <Avatar :src="data?.avatar ?? ''" class="avatar-preview">
          <span class="avatar-placeholder">
            {{ data?.username?.substring(0, 1) || '?' }}
          </span>
        </Avatar>
        <div class="avatar-actions">
          <Upload
            :custom-request="handleUploadAvatar"
            :before-upload="beforeAvatarUpload"
            :show-upload-list="false"
            :multiple="false"
            accept="image/*"
          >
            <a-button type="primary">
              {{ $t('page.user.button.uploadAvatar') }}
            </a-button>
          </Upload>
          <a-button v-if="data?.avatar" danger @click="handleDeleteAvatar">
            {{ $t('page.user.button.deleteAvatar') }}
          </a-button>
        </div>
      </Col>
    </Row>
    <a-button type="primary" @click="handleSubmit">
      {{ $t('page.user.button.updateUserInfo') }}
    </a-button>
  </Page>
</template>

<style lang="less" scoped>
.avatar-preview {
  flex-shrink: 0;
  width: 96px;
  height: 96px;
  border-radius: 50%;

  // antd Avatar 无图时的默认底色是库内写死的 #ccc，配它自己的白字只有 1.60:1；
  // 占位字母要落到主题色上（与 detail/basic-info-page.vue 的 .avatar-placeholder 同规则）。
  background: hsl(var(--muted));
}

.avatar-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 40px;
  font-weight: 700;
  color: hsl(var(--foreground));
  line-height: 1;
  text-transform: uppercase;
}

.avatar-actions {
  display: flex;
  gap: 8px;
  margin-top: 12px;
}

.edge-card {
  .ant-card-body {
    padding: 0 !important;
  }
}
</style>
