<template>
  <ProModal
    v-model:visible="drawer.visible.value"
    :close-guard="drawer.closeGuard"
    :title="drawer.title.value"
    :loading="drawer.pageLoading.value"
    :config="{
      component: 'drawer',
      drawer: { size: drawer.drawerWidth },
    }"
  >
    <ElForm
      ref="formRef"
      :model="drawer.formData"
      :rules="formRules"
      label-width="120px"
      class="drawer-form"
    >
      <!-- 基本信息 -->
      <ElDivider content-position="left">{{ $t("common.section.basic") }}</ElDivider>

      <ElFormItem :label="$t('pages.position.name')" prop="name">
        <ElInput
          v-model="drawer.formData.name"
          :placeholder="$t('common.placeholder.input')"
          clearable
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.position.code')" prop="code">
        <ElInput
          v-model="drawer.formData.code"
          :placeholder="$t('common.placeholder.input')"
          clearable
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.position.type')" prop="type">
        <ElSelect
          v-model="drawer.formData.type"
          :placeholder="$t('common.placeholder.select')"
          filterable
          clearable
          style="width: 100%"
        >
          <ElOption
            v-for="item in positionTypeList"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </ElSelect>
      </ElFormItem>

      <ElFormItem :label="$t('pages.position.orgUnit')" prop="orgUnitId">
        <ElTreeSelect
          v-model="drawer.formData.orgUnitId"
          :data="orgUnitTreeData"
          node-key="id"
          check-strictly
          :render-after-expand="false"
          default-expand-all
          filterable
          clearable
          :props="{ label: 'name', value: 'id', children: 'children' } as any"
          :placeholder="$t('common.placeholder.select')"
          style="width: 100%"
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.position.headcount')" prop="headcount">
        <ElInputNumber
          v-model="drawer.formData.headcount"
          :min="1"
          :placeholder="$t('common.placeholder.input')"
          style="width: 100%"
        />
      </ElFormItem>

      <ElFormItem :label="$t('common.table.sortOrder')" prop="sortOrder">
        <ElInputNumber
          v-model="drawer.formData.sortOrder"
          :min="1"
          :placeholder="$t('common.placeholder.input')"
          style="width: 100%"
        />
      </ElFormItem>

      <ElFormItem :label="$t('common.table.status')" prop="status">
        <ElRadioGroup v-model="drawer.formData.status">
          <ElRadioButton v-for="item in statusList" :key="item.value" :value="item.value">
            {{ item.label }}
          </ElRadioButton>
        </ElRadioGroup>
      </ElFormItem>

      <!-- 其他信息 -->
      <ElDivider content-position="left">{{ $t("common.section.other") }}</ElDivider>

      <ElFormItem :label="$t('pages.position.description')" prop="description">
        <ElInput
          v-model="drawer.formData.description"
          type="textarea"
          :rows="3"
          :placeholder="$t('common.placeholder.input')"
        />
      </ElFormItem>

      <ElFormItem :label="$t('common.table.remark')" prop="remark">
        <ElInput
          v-model="drawer.formData.remark"
          type="textarea"
          :rows="3"
          :placeholder="$t('common.placeholder.input')"
        />
      </ElFormItem>
    </ElForm>

    <template #footer>
      <div class="drawer-footer">
        <ElButton @click="drawer.guard.requestClose(drawer.close)">{{ $t("common.button.cancel") }}</ElButton>
        <ElButton
          type="primary"
          :loading="drawer.submitLoading.value"
          @click="drawer.handleSubmit(formRef, () => emit('success'))"
        >
          {{ $t("common.button.confirm") }}
        </ElButton>
      </div>
    </template>
  </ProModal>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import ProModal from "@/components/Pro/ProModal/index.vue";
import { useDrawerForm } from "@/components/Pro/composables/useDrawerForm";
import {
  useCreatePosition,
  useUpdatePosition,
  fetchListOrgUnits,
  positionTypeList,
  statusList,
} from "@/api/composables";
import { PaginationQuery } from "@go-wind/admin-core";
import { $t } from "@go-wind/admin-element";

const emit = defineEmits<{
  success: [];
}>();

const { mutateAsync: createPosition } = useCreatePosition();
const { mutateAsync: updatePosition } = useUpdatePosition();

const formRef = ref();
const orgUnitTreeData = ref<any[]>([]);

const drawer = useDrawerForm({
  moduleKey: "pages.position.moduleName",
  defaults: {
    name: "",
    code: "",
    type: "REGULAR",
    orgUnitId: undefined as number | undefined,
    headcount: 1,
    sortOrder: 1,
    status: "ON",
    description: "",
    remark: "",
  },
  createFn: createPosition,
  updateFn: (id, values) => updatePosition({ id, values }),
  asyncSetup: async () => {
    const result = await fetchListOrgUnits(new PaginationQuery({ formValues: { status: "ON" } }));
    orgUnitTreeData.value = result.items || [];
  },
});

// 表单验证规则
const formRules = {
  name: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  code: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  type: [{ required: true, message: $t("common.validation.selectRequired"), trigger: "change" }],
  orgUnitId: [
    { required: true, message: $t("common.validation.selectRequired"), trigger: "change" },
  ],
  headcount: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  sortOrder: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  status: [{ required: true, message: $t("common.validation.selectRequired"), trigger: "change" }],
};

// 包装 open：编辑模式下显式回填表单声明字段（避免 useDrawerForm 默认不回填导致编辑态空白）
function open(data?: { create?: boolean; row?: any }) {
  drawer.open(data, (row: any) => {
    if (!row) return;
    // 仅回填 defaults 声明的字段，不拷贝 id/createdAt 等不可变字段
    drawer.formData.name = row.name ?? "";
    drawer.formData.code = row.code ?? "";
    drawer.formData.type = row.type ?? "REGULAR";
    drawer.formData.orgUnitId = row.orgUnitId ?? undefined;
    drawer.formData.headcount = row.headcount ?? 1;
    drawer.formData.sortOrder = row.sortOrder ?? 1;
    drawer.formData.status = row.status ?? "ON";
    drawer.formData.description = row.description ?? "";
    drawer.formData.remark = row.remark ?? "";
  });
}

// 暴露方法给父组件
defineExpose({ open });
</script>

<style lang="scss" scoped>
.drawer-form {
  padding-right: 10px;
}

.drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
</style>
