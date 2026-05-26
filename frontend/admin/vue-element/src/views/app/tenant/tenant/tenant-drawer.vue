<template>
  <ElDrawer
    v-model="visible"
    :title="title"
    size="600px"
    :close-on-click-modal="false"
    :append-to-body="true"
    :destroy-on-close="true"
    @close="handleClose"
  >
    <ElForm :model="formData" label-width="120px">
      <!-- 基本信息 -->
      <ElFormItem :label="$t('pages.tenant.name')" required>
        <ElInput v-model="formData.name" :placeholder="$t('common.placeholder.input')" clearable />
      </ElFormItem>

      <ElFormItem :label="$t('pages.tenant.code')" required>
        <ElInput v-model="formData.code" :placeholder="$t('common.placeholder.input')" clearable />
      </ElFormItem>

      <ElFormItem :label="$t('pages.tenant.type')" required>
        <ElSelect
          v-model="formData.type"
          :placeholder="$t('common.placeholder.select')"
          filterable
          class="w-full"
        >
          <ElOption
            v-for="item in tenantTypeList"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </ElSelect>
      </ElFormItem>

      <ElFormItem :label="$t('pages.tenant.auditStatus')" required>
        <ElSelect
          v-model="formData.auditStatus"
          :placeholder="$t('common.placeholder.select')"
          filterable
          class="w-full"
        >
          <ElOption
            v-for="item in tenantAuditStatusList"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </ElSelect>
      </ElFormItem>

      <ElFormItem :label="$t('common.table.status')" required>
        <ElSelect
          v-model="formData.status"
          :placeholder="$t('common.placeholder.select')"
          filterable
          class="w-full"
        >
          <ElOption
            v-for="item in tenantStatusList"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </ElSelect>
      </ElFormItem>

      <ElFormItem :label="$t('common.table.remark')">
        <ElInput
          v-model="formData.remark"
          type="textarea"
          :rows="3"
          :placeholder="$t('common.placeholder.input')"
        />
      </ElFormItem>

      <!-- 管理员设置（仅创建时显示） -->
      <ElDivider v-if="isCreate">{{ $t("pages.tenant.adminSetting") }}</ElDivider>

      <ElFormItem v-if="isCreate" :label="$t('pages.tenant.adminUserName')" required>
        <ElInput
          v-model="formData.user.username"
          :placeholder="$t('common.placeholder.input')"
          clearable
        />
      </ElFormItem>

      <ElFormItem v-if="isCreate" :label="$t('pages.tenant.adminPassword')" required>
        <ElInput
          v-model="formData.password"
          type="password"
          show-password
          :placeholder="$t('common.placeholder.input')"
        />
      </ElFormItem>

      <ElFormItem v-if="isCreate" :label="$t('pages.tenant.adminPasswordConfirm')" required>
        <ElInput
          v-model="formData.passwordConfirm"
          type="password"
          show-password
          :placeholder="$t('common.placeholder.input')"
        />
      </ElFormItem>

      <ElFormItem v-if="isCreate" :label="$t('pages.tenant.adminMobile')" required>
        <ElInput
          v-model="formData.user.mobile"
          :placeholder="$t('common.placeholder.input')"
          clearable
        />
      </ElFormItem>

      <ElFormItem v-if="isCreate" :label="$t('pages.tenant.adminEmail')" required>
        <ElInput
          v-model="formData.user.email"
          :placeholder="$t('common.placeholder.input')"
          clearable
        />
      </ElFormItem>
    </ElForm>

    <template #footer>
      <div class="drawer-footer">
        <ElButton @click="handleClose">{{ $t("common.button.cancel") }}</ElButton>
        <ElButton type="primary" :loading="loading" @click="handleSubmit">
          {{ $t("common.button.confirm") }}
        </ElButton>
      </div>
    </template>
  </ElDrawer>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";

import { ElMessage } from "element-plus";

