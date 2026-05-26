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
    <ElForm
      ref="formRef"
      :model="formData"
      :rules="formRules"
      label-width="120px"
      class="drawer-form"
    >
      <!-- 基本信息 -->
      <ElDivider content-position="left">{{ $t("common.section.basic") }}</ElDivider>

      <ElFormItem :label="$t('pages.org_unit.name')" prop="name">
        <ElInput v-model="formData.name" :placeholder="$t('common.placeholder.input')" clearable />
      </ElFormItem>

      <ElFormItem :label="$t('pages.org_unit.code')" prop="code">
        <ElInput v-model="formData.code" :placeholder="$t('common.placeholder.input')" clearable />
      </ElFormItem>

      <ElFormItem :label="$t('pages.org_unit.parentId')" prop="parentId">
        <ElTreeSelect
          v-model="formData.parentId"
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
          v-model="formData.leaderId"
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
          v-model="formData.type"
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
          v-model="formData.sortOrder"
          :min="1"
          :placeholder="$t('common.placeholder.input')"
          style="width: 100%"
        />
      </ElFormItem>

      <ElFormItem :label="$t('common.table.status')" prop="status">
        <ElRadioGroup v-model="formData.status">
          <ElRadioButton v-for="item in orgUnitStatusList" :key="item.value" :value="item.value">
            {{ item.label }}
          </ElRadioButton>
        </ElRadioGroup>
      </ElFormItem>

      <!-- 法人实体信息 -->
      <ElDivider content-position="left">{{ $t("pages.org_unit.legalEntity") }}</ElDivider>

      <ElFormItem :label="$t('pages.org_unit.isLegalEntity')" prop="isLegalEntity">
        <ElSwitch v-model="formData.isLegalEntity" />
      </ElFormItem>

      <ElFormItem
        v-if="formData.isLegalEntity"
        :label="$t('pages.org_unit.registrationNumber')"
        prop="registrationNumber"
      >
        <ElInput
          v-model="formData.registrationNumber"
          :placeholder="$t('common.placeholder.input')"
          clearable
        />
      </ElFormItem>

      <ElFormItem v-if="formData.isLegalEntity" :label="$t('pages.org_unit.taxId')" prop="taxId">
        <ElInput v-model="formData.taxId" :placeholder="$t('common.placeholder.input')" clearable />
      </ElFormItem>

      <ElFormItem
        v-if="formData.isLegalEntity"
        :label="$t('pages.org_unit.address')"
        prop="address"
      >
        <ElInput
          v-model="formData.address"
          :placeholder="$t('common.placeholder.input')"
          clearable
        />
      </ElFormItem>

      <!-- 其他信息 -->
      <ElDivider content-position="left">{{ $t("common.section.other") }}</ElDivider>

      <ElFormItem :label="$t('pages.org_unit.description')" prop="description">
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
        <ElButton type="primary" :loading="submitLoading" @click="handleSubmit">
          {{ $t("common.button.confirm") }}
        </ElButton>
      </div>
    </template>
  </ElDrawer>
</template>

<script lang="ts" setup>
import { computed, reactive, ref } from "vue";
import { ElMessage } from "element-plus";

import {
  fetchListOrgUnits,
  useCreateOrgUnit,
  useUpdateOrgUnit,
  fetchListUsers,
  orgUnitTypeList,
  orgUnitStatusList,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/i18n";

const emit = defineEmits<{
  success: [];
}>();

const { mutateAsync: createOrgUnit } = useCreateOrgUnit();
const { mutateAsync: updateOrgUnit } = useUpdateOrgUnit();

const visible = ref(false);
const submitLoading = ref(false);
const isCreate = ref(true);
const currentId = ref<number>();
const formRef = ref();

// 表单数据
const formData = reactive({
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
});

// 表单验证规则
const formRules = {
  name: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  code: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  type: [{ required: true, message: $t("common.validation.selectRequired"), trigger: "change" }],
  sortOrder: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  status: [{ required: true, message: $t("common.validation.selectRequired"), trigger: "change" }],
};

// 标题
const title = computed(() =>
  isCreate.value
    ? $t("common.modal.create", { moduleName: $t("pages.org_unit.moduleName") })
    : $t("common.modal.update", { moduleName: $t("pages.org_unit.moduleName") })
);

// 组织树数据
const orgUnitTreeData = ref<any[]>([]);

// 用户列表
const userList = ref<any[]>([]);

// 加载组织树
async function loadOrgUnitTree() {
  try {
    const result = await fetchListOrgUnits(new PaginationQuery({ formValues: { status: "ON" } }));
    orgUnitTreeData.value = result.items || [];
  } catch (error) {
    console.error("Failed to load org unit tree:", error);
  }
}

// 加载用户列表
async function loadUserList() {
  try {
    const result = await fetchListUsers(new PaginationQuery({}));
    userList.value = result.items || [];
  } catch (error) {
    console.error("Failed to load user list:", error);
  }
}

// 打开抽屉
async function open(data?: { create: boolean; row?: any }) {
  visible.value = true;
  isCreate.value = data?.create ?? true;
  currentId.value = data?.row?.id;

  // 重置表单
  resetForm();

  // 加载数据
  await Promise.all([loadOrgUnitTree(), loadUserList()]);

  // 如果是编辑模式，填充数据
  if (!isCreate.value && data?.row) {
    Object.assign(formData, data.row);
  }
}

// 关闭抽屉
function handleClose() {
  visible.value = false;
  resetForm();
}

// 重置表单
function resetForm() {
  formData.name = "";
  formData.code = "";
  formData.parentId = undefined;
  formData.leaderId = undefined;
  formData.type = "";
  formData.sortOrder = 1;
  formData.status = "ON";
  formData.isLegalEntity = false;
  formData.registrationNumber = "";
  formData.taxId = "";
  formData.address = "";
  formData.description = "";
  formData.remark = "";

  formRef.value?.clearValidate();
}

// 提交表单
async function handleSubmit() {
  if (!formRef.value) return;

  try {
    await formRef.value.validate();
    submitLoading.value = true;

    const values = { ...formData };

    if (isCreate.value) {
      await createOrgUnit(values);
      ElMessage.success($t("common.notification.createSuccess"));
    } else {
      await updateOrgUnit({ id: currentId.value!, values });
      ElMessage.success($t("common.notification.updateSuccess"));
    }

    emit("success");
    handleClose();
  } catch (error) {
    if (error !== false) {
      // 不是验证错误
      ElMessage.error(
        isCreate.value
          ? $t("common.notification.createFailed")
          : $t("common.notification.updateFailed")
      );
    }
  } finally {
    submitLoading.value = false;
  }
}

// 暴露方法给父组件
defineExpose({
  open,
});
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
