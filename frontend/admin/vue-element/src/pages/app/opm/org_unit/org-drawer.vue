<template>
  <ProModal
    v-model:visible="drawer.visible.value"
    :title="drawer.title.value"
    :loading="drawer.pageLoading.value"
    :config="{
      component: 'drawer',
      drawer: { size: drawer.drawerWidth, closeOnClickModal: false },
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

      <ElFormItem :label="$t('pages.org_unit.name')" prop="name">
        <ElInput v-model="drawer.formData.name" :placeholder="$t('common.placeholder.input')" clearable />
      </ElFormItem>

      <ElFormItem :label="$t('pages.org_unit.code')" prop="code">
        <ElInput v-model="drawer.formData.code" :placeholder="$t('common.placeholder.input')" clearable />
      </ElFormItem>

      <ElFormItem :label="$t('pages.org_unit.parentId')" prop="parentId">
        <ElTreeSelect
          v-model="drawer.formData.parentId"
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

      <ElFormItem :label="$t('pages.org_unit.leaderId')" prop="leaderId">
        <ElSelect
          v-model="drawer.formData.leaderId"
          :placeholder="$t('common.placeholder.select')"
          filterable
          clearable
          style="width: 100%"
        >
          <ElOption
            v-for="user in userList"
            :key="user.id"
            :label="user.nickname"
            :value="user.id"
          />
        </ElSelect>
      </ElFormItem>

      <ElFormItem :label="$t('pages.org_unit.type')" prop="type">
        <ElSelect
          v-model="drawer.formData.type"
          :placeholder="$t('common.placeholder.select')"
          filterable
          clearable
          style="width: 100%"
        >
          <ElOption
            v-for="item in orgUnitTypeList"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </ElSelect>
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
          <ElRadioButton v-for="item in orgUnitStatusList" :key="item.value" :value="item.value">
            {{ item.label }}
          </ElRadioButton>
        </ElRadioGroup>
      </ElFormItem>

      <!-- 法人实体信息 -->
      <ElDivider content-position="left">{{ $t("pages.org_unit.legalEntity") }}</ElDivider>

      <ElFormItem :label="$t('pages.org_unit.isLegalEntity')" prop="isLegalEntity">
        <ElSwitch v-model="drawer.formData.isLegalEntity" />
      </ElFormItem>

      <ElFormItem
        v-if="drawer.formData.isLegalEntity"
        :label="$t('pages.org_unit.registrationNumber')"
        prop="registrationNumber"
      >
        <ElInput
          v-model="drawer.formData.registrationNumber"
          :placeholder="$t('common.placeholder.input')"
          clearable
        />
      </ElFormItem>

      <ElFormItem v-if="drawer.formData.isLegalEntity" :label="$t('pages.org_unit.taxId')" prop="taxId">
        <ElInput v-model="drawer.formData.taxId" :placeholder="$t('common.placeholder.input')" clearable />
      </ElFormItem>

      <ElFormItem
        v-if="drawer.formData.isLegalEntity"
        :label="$t('pages.org_unit.address')"
        prop="address"
      >
        <ElInput
          v-model="drawer.formData.address"
          :placeholder="$t('common.placeholder.input')"
          clearable
        />
      </ElFormItem>

      <!-- 其他信息 -->
      <ElDivider content-position="left">{{ $t("common.section.other") }}</ElDivider>

      <ElFormItem :label="$t('pages.org_unit.description')" prop="description">
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
        <ElButton @click="drawer.close">{{ $t("common.button.cancel") }}</ElButton>
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
  fetchListOrgUnits,
  useCreateOrgUnit,
  useUpdateOrgUnit,
  fetchListUsers,
  orgUnitTypeList,
  orgUnitStatusList,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/core/i18n";

const emit = defineEmits<{
  success: [];
}>();

const { mutateAsync: createOrgUnit } = useCreateOrgUnit();
const { mutateAsync: updateOrgUnit } = useUpdateOrgUnit();

const formRef = ref();
const orgUnitTreeData = ref<any[]>([]);
const userList = ref<any[]>([]);

const drawer = useDrawerForm({
  moduleKey: "pages.org_unit.moduleName",
  defaults: {
    name: "",
    code: "",
    parentId: undefined as number | undefined,
    leaderId: undefined as number | undefined,
    type: "",
    sortOrder: 1,
    status: "ON",
    isLegalEntity: false,
    registrationNumber: "",
    taxId: "",
    address: "",
    description: "",
    remark: "",
  },
  createFn: createOrgUnit,
  updateFn: (id, values) => updateOrgUnit({ id, values }),
  asyncSetup: async () => {
    const [orgResult, userResult] = await Promise.all([
      fetchListOrgUnits(new PaginationQuery({ formValues: { status: "ON" } })),
      fetchListUsers(new PaginationQuery({})),
    ]);
    orgUnitTreeData.value = orgResult.items || [];
    userList.value = userResult.items || [];
  },
});

// 表单验证规则
const formRules = {
  name: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  code: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  type: [{ required: true, message: $t("common.validation.selectRequired"), trigger: "change" }],
  sortOrder: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  status: [{ required: true, message: $t("common.validation.selectRequired"), trigger: "change" }],
};

// 暴露方法给父组件
defineExpose({ open: drawer.open });
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