import {
  tenantAuditStatusList,
  tenantStatusList,
  tenantTypeList,
  useCreateTenantWithAdminUser,
  useUpdateTenant,
  useUserExists,
  fetchListTenants,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import type { identityservicev1_Tenant as Tenant } from "@/api/generated/admin/service/v1";
import { $t } from "@/i18n";

const emit = defineEmits<{
  (e: "success"): void;
}>();

const { mutateAsync: createTenantWithAdminUserMut } = useCreateTenantWithAdminUser();
const { mutateAsync: updateTenantMut } = useUpdateTenant();
const { mutateAsync: userExists } = useUserExists();

// 弹窗可见性
const visible = ref(false);
// 加载状态
const loading = ref(false);
// 是否为创建模式
const isCreate = ref(false);
// 当前编辑的数据
const currentRow = ref<Tenant | null>(null);

// 表单数据
const formData = ref({
  name: "",
  code: "",
  type: "PAID",
  auditStatus: "APPROVED",
  status: "ON",
  remark: "",
  user: {
    username: "",
    mobile: "",
    email: "",
  },
  password: "",
  passwordConfirm: "",
});

// 弹窗标题
const title = computed(() =>
  isCreate.value
    ? $t("common.modal.create", { moduleName: $t("pages.tenant.moduleName") })
    : $t("common.modal.update", { moduleName: $t("pages.tenant.moduleName") })
);

// 打开弹窗
const open = (row?: Tenant) => {
  visible.value = true;

  if (row) {
    // 编辑模式
    isCreate.value = false;
    currentRow.value = row;
    formData.value = {
      name: row.name || "",
      code: row.code || "",
      type: row.type || "PAID",
      auditStatus: row.auditStatus || "APPROVED",
      status: row.status || "ON",
      remark: row.remark || "",
      user: {
        username: "",
        mobile: "",
        email: "",
      },
      password: "",
      passwordConfirm: "",
    };
  } else {
    // 创建模式
    isCreate.value = true;
    currentRow.value = null;
    resetForm();
  }
};

// 重置表单
const resetForm = () => {
  formData.value = {
    name: "",
    code: "",
    type: "PAID",
    auditStatus: "APPROVED",
    status: "ON",
    remark: "",
    user: {
      username: "",
      mobile: "",
      email: "",
    },
    password: "",
    passwordConfirm: "",
  };
};

// 关闭弹窗
const handleClose = () => {
  visible.value = false;
  resetForm();
};

// 提交表单
const handleSubmit = async () => {
  try {
    loading.value = true;

    // 验证必填字段
    if (!formData.value.name || !formData.value.code) {
      ElMessage.error($t("common.placeholder.input"));
      return;
    }

    if (isCreate.value) {
      await createTenantWithAdminUser();
    } else {
      await updateTenant();
    }

    // 成功回调
    emit("success");
    handleClose();
  } catch (error) {
    console.error("Submit error:", error);
  } finally {
    loading.value = false;
  }
};

// 创建租户和管理员用户
async function createTenantWithAdminUser() {
  // 检查密码和确认密码是否一致
  if (formData.value.password !== formData.value.passwordConfirm) {
    ElMessage.error($t("pages.notification.password_mismatch"));
    return;
  }

  // 检查租户编码是否存在
  try {
    const result = await fetchListTenants(
      new PaginationQuery({ formValues: { code: formData.value.code, name: formData.value.name } })
    );
    if (result.items && result.items.length > 0) {
      throw new Error("Tenant code already exists");
    }
  } catch {
    ElMessage.error($t("pages.tenant.tenant_code_exists"));
    return;
  }

  // 检查用户名是否存在
  try {
    await userExists({ username: formData.value.user.username });
  } catch {
    ElMessage.error($t("pages.tenant.notification.user_username_exists"));
    return;
  }

  await createTenantWithAdminUserMut({
    tenant: {
      name: formData.value.name,
      code: formData.value.code,
      type: formData.value.type as any,
      auditStatus: formData.value.auditStatus as any,
      status: formData.value.status as any,
      remark: formData.value.remark,
    },
    user: formData.value.user as any,
    password: formData.value.password,
  });

  ElMessage.success($t("common.notification.create_success"));
}

// 更新租户
async function updateTenant() {
  if (!currentRow.value?.id) {
    ElMessage.error($t("common.notification.update_failed"));
    return;
  }

  await updateTenantMut({
    id: currentRow.value.id,
    values: {
      name: formData.value.name,
      code: formData.value.code,
      type: formData.value.type,
      auditStatus: formData.value.auditStatus,
      status: formData.value.status,
      remark: formData.value.remark,
    },
  });

  ElMessage.success($t("common.notification.update_success"));
}

// 暴露方法给父组件
defineExpose({
  open,
});
</script>

<style scoped>
.drawer-footer {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}
</style>
