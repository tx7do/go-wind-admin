<template>
  <div v-loading="pageLoading" class="page-container">
    <ElRow :gutter="24">
      <ElCol :span="14">
        <ElForm ref="formRef" :model="formData" label-width="120px" class="profile-form">
          <ElFormItem :label="$t('pages.user.profile.field.nickname')">
            <ElInput
              v-model="formData.nickname"
              :placeholder="$t('common.placeholder.input')"
              clearable
            />
          </ElFormItem>

          <ElFormItem :label="$t('pages.user.profile.field.realname')">
            <ElInput
              v-model="formData.realname"
              :placeholder="$t('common.placeholder.input')"
              clearable
            />
          </ElFormItem>

          <ElFormItem :label="$t('pages.user.profile.field.email')">
            <ElInput
              v-model="formData.email"
              :placeholder="$t('common.placeholder.input')"
              clearable
            />
          </ElFormItem>

          <ElFormItem :label="$t('pages.user.profile.field.mobile')">
            <ElInput
              v-model="formData.mobile"
              :placeholder="$t('common.placeholder.input')"
              clearable
            />
          </ElFormItem>

          <ElFormItem :label="$t('pages.user.profile.field.telephone')">
            <ElInput
              v-model="formData.telephone"
              :placeholder="$t('common.placeholder.input')"
              clearable
            />
          </ElFormItem>

          <ElFormItem :label="$t('pages.user.profile.field.gender')">
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

          <ElFormItem :label="$t('pages.user.profile.field.region')">
            <ElInput
              v-model="formData.region"
              :placeholder="$t('common.placeholder.input')"
              clearable
            />
          </ElFormItem>

          <ElFormItem :label="$t('pages.user.profile.field.address')">
            <ElInput
              v-model="formData.address"
              :placeholder="$t('common.placeholder.input')"
              clearable
            />
          </ElFormItem>

          <ElFormItem :label="$t('pages.user.profile.field.description')">
            <ElInput
              v-model="formData.description"
              type="textarea"
              :rows="3"
              :placeholder="$t('common.placeholder.input')"
            />
          </ElFormItem>

          <!-- 偏好语言：事务性邮件（找回密码/换绑验证码）按此渲染，留空 = 跟随请求头 -->
          <ElFormItem :label="$t('pages.user.profile.field.locale')">
            <ElSelect v-model="formData.locale" clearable style="width: 100%">
              <ElOption label="中文" value="zh-CN" />
              <ElOption label="English" value="en-US" />
            </ElSelect>
          </ElFormItem>
        </ElForm>
      </ElCol>

      <ElCol :span="10">
        <div class="change-avatar">
          <div class="avatar-label">{{ $t("pages.user.profile.field.avatar") }}</div>
          <ElAvatar :size="96" :src="userInfo?.avatar || undefined" class="avatar-preview">
            {{ firstChar }}
          </ElAvatar>
          <div class="avatar-actions">
            <ElUpload
              :show-file-list="false"
              accept="image/*"
              :before-upload="beforeAvatarUpload"
              :http-request="handleUploadAvatar"
            >
              <ElButton type="primary">{{ $t("pages.user.button.uploadAvatar") }}</ElButton>
            </ElUpload>
            <ElButton v-if="userInfo?.avatar" type="danger" @click="handleDeleteAvatar">
              {{ $t("pages.user.button.deleteAvatar") }}
            </ElButton>
          </div>
        </div>

        <!-- 账号归属信息（只读）：自助页里唯一能看到自己的角色/组织/岗位/租户的入口 -->
        <ElDescriptions :column="1" size="small" class="account-info">
          <ElDescriptionsItem :label="$t('pages.user.profile.info.username')">
            {{ userInfo?.username || "-" }}
          </ElDescriptionsItem>
          <ElDescriptionsItem :label="$t('pages.user.profile.info.roleNames')">
            <template v-if="userInfo?.roleNames?.length">
              <ElTag
                v-for="role in userInfo.roleNames"
                :key="role"
                size="small"
                effect="plain"
                class="info-tag"
              >
                {{ role }}
              </ElTag>
            </template>
            <span v-else>-</span>
          </ElDescriptionsItem>
          <ElDescriptionsItem :label="$t('pages.user.profile.info.orgUnitNames')">
            <template v-if="userInfo?.orgUnitNames?.length">
              <ElTag
                v-for="org in userInfo.orgUnitNames"
                :key="org"
                size="small"
                effect="plain"
                class="info-tag"
              >
                {{ org }}
              </ElTag>
            </template>
            <span v-else>-</span>
          </ElDescriptionsItem>
          <ElDescriptionsItem :label="$t('pages.user.profile.info.positionNames')">
            <template v-if="userInfo?.positionNames?.length">
              <ElTag
                v-for="pos in userInfo.positionNames"
                :key="pos"
                size="small"
                effect="plain"
                class="info-tag"
              >
                {{ pos }}
              </ElTag>
            </template>
            <span v-else>-</span>
          </ElDescriptionsItem>
          <ElDescriptionsItem :label="$t('pages.user.profile.info.tenantName')">
            {{ userInfo?.tenantName || "-" }}
          </ElDescriptionsItem>
          <ElDescriptionsItem :label="$t('pages.user.profile.info.lastLoginAt')">
            {{ formatDateTime(userInfo?.lastLoginAt ?? "") || "-" }}
          </ElDescriptionsItem>
          <ElDescriptionsItem :label="$t('pages.user.profile.info.lastLoginIp')">
            {{ userInfo?.lastLoginIp || "-" }}
          </ElDescriptionsItem>
          <ElDescriptionsItem :label="$t('pages.user.profile.info.createdAt')">
            {{ formatDateTime(userInfo?.createdAt ?? "") || "-" }}
          </ElDescriptionsItem>
        </ElDescriptions>
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
import { computed, reactive, ref } from "vue";
import { ElMessage } from "element-plus";
import type { UploadRequestOptions, UploadRawFile } from "element-plus";

