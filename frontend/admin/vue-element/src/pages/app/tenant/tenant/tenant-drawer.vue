<template>
  <ProModal
    v-model:visible="visible"
    :title="title"
    :config="{ component: 'drawer', drawer: { size: DRAWER_WIDTH, closeOnClickModal: false } }"
  >
    <ElForm ref="formRef" :model="formData" :rules="formRules" label-width="120px">
      <!-- 基本信息 -->
      <ElFormItem :label="$t('pages.tenant.name')" prop="name" required>
        <ElInput v-model="formData.name" :placeholder="$t('common.placeholder.input')" clearable />
      </ElFormItem>

      <ElFormItem :label="$t('pages.tenant.code')" prop="code" required>
        <ElInput v-model="formData.code" :placeholder="$t('common.placeholder.input')" clearable />
      </ElFormItem>

      <ElFormItem :label="$t('pages.tenant.type')" prop="type" required>
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

      <ElFormItem :label="$t('pages.tenant.auditStatus')" prop="auditStatus" required>
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

      <ElFormItem :label="$t('common.table.status')" prop="status" required>
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

      <ElFormItem :label="$t('pages.tenant.subscriptionPlan')" prop="subscriptionPlan">
        <ElSelect
          v-model="formData.subscriptionPlan"
          :placeholder="$t('common.placeholder.select')"
          filterable
          clearable
          class="w-full"
        >
          <ElOption
            v-for="item in planOptions"
            :key="item.id"
            :label="item.name"
            :value="item.id as number"
          />
        </ElSelect>
      </ElFormItem>

      <ElFormItem :label="$t('pages.tenant.expiredAt')" prop="expiredAt">
        <ElDatePicker
          v-model="formData.expiredAt"
          type="datetime"
          :placeholder="$t('common.placeholder.select')"
          class="w-full"
        />
      </ElFormItem>

      <!-- 编辑模式下显示用量与配额 + 清理数据按钮 -->
      <template v-if="!isCreate && usageData">
        <ElDivider>{{ $t("pages.tenant.usageSection") }}</ElDivider>
        <ElDescriptions :column="1" size="small" border>
          <ElDescriptionsItem :label="$t('pages.tenant.usageUserCount')">
            {{ usageData.userCount ?? 0 }}
          </ElDescriptionsItem>
          <ElDescriptionsItem :label="$t('pages.tenant.usageStorage')">
            {{ usageData.storageUsedBytes ?? 0 }} bytes
          </ElDescriptionsItem>
          <ElDescriptionsItem :label="$t('pages.tenant.usageApiCalls')">
            {{ usageData.apiCallCount ?? 0 }}
          </ElDescriptionsItem>
          <ElDescriptionsItem :label="$t('pages.tenant.usagePlan')">
            {{ usageData.planName ?? $t("pages.tenant.usageNoPlan") }}
          </ElDescriptionsItem>
        </ElDescriptions>
        <div v-if="usageData.quotas && usageData.quotas.length" style="margin-top: 16px">
          <div v-for="(q, idx) in usageData.quotas" :key="idx" style="margin-bottom: 12px">
            <div style="margin-bottom: 4px">
              {{ getQuotaLabel(q.quotaType) }}: {{ getQuotaCurrent(q.quotaType) }} /
              {{ q.quotaValue ?? 0 }}
            </div>
            <ElProgress
              :percentage="getQuotaPercent(q.quotaType, q.quotaValue ?? 0)"
              :stroke-width="6"
            />
          </div>
        </div>
        <div style="margin-top: 24px">
          <ElPopconfirm
            :title="$t('pages.tenant.cleanupConfirmTitle')"
            :content="$t('pages.tenant.cleanupConfirmDesc')"
            confirm-button-type="danger"
            @confirm="handleCleanup"
          >
            <template #reference>
              <ElButton type="danger" :loading="cleanupLoading">
                {{ $t("pages.tenant.cleanupData") }}
              </ElButton>
            </template>
          </ElPopconfirm>
        </div>
      </template>

      <!-- 管理员设置（仅创建时显示） -->
      <ElDivider v-if="isCreate">{{ $t("pages.tenant.adminSetting") }}</ElDivider>

      <ElFormItem
        v-if="isCreate"
        :label="$t('pages.tenant.adminUserName')"
        prop="user.username"
        required
      >
        <ElInput
          v-model="formData.user.username"
          :placeholder="$t('common.placeholder.input')"
          clearable
        />
      </ElFormItem>

      <ElFormItem
        v-if="isCreate"
        :label="$t('pages.tenant.adminPassword')"
        prop="password"
        required
      >
        <ElInput
          v-model="formData.password"
          type="password"
          show-password
          :placeholder="$t('common.placeholder.input')"
        />
      </ElFormItem>

      <ElFormItem
        v-if="isCreate"
        :label="$t('pages.tenant.adminPasswordConfirm')"
        prop="passwordConfirm"
        required
      >
        <ElInput
          v-model="formData.passwordConfirm"
          type="password"
          show-password
          :placeholder="$t('common.placeholder.input')"
        />
      </ElFormItem>

      <ElFormItem
        v-if="isCreate"
        :label="$t('pages.tenant.adminMobile')"
        prop="user.mobile"
        required
      >
        <ElInput
          v-model="formData.user.mobile"
          :placeholder="$t('common.placeholder.input')"
          clearable
        />
      </ElFormItem>

      <ElFormItem v-if="isCreate" :label="$t('pages.tenant.adminEmail')" prop="user.email" required>
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
  </ProModal>
