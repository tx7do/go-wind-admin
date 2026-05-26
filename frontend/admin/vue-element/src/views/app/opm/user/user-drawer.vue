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

      <ElFormItem :label="$t('pages.user.table.username')" prop="username">
        <ElInput
          v-model="formData.username"
          :placeholder="$t('common.placeholder.input')"
          :disabled="!isCreate"
          clearable
        />
      </ElFormItem>

      <ElFormItem v-if="isCreate" :label="$t('pages.user.table.password')" prop="password">
        <ElInput
          v-model="formData.password"
          type="password"
          :placeholder="$t('common.placeholder.input')"
          show-password
          clearable
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.user.form.role')" prop="roleIds">
        <ElTreeSelect
          v-model="formData.roleIds"
          :data="roleTreeData"
          node-key="id"
          multiple
          check-strictly
          :render-after-expand="false"
          filterable
          clearable
          :props="{ label: 'name', value: 'id', children: 'children' } as any"
          :placeholder="$t('common.placeholder.select')"
          style="width: 100%"
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.user.form.orgUnit')" prop="orgUnitIds">
        <ElTreeSelect
          v-model="formData.orgUnitIds"
          :data="orgUnitTreeData"
          node-key="id"
          multiple
          check-strictly
          :render-after-expand="false"
          default-expand-all
          filterable
          clearable
          :props="{ label: 'name', value: 'id', children: 'children' } as any"
          :placeholder="$t('common.placeholder.select')"
          style="width: 100%"
          @change="handleOrgUnitChange"
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.user.form.position')" prop="positionIds">
        <ElTreeSelect
          v-model="formData.positionIds"
          :data="positionTreeData"
          node-key="id"
          multiple
          check-strictly
          :render-after-expand="false"
          filterable
          clearable
          :props="{ label: 'name', value: 'id', children: 'children' } as any"
          :placeholder="$t('common.placeholder.select')"
          style="width: 100%"
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.user.table.gender')" prop="gender">
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

      <ElFormItem :label="$t('pages.user.table.nickname')" prop="nickname">
        <ElInput
          v-model="formData.nickname"
          :placeholder="$t('common.placeholder.input')"
          clearable
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.user.table.realname')" prop="realname">
        <ElInput
          v-model="formData.realname"
          :placeholder="$t('common.placeholder.input')"
          clearable
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.user.table.email')" prop="email">
        <ElInput v-model="formData.email" :placeholder="$t('common.placeholder.input')" clearable />
      </ElFormItem>

      <ElFormItem :label="$t('pages.user.table.mobile')" prop="mobile">
        <ElInput
          v-model="formData.mobile"
          :placeholder="$t('common.placeholder.input')"
          clearable
        />
      </ElFormItem>

      <!-- 其他信息 -->
      <ElDivider content-position="left">{{ $t("common.section.other") }}</ElDivider>

      <ElFormItem :label="$t('common.table.status')" prop="status">
        <ElRadioGroup v-model="formData.status">
          <ElRadioButton v-for="item in userStatusList" :key="item.value" :value="item.value">
            {{ item.label }}
          </ElRadioButton>
        </ElRadioGroup>
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
  genderList,
  userStatusList,
  useCreateUser,
  useUpdateUser,
  fetchListRoles,
  fetchListOrgUnits,
  fetchListPositions,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/i18n";

const emit = defineEmits<{
  success: [];
}>();

const { mutateAsync: createUser } = useCreateUser();
const { mutateAsync: updateUser } = useUpdateUser();

const visible = ref(false);
const submitLoading = ref(false);
const isCreate = ref(true);
const currentId = ref<number>();
const formRef = ref();

// 表单数据
const formData = reactive({
  username: "",
  password: "",
  roleIds: [] as number[],
  orgUnitIds: [] as number[],
  positionIds: [] as number[],
  gender: "SECRET",
  nickname: "",
  realname: "",
  email: "",
  mobile: "",
  status: "NORMAL",
  remark: "",
});

