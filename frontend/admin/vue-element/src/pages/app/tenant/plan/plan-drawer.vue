<template>
  <ProModal
    v-model:visible="visible"
    :title="title"
    :config="{ component: 'drawer', drawer: { size: DRAWER_WIDTH, closeOnClickModal: false } }"
  >
    <ElForm ref="formRef" :model="formData" :rules="formRules" label-width="120px">
      <ElFormItem :label="$t('pages.plan.name')" prop="name" required>
        <ElInput v-model="formData.name" :placeholder="$t('common.placeholder.input')" clearable />
      </ElFormItem>

      <ElFormItem :label="$t('pages.plan.version')" prop="version" required>
        <ElSelect
          v-model="formData.version"
          :placeholder="$t('common.placeholder.select')"
          filterable
          class="w-full"
        >
          <ElOption
            v-for="item in planVersionList"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </ElSelect>
      </ElFormItem>

      <ElFormItem :label="$t('pages.plan.expiryPolicy')" prop="expiryPolicy" required>
        <ElSelect
          v-model="formData.expiryPolicy"
          :placeholder="$t('common.placeholder.select')"
          filterable
          class="w-full"
        >
          <ElOption
            v-for="item in planExpiryPolicyList"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </ElSelect>
      </ElFormItem>

      <ElFormItem :label="$t('pages.plan.moduleWhitelist')" prop="moduleWhitelist">
        <ElSelect
          v-model="formData.moduleWhitelist"
          multiple
          filterable
          :placeholder="$t('pages.plan.moduleWhitelistPlaceholder')"
          class="w-full"
        >
          <ElOption
            v-for="item in planModuleList"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </ElSelect>
      </ElFormItem>

      <ElFormItem :label="$t('pages.plan.dataRetentionDays')" prop="dataRetentionDays">
        <ElInputNumber
          v-model="formData.dataRetentionDays"
          :min="0"
          :placeholder="$t('common.placeholder.input')"
          class="w-full"
        />
      </ElFormItem>

      <ElFormItem :label="$t('common.table.description')" prop="description">
        <ElInput
          v-model="formData.description"
          type="textarea"
          :rows="3"
          :placeholder="$t('common.placeholder.input')"
        />
      </ElFormItem>

      <ElFormItem :label="$t('common.table.remark')" prop="remark">
        <ElInput
          v-model="formData.remark"
          type="textarea"
          :rows="3"
          :placeholder="$t('common.placeholder.input')"
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
  planExpiryPolicyList,
  planVersionList,
  planModuleList,
  useCreatePlan,
  useUpdatePlan,
  useListPlanModules,
  useCreatePlanModule,
  useDeletePlanModule,
  fetchListPlanModules,
} from "@/api/composables";
import type { identityservicev1_Plan as Plan } from "@/api/generated/admin/service/v1";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/core/i18n";
import { DRAWER_WIDTH } from "@/constants";
import { injectProModalApi } from "@/components/Pro";
import ProModal from "@/components/Pro/ProModal/index.vue";

// 通过 inject 获取列表页传入的 modalApi
const modalApi = injectProModalApi();

// 注意：不能使用 modalApi.getData()，其内部对 toRaw(store).sharedData 取值会丢失响应式，
// 导致多次 open（新增↔编辑切换、编辑不同套餐）时 data/isCreate 停留在首次快照，造成串数据。
// 这里直接读取响应式的 store.sharedData。
const data = computed(() => modalApi.store.sharedData as { create?: boolean; row?: Plan });
const isCreate = computed(() => !!data.value.create);

const visible = computed({
  get: () => modalApi.store.isOpen,
  set: (v) => {
    if (!v) modalApi.close();
  },
});

const { mutateAsync: createPlanMut } = useCreatePlan();
const { mutateAsync: updatePlanMut } = useUpdatePlan();
const { mutateAsync: createModuleMut } = useCreatePlanModule();
const { mutateAsync: deleteModuleMut } = useDeletePlanModule();

// 加载状态
const loading = ref(false);

// 当前套餐已有的模块白名单（编辑模式下从后端加载，用于 diff）
const existingModules = ref<string[]>([]);