</template>

<script lang="ts" setup>
import { computed, ref, watch } from "vue";

import { ElMessage } from "element-plus";
import type { FormInstance, FormRules } from "element-plus";

import {
  tenantAuditStatusList,
  tenantStatusList,
  tenantTypeList,
  useCreateTenantWithAdminUser,
  useUpdateTenant,
  useUserExists,
  fetchTenantUsage,
  useCleanupTenantData,
  fetchListTenants,
  fetchListPlans,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import type { identityservicev1_Plan as Plan } from "@/api/generated/admin/service/v1";
import type { identityservicev1_Tenant as Tenant } from "@/api/generated/admin/service/v1";
import { $t } from "@/core/i18n";
import { DRAWER_WIDTH } from "@/constants";
import { injectProModalApi } from "@/components/Pro";
import ProModal from "@/components/Pro/ProModal/index.vue";

// 通过 inject 获取列表页传入的 modalApi
const modalApi = injectProModalApi();

// 注意：不能使用 modalApi.getData()，其内部对 toRaw(store).sharedData 取值会丢失响应式，
// 导致多次 open（新增↔编辑切换、编辑不同租户）时 data/isCreate 停留在首次快照，造成串数据。
// 这里直接读取响应式的 store.sharedData。
const data = computed(() => modalApi.store.sharedData as { create?: boolean; row?: Tenant });
const isCreate = computed(() => !!data.value.create);

const visible = computed({
  get: () => modalApi.store.isOpen,
  set: (v) => {
    if (!v) modalApi.close();
  },
});

const { mutateAsync: createTenantWithAdminUserMut } = useCreateTenantWithAdminUser();
const { mutateAsync: updateTenantMut } = useUpdateTenant();
const { mutateAsync: userExists } = useUserExists();

// 编辑模式下按需加载租户用量数据（usageData）。
// 不在 setup 顶层发起请求：仅当抽屉打开且选中真实租户行时，用 fetchTenantUsage 命令式拉取，
// 避免 row.id 为空时以 id=0 发请求触发 400。与 vue-vben 实现保持一致。
const usageData = ref<any>(null);
const cleanupLoading = ref(false);
const { mutateAsync: cleanupMut } = useCleanupTenantData();

// 辅助函数：根据配额类型返回标签/当前值/百分比。

function getQuotaLabel(qt: any): string {
  if (qt === 1) return $t("pages.tenant.usageUserCount");
  if (qt === 2) return $t("pages.tenant.usageStorage");
  if (qt === 3) return $t("pages.tenant.usageApiCalls");
  return $t("pages.tenant.quotaType");
}
function getQuotaCurrent(qt: any): number {
  const ud = usageData.value as any;
  if (!ud) return 0;
  if (qt === 1) return ud.userCount ?? 0;
  if (qt === 2) return ud.storageUsedBytes ?? 0;
  if (qt === 3) return ud.apiCallCount ?? 0;
  return 0;
}
function getQuotaPercent(qt: any, limit: number): number {
  const current = getQuotaCurrent(qt);
  if (limit <= 0) return 0;
  return Math.min(100, (current / limit) * 100);
}

async function handleCleanup() {
  if (!data.value.row?.id) return;
  try {
    cleanupLoading.value = true;
    await cleanupMut({ id: data.value.row.id });
    ElMessage.success($t("pages.tenant.cleanupSuccess"));
    modalApi.close();
  } catch (err: any) {
    ElMessage.error(err?.message || $t("pages.tenant.cleanupFailed"));
  } finally {
    cleanupLoading.value = false;
  }
}

// 加载状态
const loading = ref(false);

// 套餐下拉选项（订阅套餐）
const planOptions = ref<(Plan & { id?: number; name?: string })[]>([]);

// 表单数据
const formData = ref({
  name: "",
  code: "",
  type: "PAID",
  auditStatus: "APPROVED",
  status: "ON",
  remark: "",
  subscriptionPlan: undefined as number | undefined,
  expiredAt: "" as string,
  user: {
    username: "",
    mobile: "",
    email: "",
  },
  password: "",
  passwordConfirm: "",
});

const formRef = ref<FormInstance>();

// 表单校验规则（补齐 required + 格式校验，与 React 版对齐）
const formRules: FormRules = {
  name: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  code: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  type: [{ required: true, message: $t("common.validation.selectRequired"), trigger: "change" }],
  auditStatus: [
    { required: true, message: $t("common.validation.selectRequired"), trigger: "change" },
  ],
  status: [{ required: true, message: $t("common.validation.selectRequired"), trigger: "change" }],
  "user.username": [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  "user.mobile": [
    { required: true, message: $t("common.validation.required"), trigger: "blur" },
    {
      pattern: /^1[3-9]\d{9}$/,
      message: $t("common.validation.phoneFormat"),
      trigger: "blur",
    },
  ],
  "user.email": [
    { required: true, message: $t("common.validation.required"), trigger: "blur" },
    { type: "email", message: $t("common.validation.email"), trigger: "blur" },
  ],
  password: [
    { required: true, message: $t("common.validation.required"), trigger: "blur" },
    { min: 6, message: $t("common.validation.passwordMin"), trigger: "blur" },
  ],
  passwordConfirm: [
    { required: true, message: $t("common.validation.required"), trigger: "blur" },
    { min: 6, message: $t("common.validation.passwordMin"), trigger: "blur" },
  ],
};

// 弹窗标题
const title = computed(() =>
  isCreate.value
    ? $t("common.modal.create", { moduleName: $t("pages.tenant.moduleName") })
    : $t("common.modal.update", { moduleName: $t("pages.tenant.moduleName") })
);

// 监听弹窗打开/关闭
watch(visible, async (val) => {
  if (val) {
    if (!isCreate.value && data.value.row) {
      // 编辑模式
      const row = data.value.row;
      formData.value = {
        name: row.name || "",
        code: row.code || "",
        type: row.type || "PAID",
        auditStatus: row.auditStatus || "APPROVED",
        status: row.status || "ON",
        remark: row.remark || "",
        subscriptionPlan: row.planId,
        expiredAt: "",
        user: {
          username: "",
          mobile: "",
          email: "",
        },
        password: "",
        passwordConfirm: "",
      };

      // 加载该租户的用量与配额对比数据
      // 仅当存在真实租户 ID 时按需拉取，避免 id=0 触发 400
      const rowId = data.value.row?.id;
      if (rowId) {
        try {
          const res = await fetchTenantUsage({ id: rowId });
          usageData.value = res ?? null;
        } catch {
          usageData.value = null;
        }
      } else {
        usageData.value = null;
      }
    } else {
      // 创建模式
      resetForm();
      usageData.value = null;
    }

    // 加载套餐下拉选项（订阅套餐）
    try {
      const res = await fetchListPlans(new PaginationQuery({ paging: { page: 1, pageSize: 100 } }));
      planOptions.value = (res.items || []) as (Plan & { id?: number; name?: string })[];
    } catch (err) {
      console.error("Failed to load plan options:", err);
      planOptions.value = [];
    }
  } else {
    // ProModal 关闭时自动重置表单
    resetForm();
  }
});

// 重置表单
const resetForm = () => {
  formData.value = {
    name: "",
    code: "",
    type: "PAID",
    auditStatus: "APPROVED",
    status: "ON",
    remark: "",
    subscriptionPlan: undefined,
    expiredAt: "",
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
  modalApi.close();
  resetForm();
};

// 提交表单
const handleSubmit = async () => {
  try {
    loading.value = true;

    // 表单校验（required + 手机/邮箱格式 + 密码长度，由 formRules 驱动）
    if (!formRef.value) return;
    await formRef.value.validate();

    if (isCreate.value) {
      await createTenantWithAdminUser();
    } else {
      await updateTenant();
    }

    // 成功回调
    modalApi.close();
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
      new PaginationQuery({ formValues: { code: formData.value.code } })
    );
    if (result.items && result.items.length > 0) {
      // 查询成功且存在重复，提示编码已存在
      ElMessage.error($t("pages.tenant.tenant_code_exists"));
      return;
    }
  } catch {
    // 查询本身失败（网络/500 等），不能误判为编码已存在
    ElMessage.error($t("common.operationFailed"));
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
  if (!data.value.row?.id) {
    ElMessage.error($t("common.notification.update_failed"));
    return;
  }

  await updateTenantMut({
    id: data.value.row!.id,
    values: {
      name: formData.value.name,
      code: formData.value.code,
      type: formData.value.type,
      auditStatus: formData.value.auditStatus,
      status: formData.value.status,
      remark: formData.value.remark,
      planId: formData.value.subscriptionPlan,
      expiredAt: formData.value.expiredAt,
    },
  });

  ElMessage.success($t("common.notification.update_success"));
}
</script>

<style scoped>
.drawer-footer {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}
</style>