import {
  genderList,
  getMe,
  useDeleteAvatar,
  useUpdateUserProfile,
  useUploadAvatar,
} from "@/api/composables";
import { queryClient } from "@/plugins/vue-query";
import { formatDateTime } from "@/utils";
import type { identityservicev1_User } from "@/api/generated/admin/service/v1";
import { $t } from "@/core/i18n";

const { mutateAsync: updateUserProfile } = useUpdateUserProfile();
const { mutateAsync: uploadAvatar } = useUploadAvatar();
const { mutateAsync: deleteAvatar } = useDeleteAvatar();

const currentUserId = ref<string>();
const submitLoading = ref(false);
const pageLoading = ref(true);
const formRef = ref();
const userInfo = ref<identityservicev1_User | null>(null);

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
  locale: "",
});

const firstChar = computed(
  () => userInfo.value?.username?.slice(0, 1).toUpperCase() || "?",
);

// 加载用户信息：资料表单与右侧只读块/头像同源，故直接取 getMe 的完整 User
async function loadUserData() {
  pageLoading.value = true;
  try {
    const data = await getMe();
    if (data) {
      userInfo.value = data;
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
        locale: data.locale || "",
      });
    }
  } catch (error) {
    console.error("Failed to load user data:", error);
  } finally {
    pageLoading.value = false;
  }
}

// 顶栏头像等消费方读的是 getMe / userProfile 两个键，改动后一起失效
function refreshUserProfile() {
  queryClient.invalidateQueries({ queryKey: ["getMe"] });
  queryClient.invalidateQueries({ queryKey: ["userProfile"] });
}

// 提交表单
async function handleSubmit() {
  submitLoading.value = true;

  try {
    await updateUserProfile({ id: currentUserId.value!, values: formData });
    ElMessage.success($t("common.notification.updateSuccess"));
    refreshUserProfile();
  } catch (error: any) {
    console.error("Failed to update profile:", error);
    ElMessage.error(error?.message || $t("common.notification.updateFailed"));
  } finally {
    submitLoading.value = false;
  }
}

/**
 * 头像上传前置校验：仅允许图片，且不超过 5MB（后端上限 50MB，此处更严格以避免无谓的 base64 转换开销）
 */
function beforeAvatarUpload(file: UploadRawFile) {
  const isImage = file.type.startsWith("image/");
  const withinSize = file.size / 1024 / 1024 < 5;
  if (!isImage || !withinSize) {
    ElMessage.error($t("pages.user.profile.notice.uploadFailed"));
    return false;
  }
  return true;
}

/**
 * 头像上传：选中文件转 base64（剥离 dataURL 前缀）后提交。
 * 后端用 base64.StdEncoding.DecodeString 解码，不识别 dataURL 前缀，故必须传纯 base64。
 */
async function handleUploadAvatar(options: UploadRequestOptions) {
  const reader = new FileReader();
  reader.addEventListener("load", async () => {
    try {
      const dataUrl = reader.result as string;
      const base64 = dataUrl.replace(/^data:[^;]+;base64,/, "");
      await uploadAvatar({ imageBase64: base64 });
      await loadUserData();
      refreshUserProfile();
      ElMessage.success($t("pages.user.profile.notice.uploadSuccess"));
    } catch (error) {
      console.error("Failed to upload avatar:", error);
      ElMessage.error($t("pages.user.profile.notice.uploadFailed"));
    }
  });
  reader.addEventListener("error", () => {
    console.error("Failed to read avatar file:", reader.error);
    ElMessage.error($t("pages.user.profile.notice.uploadFailed"));
  });
  reader.readAsDataURL(options.file);
}

async function handleDeleteAvatar() {
  try {
    await deleteAvatar();
    await loadUserData();
    refreshUserProfile();
    ElMessage.success($t("common.notification.deleteSuccess"));
  } catch (error) {
    console.error("Failed to delete avatar:", error);
    ElMessage.error($t("common.notification.deleteFailed"));
  }
}

// 初始化
loadUserData();
</script>

<style lang="scss" scoped>
.page-container {
  width: 100%;
  max-width: 100%;
  overflow-x: hidden;
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

  // 无图时的占位首字母：字号随 96px 头像放大，底色用 fill 档（暗亮两端均由 EP 变量兜底）
  .avatar-preview {
    background-color: var(--el-fill-color);
    font-size: 40px;
    font-weight: 700;
    text-transform: uppercase;
  }

  .avatar-actions {
    display: flex;
    gap: 8px;
    margin-top: 12px;
  }
}

.account-info {
  margin-top: 24px;

  .info-tag {
    margin: 0 4px 4px 0;
  }
}
</style>