// 表单数据
const formData = ref({
  name: "",
  version: "PLAN_VERSION_UNSPECIFIED",
  expiryPolicy: "PLAN_EXPIRY_POLICY_UNSPECIFIED",
  dataRetentionDays: 0,
  description: "",
  remark: "",
  moduleWhitelist: [] as string[],
});

const formRef = ref<FormInstance>();

// 表单校验规则
const formRules: FormRules = {
  name: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  version: [{ required: true, message: $t("common.validation.selectRequired"), trigger: "change" }],
  expiryPolicy: [
    { required: true, message: $t("common.validation.selectRequired"), trigger: "change" },
  ],
};

// 弹窗标题
const title = computed(() =>
  isCreate.value
    ? $t("common.modal.create", { moduleName: $t("pages.plan.moduleName") })
    : $t("common.modal.update", { moduleName: $t("pages.plan.moduleName") })
);

// 监听弹窗打开/关闭
watch(visible, async (val) => {
  if (val) {
    if (!isCreate.value && data.value.row) {
      // 编辑模式
      const row = data.value.row;
      formData.value = {
        name: row.name || "",
        version: row.version || "PLAN_VERSION_UNSPECIFIED",
        expiryPolicy: row.expiryPolicy || "PLAN_EXPIRY_POLICY_UNSPECIFIED",
        dataRetentionDays: row.dataRetentionDays ?? 0,
        description: row.description || "",
        remark: row.remark || "",
        moduleWhitelist: [],
      };

      // 加载该套餐已有的模块白名单
      try {
        const resp = await fetchListPlanModules(
          new PaginationQuery({
            paging: { page: 1, pageSize: 999 },
            formValues: { plan_id: row.id },
          })
        );
        const names: string[] = [];
        if (resp?.items) {
          for (const item of resp.items) {
            const mod = (item as any)?.module;
            if (typeof mod === "string" && mod.length > 0) {
              names.push(mod);
            }
          }
        }
        existingModules.value = names;
        formData.value.moduleWhitelist = [...names];
      } catch {
        existingModules.value = [];
      }
    } else {
      // 创建模式
      resetForm();
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
    version: "PLAN_VERSION_UNSPECIFIED",
    expiryPolicy: "PLAN_EXPIRY_POLICY_UNSPECIFIED",
    dataRetentionDays: 0,
    description: "",
    remark: "",
    moduleWhitelist: [],
  };
  existingModules.value = [];
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

    if (isCreate.value) {
      await createPlanMut({
        name: formData.value.name,
        version: formData.value.version,
        expiryPolicy: formData.value.expiryPolicy,
        dataRetentionDays: formData.value.dataRetentionDays,
        description: formData.value.description,
        remark: formData.value.remark,
      });
      ElMessage.success($t("common.notification.createSuccess"));
    } else {
      if (!data.value.row?.id) {
        ElMessage.error($t("common.notification.update_failed"));
        return;
      }
      await updatePlanMut({
        id: data.value.row!.id,
        values: {
          name: formData.value.name,
          version: formData.value.version,
          expiryPolicy: formData.value.expiryPolicy,
          dataRetentionDays: formData.value.dataRetentionDays,
          description: formData.value.description,
          remark: formData.value.remark,
        },
      });
      ElMessage.success($t("common.notification.update_success"));

      // 同步模块白名单 diff（编辑模式）
      const planId = data.value.row!.id;
      const selected = formData.value.moduleWhitelist;
      const toAdd = selected.filter((m) => !existingModules.value.includes(m));
      const toRemove = existingModules.value.filter((m) => !selected.includes(m));

      // 查找已有模块记录以获取删除所需的 ID
      let existingItems: any[] = [];
      try {
        const resp = await fetchListPlanModules(
          new PaginationQuery({
            paging: { page: 1, pageSize: 999 },
            formValues: { plan_id: planId },
          })
        );
        existingItems = resp?.items || [];
      } catch {
        existingItems = [];
      }

      for (const mod of toAdd) {
        await createModuleMut({ planId, module: mod as any } as any);
      }
      for (const mod of toRemove) {
        const item = existingItems.find((it: any) => it?.module === mod);
        if (item?.id) {
          await deleteModuleMut({ id: item.id } as any);
        }
      }
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