// 表单验证规则
const formRules = {
  username: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  password: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  nickname: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  email: [
    { required: true, message: $t("common.validation.required"), trigger: "blur" },
    { type: "email", message: $t("common.validation.email"), trigger: "blur" },
  ],
  gender: [{ required: true, message: $t("common.validation.selectRequired"), trigger: "change" }],
  status: [{ required: true, message: $t("common.validation.selectRequired"), trigger: "change" }],
};

// 标题
const title = computed(() =>
  isCreate.value
    ? $t("common.modal.create", { moduleName: $t("pages.user.moduleName") })
    : $t("common.modal.update", { moduleName: $t("pages.user.moduleName") })
);

// 树形数据
const roleTreeData = ref<any[]>([]);
const orgUnitTreeData = ref<any[]>([]);
const positionTreeData = ref<any[]>([]);

// 加载角色树
async function loadRoleTree() {
  try {
    const result = await fetchListRoles(
      new PaginationQuery({ formValues: { status: "ON", type__not: "TEMPLATE" } })
    );
    roleTreeData.value = result.items || [];
  } catch (error) {
    console.error("Failed to load role tree:", error);
  }
}

// 加载组织树
async function loadOrgUnitTree() {
  try {
    const result = await fetchListOrgUnits(new PaginationQuery({ formValues: { status: "ON" } }));
    orgUnitTreeData.value = result.items || [];
  } catch (error) {
    console.error("Failed to load org unit tree:", error);
  }
}

// 加载职位树
async function loadPositionTree() {
  try {
    const result = await fetchListPositions(new PaginationQuery({ formValues: { status: "ON" } }));
    positionTreeData.value = result.items || [];
  } catch (error) {
    console.error("Failed to load position tree:", error);
  }
}

// 组织单元变化
function handleOrgUnitChange(value: any) {
  if (!value || value.length === 0) {
    formData.positionIds = [];
  }
}

// 重置表单
function resetForm() {
  Object.assign(formData, {
    username: "",
    password: "",
    roleIds: [],
    orgUnitIds: [],
    positionIds: [],
    gender: "SECRET",
    nickname: "",
    realname: "",
    email: "",
    mobile: "",
    status: "NORMAL",
    remark: "",
  });
  formRef.value?.clearValidate();
}

// 打开抽屉
async function open(data?: { create: boolean; row?: any }) {
  visible.value = true;
  isCreate.value = data?.create ?? true;
  currentId.value = data?.row?.id;

  // 重置表单
  resetForm();

  // 加载树形数据
  await Promise.all([loadRoleTree(), loadOrgUnitTree(), loadPositionTree()]);

  // 编辑时填充数据
  if (data?.row && !isCreate.value) {
    Object.assign(formData, {
      username: data.row.username || "",
      password: "",
      roleIds: data.row.roleIds || [],
      orgUnitIds: data.row.orgUnitIds || [],
      positionIds: data.row.positionIds || [],
      gender: data.row.gender || "SECRET",
      nickname: data.row.nickname || "",
      realname: data.row.realname || "",
      email: data.row.email || "",
      mobile: data.row.mobile || "",
      status: data.row.status || "NORMAL",
      remark: data.row.remark || "",
    });
  }
}

// 关闭抽屉
function handleClose() {
  visible.value = false;
  resetForm();
}

// 提交表单
async function handleSubmit() {
  if (!formRef.value) return;

  try {
    await formRef.value.validate();
    submitLoading.value = true;

    const submitData = {
      ...formData,
      password: isCreate.value ? formData.password : undefined,
    };

    if (isCreate.value) {
      await createUser({ data: submitData as any, password: formData.password });
      ElMessage.success($t("common.notification.create_success"));
    } else {
      await updateUser({ id: currentId.value!, values: submitData });
      ElMessage.success($t("common.notification.update_success"));
    }

    handleClose();
    emit("success");
  } catch (error) {
    if (error !== false) {
      ElMessage.error(
        isCreate.value
          ? $t("common.notification.create_failed")
          : $t("common.notification.update_failed")
      );
    }
  } finally {
    submitLoading.value = false;
  }
}

// 暴露方法
defineExpose({
  open,
});
</script>

<style lang="scss" scoped>
.drawer-form {
  padding-right: 12px;
}

.drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>
