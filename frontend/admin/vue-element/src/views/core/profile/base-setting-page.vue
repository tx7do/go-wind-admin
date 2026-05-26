<template>
  <div class="page-container">
    <ElRow :gutter="24">
      <ElCol :span="14">
        <ElForm ref="formRef" :model="formData" label-width="120px" class="profile-form">
          <ElFormItem :label="$t('pages.user.table.nickname')">
            <ElInput
              v-model="formData.nickname"
              :placeholder="$t('common.placeholder.input')"
              clearable
            />
          </ElFormItem>

          <ElFormItem :label="$t('pages.user.table.realname')">
            <ElInput
              v-model="formData.realname"
              :placeholder="$t('common.placeholder.input')"
              clearable
            />
          </ElFormItem>

          <ElFormItem :label="$t('pages.user.table.email')">
            <ElInput
              v-model="formData.email"
              :placeholder="$t('common.placeholder.input')"
              clearable
            />
          </ElFormItem>

          <ElFormItem :label="$t('pages.user.table.mobile')">
            <ElInput
              v-model="formData.mobile"
              :placeholder="$t('common.placeholder.input')"
              clearable
            />
          </ElFormItem>

          <ElFormItem :label="$t('pages.user.table.telephone')">
            <ElInput
              v-model="formData.telephone"
              :placeholder="$t('common.placeholder.input')"
              clearable
            />
          </ElFormItem>

          <ElFormItem :label="$t('pages.user.table.gender')">
            <ElSelect
              v-model="formData.gender"
              :placeholder="$t('common.placeholder.select')"
              filterable
              clearable
              style="width: 100%"
            >
              <ElOption
                v-for="item in genderList"
                :key="item.value"
                :label="item.label"
                :value="item.value"
              />
            </ElSelect>
          </ElFormItem>

          <ElFormItem :label="$t('pages.user.table.region')">
            <ElInput
              v-model="formData.region"
              :placeholder="$t('common.placeholder.input')"
              clearable
            />
          </ElFormItem>

          <ElFormItem :label="$t('pages.user.table.address')">
            <ElInput
              v-model="formData.address"
              :placeholder="$t('common.placeholder.input')"
              clearable
            />
          </ElFormItem>

          <ElFormItem :label="$t('pages.user.table.description')">
            <ElInput
              v-model="formData.description"
              type="textarea"
              :rows="3"
              :placeholder="$t('common.placeholder.input')"
            />
          </ElFormItem>
        </ElForm>
      </ElCol>

      <ElCol :span="10">
        <div class="change-avatar">
          <div class="avatar-label">{{ $t("pages.user.table.avatar") }}</div>
          <!-- 头像显示区域（可根据需要添加上传功能） -->
          <div class="avatar-placeholder">
            <ElIcon :size="64" color="#909399">
              <UserFilled />
            </ElIcon>
          </div>
        </div>
      </ElCol>
    </ElRow>

    <div class="form-actions">
      <ElButton type="primary" :loading="submitLoading" @click="handleSubmit">
        {{ $t("pages.user.button.updateUserInfo") }}
      </ElButton>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { reactive, ref } from "vue";
import { ElMessage } from "element-plus";
import { UserFilled } from "@element-plus/icons-vue";

import { genderList, fetchUserProfile, useUpdateUserProfile } from "@/api/composables";
import { $t } from "@/i18n";

const { mutateAsync: updateUserProfile } = useUpdateUserProfile();

const currentUserId = ref<number>();
const submitLoading = ref(false);
const formRef = ref();

// 表单数据
const formData = reactive({
  nickname: "",
  realname: "",
  email: "",
  mobile: "",
  telephone: "",
  gender: undefined as string | undefined,
  region: "",
  address: "",
  description: "",
});

// 加载用户信息
async function loadUserData() {
  try {
    const data = await fetchUserProfile();
    if (data) {
      currentUserId.value = data.id;
      Object.assign(formData, {
        nickname: data.nickname || "",
        realname: data.realname || "",
        email: data.email || "",
        mobile: data.mobile || "",
        telephone: data.telephone || "",
        gender: data.gender,
        region: data.region || "",
        address: data.address || "",
        description: data.description || "",
      });
    }
  } catch (_error) {
    console.error("Failed to load user data:", _error);
  }
}

// 提交表单
async function handleSubmit() {
  submitLoading.value = true;

  try {
    await updateUserProfile({ id: currentUserId.value!, values: formData });
    ElMessage.success($t("common.notification.updateSuccess"));
  } catch {
    ElMessage.error($t("common.notification.updateFailed"));
  } finally {
    submitLoading.value = false;
  }
}

// 初始化
loadUserData();
</script>

<style lang="scss" scoped>
.page-container {
  width: 100%;
  max-width: 1200px;
}

.profile-form {
  padding-top: 20px;
}

.form-actions {
  padding-top: 20px;
  text-align: left;
}

.change-avatar {
  padding-top: 20px;

  .avatar-label {
    font-size: 14px;
    font-weight: 500;
    margin-bottom: 16px;
  }

  .avatar-placeholder {
    width: 120px;
    height: 120px;
    border: 1px dashed var(--el-border-color);
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    background-color: var(--el-fill-color-light);
  }
}
</style>
