<template>
  <ProModal
    v-model:visible="visible"
    :title="title"
    :config="{ component: 'drawer', drawer: { size: DRAWER_WIDTH } }"
    :close-guard="guard.allowClose"
  >
    <ElForm ref="formRef" :model="formData" :rules="formRules" label-width="120px">
      <!-- 所属套餐（只读，来自主表选中项） -->
      <ElFormItem :label="$t('pages.plan.planId')">
        <ElInput :model-value="String(parentPlanId ?? '')" readonly />
      </ElFormItem>

      <ElFormItem :label="$t('pages.plan.quotaType')" prop="quotaType" required>
        <ElSelect
          v-model="formData.quotaType"
          :placeholder="$t('common.placeholder.select')"
          filterable
          class="w-full"
        >
          <ElOption
            v-for="item in planQuotaTypeList"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </ElSelect>
      </ElFormItem>

      <ElFormItem :label="$t('pages.plan.quotaValue')" prop="quotaValue">
        <ElInputNumber
          v-model="formData.quotaValue"
          :min="0"
          :placeholder="$t('common.placeholder.input')"
          class="w-full"
        />
      </ElFormItem>
    </ElForm>

    <template #footer>
      <div class="drawer-footer">
        <ElButton @click="guard.requestClose(handleClose)">{{ $t("common.button.cancel") }}</ElButton>
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

import { planQuotaTypeList, useCreatePlanQuota, useUpdatePlanQuota } from "@/api/composables";
import type { identityservicev1_PlanQuota as PlanQuota } from "@/api/generated/admin/service/v1";
import { $t } from "@/core/i18n";
import { DRAWER_WIDTH } from "@/constants";
import { injectProModalApi } from "@/components/Pro";
import ProModal from "@/components/Pro/ProModal/index.vue";
import { useDrawerCloseGuard } from "@/components/Pro/composables/use-drawer-close-guard";
import { usePlanViewStore } from "@/pages/app/tenant/plan/plan-view.state";

// 通过 inject 获取列表页传入的 modalApi
const modalApi = injectProModalApi();

// 主表选中套餐 ID：从 store 读取，不经 setData 传入。
const planViewStore = usePlanViewStore();
const parentPlanId = computed(() => planViewStore.currentPlanId);

// 直接读取响应式的 store.sharedData。
const data = computed(() => modalApi.store.sharedData as { create?: boolean; row?: PlanQuota });
const isCreate = computed(() => !!data.value.create);

const visible = computed({
  get: () => modalApi.store.isOpen,
  set: (v) => {
    if (!v) modalApi.close();
  },
});

const { mutateAsync: createPlanQuotaMut } = useCreatePlanQuota();
const { mutateAsync: updatePlanQuotaMut } = useUpdatePlanQuota();

// 加载状态
const loading = ref(false);

// 表单数据
const formData = ref({
  quotaType: "PLAN_QUOTA_TYPE_UNSPECIFIED",
  quotaValue: 0,
});

const formRef = ref<FormInstance>();

// 表单校验规则
const formRules: FormRules = {
  quotaType: [
    { required: true, message: $t("common.validation.selectRequired"), trigger: "change" },
  ],
};

// 弹窗标题
const title = computed(() =>
  isCreate.value
    ? $t("common.modal.create", { moduleName: $t("pages.plan.quotaName") })
    : $t("common.modal.update", { moduleName: $t("pages.plan.quotaName") })
);

// 监听弹窗打开/关闭
// 关闭守卫：遮罩/ESC/X/取消统一，脏数据先确认（docs/form_drawer_close_guard.md）
const guard = useDrawerCloseGuard(() => ({ ...formData.value }));

watch(visible, (val) => {
  if (val) {
    if (!isCreate.value && data.value.row) {
      // 编辑模式
      const row = data.value.row;
      formData.value = {
        quotaType: row.quotaType || "PLAN_QUOTA_TYPE_UNSPECIFIED",
        // uint64 经 protojson 是字符串，表单数值输入框需要数字
        quotaValue: Number(row.quotaValue ?? 0),
      };
    } else {
      // 创建模式
      resetForm();
    }

    // 填充完成后记录基线快照（回填不算脏）
    guard.syncBaseline();
  } else {
    // ProModal 关闭时自动重置表单
    resetForm();
  }
});

// 重置表单
const resetForm = () => {
  formData.value = {
    quotaType: "PLAN_QUOTA_TYPE_UNSPECIFIED",
    quotaValue: 0,
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

    if (!formRef.value) return;
    await formRef.value.validate();

    const planId = parentPlanId.value;
    if (!planId) {
      ElMessage.error($t("common.notification.update_failed"));
      return;
    }

    if (isCreate.value) {
      await createPlanQuotaMut({
        planId,
        quotaType: formData.value.quotaType,
        quotaValue: formData.value.quotaValue,
      });
      ElMessage.success($t("common.notification.createSuccess"));
    } else {
      if (!data.value.row?.id) {
        ElMessage.error($t("common.notification.update_failed"));
        return;
      }
      await updatePlanQuotaMut({
        id: data.value.row!.id,
        values: {
          planId,
          quotaType: formData.value.quotaType,
          quotaValue: formData.value.quotaValue,
        },
      });
      ElMessage.success($t("common.notification.update_success"));
    }

    modalApi.close();
  } catch (error) {
    console.error("Submit error:", error);
  } finally {
    loading.value = false;
  }
};
</script>

<style scoped>
.drawer-footer {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}
</style>
